import { describe, expect, it } from "vitest";
import { fromCents, isMoney, toCents } from "./money";

describe("money", () => {
  it("converts decimal strings to integer cents without floating point errors", () => {
    expect(toCents("0.1") + toCents("0.2")).toBe(toCents("0.3"));
    expect(toCents("89.99") * 3).toBe(26997);
    expect(toCents("5")).toBe(500);
  });

  it("formats cents back to the API format", () => {
    expect(fromCents(26997)).toBe("269.97");
    expect(fromCents(5)).toBe("0.05");
  });

  it("accepts only non-negative amounts with up to two decimals", () => {
    expect(isMoney("10")).toBe(true);
    expect(isMoney("10.5")).toBe(true);
    expect(isMoney("10.555")).toBe(false);
    expect(isMoney("-1")).toBe(false);
    expect(isMoney("1e3")).toBe(false);
  });
});
