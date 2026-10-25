import { TriangleAlertIcon } from "lucide-react";
import type { Metadata } from "next";
import { AuthCard } from "@/components/auth/auth-card";
import { ChangePasswordForm } from "@/components/auth/change-password-form";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { requireUser } from "@/lib/server/api";

export const metadata: Metadata = { title: "Change password" };

export default async function ChangePasswordPage() {
  const user = await requireUser(undefined, "/account/password");
  const temporary = user.must_change_password;
  return (
    <AuthCard
      title={temporary ? "Set a new password" : "Change password"}
      description="All other sessions will be signed out"
    >
      {temporary && (
        <Alert className="mb-6">
          <TriangleAlertIcon />
          <AlertTitle>Temporary password</AlertTitle>
          <AlertDescription>Choose your own password before you can use the store.</AlertDescription>
        </Alert>
      )}
      <ChangePasswordForm temporary={temporary} />
    </AuthCard>
  );
}
