import type { ProductQuery, ProductSort } from "@/lib/api/types";
import { isMoney } from "@/lib/money";

export const PAGE_SIZE = 24;

export const SORT_OPTIONS: { value: ProductSort; label: string }[] = [
  { value: "relevance", label: "Best match" },
  { value: "newest", label: "Newest" },
  { value: "price_asc", label: "Price: low to high" },
  { value: "price_desc", label: "Price: high to low" },
  { value: "name_asc", label: "Name: A to Z" },
];

const SORTS = new Set<string>(SORT_OPTIONS.map((option) => option.value));
const UUID_PATTERN = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;
const MAX_PAGE = 10_000;
const MAX_QUERY_LENGTH = 200;

type Params = Record<string, string | string[] | undefined>;

function first(params: Params, key: string): string | undefined {
  const value = params[key];
  const text = Array.isArray(value) ? value[0] : value;
  return text?.trim() || undefined;
}

export function parseCatalogQuery(params: Params): ProductQuery {
  const query: ProductQuery = { page_size: PAGE_SIZE };
  const q = first(params, "q");
  if (q) {
    query.q = q.slice(0, MAX_QUERY_LENGTH);
  }
  const category = first(params, "category_id");
  if (category && UUID_PATTERN.test(category)) {
    query.category_id = category;
  }
  const min = first(params, "min_price");
  if (min && isMoney(min)) {
    query.min_price = min;
  }
  const max = first(params, "max_price");
  if (max && isMoney(max)) {
    query.max_price = max;
  }
  if (first(params, "in_stock") === "true") {
    query.in_stock = true;
  }
  const sort = first(params, "sort");
  if (sort && SORTS.has(sort) && (sort !== "relevance" || query.q)) {
    query.sort = sort as ProductSort;
  }
  const page = Number(first(params, "page"));
  if (Number.isInteger(page) && page > 1) {
    query.page = Math.min(page, MAX_PAGE);
  }
  return query;
}

export function defaultSort(query: Pick<ProductQuery, "q">): ProductSort {
  return query.q ? "relevance" : "newest";
}

export function catalogHref(query: ProductQuery, changes: Partial<ProductQuery> = {}): string {
  const merged: ProductQuery = { ...query, ...changes };
  if (!("page" in changes)) {
    delete merged.page;
  }
  const params = new URLSearchParams();
  if (merged.q) params.set("q", merged.q);
  if (merged.category_id) params.set("category_id", merged.category_id);
  if (merged.min_price) params.set("min_price", merged.min_price);
  if (merged.max_price) params.set("max_price", merged.max_price);
  if (merged.in_stock) params.set("in_stock", "true");
  if (merged.sort && merged.sort !== defaultSort(merged)) params.set("sort", merged.sort);
  if (merged.page && merged.page > 1) params.set("page", String(merged.page));
  const text = params.toString();
  return text ? `/?${text}` : "/";
}

export function hasFilters(query: ProductQuery): boolean {
  return Boolean(query.q || query.category_id || query.min_price || query.max_price || query.in_stock);
}

export function pageWindow(current: number, total: number, size = 5): (number | "gap")[] {
  if (total <= size + 2) {
    return Array.from({ length: total }, (_, i) => i + 1);
  }
  const half = Math.floor(size / 2);
  const start = Math.max(2, Math.min(current - half, total - size));
  const end = Math.min(total - 1, start + size - 1);
  const pages: (number | "gap")[] = [1];
  if (start > 2) pages.push("gap");
  for (let page = start; page <= end; page++) pages.push(page);
  if (end < total - 1) pages.push("gap");
  pages.push(total);
  return pages;
}
