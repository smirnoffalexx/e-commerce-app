import { StoreIcon } from "lucide-react";
import Link from "next/link";
import { canPurchase, isStaff } from "@/lib/roles";
import { getCurrentUser } from "@/lib/server/api";
import { CartButton } from "@/components/cart/cart-button";
import { Button } from "@/components/ui/button";
import { ThemeToggle } from "./theme-toggle";
import { UserMenu } from "./user-menu";

export async function SiteHeader() {
  const user = await getCurrentUser();
  return (
    <header className="sticky top-0 z-40 border-b bg-background/80 backdrop-blur supports-[backdrop-filter]:bg-background/60">
      <div className="mx-auto flex h-16 max-w-7xl items-center gap-6 px-4 sm:px-6">
        <Link href="/" className="flex items-center gap-2 font-semibold tracking-tight">
          <span className="flex size-8 items-center justify-center rounded-lg bg-primary text-primary-foreground">
            <StoreIcon className="size-4" />
          </span>
          E-Commerce App
        </Link>
        <div className="ml-auto flex items-center gap-1">
          <ThemeToggle />
          {canPurchase(user) && <CartButton />}
          {user && isStaff(user.role) && (
            <Button variant="outline" size="sm" asChild className="mr-1">
              <Link href="/admin">Admin</Link>
            </Button>
          )}
          {user ? (
            <UserMenu user={user} />
          ) : (
            <>
              <Button variant="ghost" asChild>
                <Link href="/login">Sign in</Link>
              </Button>
              <Button asChild className="hidden sm:inline-flex">
                <Link href="/register">Create account</Link>
              </Button>
            </>
          )}
        </div>
      </div>
    </header>
  );
}
