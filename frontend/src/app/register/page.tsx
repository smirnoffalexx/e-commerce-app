import type { Metadata } from "next";
import { redirect } from "next/navigation";
import { AuthCard } from "@/components/auth/auth-card";
import { RegisterForm } from "@/components/auth/register-form";
import { getCurrentUser } from "@/lib/server/api";
import { firstParam } from "@/lib/search-params";
import { safeNextPath } from "@/lib/session/routes";

export const metadata: Metadata = { title: "Create account" };

export default async function RegisterPage({ searchParams }: PageProps<"/register">) {
  const next = firstParam((await searchParams).next);
  if (await getCurrentUser()) {
    redirect(safeNextPath(next));
  }
  return (
    <AuthCard title="Create your account" description="Shop the catalog and track your orders">
      <RegisterForm next={next} />
    </AuthCard>
  );
}
