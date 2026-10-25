import { CircleCheckIcon, CircleXIcon, CreditCardIcon } from "lucide-react";
import type { Metadata } from "next";
import Link from "next/link";
import { notFound } from "next/navigation";
import { OrderStatusBadge } from "@/components/orders/order-status-badge";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Separator } from "@/components/ui/separator";
import { ApiError, errorMessage } from "@/lib/api/problem";
import type { Order } from "@/lib/api/types";
import { formatDateTime, formatMoney } from "@/lib/format";
import { shortOrderId } from "@/lib/orders";
import { isStaff } from "@/lib/roles";
import { firstParam } from "@/lib/search-params";
import { requireUser, serverApi } from "@/lib/server/api";

export const metadata: Metadata = { title: "Order" };

export default async function OrderPage({ params, searchParams }: PageProps<"/orders/[id]">) {
  const { id } = await params;
  const user = await requireUser(undefined, `/orders/${id}`);
  const placed = firstParam((await searchParams).placed) === "1";
  let order: Order;
  try {
    order = await serverApi<Order>(`/orders/${encodeURIComponent(id)}`);
  } catch (error) {
    if (error instanceof ApiError && (error.status === 404 || error.status === 400 || error.status === 422)) {
      notFound();
    }
    throw error;
  }
  const staff = isStaff(user.role);
  const failure = order.failure_code ? errorMessage(new ApiError({ status: 402, code: order.failure_code })) : null;

  return (
    <div className="mx-auto w-full max-w-4xl px-4 py-8 sm:px-6">
      {placed && order.status === "PAID" && (
        <Alert className="mb-6 border-emerald-500/40 bg-emerald-500/5">
          <CircleCheckIcon className="text-emerald-600" />
          <AlertTitle>Thank you for your order!</AlertTitle>
          <AlertDescription>Payment received. You can find this order in your order history at any time.</AlertDescription>
        </Alert>
      )}
      {order.status === "FAILED" && (
        <Alert variant="destructive" className="mb-6">
          <CircleXIcon />
          <AlertTitle>Payment failed</AlertTitle>
          <AlertDescription>{failure ?? "The payment could not be completed."} No money was taken.</AlertDescription>
        </Alert>
      )}
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div>
          {!staff && (
            <p className="mb-1 text-sm text-muted-foreground">
              <Link href="/orders" className="hover:text-foreground">
                Orders
              </Link>{" "}
              / #{shortOrderId(order.id)}
            </p>
          )}
          <h1 className=" text-3xl font-semibold tracking-tight">
            Order <span className="font-mono">#{shortOrderId(order.id)}</span>
          </h1>
          <p className="mt-1 text-muted-foreground">Placed {formatDateTime(order.created_at)}</p>
        </div>
        <OrderStatusBadge status={order.status} />
      </div>

      <div className="mt-8 grid gap-6 lg:grid-cols-[1fr_300px]">
        <Card>
          <CardHeader>
            <CardTitle>Items</CardTitle>
          </CardHeader>
          <CardContent>
            <ul className="divide-y">
              {order.items.map((item) => (
                <li key={item.product_id} className="flex items-center justify-between gap-4 py-3 first:pt-0">
                  <div className="min-w-0">
                    <Link href={`/products/${item.product_id}`} className="line-clamp-1 font-medium hover:underline">
                      {item.name}
                    </Link>
                    <p className="text-sm text-muted-foreground tabular-nums">
                      <span className="font-mono">{item.sku}</span> · {item.quantity} × {formatMoney(item.unit_price)}
                    </p>
                  </div>
                  <span className="font-medium tabular-nums">{formatMoney(item.line_total)}</span>
                </li>
              ))}
            </ul>
            <Separator className="my-4" />
            <div className="flex justify-between text-lg font-semibold">
              <span>Total</span>
              <span className="tabular-nums">{formatMoney(order.total)}</span>
            </div>
          </CardContent>
        </Card>
        <Card className="h-fit">
          <CardHeader>
            <CardTitle>Payment</CardTitle>
            <CardDescription>{order.customer_email}</CardDescription>
          </CardHeader>
          <CardContent className="flex flex-col gap-3 text-sm">
            <p className="flex items-center gap-2">
              <CreditCardIcon className="size-4 text-muted-foreground" />
              Card ending in <span className="font-mono">{order.card_last4}</span>
            </p>
            {order.paid_at && (
              <p>
                <span className="text-muted-foreground">Paid </span>
                {formatDateTime(order.paid_at)}
              </p>
            )}
            {order.payment_id && (
              <p className="break-all">
                <span className="text-muted-foreground">Payment ID </span>
                <span className="font-mono">{order.payment_id}</span>
              </p>
            )}
            <p className="break-all">
              <span className="text-muted-foreground">Order ID </span>
              <span className="font-mono">{order.id}</span>
            </p>
          </CardContent>
        </Card>
      </div>
      {!staff && (
        <div className="mt-8 flex gap-3">
          <Button variant="outline" asChild>
            <Link href="/orders">All orders</Link>
          </Button>
          <Button asChild>
            <Link href="/">Continue shopping</Link>
          </Button>
        </div>
      )}
    </div>
  );
}
