import Link from "next/link";
import type { OrderStatus } from "@/lib/api/types";
import { ORDER_STATUSES, orderListHref, orderStatusLabels, type OrderListQuery } from "@/lib/orders";
import { cn } from "@/lib/utils";

export function OrderStatusTabs({ base, query }: { base: string; query: OrderListQuery }) {
  const tabs: { value?: OrderStatus; label: string }[] = [{ label: "All" }, ...ORDER_STATUSES.map((value) => ({ value, label: orderStatusLabels[value] }))];
  return (
    <nav className="inline-flex rounded-lg bg-muted p-1" aria-label="Filter by status">
      {tabs.map((tab) => (
        <Link
          key={tab.label}
          href={orderListHref(base, { user_id: query.user_id, status: tab.value })}
          aria-current={tab.value === query.status ? "page" : undefined}
          className={cn(
            "rounded-md px-3 py-1.5 text-sm font-medium text-muted-foreground transition-colors hover:text-foreground",
            tab.value === query.status && "bg-background text-foreground shadow-sm",
          )}
        >
          {tab.label}
        </Link>
      ))}
    </nav>
  );
}
