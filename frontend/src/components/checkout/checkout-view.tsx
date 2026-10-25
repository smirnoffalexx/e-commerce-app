"use client";

import { CreditCardIcon, LockIcon, ShoppingBagIcon } from "lucide-react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useRef, useState, useTransition, type FormEvent } from "react";
import { toast } from "sonner";
import { useCart, useCartHydrated } from "@/hooks/use-cart";
import { useCartRevalidation } from "@/hooks/use-cart-revalidation";
import { placeOrder } from "@/lib/api/orders";
import { errorMessage } from "@/lib/api/problem";
import type { CheckoutRequest } from "@/lib/api/types";
import { itemCount, lineTotal, subtotal } from "@/lib/cart/cart";
import { attemptFor, fingerprintOf, isRetryableFailure, type CheckoutAttempt } from "@/lib/checkout/attempt";
import { TEST_CARDS, cardSchema, formatCardNumber, formatExpiry } from "@/lib/checkout/card";
import { cardFieldErrors, isCartConflict, isPaymentFailure } from "@/lib/checkout/errors";
import { applyCheckoutIssues } from "@/lib/checkout/issues";
import { formatMoney } from "@/lib/format";
import { zodFieldErrors } from "@/lib/validation/errors";
import { FormAlert } from "@/components/auth/form-alert";
import { FormField } from "@/components/auth/form-field";
import { CartChanges } from "@/components/cart/cart-changes";
import { ProductArt } from "@/components/catalog/product-art";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from "@/components/ui/card";
import { Empty, EmptyContent, EmptyDescription, EmptyHeader, EmptyMedia, EmptyTitle } from "@/components/ui/empty";
import { FieldGroup } from "@/components/ui/field";
import { Separator } from "@/components/ui/separator";
import { Skeleton } from "@/components/ui/skeleton";
import { Spinner } from "@/components/ui/spinner";

interface PaymentFailure {
  message: string;
  orderId?: string;
}

function testExpiry() {
  const year = (new Date().getFullYear() + 3) % 100;
  return `12/${String(year).padStart(2, "0")}`;
}

