import { z } from "zod";

export const TEST_CARDS = [
  { number: "4242 4242 4242 4242", label: "Successful payment" },
  { number: "4000 0000 0000 0002", label: "Card declined" },
  { number: "4000 0000 0000 9995", label: "Insufficient funds" },
  { number: "4000 0000 0000 0119", label: "Payment timeout" },
];

const MAX_YEARS_AHEAD = 20;

export function cardDigits(value: string): string {
  return value.replace(/\D/g, "");
}

export function formatCardNumber(value: string): string {
  return cardDigits(value)
    .slice(0, 19)
    .replace(/(\d{4})(?=\d)/g, "$1 ");
}

export function formatExpiry(value: string): string {
  const digits = cardDigits(value).slice(0, 4);
  return digits.length > 2 ? `${digits.slice(0, 2)}/${digits.slice(2)}` : digits;
}

export function luhnValid(digits: string): boolean {
  let sum = 0;
  let double = false;
  for (let i = digits.length - 1; i >= 0; i--) {
    let digit = Number(digits[i]);
    if (double) {
      digit *= 2;
      if (digit > 9) {
        digit -= 9;
      }
    }
    sum += digit;
    double = !double;
  }
  return digits.length > 0 && sum % 10 === 0;
}

export function parseExpiry(value: string): { month: number; year: number } | null {
  const match = /^(\d{2})\s*\/\s*(\d{2}|\d{4})$/.exec(value.trim());
  if (!match) {
    return null;
  }
  const year = match[2].length === 2 ? 2000 + Number(match[2]) : Number(match[2]);
  return { month: Number(match[1]), year };
}

export function cardSchema(now: Date = new Date()) {
  const currentYear = now.getUTCFullYear();
  const currentMonth = now.getUTCMonth() + 1;
  return z.object({
    card_number: z
      .string()
      .transform(cardDigits)
      .refine((digits) => digits.length > 0, "Card number is required.")
      .refine((digits) => digits.length >= 12 && digits.length <= 19, "Card number must have 12 to 19 digits.")
      .refine(luhnValid, "Card number is invalid."),
    expiry: z
      .string()
      .transform((value, ctx) => {
        const parsed = parseExpiry(value);
        if (!parsed || parsed.month < 1 || parsed.month > 12) {
          ctx.addIssue({ code: "custom", message: "Use the MM/YY format." });
          return z.NEVER;
        }
        if (parsed.year < currentYear || (parsed.year === currentYear && parsed.month < currentMonth)) {
          ctx.addIssue({ code: "custom", message: "The card has expired." });
          return z.NEVER;
        }
        if (parsed.year > currentYear + MAX_YEARS_AHEAD) {
          ctx.addIssue({ code: "custom", message: "Check the expiry year." });
          return z.NEVER;
        }
        return parsed;
      }),
    cvc: z.string().trim().regex(/^\d{3,4}$/, "CVC must be 3 or 4 digits."),
  });
}

export type CardInput = z.output<ReturnType<typeof cardSchema>>;
