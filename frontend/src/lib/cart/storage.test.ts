import { describe, expect, it } from "vitest";
import { parseCart } from "./storage";

const line = { productId: "p1", sku: "S", name: "Mug", price: "12.50", stock: 5, quantity: 2 };

describe("parseCart", () => {
  it("restores a saved cart", () => {
    expect(parseCart(JSON.stringify([line]))).toEqual([line]);
  });

  it("survives corrupted or tampered storage", () => {
    expect(parseCart("{not json")).toEqual([]);
    expect(parseCart(JSON.stringify({ items: [] }))).toEqual([]);
    expect(
      parseCart(JSON.stringify([line, { ...line, productId: "p2", quantity: 1000 }, { ...line, productId: "p3", price: "abc" }, line])),
    ).toEqual([line]);
  });
});
