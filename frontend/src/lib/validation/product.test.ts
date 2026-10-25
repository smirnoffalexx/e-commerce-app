import { describe, expect, it } from "vitest";
import { zodFieldErrors } from "./errors";
import { productSchema } from "./product";

const valid = {
  sku: " rs-001 ",
  name: "Running Shoes",
  description: "Line one\nLine two",
  category_id: "c1",
  price: "89.99",
  stock: "150",
  weight_kg: "0.35",
};

function errors(input: Record<string, string>) {
  const result = productSchema.safeParse(input);
  return result.success ? {} : zodFieldErrors(result.error);
}

describe("productSchema", () => {
  it("normalises the SKU and converts stock to a number", () => {
    expect(productSchema.parse(valid)).toEqual({ ...valid, sku: "RS-001", stock: 150 });
  });

  it("mirrors backend SKU rules", () => {
    expect(errors({ ...valid, sku: "-RS" })).toHaveProperty("sku");
    expect(errors({ ...valid, sku: "RS_001" })).toHaveProperty("sku");
    expect(errors({ ...valid, sku: "A".repeat(65) })).toHaveProperty("sku");
  });

  it("checks decimal places and limits", () => {
    expect(errors({ ...valid, price: "1.999", weight_kg: "0.1234" })).toEqual({
      price: "Price must be a non-negative number with at most 2 decimals.",
      weight_kg: "Weight must be a non-negative number with at most 3 decimals.",
    });
    expect(errors({ ...valid, price: "10000000000.00" })).toEqual({ price: "Price must be at most 9999999999.99." });
    expect(errors({ ...valid, price: "0" })).toEqual({});
  });

  it("rejects negative or fractional stock and missing category", () => {
    expect(errors({ ...valid, stock: "-1", category_id: "" })).toEqual({
      stock: "Stock must be a whole non-negative number.",
      category_id: "Choose a category.",
    });
    expect(errors({ ...valid, stock: "1.5" })).toHaveProperty("stock");
  });

  it("rejects control characters but allows new lines and tabs in the description", () => {
    expect(errors({ ...valid, name: "Bad\u0007name" })).toHaveProperty("name");
    expect(errors({ ...valid, description: "ok\ttab\r\nnext" })).toEqual({});
    expect(errors({ ...valid, description: "bell\u0007" })).toHaveProperty("description");
  });
});
