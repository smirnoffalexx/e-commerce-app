import { PackageIcon, ScaleIcon, TagIcon } from "lucide-react";
import type { Metadata } from "next";
import Link from "next/link";
import { notFound } from "next/navigation";
import { cache } from "react";
import { ProductArt } from "@/components/catalog/product-art";
import { ProductCard } from "@/components/catalog/product-card";
import { ProductPurchase } from "@/components/catalog/product-purchase";
import { StockBadge } from "@/components/catalog/stock-badge";
import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbLink,
  BreadcrumbList,
  BreadcrumbPage,
  BreadcrumbSeparator,
} from "@/components/ui/breadcrumb";
import { Separator } from "@/components/ui/separator";
import { ApiError } from "@/lib/api/problem";
import { getProduct, listProducts } from "@/lib/catalog/products";
import { formatMoney } from "@/lib/format";
import { canPurchase } from "@/lib/roles";
import { getCurrentUser } from "@/lib/server/api";

const loadProduct = cache(async (id: string) => {
  try {
    return await getProduct(id);
  } catch (error) {
    if (error instanceof ApiError && (error.status === 404 || error.status === 400 || error.status === 422)) {
      notFound();
    }
    throw error;
  }
});

export async function generateMetadata({ params }: PageProps<"/products/[id]">): Promise<Metadata> {
  const product = await loadProduct((await params).id);
  return { title: product.name, description: product.description };
}

export default async function ProductPage({ params }: PageProps<"/products/[id]">) {
  const product = await loadProduct((await params).id);
  const [user, related] = await Promise.all([
    getCurrentUser(),
    listProducts({ category_id: product.category_id, in_stock: true, page_size: 5 }).catch(() => null),
  ]);
  const canBuy = canPurchase(user);
  const others = related?.items.filter((item) => item.id !== product.id).slice(0, 4) ?? [];

  return (
    <div className="mx-auto w-full max-w-7xl px-4 py-8 sm:px-6">
      <Breadcrumb>
        <BreadcrumbList>
          <BreadcrumbItem>
            <BreadcrumbLink asChild>
              <Link href="/">Catalog</Link>
            </BreadcrumbLink>
          </BreadcrumbItem>
          {product.category && (
            <>
              <BreadcrumbSeparator />
              <BreadcrumbItem>
                <BreadcrumbLink asChild>
                  <Link href={`/?category_id=${product.category.id}`}>{product.category.name}</Link>
                </BreadcrumbLink>
              </BreadcrumbItem>
            </>
          )}
          <BreadcrumbSeparator />
          <BreadcrumbItem>
            <BreadcrumbPage className="line-clamp-1">{product.name}</BreadcrumbPage>
          </BreadcrumbItem>
        </BreadcrumbList>
      </Breadcrumb>

      <div className="mt-6 grid gap-10 lg:grid-cols-2">
        <ProductArt slug={product.category?.slug} name={product.name} className="aspect-square rounded-2xl border" />
        <div className="flex flex-col gap-6">
          <div className="flex flex-col gap-3">
            <div className="flex items-center gap-2">
              {product.category && <span className="text-sm font-medium text-primary">{product.category.name}</span>}
              <StockBadge stock={product.available} />
            </div>
            <h1 className="text-3xl font-semibold tracking-tight text-balance sm:text-4xl">{product.name}</h1>
            <p className="text-3xl font-semibold tabular-nums">{formatMoney(product.price)}</p>
          </div>
          <p className="text-lg leading-relaxed text-muted-foreground">{product.description}</p>
          <ProductPurchase product={product} canBuy={canBuy} />
          <Separator />
          <dl className="grid grid-cols-1 gap-4 text-sm sm:grid-cols-3">
            <Spec icon={<TagIcon />} label="SKU" value={<span className="font-mono">{product.sku}</span>} />
            <Spec icon={<ScaleIcon />} label="Weight" value={`${Number(product.weight_kg)} kg`} />
            <Spec icon={<PackageIcon />} label="Available" value={product.available > 0 ? `${product.available} pcs` : "None"} />
          </dl>
        </div>
      </div>

      {others.length > 0 && (
        <section className="mt-16">
          <h2 className="text-xl font-semibold tracking-tight">More from {product.category?.name}</h2>
          <div className="mt-5 grid gap-5 sm:grid-cols-2 lg:grid-cols-4">
            {others.map((item) => (
              <ProductCard key={item.id} product={item} canBuy={canBuy} />
            ))}
          </div>
        </section>
      )}
    </div>
  );
}

function Spec({ icon, label, value }: { icon: React.ReactNode; label: string; value: React.ReactNode }) {
  return (
    <div className="flex flex-col gap-1 rounded-lg border bg-card p-3">
      <dt className="flex items-center gap-1.5 text-muted-foreground [&_svg]:size-4">
        {icon}
        {label}
      </dt>
      <dd className="font-medium">{value}</dd>
    </div>
  );
}