export function CheckoutView({ email }: { email: string }) {
  const router = useRouter();
  const { lines, snapshot, replace, clear } = useCart();
  const hydrated = useCartHydrated();
  const { changes, setChanges, checking } = useCartRevalidation();
  const [card, setCard] = useState({ card_number: "", expiry: "", cvc: "" });
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [formError, setFormError] = useState<string | null>(null);
  const [failure, setFailure] = useState<PaymentFailure | null>(null);
  const [placed, setPlaced] = useState(false);
  const [pending, startTransition] = useTransition();
  const attempt = useRef<CheckoutAttempt | null>(null);

  function onSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const parsed = cardSchema().safeParse(card);
    setFormError(null);
    setFailure(null);
    if (!parsed.success) {
      setErrors(zodFieldErrors(parsed.error));
      return;
    }
    setErrors({});
    const body: CheckoutRequest = {
      items: snapshot().map((line) => ({ product_id: line.productId, quantity: line.quantity, unit_price: line.price })),
      payment: {
        card_number: parsed.data.card_number,
        exp_month: parsed.data.expiry.month,
        exp_year: parsed.data.expiry.year,
        cvc: parsed.data.cvc,
      },
    };

    startTransition(async () => {
      const current = attemptFor(attempt.current, await fingerprintOf(body), () => crypto.randomUUID());
      attempt.current = current;
      try {
        const order = await placeOrder(body, current.key);
        attempt.current = null;
        setPlaced(true);
        clear();
        toast.success("Order placed, thank you!");
        router.replace(`/orders/${order.id}?placed=1`);
      } catch (error) {
        if (!isRetryableFailure(error)) {
          attempt.current = null;
        }
        handleError(error);
      }
    });
  }

  function handleError(error: unknown) {
    if (isCartConflict(error)) {
      const result = applyCheckoutIssues(snapshot(), error.problem.items ?? []);
      replace(result.lines);
      setChanges(result.changes);
      setFormError("Your cart changed while you were shopping. Review the updates and place the order again.");
      return;
    }
    if (isPaymentFailure(error)) {
      setFailure({ message: errorMessage(error), orderId: error.problem.order_id });
      return;
    }
    const fields = cardFieldErrors(error);
    if (Object.keys(fields).length > 0) {
      setErrors(fields);
      return;
    }
    if (isRetryableFailure(error)) {
      setFormError(`${errorMessage(error)} Submitting again is safe: you will not be charged twice.`);
      return;
    }
    setFormError(errorMessage(error));
  }

  if (!hydrated || placed) {
    return <Skeleton className="h-96 w-full rounded-xl" />;
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
            <EmptyTitle>Nothing to check out</EmptyTitle>
            <EmptyDescription>Your cart is empty.</EmptyDescription>
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
    <form onSubmit={onSubmit} noValidate className="grid gap-8 lg:grid-cols-[1fr_380px]">
      <div className="flex flex-col gap-6">
        <CartChanges changes={changes} title="Please review your order" />
        <FormAlert message={formError} />
        {failure && (
          <FormAlert
            message={`${failure.message} No money was taken. Check the card details or use another card.`}
          />
        )}
        {failure?.orderId && (
          <p className="-mt-3 text-sm text-muted-foreground">
            The attempt was saved as a{" "}
            <Link href={`/orders/${failure.orderId}`} className="text-primary underline-offset-4 hover:underline">
              failed order
            </Link>
            .
          </p>
        )}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <CreditCardIcon className="size-5" />
              Payment
            </CardTitle>
            <CardDescription>Receipt goes to {email}</CardDescription>
          </CardHeader>
          <CardContent>
            <FieldGroup>
              <FormField
                name="card_number"
                label="Card number"
                inputMode="numeric"
                autoComplete="cc-number"
                placeholder="1234 1234 1234 1234"
                value={card.card_number}
                onChange={(event) => setCard({ ...card, card_number: formatCardNumber(event.target.value) })}
                error={errors.card_number}
                disabled={pending}
              />
              <div className="grid grid-cols-2 gap-4">
                <FormField
                  name="expiry"
                  label="Expiry"
                  inputMode="numeric"
                  autoComplete="cc-exp"
                  placeholder="MM/YY"
                  value={card.expiry}
                  onChange={(event) => setCard({ ...card, expiry: formatExpiry(event.target.value) })}
                  error={errors.expiry}
                  disabled={pending}
                />
                <FormField
                  name="cvc"
                  label="CVC"
                  inputMode="numeric"
                  autoComplete="cc-csc"
                  placeholder="123"
                  maxLength={4}
                  value={card.cvc}
                  onChange={(event) => setCard({ ...card, cvc: event.target.value.replace(/\D/g, "") })}
                  error={errors.cvc}
                  disabled={pending}
                />
              </div>
            </FieldGroup>
          </CardContent>
          <CardFooter className="flex flex-col items-start gap-2 border-t">
            <p className="text-sm font-medium">Test cards</p>
            <div className="flex flex-wrap gap-2">
              {TEST_CARDS.map((test) => (
                <Button
                  key={test.number}
                  type="button"
                  variant="outline"
                  size="sm"
                  disabled={pending}
                  onClick={() => {
                    setCard({ card_number: test.number, expiry: testExpiry(), cvc: "123" });
                    setErrors({});
                  }}
                >
                  {test.label}
                </Button>
              ))}
            </div>
          </CardFooter>
        </Card>
      </div>

      <Card className="h-fit lg:sticky lg:top-24">
        <CardHeader>
          <CardTitle>Order summary</CardTitle>
          <CardDescription>
            {itemCount(lines)} {itemCount(lines) === 1 ? "item" : "items"}
          </CardDescription>
        </CardHeader>
        <CardContent className="flex flex-col gap-4">
          <ul className="flex flex-col gap-3">
            {lines.map((line) => (
              <li key={line.productId} className="flex items-center gap-3 text-sm">
                <ProductArt slug={line.categorySlug} name={line.name} className="size-12 shrink-0 rounded-md" />
                <div className="min-w-0 flex-1">
                  <p className="line-clamp-1 font-medium">{line.name}</p>
                  <p className="text-muted-foreground tabular-nums">
                    {line.quantity} × {formatMoney(line.price)}
                  </p>
                </div>
                <span className="font-medium tabular-nums">{formatMoney(lineTotal(line))}</span>
              </li>
            ))}
          </ul>
          <Separator />
          <div className="flex justify-between text-sm">
            <span className="text-muted-foreground">Shipping</span>
            <span>Free</span>
          </div>
          <div className="flex justify-between text-base font-semibold">
            <span>Total</span>
            <span className="tabular-nums">{formatMoney(subtotal(lines))}</span>
          </div>
        </CardContent>
        <CardFooter className="flex flex-col gap-3">
          <Button type="submit" size="lg" className="w-full" disabled={pending || checking}>
            {pending ? <Spinner /> : <LockIcon />}
            {pending ? "Processing payment" : `Pay ${formatMoney(subtotal(lines))}`}
          </Button>
          <Button type="button" variant="ghost" size="sm" asChild>
            <Link href="/cart">Back to cart</Link>
          </Button>
        </CardFooter>
      </Card>
    </form>
  );
}
