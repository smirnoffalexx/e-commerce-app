"use client";

import { FolderTreeIcon, PackageIcon, ReceiptIcon, ShieldIcon, UploadIcon } from "lucide-react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import type { Role } from "@/lib/api/types";
import { cn } from "@/lib/utils";

const items = [
  { href: "/admin/products", label: "Products", icon: PackageIcon },
  { href: "/admin/categories", label: "Categories", icon: FolderTreeIcon },
  { href: "/admin/import", label: "Import CSV", icon: UploadIcon },
  { href: "/admin/orders", label: "Orders", icon: ReceiptIcon },
  { href: "/admin/admins", label: "Administrators", icon: ShieldIcon, superadmin: true },
];

export function AdminNav({ role }: { role: Role }) {
  const pathname = usePathname();
  return (
    <nav aria-label="Admin" className="flex gap-1 overflow-x-auto lg:flex-col">
      {items
        .filter((item) => !item.superadmin || role === "SUPERADMIN")
        .map(({ href, label, icon: Icon }) => {
          const active = pathname === href || pathname.startsWith(`${href}/`);
          return (
            <Link
              key={href}
              href={href}
              aria-current={active ? "page" : undefined}
              className={cn(
                "flex shrink-0 items-center gap-2 rounded-lg px-3 py-2 text-sm font-medium text-muted-foreground transition-colors hover:bg-muted hover:text-foreground",
                active && "bg-primary/10 text-primary hover:bg-primary/10 hover:text-primary",
              )}
            >
              <Icon className="size-4" />
              {label}
            </Link>
          );
        })}
    </nav>
  );
}
