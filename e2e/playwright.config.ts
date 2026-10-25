import { defineConfig, devices } from "@playwright/test";

// Runs against the whole docker compose stack (Caddy -> Next.js BFF -> Go API).
export default defineConfig({
  testDir: "./tests",
  // Auth endpoints allow 10 requests per minute per client IP, and every request
  // from this machine shares one IP, so the suite runs serially and keeps logins few.
  workers: 1,
  fullyParallel: false,
  retries: process.env.CI ? 1 : 0,
  timeout: 60_000,
  expect: { timeout: 10_000 },
  reporter: process.env.CI ? [["list"], ["html", { open: "never" }]] : "list",
  globalSetup: "./global-setup.ts",
  use: {
    baseURL: process.env.E2E_BASE_URL ?? "http://localhost:3000",
    trace: "retain-on-failure",
    screenshot: "only-on-failure",
  },
  projects: [{ name: "chromium", use: { ...devices["Desktop Chrome"] } }],
});
