import { ChevronRightIcon } from "lucide-react";
import Link from "next/link";
import type { Order } from "@/lib/api/types";
import { formatDateTime, formatMoney } from "@/lib/format";
import { shortOrderId } from "@/lib/orders";
import { OrderStatusBadge } from "./order-status-badge";

export function OrderList({ orders, showCustomer = false }: { orders: Order[]; showCustomer?: boolean }) {
  return (
    <ul className="divide-y rounded-xl border bg-card">
      {orders.map((order) => {
        const items = order.items.reduce((sum, item) => sum + item.quantity, 0);
        return (
          <li key={order.id}>
            <Link href={`/orders/${order.id}`} className="flex items-center gap-4 p-4 transition-colors hover:bg-muted/50">
              <div className="grid flex-1 gap-1 sm:grid-cols-[1fr_auto_auto] sm:items-center sm:gap-6">
                <div className="min-w-0">
                  <p className="font-medium">
                    Order <span className="font-mono">#{shortOrderId(order.id)}</span>
                  </p>
                  <p className="truncate text-sm text-muted-foreground">
                    {showCustomer ? `${order.customer_email} · ` : ""}
                    {formatDateTime(order.created_at)} · {items} {items === 1 ? "item" : "items"}
                  </p>
                </div>
                <OrderStatusBadge status={order.status} />
                <p className="font-semibold tabular-nums sm:w-28 sm:text-right">{formatMoney(order.total)}</p>
              </div>
              <ChevronRightIcon className="size-4 shrink-0 text-muted-foreground" />
            </Link>
          </li>
        );
      })}
    </ul>
  );
}
