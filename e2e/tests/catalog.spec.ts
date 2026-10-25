import { expect, test } from "@playwright/test";
import { createProduct, unique } from "../support/api";

test("guest finds a product by search and opens its page", async ({ page }) => {
  const product = await createProduct({ name: unique("Search Lamp"), price: "42.50" });

  await page.goto("/");
  await expect(page.getByRole("heading", { name: "Catalog" })).toBeVisible();
  await page.getByLabel("Search products").fill(product.name);
  await expect(page).toHaveURL(/[?&]q=/);

  const card = page.getByRole("article").filter({ hasText: product.name });
  await expect(card).toHaveCount(1);
  await expect(card).toContainText("$42.50");

  await card.getByRole("link", { name: product.name }).click();
  await expect(page).toHaveURL(`/products/${product.id}`);
  await expect(page.getByRole("heading", { name: product.name })).toBeVisible();
  await expect(page.getByRole("button", { name: "Add to cart" })).toBeEnabled();
});

test("search with no matches shows no products", async ({ page }) => {
  await page.goto(`/?q=${unique("nothing-like-this")}`);
  await expect(page.getByRole("article")).toHaveCount(0);
  await expect(page.getByText("0 products")).toBeVisible();
});

test("guest is sent to sign in for protected pages", async ({ page }) => {
  await page.goto("/orders");
  await expect(page).toHaveURL("/login?next=%2Forders");
  await page.goto("/admin");
  await expect(page).toHaveURL("/login?next=%2Fadmin");
});
