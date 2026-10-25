"use client";

import { useEffect, useState } from "react";
import { reconcile, type CartChange } from "@/lib/cart/cart";
import { fetchCurrentProducts } from "@/lib/cart/revalidate";
import { useCart, useCartHydrated } from "./use-cart";

export function useCartRevalidation() {
  const { lines, snapshot, replace } = useCart();
  const hydrated = useCartHydrated();
  const [changes, setChanges] = useState<CartChange[]>([]);
  const [checking, setChecking] = useState(true);

  useEffect(() => {
    if (!hydrated) {
      return;
    }
    const ids = snapshot().map((line) => line.productId);
    if (ids.length === 0) {
      return;
    }
    let cancelled = false;
    fetchCurrentProducts(ids)
      .then((fresh) => {
        if (cancelled) {
          return;
        }
        const result = reconcile(snapshot(), fresh);
        replace(result.lines);
        setChanges(result.changes);
      })
      .catch(() => {})
      .finally(() => {
        if (!cancelled) {
          setChecking(false);
        }
      });
    return () => {
      cancelled = true;
    };
  }, [hydrated, snapshot, replace]);

  return { changes, setChanges, checking: checking && lines.length > 0 };
}
