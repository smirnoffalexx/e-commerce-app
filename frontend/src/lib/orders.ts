import type { OrderStatus } from "@/lib/api/types";

export const ORDER_STATUSES: OrderStatus[] = ["PAID", "PENDING", "FAILED"];

export const orderStatusLabels: Record<OrderStatus, string> = {
  PAID: "Paid",
  PENDING: "Processing",
  FAILED: "Payment failed",
};

export interface OrderListQuery {
  status?: OrderStatus;
  page?: number;
}

type Params = Record<string, string | string[] | undefined>;

export function parseOrderListQuery(params: Params): OrderListQuery {
  const query: OrderListQuery = {};
  const status = Array.isArray(params.status) ? params.status[0] : params.status;
  if (status && (ORDER_STATUSES as string[]).includes(status)) {
    query.status = status as OrderStatus;
  }
  const page = Number(Array.isArray(params.page) ? params.page[0] : params.page);
  if (Number.isInteger(page) && page > 1) {
    query.page = Math.min(page, 10_000);
  }
  return query;
}

export function orderListHref(base: string, query: OrderListQuery): string {
  const params = new URLSearchParams();
  if (query.status) params.set("status", query.status);
  if (query.page && query.page > 1) params.set("page", String(query.page));
  const text = params.toString();
  return text ? `${base}?${text}` : base;
}

export function shortOrderId(id: string): string {
  return id.replace(/-/g, "").slice(-8).toUpperCase();
}
