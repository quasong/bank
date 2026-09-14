# The Bank

Go internet-banking demo. This is not a licensed bank and holds no real funds.

Phase one shipped authentication. This codebase also has the **deposit money path**: open an account, demo funding, customer-to-customer transfer, and activity from the journal.

## Architecture

Modular monolith on PostgreSQL. A `customer` is a person; an `account` is a deposit product. Signing in does not open an account.

SQL lives in `internal/db/queries` and `migrations`. Runtime execution is pgx in `internal/db/store.go` and `internal/db/money.go`.

Customer deposits are bank liabilities. A transfer is a debit/credit pair in one transaction. Amounts are integer cents (`int64`), never `float64`.

```
cmd/server          HTTP entry (API + optional static UI)
internal/auth       register / login / refresh / logout
internal/customer   customer profile
internal/audit      audit action names
internal/account    demand-deposit accounts and funding
internal/ledger     journal validation and types
internal/transfer   customer-to-customer transfers
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

Deposits are ledger liabilities. Demo funding debits vault cash and credits the customer account. A transfer debits the sender and credits the destination in one transaction. Amounts are integer cents.

| Method | Path | Notes |
|------|------|------|
| POST | `/api/v1/accounts` | Open a demand-deposit account |
| GET | `/api/v1/accounts` | List mine |
| GET | `/api/v1/accounts/{id}` | Detail and cached balance |
| POST | `/api/v1/accounts/{id}/funding` | Demo inbound credit (`amount_cents`, `idempotency_key`) |
| POST | `/api/v1/transfers` | `{from_account_id, to_account_number, amount_cents, idempotency_key}` |
| GET | `/api/v1/accounts/{id}/activity` | Journal lines for that account |

Replay the same idempotency key to receive the original journal without moving money twice.

```bash
curl -s -X POST http://127.0.0.1:8080/api/v1/auth/register \
  -H 'Content-Type: application/json' \
  -d '{"email":"demo@thebank.test","password":"password1"}'
```

## Tests

```bash
make test
```
