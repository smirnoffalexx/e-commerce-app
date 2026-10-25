"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useApiForm } from "@/hooks/use-api-form";
import { authApi } from "@/lib/api/auth";
import { homePath } from "@/lib/roles";
import { PASSWORD_CHANGE_PATH, safeNextPath } from "@/lib/session/routes";
import { loginSchema } from "@/lib/validation/auth";
import { FieldGroup } from "@/components/ui/field";
import { FormAlert } from "./form-alert";
import { FormField } from "./form-field";
import { SubmitButton } from "./submit-button";

export function LoginForm({ next }: { next?: string }) {
  const router = useRouter();
  const { errors, formError, pending, onSubmit } = useApiForm(loginSchema, async (values) => {
    const result = await authApi.login(values);
    router.replace(result.password_change_required ? `${PASSWORD_CHANGE_PATH}?required=1` : safeNextPath(next, homePath(result.user.role)));
    router.refresh();
  });

  return (
    <form onSubmit={onSubmit} noValidate>
      <FieldGroup>
        <FormAlert message={formError} />
        <FormField name="email" label="Email" type="email" autoComplete="email" autoFocus error={errors.email} />
        <FormField name="password" label="Password" type="password" autoComplete="current-password" error={errors.password} />
        <SubmitButton pending={pending}>Sign in</SubmitButton>
        <p className="text-center text-sm text-muted-foreground">
          New here?{" "}
          <Link href={next ? `/register?next=${encodeURIComponent(next)}` : "/register"} className="font-medium text-primary underline-offset-4 hover:underline">
            Create an account
          </Link>
        </p>
      </FieldGroup>
    </form>
  );
}
