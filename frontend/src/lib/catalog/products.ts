import "server-only";
import type { Category, Page, Product, ProductQuery } from "@/lib/api/types";
import { serverApi } from "@/lib/server/api";

export function listProducts(query: ProductQuery) {
  return serverApi<Page<Product>>("/products", { query: { ...query } });
}

export async function listCategories(): Promise<Category[]> {
  const { items } = await serverApi<{ items: Category[] }>("/categories");
  return items;
}

export function getProduct(id: string) {
  return serverApi<Product>(`/products/${encodeURIComponent(id)}`);
}
