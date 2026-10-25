import { describe, expect, it } from "vitest";
import { cardSchema, formatCardNumber, formatExpiry, luhnValid, TEST_CARDS } from "./card";
import { zodFieldErrors } from "@/lib/validation/errors";

const now = new Date("2026-10-25T00:00:00Z");

function errors(input: { card_number: string; expiry: string; cvc: string }) {
  const result = cardSchema(now).safeParse(input);
  return result.success ? {} : zodFieldErrors(result.error);
}

describe("card input", () => {
  it("formats the number in groups of four and the expiry as MM/YY", () => {
    expect(formatCardNumber("4242424242424242")).toBe("4242 4242 4242 4242");
    expect(formatCardNumber("4242-4242 42")).toBe("4242 4242 42");
    expect(formatExpiry("1228")).toBe("12/28");
    expect(formatExpiry("1")).toBe("1");
  });

  it("accepts every documented test card", () => {
    for (const card of TEST_CARDS) {
      expect(luhnValid(card.number.replace(/ /g, ""))).toBe(true);
    }
  });

  it("returns the payload the backend expects", () => {
    expect(cardSchema(now).parse({ card_number: "4242 4242 4242 4242", expiry: "10/26", cvc: "123" })).toEqual({
      card_number: "4242424242424242",
      expiry: { month: 10, year: 2026 },
      cvc: "123",
    });
  });

  it("mirrors backend card validation", () => {
    expect(errors({ card_number: "4242 4242 4242 4241", expiry: "12/30", cvc: "123" })).toEqual({
      card_number: "Card number is invalid.",
    });
    expect(errors({ card_number: "4242", expiry: "13/30", cvc: "12" })).toEqual({
      card_number: "Card number must have 12 to 19 digits.",
      expiry: "Use the MM/YY format.",
      cvc: "CVC must be 3 or 4 digits.",
    });
    expect(errors({ card_number: "4242424242424242", expiry: "09/26", cvc: "123" })).toEqual({ expiry: "The card has expired." });
    expect(errors({ card_number: "4242424242424242", expiry: "01/2047", cvc: "123" })).toEqual({ expiry: "Check the expiry year." });
  });
});
