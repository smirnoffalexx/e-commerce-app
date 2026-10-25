import { ReceiptIcon } from "lucide-react";
import type { Metadata } from "next";
import Link from "next/link";
import { FormAlert } from "@/components/auth/form-alert";
import { OrderList } from "@/components/orders/order-list";
import { OrderStatusTabs } from "@/components/orders/order-status-tabs";
import { Pager } from "@/components/pager";
import { Button } from "@/components/ui/button";
import { Empty, EmptyContent, EmptyDescription, EmptyHeader, EmptyMedia, EmptyTitle } from "@/components/ui/empty";
import { errorMessage } from "@/lib/api/problem";
import type { Order, Page } from "@/lib/api/types";
import { orderListHref, parseOrderListQuery } from "@/lib/orders";
import { requireUser, serverApi } from "@/lib/server/api";

export const metadata: Metadata = { title: "Orders" };

const PAGE_SIZE = 20;

export default async function OrdersPage({ searchParams }: PageProps<"/orders">) {
  await requireUser(["USER"], "/orders");
  const query = parseOrderListQuery(await searchParams);
  let page: Page<Order> | null = null;
  let error: unknown = null;
  try {
    page = await serverApi<Page<Order>>("/orders", { query: { ...query, page_size: PAGE_SIZE } });
  } catch (caught) {
    error = caught;
  }

  return (
    <div className="mx-auto w-full max-w-4xl px-4 py-8 sm:px-6">
      <div className="flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
        <div>
          <h1 className="text-3xl font-semibold tracking-tight">Orders</h1>
          <p className="mt-1 text-muted-foreground">Your purchase history</p>
        </div>
        <OrderStatusTabs base="/orders" status={query.status} />
      </div>
      <div className="mt-6">
        {error !== null && <FormAlert message={errorMessage(error)} />}
        {page && page.items.length === 0 && (
          <Empty className="border border-dashed">
            <EmptyHeader>
              <EmptyMedia variant="icon">
                <ReceiptIcon />
              </EmptyMedia>
              <EmptyTitle>No orders yet</EmptyTitle>
              <EmptyDescription>{query.status ? "No orders with this status." : "Orders you place will appear here."}</EmptyDescription>
            </EmptyHeader>
            <EmptyContent>
              <Button asChild>
                <Link href="/">Start shopping</Link>
              </Button>
            </EmptyContent>
          </Empty>
        )}
        {page && page.items.length > 0 && (
          <>
            <OrderList orders={page.items} />
            <Pager page={page.page} totalPages={page.total_pages} href={(target) => orderListHref("/orders", { ...query, page: target })} />
          </>
        )}
      </div>
    </div>
  );
}
