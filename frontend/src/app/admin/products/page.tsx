import { ExternalLinkIcon, PencilIcon, PlusIcon } from "lucide-react";
import type { Metadata } from "next";
import Link from "next/link";
import { PageHeader } from "@/components/admin/page-header";
import { DeleteProductButton } from "@/components/admin/products/delete-product-button";
import { ProductFilters } from "@/components/admin/products/product-filters";
import { FormAlert } from "@/components/auth/form-alert";
import { LOW_STOCK } from "@/components/catalog/stock-badge";
import { Pager } from "@/components/pager";
import { Button } from "@/components/ui/button";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { errorMessage } from "@/lib/api/problem";
import type { Category, Page, Product } from "@/lib/api/types";
import { catalogHref, parseCatalogQuery } from "@/lib/catalog/query";
import { listCategories, listProducts } from "@/lib/catalog/products";
import { formatDate, formatMoney } from "@/lib/format";
import { cn } from "@/lib/utils";

export const metadata: Metadata = { title: "Products" };

export default async function AdminProductsPage({ searchParams }: PageProps<"/admin/products">) {
  const query = { ...parseCatalogQuery(await searchParams), page_size: 20 };
  const [products, categories] = await Promise.allSettled([listProducts(query), listCategories()]);
  const page: Page<Product> | null = products.status === "fulfilled" ? products.value : null;
  const categoryList: Category[] = categories.status === "fulfilled" ? categories.value : [];

  return (
    <>
      <PageHeader
        title="Products"
        description={page ? `${page.total} products in the catalog` : undefined}
        actions={
          <Button asChild>
            <Link href="/admin/products/new">
              <PlusIcon />
              New product
            </Link>
          </Button>
        }
      />
      <ProductFilters query={query} categories={categoryList} />
      {products.status === "rejected" && <FormAlert message={errorMessage(products.reason)} />}
      {page && (
        <div className="rounded-xl border bg-card">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="pl-4">SKU</TableHead>
                <TableHead>Name</TableHead>
                <TableHead className="hidden md:table-cell">Category</TableHead>
                <TableHead className="text-right">Price</TableHead>
                <TableHead className="text-right">Stock</TableHead>
                <TableHead className="hidden lg:table-cell">Updated</TableHead>
                <TableHead className="pr-4 text-right">
                  <span className="sr-only">Actions</span>
                </TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {page.items.length === 0 && (
                <TableRow>
                  <TableCell colSpan={7} className="h-24 text-center text-muted-foreground">
                    No products match the filters.
                  </TableCell>
                </TableRow>
              )}
              {page.items.map((product) => (
                <TableRow key={product.id}>
                  <TableCell className="pl-4 font-mono text-xs">{product.sku}</TableCell>
                  <TableCell className="max-w-64">
                    <Link href={`/admin/products/${product.id}`} className="line-clamp-1 font-medium hover:underline">
                      {product.name}
                    </Link>
                  </TableCell>
                  <TableCell className="hidden text-muted-foreground md:table-cell">{product.category?.name}</TableCell>
                  <TableCell className="text-right tabular-nums">{formatMoney(product.price)}</TableCell>
                  <TableCell
                    className={cn(
                      "text-right tabular-nums",
                      product.stock === 0 && "text-destructive",
                      product.stock > 0 && product.stock <= LOW_STOCK && "text-amber-600",
                    )}
                  >
                    {product.stock}
                  </TableCell>
                  <TableCell className="hidden text-muted-foreground lg:table-cell">{formatDate(product.updated_at)}</TableCell>
                  <TableCell className="pr-4">
                    <div className="flex justify-end gap-1">
                      <Button variant="ghost" size="icon-sm" asChild>
                        <Link href={`/products/${product.id}`} aria-label={`View ${product.name} in the store`} target="_blank">
                          <ExternalLinkIcon />
                        </Link>
                      </Button>
                      <Button variant="ghost" size="icon-sm" asChild>
                        <Link href={`/admin/products/${product.id}`} aria-label={`Edit ${product.name}`}>
                          <PencilIcon />
                        </Link>
                      </Button>
                      <DeleteProductButton id={product.id} name={product.name} />
                    </div>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      )}
      {page && <Pager page={page.page} totalPages={page.total_pages} href={(target) => catalogHref(query, { page: target }, "/admin/products")} />}
    </>
  );
}
