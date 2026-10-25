import type { Metadata } from "next";
import { CategoryManager } from "@/components/admin/category-manager";
import { PageHeader } from "@/components/admin/page-header";
import { listCategories } from "@/lib/catalog/products";

export const metadata: Metadata = { title: "Categories" };

export default async function AdminCategoriesPage() {
  const categories = await listCategories();
  return (
    <>
      <PageHeader title="Categories" description={`${categories.length} categories`} />
      <CategoryManager categories={categories} />
    </>
  );
}
