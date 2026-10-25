import type { Metadata } from "next";
import { AdminManager } from "@/components/admin/admin-manager";
import { PageHeader } from "@/components/admin/page-header";
import { Pager } from "@/components/pager";
import type { Page, User } from "@/lib/api/types";
import { requireUser, serverApi } from "@/lib/server/api";

export const metadata: Metadata = { title: "Administrators" };

export default async function AdminAdminsPage({ searchParams }: PageProps<"/admin/admins">) {
  await requireUser(["SUPERADMIN"], "/admin/admins");
  const requested = Number((await searchParams).page);
  const pageNumber = Number.isInteger(requested) && requested > 1 ? requested : 1;
  const page = await serverApi<Page<User>>("/admins", { query: { page: pageNumber, page_size: 20 } });
  return (
    <>
      <PageHeader title="Administrators" description="Admins manage the catalog and see all orders. Only you can manage admins." />
      <AdminManager admins={page.items} />
      <Pager page={page.page} totalPages={page.total_pages} href={(target) => (target > 1 ? `/admin/admins?page=${target}` : "/admin/admins")} />
    </>
  );
}
