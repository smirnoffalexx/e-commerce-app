import { expect, test } from "@playwright/test";
import { api, createProduct, signIn, superAdminSession, unique } from "../support/api";

test.beforeEach(async ({ context, baseURL }) => {
  await signIn(context, baseURL!, superAdminSession());
});

test("staff browse the shop but cannot buy", async ({ page }) => {
  const product = await createProduct();

  await page.goto(`/products/${product.id}`);
  await expect(page.getByRole("button", { name: "Add to cart" })).toBeDisabled();
  await expect(page.getByRole("link", { name: /^Cart/ })).toHaveCount(0);
  await expect(page.getByRole("link", { name: "Admin" })).toBeVisible();
});

test("admin creates a product that shoppers can find", async ({ page }) => {
  const token = superAdminSession().access_token;
  const category = await api<{ id: string; name: string }>("POST", "/categories", { name: unique("Admin Category") }, token);
  const name = unique("Admin Chair");
  const sku = unique("ADM").toUpperCase();

  await page.goto("/admin/products/new");
  await page.getByLabel("SKU").fill(sku);
  await page.getByLabel("Name").fill(name);
  await page.getByLabel("Description").fill("Made in the e2e suite");
  await page.getByRole("combobox").filter({ hasText: "Choose a category" }).click();
  await page.getByRole("option", { name: category.name }).click();
  await page.getByLabel("Price, $").fill("149.00");
  await page.getByLabel("Stock").fill("4");
  await page.getByLabel("Weight, kg").fill("7.5");
  await page.getByRole("button", { name: "Create product" }).click();

  await expect(page).toHaveURL(/\/admin\/products(\?|$)/);
  await expect(page.getByText(`${name} created`)).toBeVisible();

  await page.goto(`/?q=${encodeURIComponent(name)}`);
  const card = page.getByRole("article").filter({ hasText: name });
  await expect(card).toContainText("$149.00");
});

test("admin form rejects invalid product data", async ({ page }) => {
  await page.goto("/admin/products/new");
  await page.getByLabel("Price, $").fill("-1");
  await page.getByRole("button", { name: "Create product" }).click();
  await expect(page).toHaveURL("/admin/products/new");
  await expect(page.locator('[aria-invalid="true"]').first()).toBeVisible();
});

test("CSV import previews without saving, then imports", async ({ page }) => {
  const category = unique("Imported");
  const sku = unique("IMP").toUpperCase();
  const name = unique("Imported Kettle");
  const csv = `name,sku,description,category,price,stock,weight_kg\n${name},${sku},From CSV,${category},55.00,9,1.2\n`;

  await page.goto("/admin/import");
  await page.locator('input[type="file"]').setInputFiles({ name: "catalog.csv", mimeType: "text/csv", buffer: Buffer.from(csv) });
  await page.getByLabel("Create missing categories").check();

  await page.getByRole("button", { name: "Preview" }).click();
  await expect(page.getByText("Preview: nothing has been saved yet")).toBeVisible();
  await expect(page.getByText("Would create")).toBeVisible();
  const before = await api<{ total: number }>("GET", `/products?q=${encodeURIComponent(sku)}`);
  expect(before.total).toBe(0);

  await page.getByRole("button", { name: "Import", exact: true }).click();
  await expect(page.getByRole("link", { name: "Go to products" })).toBeVisible();

  await page.goto(`/?q=${encodeURIComponent(name)}`);
  await expect(page.getByRole("article").filter({ hasText: name })).toContainText("$55.00");
});
