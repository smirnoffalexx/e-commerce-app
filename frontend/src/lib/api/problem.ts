import type { Problem } from "./types";

const messages: Record<string, string> = {
  network_error: "Could not reach the server. Check your connection and try again.",
  backend_unavailable: "The store is temporarily unavailable. Please try again in a moment.",
  invalid_origin: "The request was blocked for security reasons. Reload the page and try again.",
  unauthorized: "Please sign in to continue.",
  missing_token: "Please sign in to continue.",
  invalid_token: "Your session has expired. Please sign in again.",
  invalid_credentials: "Invalid email or password.",
  temp_password_expired: "Your temporary password has expired. Ask an administrator for a new one.",
  password_change_required: "You need to set a new password before continuing.",
  insufficient_permissions: "You do not have permission to do this.",
  forbidden: "You do not have permission to do this.",
  account_inactive: "Your account is deactivated.",
  email_taken: "An account with this email already exists.",
  validation_failed: "Some fields are invalid. Fix them and try again.",
  rate_limited: "Too many requests. Wait a minute and try again.",
  not_found: "Not found.",
  product_not_found: "This product no longer exists.",
  category_not_found: "This category no longer exists.",
  order_not_found: "Order not found.",
  admin_not_found: "Administrator not found.",
  sku_taken: "A product with this SKU already exists.",
  version_conflict: "Someone else changed this product. Reload it and try again.",
  category_exists: "A category with this name already exists.",
  category_in_use: "This category still has products and cannot be deleted.",
  price_changed: "Some prices have changed since you added the items.",
  items_unavailable: "Some items are no longer available in the requested quantity.",
  checkout_in_progress: "Your order is still being processed. Wait a few seconds.",
  idempotency_key_reused: "This checkout attempt was already used. Try again.",
  card_declined: "Your card was declined.",
  insufficient_funds: "Your card has insufficient funds.",
  payment_timeout: "The payment provider did not respond in time.",
  payment_error: "The payment could not be processed.",
  file_too_large: "The file is too large (5 MB maximum).",
  timeout: "The request took too long. Please try again.",
  internal_error: "Something went wrong on our side. Please try again later.",
};

const fieldMessages: Record<string, string> = {
  required: "This field is required.",
  same_as_email: "Password must not be the same as your email.",
};

export class ApiError extends Error {
  readonly problem: Problem;

  constructor(problem: Problem) {
    super(problem.detail || problem.title || problem.code);
    this.name = "ApiError";
    this.problem = problem;
  }

  get status() {
    return this.problem.status;
  }

  get code() {
    return this.problem.code;
  }
}

export function toProblem(status: number, body: unknown): Problem {
  if (body && typeof body === "object" && typeof (body as Problem).code === "string") {
    return { ...(body as Problem), status };
  }
  if (status === 502 || status === 503 || status === 504) {
    return { status, code: "backend_unavailable" };
  }
  return { status, code: status >= 500 ? "internal_error" : "error" };
}

export function errorMessage(error: unknown): string {
  if (error instanceof ApiError) {
    return messages[error.code] ?? error.problem.detail ?? error.problem.title ?? "Request failed.";
  }
  return messages.network_error;
}

export function fieldErrors(error: unknown): Record<string, string> {
  if (!(error instanceof ApiError) || !error.problem.errors) {
    return {};
  }
  const out: Record<string, string> = {};
  for (const item of error.problem.errors) {
    if (!(item.field in out)) {
      out[item.field] = fieldMessages[item.code] ?? capitalize(item.message);
    }
  }
  return out;
}

function capitalize(text: string) {
  const sentence = text.charAt(0).toUpperCase() + text.slice(1);
  return sentence.endsWith(".") ? sentence : `${sentence}.`;
}
