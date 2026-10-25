import { expect, test, type Page } from "@playwright/test";
import { PASSWORD, createProduct, getProduct, registerCustomer, setPrice, signIn, unique, type Session } from "../support/api";

// One signed-up customer for the scenarios that do not test sign-up itself (auth is rate limited).
let customer: Session;
test.beforeAll(async () => {
  customer = (await registerCustomer()).session;
});

async function addToCart(page: Page, productId: string) {
  await page.goto(`/products/${productId}`);
  await page.getByRole("button", { name: "Add to cart" }).click();
  await expect(page.getByText("added to cart")).toBeVisible();
}

async function pay(page: Page, card: "Successful payment" | "Card declined" | "Insufficient funds") {
  await page.getByRole("button", { name: card }).click();
  await page.getByRole("button", { name: /^Pay / }).click();
}

test("guest fills a cart, registers at checkout and pays", async ({ page }) => {
  const lamp = await createProduct({ price: "30.00", stock: 5 });
  const mug = await createProduct({ price: "12.25", stock: 5 });

  await addToCart(page, lamp.id);
  await addToCart(page, mug.id);
  await page.goto("/cart");
  await expect(page.getByRole("link", { name: "Cart, 2 items" })).toBeVisible();
  await page.getByRole("group", { name: `Quantity of ${lamp.name}` }).getByRole("button", { name: "Increase quantity" }).click();
  await expect(page.getByText("$72.25").first()).toBeVisible();

  await page.getByRole("link", { name: "Checkout" }).click();
  await expect(page).toHaveURL("/login?next=%2Fcheckout");
  await page.getByRole("link", { name: "Create an account" }).click();
  await expect(page).toHaveURL("/register?next=%2Fcheckout");

  const email = `${unique("buyer")}@e2e.test`;
  await page.getByLabel("Email").fill(email);
  await page.getByLabel("Password", { exact: true }).fill(PASSWORD);
  await page.getByLabel("Confirm password").fill(PASSWORD);
  await page.getByRole("button", { name: "Create account" }).click();

  await expect(page).toHaveURL("/checkout");
  await expect(page.getByText(`Receipt goes to ${email}`)).toBeVisible();
  await pay(page, "Successful payment");

  await expect(page).toHaveURL(/\/orders\/[0-9a-f-]+\?placed=1$/);
  await expect(page.getByText("Thank you for your order!")).toBeVisible();
  await expect(page.locator('[data-slot="badge"]').getByText("Paid", { exact: true })).toBeVisible();
  await expect(page.getByText("Card ending in 4242")).toBeVisible();
  await expect(page.getByRole("link", { name: "Cart" })).toBeVisible();

  expect((await getProduct(lamp.id)).available).toBe(3);
  expect((await getProduct(mug.id)).available).toBe(4);

  await page.getByRole("link", { name: "All orders" }).click();
  await expect(page).toHaveURL("/orders");
  await expect(page.getByText("$72.25")).toBeVisible();
});

test("declined card keeps the cart and stock, the next card succeeds", async ({ page, context, baseURL }) => {
  const product = await createProduct({ price: "19.99", stock: 2 });
  await signIn(context, baseURL!, customer);

  await addToCart(page, product.id);
  await page.goto("/checkout");
  await pay(page, "Card declined");
  await expect(page.getByText(/No money was taken/)).toBeVisible();
  await expect(page.getByRole("link", { name: "failed order" })).toBeVisible();
  expect((await getProduct(product.id)).available).toBe(2);

  await pay(page, "Insufficient funds");
  await expect(page.getByText(/No money was taken/)).toBeVisible();

  await pay(page, "Successful payment");
  await expect(page.getByText("Thank you for your order!")).toBeVisible();
  expect((await getProduct(product.id)).available).toBe(1);

  await page.goto("/orders");
  const orders = page.getByRole("link", { name: /^Order #/ });
  await expect(orders.filter({ hasText: "Payment failed" })).toHaveCount(2);
  await expect(orders.filter({ hasText: "Paid" })).toHaveCount(1);

  await expect(page.getByRole("link", { name: "Admin" })).toHaveCount(0);
});

test("checkout picks up a price change made after the item was added", async ({ page, context, baseURL }) => {
  const product = await createProduct({ price: "30.00", stock: 3 });
  await signIn(context, baseURL!, customer);

  await addToCart(page, product.id);
  await setPrice(product, "35.00");

  await page.goto("/checkout");
  await expect(page.getByText("Please review your order")).toBeVisible();
  await expect(page.getByRole("button", { name: "Pay $35.00" })).toBeVisible();
  await pay(page, "Successful payment");
  await expect(page.getByText("Thank you for your order!")).toBeVisible();
  await expect(page.getByText("$35.00").first()).toBeVisible();
});

test("the last item in stock can be bought only once", async ({ browser, baseURL }) => {
  const product = await createProduct({ price: "5.00", stock: 1 });

  const first = await browser.newContext();
  const second = await browser.newContext();
  try {
    const firstPage = await first.newPage();
    const secondPage = await second.newPage();
    await addToCart(firstPage, product.id);
    await addToCart(secondPage, product.id);

    await signIn(first, baseURL!, customer);
    await firstPage.goto("/checkout");
    await pay(firstPage, "Successful payment");
    await expect(firstPage.getByText("Thank you for your order!")).toBeVisible();

    await secondPage.goto(`/products/${product.id}`);
    await expect(secondPage.getByRole("button", { name: "Out of stock" })).toBeDisabled();
  } finally {
    await first.close();
    await second.close();
  }
});
