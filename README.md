# The Bank

Go internet-banking demo. This is not a licensed bank and holds no real funds.

Phase one ships **customer authentication** and a web console you can sign in to. Accounts, transfers, and activity are placeholders; there is no ledger yet.

## Architecture

Modular monolith on PostgreSQL. A `customer` is a person; an `account` is a deposit product. Signing in does not open an account.

SQL lives in `internal/db/queries` and `migrations`. Runtime execution is pgx in `internal/db/store.go`.

Later transfers use double-entry bookkeeping: customer deposits are bank liabilities, a transfer is a debit/credit pair in one transaction, and amounts are integer cents (`int64`), never `float64`. Phase one creates auth tables only.

```
cmd/server          HTTP entry (API + optional static UI)
internal/auth       register / login / refresh / logout
internal/customer   customer profile
internal/audit      audit action names
internal/account    reserved: deposit accounts
internal/ledger     reserved: journal
internal/transfer   reserved: transfer orchestration
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

```bash
curl -s -X POST http://127.0.0.1:8080/api/v1/auth/register \
  -H 'Content-Type: application/json' \
  -d '{"email":"demo@thebank.test","password":"password1"}'
```

## Tests

```bash
make test
```

## Later: accounts and transfers

1. Add `accounts` and journal tables in a new goose migration. Do not store balances on `customers`.
2. Write journal lines in `internal/ledger`. `internal/account` only projects balances.
3. Orchestrate in `internal/transfer`: frozen/closed checks, idempotency key, one transaction.
4. Turn on the Accounts / Transfers / Activity pages. Still no fake data.
