import { XIcon } from "lucide-react";
import type { Metadata } from "next";
import Link from "next/link";
import { PageHeader } from "@/components/admin/page-header";
import { FormAlert } from "@/components/auth/form-alert";
import { OrderList } from "@/components/orders/order-list";
import { OrderStatusTabs } from "@/components/orders/order-status-tabs";
import { Pager } from "@/components/pager";
import { Badge } from "@/components/ui/badge";
import { errorMessage } from "@/lib/api/problem";
import type { Order, Page } from "@/lib/api/types";
import { orderListHref, parseOrderListQuery } from "@/lib/orders";
import { serverApi } from "@/lib/server/api";

export const metadata: Metadata = { title: "Orders" };

export default async function AdminOrdersPage({ searchParams }: PageProps<"/admin/orders">) {
  const query = parseOrderListQuery(await searchParams);
  let page: Page<Order> | null = null;
  let error: unknown = null;
  try {
    page = await serverApi<Page<Order>>("/orders", { query: { ...query, page_size: 20 } });
  } catch (caught) {
    error = caught;
  }
  const customer = query.user_id ? page?.items[0]?.customer_email : undefined;

  return (
    <>
      <PageHeader title="Orders" description={page ? `${page.total} orders` : undefined} actions={<OrderStatusTabs base="/admin/orders" query={query} />} />
      {query.user_id && (
        <div className="mb-4 flex items-center gap-2 text-sm">
          <span className="text-muted-foreground">Customer</span>
          <Badge variant="secondary" className="gap-1 pr-1">
            {customer ?? query.user_id}
            <Link href={orderListHref("/admin/orders", { status: query.status })} aria-label="Show all customers" className="rounded-full p-0.5 hover:bg-background">
              <XIcon className="size-3" />
            </Link>
          </Badge>
        </div>
      )}
      {error !== null && <FormAlert message={errorMessage(error)} />}
      {page && page.items.length === 0 && (
        <p className="rounded-xl border border-dashed p-10 text-center text-muted-foreground">No orders match the filters.</p>
      )}
      {page && page.items.length > 0 && (
        <>
          <OrderList orders={page.items} showCustomer />
          <Pager page={page.page} totalPages={page.total_pages} href={(target) => orderListHref("/admin/orders", { ...query, page: target })} />
        </>
      )}
    </>
  );
}
