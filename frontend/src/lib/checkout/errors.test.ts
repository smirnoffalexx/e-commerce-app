import { describe, expect, it } from "vitest";
import { ApiError } from "@/lib/api/problem";
import { cardFieldErrors, isCartConflict, isPaymentFailure } from "./errors";

describe("cardFieldErrors", () => {
  it("maps backend payment fields onto the card form", () => {
    const error = new ApiError({
      status: 422,
      code: "validation_failed",
      errors: [
        { field: "payment.exp_month", code: "expired", message: "card has expired" },
        { field: "payment.exp_year", code: "invalid_format", message: "expiry year must be a four-digit year" },
        { field: "payment.cvc", code: "invalid_format", message: "cvc must be 3 or 4 digits" },
        { field: "items[0].quantity", code: "too_large", message: "quantity must be at most 100" },
      ],
    });
    expect(cardFieldErrors(error)).toEqual({ expiry: "Card has expired.", cvc: "Cvc must be 3 or 4 digits." });
  });
});

describe("checkout error kinds", () => {
  it("recognises cart conflicts only when the backend sent item details", () => {
    expect(isCartConflict(new ApiError({ status: 409, code: "price_changed", items: [{ product_id: "p", code: "price_changed", requested_quantity: 1 }] }))).toBe(true);
    expect(isCartConflict(new ApiError({ status: 409, code: "checkout_in_progress" }))).toBe(false);
  });

  it("recognises payment failures by status", () => {
    expect(isPaymentFailure(new ApiError({ status: 402, code: "card_declined", order_id: "o" }))).toBe(true);
    expect(isPaymentFailure(new ApiError({ status: 409, code: "price_changed" }))).toBe(false);
  });
});
