import { CircleCheckIcon, CircleXIcon, ClockIcon } from "lucide-react";
import type { OrderStatus } from "@/lib/api/types";
import { orderStatusLabels } from "@/lib/orders";
import { cn } from "@/lib/utils";
import { Badge } from "@/components/ui/badge";

const styles: Record<OrderStatus, string> = {
  PAID: "border-emerald-500/40 text-emerald-700 dark:text-emerald-400",
  PENDING: "border-amber-500/40 text-amber-700 dark:text-amber-400",
  FAILED: "border-destructive/40 text-destructive",
};

const icons = { PAID: CircleCheckIcon, PENDING: ClockIcon, FAILED: CircleXIcon };

export function OrderStatusBadge({ status }: { status: OrderStatus }) {
  const Icon = icons[status];
  return (
    <Badge variant="outline" className={cn("gap-1", styles[status])}>
      <Icon />
      {orderStatusLabels[status]}
    </Badge>
  );
}
