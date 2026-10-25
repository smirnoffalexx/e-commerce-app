import { PackageSearchIcon } from "lucide-react";
import Link from "next/link";
import { CatalogPagination } from "@/components/catalog/catalog-pagination";
import { CatalogView } from "@/components/catalog/catalog-view";
import { ProductCard } from "@/components/catalog/product-card";
import { FormAlert } from "@/components/auth/form-alert";
import { Button } from "@/components/ui/button";
import { Empty, EmptyContent, EmptyDescription, EmptyHeader, EmptyMedia, EmptyTitle } from "@/components/ui/empty";
import { errorMessage } from "@/lib/api/problem";
import type { Category, Page, Product } from "@/lib/api/types";
import { hasFilters, parseCatalogQuery } from "@/lib/catalog/query";
import { listCategories, listProducts } from "@/lib/catalog/products";
import { canPurchase } from "@/lib/roles";
import { getCurrentUser } from "@/lib/server/api";

export default async function CatalogPage({ searchParams }: PageProps<"/">) {
  const query = parseCatalogQuery(await searchParams);
  const [products, categories, user] = await Promise.allSettled([listProducts(query), listCategories(), getCurrentUser()]);
  const page: Page<Product> | null = products.status === "fulfilled" ? products.value : null;
  const categoryList: Category[] = categories.status === "fulfilled" ? categories.value : [];
  const canBuy = canPurchase(user.status === "fulfilled" ? user.value : null);

  return (
    <CatalogView query={query} categories={categoryList} total={page?.total ?? null}>
      {products.status === "rejected" && <FormAlert message={errorMessage(products.reason)} />}
      {page && page.items.length === 0 && (
        <Empty className="border border-dashed">
          <EmptyHeader>
            <EmptyMedia variant="icon">
              <PackageSearchIcon />
            </EmptyMedia>
            <EmptyTitle>No products found</EmptyTitle>
            <EmptyDescription>
              {hasFilters(query) ? "Try a different search or remove some filters." : "The catalog is empty for now."}
            </EmptyDescription>
          </EmptyHeader>
          {hasFilters(query) && (
            <EmptyContent>
              <Button variant="outline" asChild>
                <Link href="/">Clear filters</Link>
              </Button>
            </EmptyContent>
          )}
        </Empty>
      )}
      {page && page.items.length > 0 && (
        <>
          <div className="grid gap-5 sm:grid-cols-2 xl:grid-cols-3">
            {page.items.map((product) => (
              <ProductCard key={product.id} product={product} canBuy={canBuy} />
            ))}
          </div>
          <CatalogPagination query={query} page={page.page} totalPages={page.total_pages} />
        </>
      )}
    </CatalogView>
  );
}
