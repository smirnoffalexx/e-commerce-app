import { Badge } from "@/components/ui/badge";

export const LOW_STOCK = 5;

export function StockBadge({ stock }: { stock: number }) {
  if (stock <= 0) {
    return <Badge variant="secondary">Out of stock</Badge>;
  }
  if (stock <= LOW_STOCK) {
    return (
      <Badge variant="outline" className="border-amber-500/40 text-amber-700 dark:text-amber-400">
        Only {stock} left
      </Badge>
    );
  }
  return (
    <Badge variant="outline" className="border-emerald-500/40 text-emerald-700 dark:text-emerald-400">
      In stock
    </Badge>
  );
}
