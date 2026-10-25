import Link from "next/link";
import type { Product } from "@/lib/api/types";
import { formatMoney } from "@/lib/format";
import { AddToCartButton } from "@/components/cart/add-to-cart-button";
import { ProductArt } from "./product-art";
import { StockBadge } from "./stock-badge";

export function ProductCard({ product, canBuy }: { product: Product; canBuy: boolean }) {
  return (
    <article className="group relative flex flex-col overflow-hidden rounded-xl border bg-card shadow-xs transition-shadow hover:shadow-md">
      <ProductArt
        slug={product.category?.slug}
        name={product.name}
        className="aspect-[4/3] transition-transform duration-300 group-hover:scale-[1.03]"
      />
      <div className="flex flex-1 flex-col gap-3 p-4">
        <div className="flex items-start justify-between gap-2">
          <p className="text-xs font-medium tracking-wide text-muted-foreground uppercase">{product.category?.name}</p>
          <StockBadge stock={product.available} />
        </div>
        <h2 className="line-clamp-2 font-medium leading-snug">
          <Link href={`/products/${product.id}`} className="after:absolute after:inset-0">
            {product.name}
          </Link>
        </h2>
        <p className="line-clamp-2 text-sm text-muted-foreground">{product.description}</p>
        <div className="mt-auto flex items-center justify-between gap-2 pt-1">
          <span className="text-lg font-semibold tabular-nums">{formatMoney(product.price)}</span>
          <AddToCartButton product={product} canBuy={canBuy} size="sm" className="relative z-10">
            Add
          </AddToCartButton>
        </div>
      </div>
    </article>
  );
}
