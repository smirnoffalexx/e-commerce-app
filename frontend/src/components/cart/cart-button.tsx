"use client";

import { ShoppingBagIcon } from "lucide-react";
import Link from "next/link";
import { useCart } from "@/hooks/use-cart";
import { itemCount } from "@/lib/cart/cart";
import { Button } from "@/components/ui/button";

export function CartButton() {
  const { lines } = useCart();
  const count = itemCount(lines);
  return (
    <Button variant="ghost" size="icon" className="relative" asChild>
      <Link href="/cart" aria-label={count ? `Cart, ${count} items` : "Cart"}>
        <ShoppingBagIcon />
        {count > 0 && (
          <span className="absolute -top-0.5 -right-0.5 flex h-4.5 min-w-4.5 items-center justify-center rounded-full bg-primary px-1 text-[10px] font-semibold text-primary-foreground tabular-nums">
            {count > 99 ? "99+" : count}
          </span>
        )}
      </Link>
    </Button>
  );
}
