import type { Metadata } from "next";
import { PageHeader } from "@/components/admin/page-header";
import { ProductForm } from "@/components/admin/products/product-form";
import { listCategories } from "@/lib/catalog/products";

export const metadata: Metadata = { title: "New product" };

export default async function NewProductPage() {
  const categories = await listCategories();
  return (
    <>
      <PageHeader title="New product" description="It appears in the catalog as soon as it is saved" />
      <ProductForm categories={categories} />
    </>
  );
}
