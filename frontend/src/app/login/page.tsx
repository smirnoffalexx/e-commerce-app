import type { Metadata } from "next";
import { redirect } from "next/navigation";
import { AuthCard } from "@/components/auth/auth-card";
import { LoginForm } from "@/components/auth/login-form";
import { getCurrentUser } from "@/lib/server/api";
import { firstParam } from "@/lib/search-params";
import { safeNextPath } from "@/lib/session/routes";

export const metadata: Metadata = { title: "Sign in" };

export default async function LoginPage({ searchParams }: PageProps<"/login">) {
  const next = firstParam((await searchParams).next);
  if (await getCurrentUser()) {
    redirect(safeNextPath(next));
  }
  return (
    <AuthCard title="Welcome back" description="Sign in to your account to continue">
      <LoginForm next={next} />
    </AuthCard>
  );
}
