import type { Metadata } from "next";
import { notFound } from "next/navigation";
import { PageHeader } from "@/components/admin/page-header";
import { ProductForm } from "@/components/admin/products/product-form";
import { ApiError } from "@/lib/api/problem";
import type { Product } from "@/lib/api/types";
import { getProduct, listCategories } from "@/lib/catalog/products";
import { formatDateTime } from "@/lib/format";

export const metadata: Metadata = { title: "Edit product" };

export default async function EditProductPage({ params }: PageProps<"/admin/products/[id]">) {
  const { id } = await params;
  let product: Product;
  try {
    product = await getProduct(id);
  } catch (error) {
    if (error instanceof ApiError && (error.status === 404 || error.status === 400 || error.status === 422)) {
      notFound();
    }
    throw error;
  }
  const categories = await listCategories();
  return (
    <>
      <PageHeader title={product.name} description={`Version ${product.version}, last updated ${formatDateTime(product.updated_at)}`} />
      <ProductForm key={product.version} categories={categories} product={product} />
    </>
  );
}
