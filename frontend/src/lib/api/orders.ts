import { api } from "./client";
import type { CheckoutRequest, Order } from "./types";

export function placeOrder(body: CheckoutRequest, idempotencyKey: string) {
  return api<Order>("/orders", { body, headers: { "Idempotency-Key": idempotencyKey } });
}
