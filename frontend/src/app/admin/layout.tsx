import { AdminNav } from "@/components/admin/admin-nav";
import { STAFF_ROLES } from "@/lib/roles";
import { requireUser } from "@/lib/server/api";

export default async function AdminLayout({ children }: LayoutProps<"/admin">) {
  const user = await requireUser(STAFF_ROLES, "/admin");
  return (
    <div className="mx-auto grid w-full max-w-7xl flex-1 gap-6 px-4 py-6 sm:px-6 lg:grid-cols-[200px_1fr] lg:gap-10 lg:py-10">
      <aside className="lg:sticky lg:top-24 lg:self-start">
        <p className="mb-3 hidden px-3 text-xs font-semibold tracking-wider text-muted-foreground uppercase lg:block">Administration</p>
        <AdminNav role={user.role} />
      </aside>
      <div className="min-w-0">{children}</div>
    </div>
  );
}
