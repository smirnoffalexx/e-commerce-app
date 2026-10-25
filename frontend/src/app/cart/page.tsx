import type { Metadata } from "next";
import { redirect } from "next/navigation";
import { CartView } from "@/components/cart/cart-view";
import { canPurchase } from "@/lib/roles";
import { getCurrentUser } from "@/lib/server/api";

export const metadata: Metadata = { title: "Cart" };

export default async function CartPage() {
  if (!canPurchase(await getCurrentUser())) {
    redirect("/");
  }
  return (
    <div className="mx-auto w-full max-w-6xl px-4 py-8 sm:px-6">
      <h1 className="mb-6 text-3xl font-semibold tracking-tight">Cart</h1>
      <CartView />
    </div>
  );
}
