import { describe, expect, it } from "vitest";
import { orderListHref, parseOrderListQuery, shortOrderId } from "./orders";

describe("order list query", () => {
  it("keeps known statuses and valid pages", () => {
    expect(parseOrderListQuery({ status: "FAILED", page: "2" })).toEqual({ status: "FAILED", page: 2 });
    expect(parseOrderListQuery({ status: "SHIPPED", page: "-1" })).toEqual({});
  });

  it("builds links without default values", () => {
    expect(orderListHref("/orders", { status: "PAID", page: 1 })).toBe("/orders?status=PAID");
    expect(orderListHref("/orders", {})).toBe("/orders");
  });
});

describe("shortOrderId", () => {
  it("uses the random tail of the UUID v7, not the timestamp head", () => {
    expect(shortOrderId("01a1707a-4dd6-7b33-a55a-0b6f3999ed79")).toBe("3999ED79");
  });
});
