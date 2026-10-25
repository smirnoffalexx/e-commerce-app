"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useApiForm } from "@/hooks/use-api-form";
import { authApi } from "@/lib/api/auth";
import { safeNextPath } from "@/lib/session/routes";
import { PASSWORD_MIN_LENGTH, registerSchema } from "@/lib/validation/auth";
import { FieldGroup } from "@/components/ui/field";
import { FormAlert } from "./form-alert";
import { FormField } from "./form-field";
import { SubmitButton } from "./submit-button";

export function RegisterForm({ next }: { next?: string }) {
  const router = useRouter();
  const { errors, formError, pending, onSubmit } = useApiForm(registerSchema, async ({ email, password }) => {
    await authApi.register({ email, password });
    router.replace(safeNextPath(next));
    router.refresh();
  });

  return (
    <form onSubmit={onSubmit} noValidate>
      <FieldGroup>
        <FormAlert message={formError} />
        <FormField name="email" label="Email" type="email" autoComplete="email" autoFocus error={errors.email} />
        <FormField
          name="password"
          label="Password"
          type="password"
          autoComplete="new-password"
          description={`At least ${PASSWORD_MIN_LENGTH} characters.`}
          error={errors.password}
        />
        <FormField name="confirm_password" label="Confirm password" type="password" autoComplete="new-password" error={errors.confirm_password} />
        <SubmitButton pending={pending}>Create account</SubmitButton>
        <p className="text-center text-sm text-muted-foreground">
          Already have an account?{" "}
          <Link href={next ? `/login?next=${encodeURIComponent(next)}` : "/login"} className="font-medium text-primary underline-offset-4 hover:underline">
            Sign in
          </Link>
        </p>
      </FieldGroup>
    </form>
  );
}
