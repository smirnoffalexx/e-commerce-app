import { expect, test } from "@playwright/test";
import { PASSWORD, api, unique } from "../support/api";

test("customer signs in after a wrong password and signs out", async ({ page }) => {
  const email = `${unique("login")}@e2e.test`;
  await api("POST", "/auth/register", { email, password: PASSWORD });

  await page.goto("/login");
  await page.getByLabel("Email").fill(email);
  await page.getByLabel("Password").fill("wrong-password-123");
  await page.getByRole("button", { name: "Sign in" }).click();
  await expect(page.getByText("Invalid email or password.")).toBeVisible();

  await page.getByLabel("Password").fill(PASSWORD);
  await page.getByRole("button", { name: "Sign in" }).click();
  await expect(page).toHaveURL("/");

  await page.getByRole("button", { name: "Account menu" }).click();
  await expect(page.getByRole("menu")).toContainText(email);
  await expect(page.getByRole("menu")).toContainText("Customer");
  await page.getByRole("menuitem", { name: "Sign out" }).click();

  await expect(page.getByRole("link", { name: "Sign in" })).toBeVisible();
  await page.goto("/account");
  await expect(page).toHaveURL(/\/login\?next=/);
});

test("registration form validates input before calling the API", async ({ page }) => {
  await page.goto("/register");
  await page.getByLabel("Email").fill("not-an-email");
  await page.getByLabel("Password", { exact: true }).fill("short");
  await page.getByLabel("Confirm password").fill("different");
  await page.getByRole("button", { name: "Create account" }).click();

  await expect(page).toHaveURL("/register");
  await expect(page.locator('[aria-invalid="true"]')).toHaveCount(3);
});
