"use client";

import { useCallback, useSyncExternalStore } from "react";
import type { Product } from "@/lib/api/types";
import { addToCart, removeFromCart, setQuantity, type AddOutcome, type CartLine } from "@/lib/cart/cart";
import { CART_STORAGE_KEY, parseCart } from "@/lib/cart/storage";

const EMPTY: CartLine[] = [];
const listeners = new Set<() => void>();
let cachedRaw: string | null = null;
let cachedLines: CartLine[] = EMPTY;

function read(): CartLine[] {
  let raw: string | null = null;
  try {
    raw = window.localStorage.getItem(CART_STORAGE_KEY);
  } catch {
    return cachedLines;
  }
  if (raw !== cachedRaw) {
    cachedRaw = raw;
    cachedLines = parseCart(raw);
  }
  return cachedLines;
}

function write(lines: CartLine[]) {
  cachedLines = lines;
  cachedRaw = JSON.stringify(lines);
  try {
    if (lines.length === 0) {
      window.localStorage.removeItem(CART_STORAGE_KEY);
      cachedRaw = null;
    } else {
      window.localStorage.setItem(CART_STORAGE_KEY, cachedRaw);
    }
  } catch {}
  listeners.forEach((listener) => listener());
}

function subscribe(listener: () => void) {
  listeners.add(listener);
  const onStorage = (event: StorageEvent) => {
    if (event.key === CART_STORAGE_KEY || event.key === null) {
      listener();
    }
  };
  window.addEventListener("storage", onStorage);
  return () => {
    listeners.delete(listener);
    window.removeEventListener("storage", onStorage);
  };
}

export function clearCart() {
  write(EMPTY);
}

export function useCart() {
  const lines = useSyncExternalStore(subscribe, read, () => EMPTY);

  const add = useCallback((product: Product, quantity = 1): AddOutcome => {
    const result = addToCart(read(), product, quantity);
    if (result.lines !== read()) {
      write(result.lines);
    }
    return result.outcome;
  }, []);

  const update = useCallback((productId: string, quantity: number) => write(setQuantity(read(), productId, quantity)), []);
  const remove = useCallback((productId: string) => write(removeFromCart(read(), productId)), []);
  const replace = useCallback((next: CartLine[]) => write(next), []);

  return { lines, add, update, remove, replace, snapshot: read, clear: clearCart };
}

export function useCartHydrated() {
  return useSyncExternalStore(
    () => () => {},
    () => true,
    () => false,
  );
}
