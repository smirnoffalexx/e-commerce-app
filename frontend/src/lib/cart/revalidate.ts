import { api } from "@/lib/api/client";
import { ApiError } from "@/lib/api/problem";
import type { Product } from "@/lib/api/types";

export async function fetchCurrentProducts(ids: string[]): Promise<Map<string, Product | null>> {
  const results = await Promise.allSettled(ids.map((id) => api<Product>(`/products/${encodeURIComponent(id)}`)));
  const fresh = new Map<string, Product | null>();
  results.forEach((result, index) => {
    if (result.status === "fulfilled") {
      fresh.set(ids[index], result.value);
    } else if (result.reason instanceof ApiError && result.reason.status === 404) {
      fresh.set(ids[index], null);
    }
  });
  return fresh;
}
