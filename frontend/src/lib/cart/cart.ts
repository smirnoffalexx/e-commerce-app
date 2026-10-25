import type { Product } from "@/lib/api/types";
import { fromCents, toCents } from "@/lib/money";

export const MAX_CART_LINES = 50;
export const MAX_LINE_QUANTITY = 100;

export interface CartLine {
  productId: string;
  sku: string;
  name: string;
  categorySlug?: string;
  price: string;
  stock: number;
  quantity: number;
}

export type AddOutcome = "added" | "limited" | "cart_full" | "out_of_stock";

export function maxQuantity(stock: number): number {
  return Math.max(0, Math.min(MAX_LINE_QUANTITY, stock));
}

export function lineFromProduct(product: Product, quantity: number): CartLine {
  return {
    productId: product.id,
    sku: product.sku,
    name: product.name,
    categorySlug: product.category?.slug,
    price: product.price,
    stock: product.stock,
    quantity,
  };
}

export function addToCart(lines: CartLine[], product: Product, quantity: number): { lines: CartLine[]; outcome: AddOutcome } {
  const limit = maxQuantity(product.stock);
  if (limit === 0) {
    return { lines, outcome: "out_of_stock" };
  }
  const index = lines.findIndex((line) => line.productId === product.id);
  if (index === -1 && lines.length >= MAX_CART_LINES) {
    return { lines, outcome: "cart_full" };
  }
  const current = index === -1 ? 0 : lines[index].quantity;
  const wanted = current + Math.max(1, Math.floor(quantity));
  const next = Math.min(wanted, limit);
  const line = lineFromProduct(product, next);
  const updated = index === -1 ? [...lines, line] : lines.map((item, i) => (i === index ? line : item));
  return { lines: updated, outcome: next < wanted ? "limited" : "added" };
}

export function setQuantity(lines: CartLine[], productId: string, quantity: number): CartLine[] {
  return lines.map((line) => {
    if (line.productId !== productId) {
      return line;
    }
    const limit = Math.max(1, maxQuantity(line.stock));
    return { ...line, quantity: Math.min(Math.max(1, Math.floor(quantity)), limit) };
  });
}

export function removeFromCart(lines: CartLine[], productId: string): CartLine[] {
  return lines.filter((line) => line.productId !== productId);
}

export function itemCount(lines: CartLine[]): number {
  return lines.reduce((sum, line) => sum + line.quantity, 0);
}

export function lineTotal(line: CartLine): string {
  return fromCents(toCents(line.price) * line.quantity);
}

export function subtotal(lines: CartLine[]): string {
  return fromCents(lines.reduce((sum, line) => sum + toCents(line.price) * line.quantity, 0));
}

export type CartChange =
  | { kind: "price_changed"; productId: string; name: string; from: string; to: string }
  | { kind: "quantity_reduced"; productId: string; name: string; from: number; to: number }
  | { kind: "out_of_stock"; productId: string; name: string }
  | { kind: "unavailable"; productId: string; name: string };

export function reconcile(lines: CartLine[], fresh: Map<string, Product | null>): { lines: CartLine[]; changes: CartChange[] } {
  const changes: CartChange[] = [];
  const next: CartLine[] = [];
  for (const line of lines) {
    if (!fresh.has(line.productId)) {
      next.push(line);
      continue;
    }
    const product = fresh.get(line.productId);
    if (!product) {
      changes.push({ kind: "unavailable", productId: line.productId, name: line.name });
      continue;
    }
    if (toCents(product.price) !== toCents(line.price)) {
      changes.push({ kind: "price_changed", productId: line.productId, name: product.name, from: line.price, to: product.price });
    }
    const limit = maxQuantity(product.stock);
    if (limit === 0) {
      changes.push({ kind: "out_of_stock", productId: line.productId, name: product.name });
      continue;
    }
    if (line.quantity > limit) {
      changes.push({ kind: "quantity_reduced", productId: line.productId, name: product.name, from: line.quantity, to: limit });
    }
    next.push({ ...lineFromProduct(product, Math.min(line.quantity, limit)) });
  }
  return { lines: next, changes };
}
