"use client";

import { useState } from "react";
import { useCart, useCartHydrated } from "@/hooks/use-cart";
import type { Product } from "@/lib/api/types";
import { maxQuantity } from "@/lib/cart/cart";
import { AddToCartButton } from "@/components/cart/add-to-cart-button";
import { QuantityStepper } from "@/components/cart/quantity-stepper";

export function ProductPurchase({ product, canBuy }: { product: Product; canBuy: boolean }) {
  const { lines } = useCart();
  const hydrated = useCartHydrated();
  const inCart = lines.find((line) => line.productId === product.id)?.quantity ?? 0;
  const available = Math.max(0, maxQuantity(product.available) - inCart);
  const [quantity, setQuantity] = useState(1);
  const value = Math.min(quantity, Math.max(1, available));

  return (
    <div className="flex flex-col gap-3">
      <div className="flex flex-wrap items-center gap-3">
        {canBuy && product.available > 0 && (
          <QuantityStepper value={value} max={Math.max(1, available)} onChange={setQuantity} />
        )}
        <AddToCartButton
          product={product}
          quantity={value}
          canBuy={canBuy}
          size="lg"
          className="min-w-44"
          disabled={hydrated && available === 0}
          onAdded={() => setQuantity(1)}
        />
      </div>
      {hydrated && inCart > 0 && (
        <p className="text-sm text-muted-foreground">
          {inCart} already in your cart{available === 0 ? ", that is the maximum you can order" : ""}.
        </p>
      )}
    </div>
  );
}
