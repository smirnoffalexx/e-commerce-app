import Link from "next/link";
import type { OrderStatus } from "@/lib/api/types";
import { ORDER_STATUSES, orderListHref, orderStatusLabels } from "@/lib/orders";
import { cn } from "@/lib/utils";

export function OrderStatusTabs({ base, status, extra }: { base: string; status?: OrderStatus; extra?: Record<string, string> }) {
  const tabs: { value?: OrderStatus; label: string }[] = [{ label: "All" }, ...ORDER_STATUSES.map((value) => ({ value, label: orderStatusLabels[value] }))];
  return (
    <nav className="inline-flex rounded-lg bg-muted p-1" aria-label="Filter by status">
      {tabs.map((tab) => {
        const href = orderListHref(base, { status: tab.value });
        const params = extra ? new URLSearchParams(extra).toString() : "";
        return (
          <Link
            key={tab.label}
            href={params ? `${href}${href.includes("?") ? "&" : "?"}${params}` : href}
            aria-current={tab.value === status ? "page" : undefined}
            className={cn(
              "rounded-md px-3 py-1.5 text-sm font-medium text-muted-foreground transition-colors hover:text-foreground",
              tab.value === status && "bg-background text-foreground shadow-sm",
            )}
          >
            {tab.label}
          </Link>
        );
      })}
    </nav>
  );
}
