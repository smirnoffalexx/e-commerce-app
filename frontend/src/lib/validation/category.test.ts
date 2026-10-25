import { describe, expect, it } from "vitest";
import { categorySchema } from "./category";

describe("categorySchema", () => {
  it("trims the name", () => {
    expect(categorySchema.parse({ name: "  Home & Office " })).toEqual({ name: "Home & Office" });
  });

  it("requires something the backend can turn into a slug", () => {
    expect(categorySchema.safeParse({ name: "Книги" }).success).toBe(false);
    expect(categorySchema.safeParse({ name: "&&&" }).success).toBe(false);
    expect(categorySchema.safeParse({ name: "3D Printing" }).success).toBe(true);
  });

  it("limits the length", () => {
    expect(categorySchema.safeParse({ name: "a".repeat(65) }).success).toBe(false);
  });
});
