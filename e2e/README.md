# End-to-end tests

Playwright tests that drive the whole stack from `docker compose` (Caddy → Next.js BFF → Go API → Postgres/Redis) through a real browser. Test data (categories, products, customers) is created through the Go API; everything else is done in the UI.

## Run

```bash
docker compose up -d --build --wait   # from the repository root
cd e2e
npm ci
npx playwright install chromium
npm test
```

Superadmin credentials are read from `SUPERADMIN_EMAIL` / `SUPERADMIN_PASSWORD` in the environment or from the root `.env`. `E2E_BASE_URL` (default `http://localhost:3000`) and `E2E_API_URL` (default `http://localhost:8080/api/v1`) point the suite elsewhere.

Auth endpoints allow 10 requests per minute per client IP and every request from your machine shares one IP, so the suite runs on one worker and makes about 8 auth calls. Wait a minute before running it again, otherwise logins fail with 429.

## Scenarios

- **catalog**: search finds a product and opens its page, empty search result, protected pages redirect guests to sign in
- **auth**: wrong password then successful sign-in and sign-out, client-side validation on registration
- **checkout**: guest cart → sign-up at checkout → payment → order page and history, declined and insufficient-funds cards keep stock and are saved as failed orders, price change after adding to cart is shown before paying, the last item in stock sells only once
- **admin**: staff cannot buy, product created in the admin form shows up in the catalog, form validation, CSV import preview saves nothing and the real import does
