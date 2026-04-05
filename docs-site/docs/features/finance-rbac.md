---
sidebar_position: 8
title: Finance, RBAC, and audit logging
---

# Finance, RBAC, and audit logging

RideSync adds a non-destructive finance ledger, user authentication with three roles, JWT enforcement in the API Gateway, and asynchronous audit logging via RabbitMQ.

## Roles

| Role | Auth | Trip APIs | Finance |
|------|------|-----------|---------|
| `customer` | Google ID token (`POST /api/auth/google`) | `/trip/*`, `POST /api/trips/book` | `GET /api/finance/me` (ledger), `GET /api/trips/history` (ride history) |
| `business` | Email/password (`POST /api/auth/login`) | Denied | `GET /api/finance/dashboard/*` |
| `admin` | Email/password | Denied | Dashboard + `GET /api/admin/system-logs` + user provisioning |

## HTTP routes (gateway)

**Public (no JWT):** `/health`, `/api/auth/login`, `/api/auth/google`, `/api/auth/forgot-password`, `/api/auth/reset-password`, `/webhook/stripe`, `/ws/*`.

**Authenticated:** All other routes require `Authorization: Bearer <jwt>`.

- `GET /api/finance/me` — customer; lists payment ledger rows from `platform-service` (`FinanceService` gRPC).
- `GET /api/trips/history` — customer; lists MongoDB trips where the user is the **rider** (`userID`) or **assigned driver** (`driver.id`), newest first (`TripService.ListMyTrips` gRPC). The web **Ride history** page (`/finance/me`) uses this endpoint.
- `GET /api/finance/dashboard/revenue|regions|categories` — business or admin; query params `from`, `to` (RFC3339) where supported.
- `GET /api/admin/system-logs` — admin; query `limit`, `before` (RFC3339).
- `POST /api/admin/users/business`, `POST /api/admin/users/admin` — admin only.

Trip `userID` in JSON is ignored for identity: the gateway overwrites it with the JWT `sub`.

## Environment variables

**API Gateway:** `JWT_SECRET`, `JWT_ISSUER` (default `ridesync-auth`), `JWT_AUDIENCE` (default `ridesync-gateway`), `PLATFORM_SERVICE_URL` (preferred; single gRPC endpoint for finance + auth). For backward compatibility, `FINANCE_SERVICE_URL` or `USER_AUTH_SERVICE_URL` are used if `PLATFORM_SERVICE_URL` is unset. Also `TRIP_SERVICE_URL`, `RABBITMQ_URI`. **Do not set `GOOGLE_CLIENT_ID` on the gateway**—it does not verify Google tokens; it only proxies `POST /api/auth/google` to **platform-service** over gRPC.

**platform-service** (combined finance ledger + user auth + audit): `DATABASE_URL`, `RABBITMQ_URI`, `SUPER_ADMIN_EMAIL`, `SUPER_ADMIN_PASSWORD`, **`GOOGLE_CLIENT_ID`** (required server-side for `idtoken.Validate`; must be the **same Web client ID** as `NEXT_PUBLIC_GOOGLE_CLIENT_ID` on the web app), `JWT_*` (signing), `PUBLIC_GATEWAY_URL` (simulated reset email logs).

**Listen address:** If `GRPC_ADDR` is set, it is used (e.g. `:9094` for Docker Compose). If unset and **`PORT`** is set (Railway, Render, Fly), the server listens on **`:{PORT}`**. Otherwise default is **`:9094`**. Point the API gateway at the same host and port the platform exposes (on Railway, use the service’s **private** hostname and port, often the same as `PORT`).

**`SQL_SCHEMA_PATH` (important):**

- **Docker image** (`infra/production/docker/platform-service.Dockerfile`): the schema file is copied to **`/root/001_schema.sql`** and the image sets `ENV SQL_SCHEMA_PATH=/root/001_schema.sql`. On Railway, Render, etc., **do not** set `SQL_SCHEMA_PATH` to `infra/sql/001_schema.sql`—that path exists only in the Git repo, not inside the container. Either **omit** `SQL_SCHEMA_PATH` (use the image default) or set it explicitly to **`/root/001_schema.sql`**.
- **Local / monorepo run from repo root** (no Docker): default is `infra/sql/001_schema.sql`; override only if you keep the file elsewhere.

## RabbitMQ

- `finance_payment_success` — bound to `payment.event.success` (same routing key as trip payment consumer); consumed by `platform-service`.
- `audit_logs` — bound to `audit.event.write`; API Gateway publishes mutating requests; `platform-service` persists rows to `audit_logs`.

## PostgreSQL

Schema: `infra/sql/001_schema.sql` in the repo (`users`, `password_reset_tokens`, `transactions`, `audit_logs`).

## gRPC

- `platform-service` — **one** gRPC server exposes both `FinanceService` and `UserAuthService` on port **9094** (compose DNS `platform-service:9094`). The gateway opens a single connection and uses both stubs.

Install codegen plugins:

```bash
go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.11
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.6.1
make generate-proto
```

## Web UI (`web/`)

Next.js routes:

- `/login` — Google (riders/drivers) and email/password (admin/business); forgot-password request; requires `NEXT_PUBLIC_GOOGLE_CLIENT_ID` for Google.
- `/finance/me` — customer transaction table (JWT).
- `/dashboard` — business/admin finance JSON panels (revenue, regions, categories).
- `/admin` — audit log JSON, create business/admin users.
- `/reset-password` — optional `?token=` query or paste token.

The home map flows require a **customer** JWT: sign in before **I Need a Ride** / **I Want to Drive**. Trip HTTP calls send `Authorization: Bearer`.
