"use client";

import { useRouter } from "next/navigation";
import { toast } from "sonner";
import { useApiForm } from "@/hooks/use-api-form";
import { authApi } from "@/lib/api/auth";
import { homePath } from "@/lib/roles";
import { PASSWORD_MIN_LENGTH, changePasswordSchema } from "@/lib/validation/auth";
import { FieldGroup } from "@/components/ui/field";
import { FormAlert } from "./form-alert";
import { FormField } from "./form-field";
import { SubmitButton } from "./submit-button";

export function ChangePasswordForm({ temporary }: { temporary: boolean }) {
  const router = useRouter();
  const { errors, formError, pending, onSubmit } = useApiForm(changePasswordSchema, async (values) => {
    const result = await authApi.changePassword({ current_password: values.current_password, new_password: values.new_password });
    toast.success("Password changed. Other sessions were signed out.");
    router.replace(temporary ? homePath(result.user.role) : "/account");
    router.refresh();
  });

  return (
    <form onSubmit={onSubmit} noValidate>
      <FieldGroup>
        <FormAlert message={formError} />
        <FormField
          name="current_password"
          label={temporary ? "Temporary password" : "Current password"}
          type="password"
          autoComplete="current-password"
          autoFocus
          error={errors.current_password}
        />
        <FormField
          name="new_password"
          label="New password"
          type="password"
          autoComplete="new-password"
          description={`At least ${PASSWORD_MIN_LENGTH} characters, not your email.`}
          error={errors.new_password}
        />
        <FormField name="confirm_password" label="Confirm new password" type="password" autoComplete="new-password" error={errors.confirm_password} />
        <SubmitButton pending={pending}>Change password</SubmitButton>
      </FieldGroup>
    </form>
  );
}
