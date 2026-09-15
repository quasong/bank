# The Bank

Go internet-banking demo. This is not a licensed bank and holds no real funds.

Phase one shipped authentication. This codebase also has the **deposit money path**: open an account, demo funding, customer-to-customer transfer, and activity from the journal.

## Architecture

Modular monolith on PostgreSQL. A customer identity is not an account. Signing in does not open an account. The first deposit is USD; you can then open EUR and GBP balances. Each currency has its own local details (ACH routing, UK sort code, or GB IBAN). Same-currency sends stay on the ledger; cross-currency conversion uses a live ECB rate from Frankfurter and posts a balanced FX journal per currency.

SQL lives in `internal/db/queries` and `migrations`. Runtime execution is pgx in `internal/db/store.go` and `internal/db/money.go`.

Customer deposits are bank liabilities. A transfer is a debit/credit pair in one transaction. Amounts are integer cents (`int64`), never `float64`.

```
cmd/server          HTTP entry (API + optional static UI)
internal/auth       register / login / refresh / logout
internal/customer   customer profile
internal/audit      audit action names
internal/account    demand-deposit accounts and funding
internal/currency   USD / EUR / GBP codes, local details, integer FX math
internal/ledger     journal validation and types
internal/transfer   same-currency customer-to-customer transfers
internal/fx         live quotes and conversion
internal/payee      saved destinations
internal/httpapi    router and middleware
web                 React console
```

## Run locally

Needs Go 1.26+, Node 20+, and PostgreSQL 16 (Docker or a local instance).

```bash
# Option A: Docker
make compose-up

# Option B: local PostgreSQL already on 5432
psql -c "CREATE ROLE bank LOGIN PASSWORD 'bank';" || true
psql -c "CREATE DATABASE bank OWNER bank;" || true
```

Then:

```bash
make run                 # API, and the UI if web/dist exists: http://127.0.0.1:8080
make web-install
make web-dev             # UI: http://127.0.0.1:5173 (/api proxied to the backend)
```

Single-port demo: `make web-build && make run`, then open http://127.0.0.1:8080.

See [.env.example](.env.example). `JWT_SECRET` must be at least 32 bytes. Local defaults are for demo only.

## Auth

| Method | Path | Notes |
|------|------|------|
| POST | `/api/v1/auth/register` | Register and sign in |
| POST | `/api/v1/auth/login` | Sign in |
| POST | `/api/v1/auth/refresh` | Rotate refresh via HttpOnly cookie |
| POST | `/api/v1/auth/logout` | Revoke refresh, clear cookie |
| GET | `/api/v1/me` | Current customer; `Authorization: Bearer` |
| GET | `/healthz` | Liveness |

- Password: Argon2id
- Access token: ~15 minute JWT, held in frontend memory
- Refresh token: ~7 days, stored as SHA-256, cookie `HttpOnly; SameSite=Lax; Path=/api/v1/auth`
- Failed login always returns "incorrect email or password"
- Login is rate-limited by IP + email
- Error body: `{"error":{"code":"...","message":"..."}}`

## Accounts and money

Deposits are ledger liabilities. Demo funding debits vault cash in that currency and credits the customer account. A transfer debits the sender and credits the destination in one transaction, and only works in the same currency. Amounts are integer minor units (`int64`), never `float64`.

USD is an ACH destination: routing `121000248` (fictional ABA with a valid checksum) plus an 8-digit DDA, stored as `121000248########`. GBP is sort code `04-00-04` plus those 8 digits. EUR is a GB IBAN `GB##THEB040004########` (ISO 13616 checksum) with BIC `THEBGB2L`. These identifiers mimic real formats; they are not issued by a real bank.

Cross-currency conversion withdraws from the source vault and funds the destination vault in one journal, balanced per currency. Live rates come from [Frankfurter](https://api.frankfurter.dev/v1) (ECB), cached for about a minute, and converted with `rate_e8` integer math (half-up). The destination pocket is opened automatically if needed.

| Method | Path | Notes |
|------|------|------|
| POST | `/api/v1/accounts` | Open a balance. Body `{currency?}` (`USD` default, or `EUR` / `GBP`) |
| GET | `/api/v1/accounts` | List mine |
| GET | `/api/v1/accounts/{id}` | Detail, local details, and cached balance |
| POST | `/api/v1/accounts/{id}/funding` | Demo inbound credit (`amount_cents`, `idempotency_key`) |
| POST | `/api/v1/accounts/{id}/withdrawals` | Demo outbound debit to that currency's vault |
| POST | `/api/v1/accounts/{id}/freeze` | Stop funding, withdrawals, transfers, and conversion |
| POST | `/api/v1/accounts/{id}/unfreeze` | Return a frozen account to active |
| POST | `/api/v1/accounts/{id}/close` | Close when `balance_cents` is 0; terminal |
| POST | `/api/v1/transfers` | Same-currency `{from_account_id, to_account_number, amount_cents, idempotency_key, payee_name?, note?}` |
| GET | `/api/v1/fx/quote` | Live quote `from`, `to`, optional `amount_cents` |
| POST | `/api/v1/fx` | Convert `{from_account_id, to_currency, amount_cents, idempotency_key}` |
| GET | `/api/v1/accounts/{id}/activity` | Journal lines; includes `receipt`, `note`, `counterparty_account_number`, `counterparty_name` |
| GET | `/api/v1/payees` | Saved destinations, most recently used first |
| POST | `/api/v1/payees` | Upsert `{account_number, display_name?}`. Destination must exist and not be yours |
| PATCH | `/api/v1/payees/{id}` | Rename. Empty name resets to the account number |
| DELETE | `/api/v1/payees/{id}` | Remove a saved destination |
| GET | `/api/v1/audit` | Your security and money events |

Replay the same idempotency key to receive the original journal without moving money twice. A successful transfer upserts a payee for the sender; a payee write failure does not fail the transfer. Optional `note` is stored on the journal (max 40 characters) and shown in activity. Saved people can be added, renamed, and removed without sending; a removed name no longer appears on new activity, and past journal lines stay unchanged. Destination numbers may include spaces or dashes; the API stores the canonical form.

Activity `receipt` is the last 8 hex digits of `journal_id`. Copy the full `journal_id` if you need the canonical id. Funding and withdrawals have no counterparty. FX activity shows your other currency pocket as the counterparty.

Funding, withdrawals, transfers, conversion, freeze, unfreeze, and close append to `audit_logs`. Idempotent replays and no-op status changes are not recorded again.

```bash
curl -s -X POST http://127.0.0.1:8080/api/v1/auth/register \
  -H 'Content-Type: application/json' \
  -d '{"email":"demo@thebank.test","password":"password1"}'
```

## Tests

```bash
make test
```
