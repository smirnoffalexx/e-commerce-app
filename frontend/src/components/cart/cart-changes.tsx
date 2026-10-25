import { InfoIcon } from "lucide-react";
import type { CartChange } from "@/lib/cart/cart";
import { formatMoney } from "@/lib/format";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";

export function describeChange(change: CartChange): string {
  switch (change.kind) {
    case "price_changed":
      return `${change.name}: price changed from ${formatMoney(change.from)} to ${formatMoney(change.to)}.`;
    case "quantity_reduced":
      return `${change.name}: only ${change.to} available, quantity reduced from ${change.from}.`;
    case "out_of_stock":
      return `${change.name} is out of stock and was removed.`;
    case "unavailable":
      return `${change.name} is no longer sold and was removed.`;
  }
}

export function CartChanges({ changes, title = "Your cart was updated" }: { changes: CartChange[]; title?: string }) {
  if (changes.length === 0) {
    return null;
  }
  return (
    <Alert className="border-amber-500/40 bg-amber-500/5">
      <InfoIcon className="text-amber-600" />
      <AlertTitle>{title}</AlertTitle>
      <AlertDescription>
        <ul className="mt-1 list-disc space-y-0.5 pl-4">
          {changes.map((change) => (
            <li key={`${change.kind}-${change.productId}`}>{describeChange(change)}</li>
          ))}
        </ul>
      </AlertDescription>
    </Alert>
  );
}
