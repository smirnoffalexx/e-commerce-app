"use client";

import { ShoppingCartIcon } from "lucide-react";
import { useRouter } from "next/navigation";
import { toast } from "sonner";
import { useCart } from "@/hooks/use-cart";
import type { Product } from "@/lib/api/types";
import { MAX_CART_LINES, MAX_LINE_QUANTITY } from "@/lib/cart/cart";
import { Button } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";

interface AddToCartButtonProps extends Omit<React.ComponentProps<typeof Button>, "onClick"> {
  product: Product;
  quantity?: number;
  canBuy: boolean;
  onAdded?: () => void;
}

export function AddToCartButton({ product, quantity = 1, canBuy, onAdded, children, ...props }: AddToCartButtonProps) {
  const { add } = useCart();
  const router = useRouter();

  if (!canBuy) {
    return (
      <Tooltip>
        <TooltipTrigger asChild>
          <span tabIndex={0}>
            <Button {...props} disabled>
              <ShoppingCartIcon />
              {children ?? "Add to cart"}
            </Button>
          </span>
        </TooltipTrigger>
        <TooltipContent>Staff accounts cannot place orders</TooltipContent>
      </Tooltip>
    );
  }

  function handleClick(event: React.MouseEvent) {
    event.preventDefault();
    const outcome = add(product, quantity);
    const action = { label: "View cart", onClick: () => router.push("/cart") };
    switch (outcome) {
      case "added":
        toast.success(`${product.name} added to cart`, { action });
        onAdded?.();
        break;
      case "limited":
        toast.warning(`Only ${Math.min(product.stock, MAX_LINE_QUANTITY)} of ${product.name} can be ordered`, { action });
        onAdded?.();
        break;
      case "cart_full":
        toast.error(`A cart can hold up to ${MAX_CART_LINES} different products`);
        break;
      case "out_of_stock":
        toast.error(`${product.name} is out of stock`);
        break;
    }
  }

  return (
    <Button {...props} disabled={props.disabled || product.stock <= 0} onClick={handleClick}>
      <ShoppingCartIcon />
      {children ?? (product.stock > 0 ? "Add to cart" : "Out of stock")}
    </Button>
  );
}
