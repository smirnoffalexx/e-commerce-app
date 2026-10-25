import { describe, expect, it } from "vitest";
import type { Product } from "@/lib/api/types";
import {
  MAX_CART_LINES,
  addToCart,
  itemCount,
  lineFromProduct,
  reconcile,
  setQuantity,
  subtotal,
  type CartLine,
} from "./cart";

function product(overrides: Partial<Product> = {}): Product {
  return {
    id: "p1",
    sku: "SKU-1",
    name: "Mug",
    description: "",
    category_id: "c1",
    category: { id: "c1", name: "Kitchen", slug: "kitchen" },
    price: "12.50",
    stock: 10,
    in_stock: true,
    weight_kg: "0.3",
    version: 1,
    created_at: "",
    updated_at: "",
    ...overrides,
  };
}

describe("addToCart", () => {
  it("merges repeated adds of the same product into one line", () => {
    let lines: CartLine[] = [];
    lines = addToCart(lines, product(), 2).lines;
    const result = addToCart(lines, product(), 3);
    expect(result.outcome).toBe("added");
    expect(result.lines).toHaveLength(1);
    expect(result.lines[0].quantity).toBe(5);
  });

  it("caps the quantity at the stock and at 100 per line", () => {
    expect(addToCart([], product({ stock: 3 }), 5)).toMatchObject({ outcome: "limited", lines: [{ quantity: 3 }] });
    expect(addToCart([], product({ stock: 500 }), 150)).toMatchObject({ outcome: "limited", lines: [{ quantity: 100 }] });
  });

  it("refuses out-of-stock products and a 51st line", () => {
    expect(addToCart([], product({ stock: 0 }), 1).outcome).toBe("out_of_stock");
    const full = Array.from({ length: MAX_CART_LINES }, (_, i) => lineFromProduct(product({ id: `p${i}` }), 1));
    expect(addToCart(full, product({ id: "new" }), 1).outcome).toBe("cart_full");
    expect(addToCart(full, product({ id: "p0" }), 1).outcome).toBe("added");
  });
});

describe("setQuantity", () => {
  it("keeps the quantity between 1 and the known stock", () => {
    const lines = [lineFromProduct(product({ stock: 4 }), 2)];
    expect(setQuantity(lines, "p1", 0)[0].quantity).toBe(1);
    expect(setQuantity(lines, "p1", 9)[0].quantity).toBe(4);
  });
});

describe("totals", () => {
  it("sums lines in cents", () => {
    const lines = [lineFromProduct(product({ price: "0.10" }), 3), lineFromProduct(product({ id: "p2", price: "19.99" }), 2)];
    expect(itemCount(lines)).toBe(5);
    expect(subtotal(lines)).toBe("40.28");
  });
});

describe("reconcile", () => {
  const cart = [
    lineFromProduct(product({ id: "same" }), 1),
    lineFromProduct(product({ id: "pricier", price: "10.00" }), 1),
    lineFromProduct(product({ id: "scarce", stock: 10 }), 6),
    lineFromProduct(product({ id: "sold-out" }), 1),
    lineFromProduct(product({ id: "deleted" }), 1),
    lineFromProduct(product({ id: "not-checked" }), 1),
  ];

  const fresh = new Map<string, Product | null>([
    ["same", product({ id: "same" })],
    ["pricier", product({ id: "pricier", price: "12.00" })],
    ["scarce", product({ id: "scarce", stock: 2 })],
    ["sold-out", product({ id: "sold-out", stock: 0 })],
    ["deleted", null],
  ]);

  it("adopts current prices and stock, reporting every change to the buyer", () => {
    const result = reconcile(cart, fresh);
    expect(result.lines.map((line) => [line.productId, line.price, line.quantity])).toEqual([
      ["same", "12.50", 1],
      ["pricier", "12.00", 1],
      ["scarce", "12.50", 2],
      ["not-checked", "12.50", 1],
    ]);
    expect(result.changes.map((change) => [change.kind, change.productId])).toEqual([
      ["price_changed", "pricier"],
      ["quantity_reduced", "scarce"],
      ["out_of_stock", "sold-out"],
      ["unavailable", "deleted"],
    ]);
  });

  it("reports nothing when the cart is current", () => {
    expect(reconcile([cart[0]], fresh).changes).toEqual([]);
  });
});
