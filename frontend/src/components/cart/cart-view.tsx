"use client";

import { ShoppingBagIcon, Trash2Icon } from "lucide-react";
import Link from "next/link";
import { useCart, useCartHydrated } from "@/hooks/use-cart";
import { useCartRevalidation } from "@/hooks/use-cart-revalidation";
import { itemCount, lineTotal, maxQuantity, subtotal } from "@/lib/cart/cart";
import { formatMoney } from "@/lib/format";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardFooter, CardHeader, CardTitle } from "@/components/ui/card";
import { Empty, EmptyContent, EmptyDescription, EmptyHeader, EmptyMedia, EmptyTitle } from "@/components/ui/empty";
import { Separator } from "@/components/ui/separator";
import { Skeleton } from "@/components/ui/skeleton";
import { Spinner } from "@/components/ui/spinner";
import { ProductArt } from "@/components/catalog/product-art";
import { CartChanges } from "./cart-changes";
import { QuantityStepper } from "./quantity-stepper";

export function CartView() {
  const { lines, update, remove } = useCart();
  const hydrated = useCartHydrated();
  const { changes, checking } = useCartRevalidation();

  if (!hydrated) {
    return <Skeleton className="h-64 w-full rounded-xl" />;
  }

  if (lines.length === 0) {
    return (
      <div className="flex flex-col gap-6">
        <CartChanges changes={changes} />
        <Empty className="border border-dashed">
          <EmptyHeader>
            <EmptyMedia variant="icon">
              <ShoppingBagIcon />
            </EmptyMedia>
            <EmptyTitle>Your cart is empty</EmptyTitle>
            <EmptyDescription>Find something you like in the catalog.</EmptyDescription>
          </EmptyHeader>
          <EmptyContent>
            <Button asChild>
              <Link href="/">Browse catalog</Link>
            </Button>
          </EmptyContent>
        </Empty>
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-6">
      <CartChanges changes={changes} />
      <div className="grid gap-8 lg:grid-cols-[1fr_340px]">
        <ul className="divide-y rounded-xl border bg-card">
          {lines.map((line) => (
            <li key={line.productId} className="flex gap-4 p-4">
              <Link href={`/products/${line.productId}`} className="shrink-0">
                <ProductArt slug={line.categorySlug} name={line.name} className="size-20 rounded-lg sm:size-24" />
              </Link>
              <div className="flex min-w-0 flex-1 flex-col gap-2">
                <div className="flex items-start justify-between gap-3">
                  <div className="min-w-0">
                    <Link href={`/products/${line.productId}`} className="line-clamp-2 font-medium hover:underline">
                      {line.name}
                    </Link>
                    <p className="font-mono text-xs text-muted-foreground">{line.sku}</p>
                  </div>
                  <p className="font-semibold tabular-nums">{formatMoney(lineTotal(line))}</p>
                </div>
                <div className="mt-auto flex flex-wrap items-center justify-between gap-2">
                  <div className="flex items-center gap-3">
                    <QuantityStepper
                      size="sm"
                      value={line.quantity}
                      max={Math.max(1, maxQuantity(line.stock))}
                      onChange={(quantity) => update(line.productId, quantity)}
                      label={`Quantity of ${line.name}`}
                    />
                    <span className="text-sm text-muted-foreground tabular-nums">× {formatMoney(line.price)}</span>
                  </div>
                  <Button variant="ghost" size="sm" className="text-muted-foreground" onClick={() => remove(line.productId)}>
                    <Trash2Icon />
                    Remove
                  </Button>
                </div>
              </div>
            </li>
          ))}
        </ul>
        <Card className="h-fit lg:sticky lg:top-24">
          <CardHeader>
            <CardTitle>Order summary</CardTitle>
          </CardHeader>
          <CardContent className="flex flex-col gap-3 text-sm">
            <div className="flex justify-between">
              <span className="text-muted-foreground">Items</span>
              <span className="tabular-nums">{itemCount(lines)}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-muted-foreground">Shipping</span>
              <span>Free</span>
            </div>
            <Separator />
            <div className="flex justify-between text-base font-semibold">
              <span>Total</span>
              <span className="tabular-nums">{formatMoney(subtotal(lines))}</span>
            </div>
          </CardContent>
          <CardFooter className="flex flex-col gap-2">
            <Button size="lg" className="w-full" disabled={checking} asChild={!checking}>
              {checking ? (
                <span>
                  <Spinner />
                  Checking prices
                </span>
              ) : (
                <Link href="/checkout">Checkout</Link>
              )}
            </Button>
            <p className="text-center text-xs text-muted-foreground">Prices and stock are checked again when you pay.</p>
          </CardFooter>
        </Card>
      </div>
    </div>
  );
}
