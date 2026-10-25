import { describe, expect, it } from "vitest";
import type { CartLine } from "@/lib/cart/cart";
import { applyCheckoutIssues } from "./issues";

function line(productId: string, quantity = 2, price = "10.00"): CartLine {
  return { productId, sku: productId, name: productId, price, stock: 50, quantity };
}

describe("applyCheckoutIssues", () => {
  it("applies the backend 409 details to the cart and lists the changes", () => {
    const cart = [line("price"), line("short", 5), line("gone"), line("deleted"), line("ok")];
    const result = applyCheckoutIssues(cart, [
      { product_id: "price", code: "price_changed", requested_quantity: 2, expected_price: "10.00", current_price: "11.50" },
      { product_id: "short", code: "insufficient_stock", requested_quantity: 5, available_quantity: 3 },
      { product_id: "gone", code: "out_of_stock", requested_quantity: 2, available_quantity: 0 },
      { product_id: "deleted", code: "product_unavailable", requested_quantity: 2 },
    ]);

    expect(result.lines.map((item) => [item.productId, item.price, item.quantity])).toEqual([
      ["price", "11.50", 2],
      ["short", "10.00", 3],
      ["ok", "10.00", 2],
    ]);
    expect(result.changes.map((change) => change.kind)).toEqual([
      "price_changed",
      "quantity_reduced",
      "out_of_stock",
      "unavailable",
    ]);
  });

  it("removes a line when the remaining stock is zero", () => {
    const result = applyCheckoutIssues([line("a")], [{ product_id: "a", code: "insufficient_stock", requested_quantity: 2, available_quantity: 0 }]);
    expect(result.lines).toEqual([]);
    expect(result.changes[0].kind).toBe("out_of_stock");
  });
});
