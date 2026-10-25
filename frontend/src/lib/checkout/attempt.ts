import { ApiError } from "@/lib/api/problem";

export interface CheckoutAttempt {
  key: string;
  fingerprint: string;
}

export function attemptFor(previous: CheckoutAttempt | null, fingerprint: string, newKey: () => string): CheckoutAttempt {
  if (previous && previous.fingerprint === fingerprint) {
    return previous;
  }
  return { key: newKey(), fingerprint };
}

export function isRetryableFailure(error: unknown): boolean {
  if (!(error instanceof ApiError)) {
    return true;
  }
  return error.status === 0 || error.status >= 500 || error.code === "checkout_in_progress" || error.code === "rate_limited";
}

export async function fingerprintOf(body: unknown): Promise<string> {
  const bytes = new TextEncoder().encode(JSON.stringify(body));
  const digest = await crypto.subtle.digest("SHA-256", bytes);
  return Array.from(new Uint8Array(digest), (byte) => byte.toString(16).padStart(2, "0")).join("");
}
