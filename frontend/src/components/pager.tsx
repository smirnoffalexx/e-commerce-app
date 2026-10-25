import { ChevronLeftIcon, ChevronRightIcon } from "lucide-react";
import Link from "next/link";
import { pageWindow } from "@/lib/pagination";
import { cn } from "@/lib/utils";
import { buttonVariants } from "@/components/ui/button";

export function Pager({ page, totalPages, href }: { page: number; totalPages: number; href: (page: number) => string }) {
  if (totalPages <= 1) {
    return null;
  }
  return (
    <nav aria-label="Pagination" className="mt-10 flex items-center justify-center gap-1">
      <PageLink href={href(page - 1)} disabled={page <= 1} label="Previous page">
        <ChevronLeftIcon />
      </PageLink>
      {pageWindow(page, totalPages).map((item, index) =>
        item === "gap" ? (
          <span key={`gap-${index}`} className="px-2 text-muted-foreground">
            …
          </span>
        ) : (
          <PageLink key={item} href={href(item)} active={item === page} label={`Page ${item}`}>
            {item}
          </PageLink>
        ),
      )}
      <PageLink href={href(page + 1)} disabled={page >= totalPages} label="Next page">
        <ChevronRightIcon />
      </PageLink>
    </nav>
  );
}

function PageLink({
  href,
  active,
  disabled,
  label,
  children,
}: {
  href: string;
  active?: boolean;
  disabled?: boolean;
  label: string;
  children: React.ReactNode;
}) {
  const className = cn(buttonVariants({ variant: active ? "default" : "ghost", size: "icon" }), "tabular-nums");
  if (disabled) {
    return (
      <span aria-disabled className={cn(className, "pointer-events-none opacity-40")}>
        {children}
      </span>
    );
  }
  return (
    <Link href={href} aria-label={label} aria-current={active ? "page" : undefined} className={className}>
      {children}
    </Link>
  );
}
