import { describe, expect, it } from "vitest";
import { ApiError } from "@/lib/api/problem";
import { attemptFor, fingerprintOf, isRetryableFailure } from "./attempt";

describe("attemptFor", () => {
  let counter = 0;
  const newKey = () => `key-${++counter}`;

  it("reuses the key when the same order is submitted again", () => {
    const first = attemptFor(null, "abc", newKey);
    expect(attemptFor(first, "abc", newKey)).toBe(first);
  });

  it("starts a new attempt when the cart or card changed", () => {
    const first = attemptFor(null, "abc", newKey);
    expect(attemptFor(first, "def", newKey).key).not.toBe(first.key);
  });
});

describe("isRetryableFailure", () => {
  it("keeps the key when the outcome is unknown", () => {
    expect(isRetryableFailure(new ApiError({ status: 0, code: "network_error" }))).toBe(true);
    expect(isRetryableFailure(new ApiError({ status: 502, code: "backend_unavailable" }))).toBe(true);
    expect(isRetryableFailure(new ApiError({ status: 409, code: "checkout_in_progress" }))).toBe(true);
    expect(isRetryableFailure(new TypeError("fetch failed"))).toBe(true);
  });

  it("drops the key after a final answer", () => {
    expect(isRetryableFailure(new ApiError({ status: 402, code: "card_declined" }))).toBe(false);
    expect(isRetryableFailure(new ApiError({ status: 409, code: "price_changed" }))).toBe(false);
    expect(isRetryableFailure(new ApiError({ status: 422, code: "validation_failed" }))).toBe(false);
  });
});

describe("fingerprintOf", () => {
  it("is stable for the same body and changes with any field", async () => {
    const body = { items: [{ product_id: "p", quantity: 1 }], payment: { cvc: "123" } };
    expect(await fingerprintOf(body)).toBe(await fingerprintOf(structuredClone(body)));
    expect(await fingerprintOf({ ...body, payment: { cvc: "124" } })).not.toBe(await fingerprintOf(body));
  });
});
