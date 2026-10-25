import { z } from "zod";
import { isMoney } from "@/lib/money";
import { MAX_CART_LINES, MAX_LINE_QUANTITY, type CartLine } from "./cart";

export const CART_STORAGE_KEY = "ecommerce-app:cart:v1";

const lineSchema = z.object({
  productId: z.string().min(1),
  sku: z.string(),
  name: z.string(),
  categorySlug: z.string().optional(),
  price: z.string().refine(isMoney),
  stock: z.number().int().nonnegative(),
  quantity: z.number().int().min(1).max(MAX_LINE_QUANTITY),
});

export function parseCart(raw: string | null): CartLine[] {
  if (!raw) {
    return [];
  }
  try {
    const parsed = z.array(z.unknown()).parse(JSON.parse(raw));
    const lines: CartLine[] = [];
    const seen = new Set<string>();
    for (const item of parsed) {
      const result = lineSchema.safeParse(item);
      if (result.success && !seen.has(result.data.productId) && lines.length < MAX_CART_LINES) {
        seen.add(result.data.productId);
        lines.push(result.data);
      }
    }
    return lines;
  } catch {
    return [];
  }
}
