import type { CheckoutIssue } from "@/lib/api/types";
import type { CartChange, CartLine } from "@/lib/cart/cart";
import { maxQuantity } from "@/lib/cart/cart";

export function applyCheckoutIssues(lines: CartLine[], issues: CheckoutIssue[]): { lines: CartLine[]; changes: CartChange[] } {
  const changes: CartChange[] = [];
  let next = [...lines];
  for (const issue of issues) {
    const line = next.find((item) => item.productId === issue.product_id);
    if (!line) {
      continue;
    }
    const base = { productId: line.productId, name: line.name };
    switch (issue.code) {
      case "price_changed":
        if (issue.current_price) {
          changes.push({ kind: "price_changed", ...base, from: line.price, to: issue.current_price });
          next = next.map((item) => (item === line ? { ...item, price: issue.current_price! } : item));
        }
        break;
      case "insufficient_stock": {
        const available = issue.available_quantity ?? 0;
        const limit = maxQuantity(available);
        if (limit === 0) {
          changes.push({ kind: "out_of_stock", ...base });
          next = next.filter((item) => item !== line);
        } else {
          changes.push({ kind: "quantity_reduced", ...base, from: line.quantity, to: Math.min(line.quantity, limit) });
          next = next.map((item) => (item === line ? { ...item, stock: available, quantity: Math.min(item.quantity, limit) } : item));
        }
        break;
      }
      case "out_of_stock":
        changes.push({ kind: "out_of_stock", ...base });
        next = next.filter((item) => item !== line);
        break;
      case "product_unavailable":
        changes.push({ kind: "unavailable", ...base });
        next = next.filter((item) => item !== line);
        break;
    }
  }
  return { lines: next, changes };
}
