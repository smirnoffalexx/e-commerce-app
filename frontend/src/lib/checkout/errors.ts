import { ApiError, fieldErrors } from "@/lib/api/problem";

const FIELD_MAP: Record<string, string> = {
  "payment.card_number": "card_number",
  "payment.exp_month": "expiry",
  "payment.exp_year": "expiry",
  "payment.cvc": "cvc",
};

export function cardFieldErrors(error: unknown): Record<string, string> {
  const out: Record<string, string> = {};
  for (const [field, message] of Object.entries(fieldErrors(error))) {
    const target = FIELD_MAP[field];
    if (target && !(target in out)) {
      out[target] = message;
    }
  }
  return out;
}

export function isCartConflict(error: unknown): error is ApiError {
  return error instanceof ApiError && (error.code === "price_changed" || error.code === "items_unavailable") && Boolean(error.problem.items?.length);
}

export function isPaymentFailure(error: unknown): error is ApiError {
  return error instanceof ApiError && error.status === 402;
}
