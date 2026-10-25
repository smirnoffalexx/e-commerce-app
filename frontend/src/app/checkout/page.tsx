import type { Metadata } from "next";
import { CheckoutView } from "@/components/checkout/checkout-view";
import { requireUser } from "@/lib/server/api";

export const metadata: Metadata = { title: "Checkout" };

export default async function CheckoutPage() {
  const user = await requireUser(["USER"], "/checkout");
  return (
    <div className="mx-auto w-full max-w-6xl px-4 py-8 sm:px-6">
      <h1 className="mb-6 text-3xl font-semibold tracking-tight">Checkout</h1>
      <CheckoutView email={user.email} />
    </div>
  );
}
