# E-Commerce App

A small but production-minded online store: product catalog with CRUD, CSV import, search, and checkout with a fake payment provider.

| Part | Stack | Status |
|---|---|---|
| Backend | Go 1.27.3, Gin, GORM, PostgreSQL 18, Redis 8, Swagger | Catalog, search, CSV import, auth and roles, rate limiting, orders with a fake payment provider are done. Dockerfile, compose service and CI are done. |
| Frontend | Next.js 16 (App Router), React 19, TypeScript, Tailwind CSS 4, shadcn/ui | BFF with cookie sessions, sign-up, sign-in, account, catalog with search and filters, product page, cart, checkout and order history are done. Admin screens are in progress. |

Most of this document describes the backend; see [Frontend](#frontend) for the web app.

## Example CSV

The example file from the task was **downloaded on 2026-10-24** and is stored in the repository as [`example_data.csv`](example_data.csv), unchanged.

Before writing any code I went through the file row by row, because it is clearly designed to test edge cases. Line numbers below are file lines (header is line 1).

| Line(s) | What is special | How the import treats it |
|---|---|---|
| 20 | Name is `<script>alert('xss')</script>` | Stored as plain text. XSS protection belongs to the output layer: React escapes text, and the API never renders HTML. |
| 29 | Name is `Robert'); DROP TABLE products;--` | Stored as plain text. All queries are parameterized, so this is harmless. |
| 4 | Price `$29.99` | Rejected: `price must be a plain decimal number`. |
| 7 | Price `free` | Rejected. Silently turning a word into `0` would sell the product for free. |
| 47 | Price `0.00` | Accepted. Zero is a legitimate price (a promo item). |
| 16 | Stock `-5` | Rejected by domain validation. |
| 51 | Stock `0` | Accepted; the product shows as out of stock and cannot be bought. |
| 52 | Stock `99999`, empty category, weight `0` | Rejected because the category is empty. Weight `0` alone would be fine (digital good). |
| 25 | Empty name | Rejected. |
| 41 | Name made of spaces only | Rejected. Values are trimmed before validation. |
| 50 | Empty `weight_kg` | Rejected: weight is required. |
| 62–63 | `,,,,,,` | Counted as blank rows and skipped. |
| 2 / 36 | SKU `RS-001` twice with different data | Last row wins (36). Both lines are listed in the report. |
| 11 / 56 / 89 | SKU `BS-021` three times, 11 and 89 identical, 56 different | Last row wins (89). |
| 2, 36, 55 / 11, 56, 68, 89 | Same product name with different SKUs | Allowed. SKU is the identity, not the name. |
| 53, 59 | Comma and escaped quotes inside a quoted name | Handled by a real CSV parser (`encoding/csv`), not by splitting on commas. |
| 31, 36, 52 | `—` and `™` | File is read as UTF-8; a UTF-8 BOM is tolerated. |

Importing the file produces: 95 data rows, 2 blank rows, **85 products created**, 17 categories created, 7 rows rejected (lines 4, 7, 16, 25, 41, 50, 52), 2 duplicate SKUs reported. This exact result is asserted in the unit tests.

## Running locally

### Everything in Docker

Requirements: Docker with Compose.

```bash
cp example.env .env
docker compose up -d --build --wait
```

This builds the backend image, starts PostgreSQL and Redis, applies migrations, then starts the API once the database is migrated; `--wait` returns when every service is healthy. The demo catalog from `example_data.csv` is imported on the first start (the file is mounted read-only). Stop with `docker compose down` (add `-v` to wipe the data). The API listens on the same port as `make run`, so run only one of them.

The image (`backend/Dockerfile`):
- Multi-stage: `golang:1.27.3-alpine` runs `make swagger` (the spec is generated, never committed) and builds a static binary with `CGO_ENABLED=0 -trimpath`; Go module and build caches are BuildKit cache mounts.
- Runtime is `alpine` with an unprivileged user `app` (uid 10001), about 50 MB. In compose the container also runs with a read-only root filesystem and `no-new-privileges`.
- Healthcheck: BusyBox `wget` calls `/readyz` (PostgreSQL and Redis reachable); a 503 makes it fail. It is used both by the image `HEALTHCHECK` and by compose.
- Alternative considered: `distroless/static` (no shell, about 5 MB smaller). It has no `wget` or `curl`, so the healthcheck would need extra Go code (a `healthcheck` subcommand or a separate binary) only for Docker; Alpine keeps the application code free of it.
- Graceful shutdown: compose waits 20 s (`stop_grace_period`), longer than the server's 15 s shutdown timeout, so in-flight requests, including a checkout waiting for payment, finish.
- In compose `APP_ENV` defaults to `production` (JSON logs, Gin release mode).

### Go on the host

Requirements: Go 1.27.3 (with `GOTOOLCHAIN=auto` an older Go downloads it automatically), Docker with Compose, `make`.

```bash
cp example.env .env
cd backend
make infra-up
make swagger
make run
```

- `make infra-up` starts PostgreSQL and Redis and applies migrations (the one-shot `migrate` container).
- `make swagger` generates `backend/docs`. The folder is git-ignored and **must be generated before `go build`**, because the router imports it.
- `make run` reads `../.env`. On the first start with an empty catalog it imports `example_data.csv` (`SEED_DEMO_FILE`) and creates the superadmin from `SUPERADMIN_EMAIL` / `SUPERADMIN_PASSWORD`.

Then open:
- API: http://localhost:8080/api/v1
- Swagger UI: http://localhost:8080/swagger/index.html
- Health: http://localhost:8080/healthz (process is alive), http://localhost:8080/readyz (PostgreSQL and Redis reachable)

Reset the local database: `docker compose down -v`, then `make infra-up`.

### Ports

| Service | Host port | Why not the default |
|---|---|---|
| PostgreSQL | **5433** | 5432 is very likely taken by a locally installed PostgreSQL |
| Redis | **6380** | 6379 is very likely taken by a locally installed Redis |
| API | 8080 | `API_HOST_PORT` for the Docker container |

The host ports can be changed with `POSTGRES_HOST_PORT` and `REDIS_HOST_PORT` in `.env`.

### Make targets

| Target | What it does |
|---|---|
| `make infra-up` / `infra-down` | Start / stop PostgreSQL and Redis, apply migrations |
| `make migrate-up` | Apply migrations only |
| `make run`, `make build` | Run or build the API |
| `make test` | Unit tests with the race detector (no Docker needed) |
| `make test-integration` | Integration tests against real PostgreSQL in testcontainers (needs Docker) |
| `make lint` / `lint-fix` | golangci-lint v2 (pinned version, installed into `backend/bin`) |
| `make mocks` | Regenerate mocks with mockery |
| `make swagger` | Regenerate the OpenAPI spec |

### Configuration

| Variable | Default | Notes |
|---|---|---|
| `APP_ENV` | `development` | `production` switches Gin to release mode and logs to JSON |
| `HTTP_PORT` | `8080` | |
| `LOG_LEVEL` | `info` | `debug`, `info`, `warn`, `error` |
| `POSTGRES_HOST` / `PORT` / `USER` / `PASSWORD` / `DB` / `SSLMODE` | `localhost` / `5433` / `ecommerce` / `ecommerce` / `ecommerce` / `disable` | |
| `REDIS_ADDR` / `REDIS_PASSWORD` / `REDIS_DB` | `localhost:6380` / empty / `0` | |
| `JWT_SECRET` | **required**, at least 32 bytes | No default on purpose, so a weak secret can never reach production |
| `SUPERADMIN_EMAIL`, `SUPERADMIN_PASSWORD` | empty | Creates the first superadmin once; an existing account is never modified |
| `SEED_DEMO_FILE` | empty | CSV imported on start if the catalog is empty |
| `TRUSTED_PROXIES` | empty | Comma-separated IPs/CIDRs whose `X-Forwarded-For` is trusted. Must include the frontend BFF in Docker |

## Architecture

```
backend/
  cmd/api/                 entry point: load config, handle signals, run the app
  config/                  environment configuration (envconfig)
  migrations/              forward-only SQL migrations
  internal/
    app/                   wiring, HTTP server, graceful shutdown, demo seed
    domain/                entities, value rules, roles and permissions, domain errors
    usecase/               services + interfaces they depend on (repositories, hasher, tokens, parser)
    repository/postgres/   GORM models, mappers, repositories, transactor
    repository/redis/      refresh tokens, rate limiter
    security/              bcrypt, JWT, temporary password generator
    csvimport/             CSV parser (format only, no business rules)
    delivery/http/         router, handlers, DTOs, middleware, RFC 9457 responses
```

Dependency rule: `domain` depends on nothing; `usecase` depends only on `domain` and on interfaces it declares itself; `repository`, `security`, `csvimport` and `delivery` depend on `usecase`/`domain`, never on each other.

- **Interfaces live with the consumer.** Repository interfaces are declared in `usecase`, service interfaces in `delivery/http/handler`. Every implementation has a compile-time check (`var _ usecase.ProductRepository = (*ProductRepository)(nil)`).
- **Each layer has its own structs.** Handlers use DTOs (`delivery/http/dto`), repositories use GORM models (`repository/postgres/models.go`), and both map to domain structs explicitly. A JSON field rename or a column rename never leaks into the domain.
- **Validation in two places, by responsibility.** DTOs check the request shape (types, required fields, UUID format). Domain constructors (`domain.NewProduct`, `Product.Update`) check business rules. Database `CHECK` constraints repeat the critical rules as a last line of defense.
- **Transactions without leaking GORM.** `usecase.Transactor` runs a function in a transaction and passes it through `context`; repositories pick it up transparently. Services never see `*gorm.DB`.
- **Graceful shutdown.** On SIGINT/SIGTERM the server stops accepting connections, waits up to 15 s for in-flight requests, then closes PostgreSQL and Redis.

### Comments policy

The task asks to remove comments from AI-assisted code. The only comments left are machine-readable directives, not explanations:
- `swag` annotations on handlers (they *are* the Swagger spec);
- `//go:build integration` on integration test files.

All reasoning lives in this README instead.

## API

Full contract: Swagger UI at `/swagger/index.html`.

| Area | Endpoints | Access |
|---|---|---|
| Products | `GET /api/v1/products`, `GET /api/v1/products/{id}` | public |
| | `POST`, `PUT /{id}`, `DELETE /{id}` | ADMIN, SUPERADMIN |
| | `POST /api/v1/products/import` | ADMIN, SUPERADMIN |
| Categories | `GET /api/v1/categories`, `GET /{id}` | public |
| | `POST`, `PUT /{id}`, `DELETE /{id}` | ADMIN, SUPERADMIN |
| Auth | `POST /api/v1/auth/register`, `login`, `refresh`, `logout` | public |
| | `GET /api/v1/auth/me`, `POST /api/v1/auth/change-password` | any signed-in user, also with a temporary password |
| Admins | `GET/POST /api/v1/admins`, `POST /{id}/reset-password`, `/deactivate`, `/activate` | SUPERADMIN |
| Orders | `POST /api/v1/orders` (header `Idempotency-Key`) | USER |
| | `GET /api/v1/orders`, `GET /api/v1/orders/{id}` | USER (own orders), ADMIN and SUPERADMIN (all, filter by `user_id`, `status`) |
| Health | `GET /healthz`, `GET /readyz` | public |

### Error handling

Every error is `application/problem+json` (RFC 9457) with a stable machine-readable `code`, a `request_id` that also appears in the logs, and per-field `errors` when relevant:

```json
{
  "type": "about:blank",
  "title": "Validation failed",
  "status": 422,
  "code": "validation_failed",
  "detail": "request contains invalid fields",
  "instance": "/api/v1/products",
  "request_id": "0192f6b8-3c5e-7a51-9d3f-2b1c4e5f6a7b",
  "errors": [
    { "field": "price", "code": "invalid_format", "message": "price must be a decimal number" },
    { "field": "stock", "code": "negative", "message": "stock must not be negative" }
  ]
}
```

| Situation | Status |
|---|---|
| Malformed JSON, empty body, unknown field, wrong type, two JSON values, invalid UUID or query parameter | 400 |
| Missing or invalid token | 401 |
| Payment declined or failed (`order_id` points to the `FAILED` order) | 402 |
| Not allowed, or temporary password must be changed first | 403 |
| Not found (also unknown route) | 404 |
| Wrong method | 405 |
| SKU taken, category exists or in use, version conflict, cart prices or stock changed, checkout with this key still running | 409 |
| Body too large (JSON 1 MB, CSV 5 MB) | 413 |
| Wrong `Content-Type` | 415 |
| Business validation failed (all fields reported at once), `Idempotency-Key` reused with a different body | 422 |
| Rate limited (`Retry-After` set) | 429 |
| Unexpected failure | 500, generic text only; details go to the log with the `request_id` |

Principles: a bad request never looks like a server failure; all invalid fields are reported in one response instead of one by one; a missing value is never silently treated as zero (`price` missing or `null` is `required`, not a free product); unknown JSON fields are rejected so typos do not go unnoticed; expected conflicts (duplicate key) are not logged as SQL errors.

## Decisions and alternatives

### Single store, not a marketplace

The task speaks about "CRUD for Products" and "Purchase products" with no sellers, and the CSV has no owner column. SKU duplicates in the example only make sense if SKU is globally unique. So this is one store with staff and customers.

A marketplace would need `seller_id` on products, SKU unique per seller, splitting an order into per-seller sub-orders, payouts and seller moderation. It is a reasonable next step but a different product.

### Roles and permissions

| Role | Who | Can |
|---|---|---|
| `USER` | self-registered customer | browse, search, buy, see own orders |
| `ADMIN` | store staff, created by a superadmin | manage catalog and categories, import CSV, see all orders |
| `SUPERADMIN` | bootstrapped from env | everything an admin can, plus manage admins |

Middleware checks **permissions**, not roles (`auth.Require(domain.PermProductsWrite)`). The role-to-permission table lives in one place (`domain/role.go`), so adding a role like `SUPPORT` is one line and no handler changes.

Admins are onboarded with a **server-generated temporary password**: the superadmin creates the admin, gets a 16-character password once (`Cache-Control: no-store`), passes it to the person out of band. It is valid for 24 hours. With it, the access token only allows `/auth/me` and `/auth/change-password`; everything else returns `403 password_change_required`.
- Alternative considered: invitation link by email. Better (the superadmin never knows the secret and email ownership is verified), but needs SMTP; planned as a next step.
- Rejected: "pre-register an email and let that person sign up". Without email verification anyone who knows the address can claim an admin account.

Admin management only touches accounts with role `ADMIN`; targeting a `USER` or `SUPERADMIN` returns 404, so the endpoint does not reveal other roles.

### Authentication

- Access token: JWT HS256, 15 minutes. Refresh token: 32 random bytes, 7 days, stored in Redis **as a SHA-256 hash only**, rotated on every use with atomic `GETDEL`, so a reused refresh token is rejected.
- Password change, password reset and deactivation revoke all refresh tokens of the user (atomic Lua script).
- Passwords: bcrypt cost 12, 10 characters to 72 bytes. The upper limit exists because bcrypt silently ignores everything after 72 bytes. A password equal to the email is rejected.
- Login does not reveal whether an account exists: unknown email, wrong password and deactivated account return the same `invalid_credentials`, and an unknown email still runs a bcrypt comparison so response time is the same.
- JWT parsing pins the algorithm (no `alg: none`), checks issuer and requires expiration.
- Limitation: after deactivation or a role change an already issued access token stays valid for up to 15 minutes. Instant revocation would need a per-request session version check.

### Money, identifiers, SKU

- Prices are `NUMERIC(12,2)` in PostgreSQL and `shopspring/decimal` in Go, never `float`. In JSON money is a **string** (`"89.99"`); input accepts a number or a string.
- IDs are UUID v7: globally unique like v4 but time-ordered, so B-tree indexes do not fragment.
- SKU is trimmed and upper-cased, so `rs-001` and `RS-001` are the same product.
- Names may contain any text including HTML and SQL; control characters are rejected (newlines are allowed only in descriptions).

### Deleting and concurrent edits

- Products and categories are **soft-deleted**. Uniqueness of SKU and category slug is enforced by partial indexes `WHERE deleted_at IS NULL`, so a SKU can be reused after deletion. Order items store a snapshot of name, SKU and price, so history is not affected by later edits.
- A category with active products cannot be deleted (`409 category_in_use`). The race "delete a category while a product is being created in it" is closed with row locks: deletion takes `FOR UPDATE` on the category, product create/update takes `FOR SHARE`. A foreign key alone does not help because a soft-deleted category still exists physically. An integration test runs 30 such races and checks that no active product ends up in a deleted category.
- Product updates use **optimistic locking**: `PUT` must carry the `version` the client last saw; `UPDATE ... WHERE id = ? AND version = ?` affects zero rows if someone else saved first, and the client gets `409 version_conflict` instead of silently overwriting. Tested with 10 concurrent writers: exactly one wins.

### Checkout and payment

`POST /api/v1/orders` with the cart, the unit prices the customer saw, and a card. Only `USER` accounts buy; staff accounts get 403.

```json
{
  "items": [{ "product_id": "0192f6b8-...", "quantity": 2, "unit_price": "89.99" }],
  "payment": { "card_number": "4242 4242 4242 4242", "exp_month": 12, "exp_year": 2030, "cvc": "123" }
}
```

Flow:
1. **Validate** everything at once: 1 to 50 lines, quantity 1 to 100, each product once, prices with at most 2 decimals, card number (12 to 19 digits, Luhn), expiry not in the past, CVC 3 or 4 digits.
2. **Reserve** in one transaction: lock the product rows `FOR UPDATE` **in id order** (two carts with the same products in a different order cannot deadlock), compare stock and prices, decrement with `UPDATE ... SET stock = stock - n WHERE stock >= n` (a second guard on top of the lock), insert the order as `PENDING` with item snapshots. Decrementing stock also bumps the product `version`, so an admin edit based on the old stock fails with `409 version_conflict` instead of overwriting a purchase.
3. **Charge** the payment provider **outside** the transaction, so row locks are never held during a network call; 10 second timeout. The order id is the provider idempotency key. If the client disconnects after the reservation the payment still runs to the end (`context.WithoutCancel`).
4. **Finalize** in a second transaction: `PAID` with the payment id, or `FAILED` with a reason and the stock returned. The status update is guarded with `WHERE status = 'PENDING'`.

Prices: the client sends the price it showed; if any differs, nothing is reserved or charged and the response is `409 price_changed` with the current price per item. Missing, deleted, sold out (`stock = 0`) or short items give `409 items_unavailable`. All problems are reported in one response in `items`:

```json
{ "status": 409, "code": "price_changed",
  "items": [{ "product_id": "0192f6b8-...", "code": "price_changed", "requested_quantity": 2,
              "available_quantity": 5, "expected_price": "89.99", "current_price": "94.50" }] }
```

Idempotency (`Idempotency-Key` header, required):
- The key is unique per user in the database (`UNIQUE (user_id, idempotency_key)`), together with a SHA-256 fingerprint of the request (lines, quantities, prices, card last 4 digits).
- Same key and same body: the stored result is returned with `Idempotent-Replayed: true`, nothing is charged again. Same key, different body: `422 idempotency_key_reused`. First request still running: `409 checkout_in_progress`.
- A Redis lock (`SET NX` with a 30 second TTL, released by a Lua compare-and-delete so an expired holder cannot release someone else's lock) answers concurrent duplicates fast. If Redis is down checkout keeps working: the unique index is the real guarantee, a racing duplicate rolls back its reservation and gets the stored result.
- A failed payment is final for its key; retry with a new key.

Fake provider (`internal/payment`), deterministic by card number, behind the `PaymentProvider` interface:

| Card | Result |
|---|---|
| `4242 4242 4242 4242` (and any other valid number) | approved |
| ends with `0002` | `402 card_declined` |
| ends with `9995` | `402 insufficient_funds` |
| ends with `0119` | `402 payment_timeout` |

Card data is never stored or logged: the order keeps only the last 4 digits.

Alternatives considered:
- Charging inside the reservation transaction: simpler, but holds row locks for the whole provider call and blocks every buyer of the same products.
- Optimistic stock check (`version`) instead of row locks: under a sale on one hot product most buyers would retry; a short `FOR UPDATE` serializes them cheaply.
- Silently charging the new price: the customer must confirm any change, so the API refuses and returns the new prices.

### Search

`GET /api/v1/products?q=...` with filters `category_id`, `min_price`, `max_price`, `in_stock` and `sort` (`relevance`, `newest`, `price_asc`, `price_desc`, `name_asc`).

It is built on PostgreSQL itself, in two stages:
1. **Exact stage**: full-text search over a generated `tsvector` column (name weight A, description weight C, English stemming, GIN index) with `websearch_to_tsquery`, so users get `"phrases"`, `-exclusions` and `OR`. Plus exact and prefix match on SKU (`RS-0` finds `RS-001`); an exact SKU always ranks first.
2. **Fuzzy fallback**, only when the exact stage finds nothing: trigram word similarity (`pg_trgm`, GIN index) between each query word and the product name. It catches typos and partial words: `blutooth` finds Bluetooth Speaker, `waterprof tent` finds Camping Tent, `cofee beens` finds Organic Coffee Beans.

Why two stages: the first version combined everything with `OR`, and testing on the example data showed that trigram matches broke the operators (`speaker -portable` still returned the portable product). With a fallback, precise queries stay precise and typos still find something.

Other details: LIKE wildcards in the query are escaped (`RS_0` and `%` are literal); results are ordered by rank and then by creation date and id, so pagination is stable for equal ranks; filters and paging errors are reported together.

Alternatives considered:
- `ILIKE '%term%'`: no ranking, no typo tolerance, no stemming, and it cannot use a B-tree index.
- Elasticsearch / OpenSearch / Meilisearch: better relevance tuning, synonyms, facets. But it is one more service to run and keep in sync with the database (outbox or CDC), which is not justified for a catalog of this size. The repository hides the implementation behind `ProductRepository.List`, so it can be swapped later.

Known limitations: the category name is not part of the full-text index (a generated column cannot read another table); use the `category_id` filter. A query made only of stop words (`the`) falls through to the fuzzy stage and may return loosely related names.

**Not implemented: Redis cache for search.** With ~100 products PostgreSQL answers in a few milliseconds, so a cache would add complexity without measurable benefit. If load grows: cache by a hash of normalized query parameters with a short TTL, invalidate by bumping a catalog version key (`INCR`) on every product change instead of scanning and deleting keys; stock is shown only as "in stock" and the real check happens atomically at checkout, so a stale cache can never oversell.

### CSV import

`POST /api/v1/products/import`, multipart field `file`. Parsing happens **on the backend**: the API must protect itself anyway (it can be called without the UI), the same domain rules apply to the form and the file, and upserts, atomicity and concurrency can only be guaranteed in a database transaction. The frontend only uploads the file and shows the report; a client-side parser (e.g. Papa Parse) would duplicate the rules.

Options:
- `mode=upsert` (default) updates existing SKUs, `skip` leaves them untouched, `fail` reports them as row errors.
- `dry_run=true` runs the whole import inside a transaction and rolls it back, so the preview is exact, including categories that would be created.
- `atomic=true` saves nothing if any row fails. Default is to save valid rows and report the rest.
- `create_missing_categories=true` creates unknown categories (matched by slug, so `Toys` and `toys` are one category). Categories are created only for rows that are otherwise valid.

Rules:
- Strict header: required columns must be present; unknown and duplicate columns are an error; case, surrounding spaces, column order and a UTF-8 BOM are tolerated.
- Strict numbers: only plain decimals (`29.99`). `$29.99`, `free`, `1,299.00`, `1e3` are rejected with the reason. Stock must be a whole number.
- Duplicate SKUs inside one file: the last row wins, all lines are listed in `duplicates`.
- Limits: 5 MB and 10 000 rows. The file is streamed, not loaded into memory.
- Line numbers match the file (header is line 1), and a quoted multi-line value is reported at the line where the record starts. A broken row does not stop the import.
- Response is `200` whenever the file itself could be processed; `committed` says whether anything was saved. Errors of the file or request (no file, not multipart, too large, bad header) are regular 4xx problems.
- Concurrency: two imports never run at the same time (`pg_advisory_xact_lock`), existing products are locked `FOR UPDATE`, categories `FOR SHARE`, new products are inserted in batches of 500.

### Migrations

Plain SQL files in `backend/migrations`, applied by the `migrate/migrate` container. GORM `AutoMigrate` is not used: it cannot express generated `tsvector` columns, GIN/trigram/partial indexes or `CHECK` constraints, and it changes the schema implicitly on startup.

Migrations are **forward-only** (no down files). In production a down migration that drops a column loses data written after the migration; mistakes are fixed with a new migration, and schema changes follow expand/contract so the previous app version keeps working on the new schema. Locally the database is reset with `docker compose down -v`; integration tests always start from a fresh container.

### Rate limiting

Sliding-window log in Redis (one atomic Lua script on a sorted set), per client IP:

| Policy | Routes | Limit | If Redis is down |
|---|---|---|---|
| `auth` | login, register, refresh, change-password | 10 / minute | **fail closed** (503): brute-force protection must not disappear |
| `api` | all `/api/v1` | 300 / minute | **fail open** with a warning: the catalog stays available |

Responses carry `RateLimit-Limit`, `RateLimit-Remaining`, `RateLimit-Reset`; a blocked request gets `429` with `Retry-After`. A sliding window avoids the fixed-window burst (10 attempts at 0:59 and 10 more at 1:00), and rejected requests are not recorded, so a client hammering while blocked is still released on time.

`X-Forwarded-For` is trusted **only** from `TRUSTED_PROXIES`. Gin trusts every proxy by default, which would let anyone bypass the limit with a forged header. When the Next.js frontend proxies API calls, its address must be listed there, otherwise all users share one limit.

Limitation: the limit is per IP. A distributed attack on one account from many IPs needs an additional per-email limit inside login; that is a next step.

## Frontend

### Running the frontend

Requirements: Node.js 22.12+ (`frontend/.nvmrc` pins 24) and npm 11 (npm 10 fails to install the dependency tree with `Cannot read properties of null (reading 'edgesOut')`). The backend must be running (see above).

```bash
cd frontend
cp example.env .env.local
npm install
npm run dev
```

Open http://localhost:3000. Sign in as the superadmin from `.env` or create a customer account.

| Script | What it does |
|---|---|
| `npm run dev` | Development server with hot reload |
| `npm run build`, `npm start` | Production build; `start` runs `server.mjs` (see Client IP) |
| `npm run lint` | ESLint (`eslint-config-next`) |
| `npm run typecheck` | Generates route types (`next typegen`) and runs `tsc` |
| `npm test` | Vitest unit tests |

| Variable | Default | Notes |
|---|---|---|
| `BACKEND_URL` | `http://localhost:8080` | Backend base URL, used only on the server |
| `APP_ORIGIN` | the request origin | Comma-separated origins allowed to send mutating requests (CSRF check) |
| `COOKIE_SECURE` | `true` in production | Set `false` to use the production build over plain HTTP on localhost |
| `TRUST_PROXY` | `false` | `true` only when a reverse proxy in front of the frontend sets `X-Forwarded-For` |

### Architecture

The browser talks only to Next.js; Next.js is a backend-for-frontend (BFF) and the only client of the Go API. The backend has no CORS on purpose, and tokens never reach JavaScript.

```
browser ──cookies──▶ Next.js (proxy.ts → pages / route handlers) ──Bearer + X-Forwarded-For──▶ Go API
```

```
frontend/
  server.mjs              production entry: Next.js request handler + trusted client IP
  src/proxy.ts            runs before every page and /api request: CSRF check, token refresh, auth redirects
  src/app/api/auth/*      login, register, logout, change-password: exchange credentials for httpOnly cookies
  src/app/api/[...path]   generic proxy to /api/v1/* with the access token from the cookie
  src/app/**/page.tsx     server components; read data with serverApi() directly from the backend
  src/lib/api             types from the API responses, browser client, RFC 9457 error mapping
  src/lib/session         cookie policy, refresh coordinator, origin check, route rules (pure, unit-tested)
  src/lib/catalog         URL query parsing and links for the catalog
  src/lib/cart            cart rules, localStorage parsing, revalidation against current products
  src/lib/checkout        card validation, idempotency attempts, applying 409 details to the cart
  src/lib/server          server-only helpers: backend calls, current user, BFF responses
  src/components/ui       shadcn/ui components (Radix primitives + Tailwind)
```

- **Reads** happen in server components (`serverApi`), so pages render with data and the token stays on the server. **Mutations** go from client components to `/api/*` route handlers.
- **Types** are written by hand from the backend DTOs (`src/lib/api/types.ts`). The backend publishes Swagger 2.0, and OpenAPI generators for TypeScript expect OpenAPI 3; a conversion step is a next step if the API grows.
- **UI**: shadcn/ui (components are copied into the repo, so they can be adjusted freely), lucide icons, sonner toasts, light and dark theme via `next-themes`. Forms validate with zod on the client using the same rules as the backend (password 10 characters to 72 bytes, not equal to the email), and server-side 422 errors are mapped back to the fields by `field`.
- **Errors**: every non-2xx response is parsed as RFC 9457 into an `ApiError`; messages are chosen by the stable `code`, with the backend `detail` as a fallback.
- **Cache Components** (enabled by default in the Next.js 16 template) are turned off: almost every page depends on the session cookie, so there is little to prerender, and per-user data must never be cached across users.

### Session, refresh and CSRF

- Login, register (register, then login with the same credentials) and change-password are handled by BFF route handlers that store the tokens in cookies and return only the user: `shop_access` (lives 30 s less than the 15-minute JWT), `shop_refresh` (until the refresh token expires, 7 days) and `shop_password_change` (set while an admin still has a temporary password). All are `HttpOnly`, `SameSite=Lax`, `Secure` in production, `Path=/`.
- `proxy.ts` refreshes the session **before** the request reaches a page or route handler: when the access cookie has expired and a refresh cookie exists, it calls `/auth/refresh`, writes the new cookies to the response and to the current request, so server components already see the new token. Server components cannot set cookies, so refreshing there would lose the rotated token.
- Refresh tokens rotate and a reused token is rejected, so parallel requests with the same expired session must not all refresh. The proxy keeps one in-flight refresh per token and reuses its result for 15 s, so requests that were already on the way with the old cookie get the same new session. If the backend is unreachable the result is not cached and the cookies are kept; a rejected token clears them.
- `/api/auth/refresh` is not exposed through the generic proxy, so refresh tokens never leave the server.
- A backend 401 on any proxied call clears the cookies. Protected pages (`/account`, `/orders`, `/checkout`, `/admin`) redirect guests to `/login?next=…`; `next` accepts only same-site relative paths. While `shop_password_change` is set every page redirects to `/account/password`, mirroring the backend's `403 password_change_required`.
- CSRF: cookies are `SameSite=Lax`, so other sites cannot send them with `POST`/`PUT`/`DELETE`, and the proxy additionally rejects every mutating `/api/*` request whose `Origin` is not in `APP_ORIGIN` (or, without `Origin`, whose `Sec-Fetch-Site` is not `same-origin`) with `403 invalid_origin`. A separate CSRF token is not needed: `SameSite` keeps the cookies off cross-site requests, and the origin check rejects whatever a browser still sends (for example from a sibling subdomain, which counts as same-site).
- Alternative considered: the access token in memory and the refresh token in a cookie (SPA pattern). It still needs a BFF-like refresh endpoint to keep the refresh token away from JavaScript, loses the session on reload until a refresh, and cannot render authenticated pages on the server.

### Client IP

The backend rate-limits by IP and trusts `X-Forwarded-For` only from `TRUSTED_PROXIES`. The BFF forwards `X-Forwarded-For`, so the frontend's address must be in `TRUSTED_PROXIES`; otherwise all users share one limit.

Next.js sets `X-Forwarded-For` only when the request does not already have one, so a client could send its own header and get a fresh rate-limit bucket on every request. `npm start` therefore runs `server.mjs`, a thin wrapper around the Next.js request handler that replaces `X-Forwarded-For` with the socket address and drops other `X-Forwarded-*` headers. Behind a reverse proxy that sets the header itself, set `TRUST_PROXY=true`. `npm run dev` uses the plain Next.js dev server.

### Screens

| Screen | Path | Status |
|---|---|---|
| Sign in, create account | `/login`, `/register` | Done |
| Account, change password (also forced for temporary admin passwords) | `/account`, `/account/password` | Done |
| Catalog with search, category, price range, in-stock filter, sorting and pagination | `/` | Done |
| Product page with quantity, stock state and more products from the category | `/products/[id]` | Done |
| Cart | `/cart` | Done |
| Checkout with test card shortcuts | `/checkout` | Done |
| Order history with status filter, order details | `/orders`, `/orders/[id]` | Done |
| Admin: products, categories, CSV import with dry-run preview, all orders | `/admin/*` | Next step |
| Superadmin: administrators | `/admin/admins` | Planned |

### Catalog and cart

- The catalog state lives in the URL (`/?q=lamp&category_id=…&in_stock=true&sort=price_asc&page=2`), so a search can be shared, bookmarked and restored with Back. The page is a server component; filters are client components that replace the URL, and the previous results stay visible (dimmed) until the new ones arrive. The search box waits 350 ms after typing; price limits apply on blur or Enter and are validated first (format, minimum not above maximum). Parameters from the URL are sanitised before they reach the backend, so a hand-edited link shows the catalog instead of an error.
- There are no product images in the data, so every card shows a category icon on a gradient derived from the category slug.
- The cart is kept in `localStorage` and shared by tabs (the `storage` event). Each line stores the price the buyer saw, because checkout sends it as `unit_price` and the backend rejects the order with `409 price_changed` if it differs. The cart enforces the backend limits up front: at most 50 lines, 1 to 100 items per line, never more than the known stock. Stored data is validated on read, so a corrupted or edited value cannot break the page.
- Opening the cart re-reads every product: changed prices are adopted, quantities are reduced to the available stock, sold-out and deleted products are removed, and the buyer sees a list of what changed. The stock can still change before payment, so checkout handles `409` again.
- Money is calculated in integer cents (`"0.10" * 3` is exactly `"0.30"`); amounts stay strings everywhere else, as in the API.
- Staff accounts cannot buy: the cart icon is hidden, add-to-cart buttons are disabled with an explanation and `/cart` redirects to the catalog. The cart is cleared on sign-out, so the next person on a shared computer does not inherit it.

### Checkout

- Only customers can check out: `/checkout` and `/orders` redirect staff, guests are sent to sign in and come back. The cart is revalidated on the page as on `/cart`, and the pay button waits for it.
- The card form formats the number and the expiry while typing and validates like the backend (12 to 19 digits, Luhn check, `MM/YY` not in the past and at most 20 years ahead, 3 or 4 digit CVC). Buttons fill in each fake-provider test card. Backend 422 errors for `payment.*` fields are shown under the matching inputs.
- **Idempotency key per attempt.** A key is a random UUID, created when the buyer submits and tied to a SHA-256 fingerprint of the request body. Submitting the same body again after an unknown outcome (network error, 5xx, `409 checkout_in_progress`, rate limit) reuses the key, so the backend replays the original result instead of charging twice. A final answer (201, 402, 409 with cart changes, 422) drops the key, and any change to the cart or card produces a new fingerprint and a new key, so the backend never sees one key with two bodies (`422 idempotency_key_reused`).
- The key lives in memory only, together with the fingerprint, because the body contains the card number. Reloading the page after a network error therefore starts a new attempt; the backend still cannot double-charge the same attempt, but a reload could create a second order if the first one had in fact succeeded. Keeping keys across reloads would need storing them per cart without card data.
- **`409 price_changed` / `items_unavailable`:** the `items` from the response are applied to the cart (new price, reduced quantity, removed products), the buyer sees the list of changes and confirms by paying again.
- **`402`:** the payment error is shown with a link to the `FAILED` order; the form keeps the cart and card so another card can be tried.
- **`201`:** the cart is cleared and the buyer lands on the order page with a confirmation.
- Order pages show status, items with the prices fixed at purchase, total, the last four card digits and the payment ID. Order numbers are the last 8 characters of the UUID v7: the first characters are a timestamp and would look almost the same for every order.

### Frontend tests

Vitest covers the logic where a mistake would break sessions or security, not the markup: the refresh coordinator (one backend call for parallel requests, reuse window, no caching of outages), cookie lifetimes and flags, the origin check, redirect rules and open-redirect protection, RFC 9457 error mapping, the password rules shared with the backend, money arithmetic, cart limits and merging, cart revalidation, tolerant parsing of stored carts, catalog URL parsing and pagination, card validation against the test cards, idempotency-key reuse rules, applying checkout 409 details and mapping payment field errors.

## Testing

```bash
make test                # unit tests, no Docker
make test-integration    # + PostgreSQL in testcontainers
```

The focus is business rules and the unusual cases, not line coverage:
- **Domain**: every validation rule and boundary (price scale, max values, SKU format, control characters, password byte limit, email edge cases), role-permission matrix, temporary password expiry.
- **Services** (mockery mocks): invalid input never reaches the repository; optimistic locking including the race between read and write; login does not reveal accounts; refresh rotation and revocation; superadmin bootstrap including two instances starting at once; admin management cannot touch other roles; search defaults and validation; CSV import on the real example file, all three modes, dry-run and atomic rollback, category creation, conflicts.
- **HTTP**: through the full router — every kind of bad request gets its own 4xx and never calls the service, internal error text never leaks, authentication and permissions for every write endpoint, upload limits.
- **Security**: forged, expired, `alg: none` and tampered JWTs; temporary password alphabet and uniqueness.
- **Rate limiter**: the Lua script on miniredis, including 50 concurrent requests against a limit of 10 (exactly 10 pass).
- **Checkout**: cart limits, duplicates and card rules; price and stock conflicts stop before any payment; every payment outcome (approved, declined, insufficient funds, timeout, provider error) and stock release on failure; payment survives a disconnected client; idempotent replay, reused key, request in progress, Redis outage falling back to the unique index; customers never see other customers' orders.
- **Integration** (real PostgreSQL 18): repositories, soft delete and SKU reuse, database constraints, 10 concurrent updates with the same version, 30 races between category delete and product create, search ranking, typos, operators and pagination stability; checkout end to end with the fake provider, and **20 buyers racing for the last item: exactly one order is paid, stock ends at 0**.

## CI

GitHub Actions, `.github/workflows/backend.yml`. Runs on pushes to `master` and on pull requests that touch `backend/`, `docker-compose.yml`, `example_data.csv` or the workflow itself; parallel runs of the same pull request are cancelled. Three jobs run one after another (`needs`), so a lint or build error stops the pipeline before the slower tests:

| Job | Steps |
|---|---|
| **build** | `make swagger`, `make lint` (linter errors are printed in the job log), `go build` |
| **Unit tests** | `make swagger`, `make test` (race detector) |
| **Integration tests** | `make swagger`, `make test-integration` (PostgreSQL 18 in testcontainers, Docker is available on the runner) |

CI uses the same `make` targets as a developer, so tool versions (golangci-lint, mockery, swag) are pinned in one place, the Makefile; the installed tools are cached by its hash. Go comes from `go.mod` and `GOTOOLCHAIN=local` forbids silently downloading another version. The workflow has read-only permissions and pushes no images; publishing to a registry is a next step together with deployment.

## Known limitations and next steps

- OpenTelemetry tracing: spans for HTTP, SQL, Redis and checkout steps (reserve, payment, finalize), `trace_id` in logs, Jaeger behind `docker compose --profile tracing`.
- A crash between reservation and payment leaves a `PENDING` order holding stock. A background job should expire old `PENDING` orders (there is a partial index for it) after checking the provider with the order id. With a real provider a timeout is ambiguous (the charge may have gone through), so the order would stay `PENDING` until reconciliation instead of failing immediately; the fake provider guarantees that a timeout charged nothing.
- One currency; no taxes, discounts or shipping in the total.
- Cart lives in the browser (`localStorage`); prices and stock are re-checked at checkout. If carts must follow the user across devices, move them to the database.
- Order cancellation and refunds (return stock as a compensating action).
- Shipping cost by `weight_kg` (the weight is stored and returned, but not used yet).
- Product images: upload through presigned URLs to S3/MinIO, multipart upload for large files.
- Email (SMTP) for password recovery, order notifications and admin invitations, sent through a transactional outbox so a mail outage never breaks a purchase.
- Audit log of admin actions (who changed a role, who deleted a product).
- Per-account login rate limit; instant access-token revocation.
- Search cache in Redis (see Search).
- The refresh coordinator in the BFF is in-memory, so it deduplicates refreshes only within one frontend instance; with several replicas use sticky sessions or a shared lock in Redis.
- i18n: the UI and API messages are English only; error `code`s are stable and can be translated on the client.
- Marketplace mode (see "Single store").
