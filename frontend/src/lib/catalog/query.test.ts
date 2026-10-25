import { describe, expect, it } from "vitest";
import { PAGE_SIZE, catalogHref, parseCatalogQuery } from "./query";

const category = "01a1707a-4dd4-71dc-b154-789730a74595";

describe("parseCatalogQuery", () => {
  it("keeps valid filters from the URL", () => {
    expect(
      parseCatalogQuery({ q: " lamp ", category_id: category, min_price: "10", max_price: "99.99", in_stock: "true", sort: "price_asc", page: "3" }),
    ).toEqual({ q: "lamp", category_id: category, min_price: "10", max_price: "99.99", in_stock: true, sort: "price_asc", page: 3, page_size: PAGE_SIZE });
  });

  it("drops malformed values instead of sending them to the backend", () => {
    expect(
      parseCatalogQuery({ category_id: "abc", min_price: "-5", max_price: "1e9", in_stock: "yes", sort: "random", page: "0" }),
    ).toEqual({ page_size: PAGE_SIZE });
  });

  it("ignores relevance sort without a search text", () => {
    expect(parseCatalogQuery({ sort: "relevance" }).sort).toBeUndefined();
    expect(parseCatalogQuery({ q: "mug", sort: "relevance" }).sort).toBe("relevance");
  });
});

describe("catalogHref", () => {
  it("resets the page when a filter changes and omits default values", () => {
    const query = parseCatalogQuery({ q: "mug", page: "4", sort: "relevance" });
    expect(catalogHref(query, { in_stock: true })).toBe("/?q=mug&in_stock=true");
    expect(catalogHref(query, { page: 5 })).toBe("/?q=mug&page=5");
    expect(catalogHref({}, { page: 1 })).toBe("/");
  });
});
