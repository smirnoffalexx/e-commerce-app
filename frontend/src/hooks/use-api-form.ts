"use client";

import { useState, useTransition, type FormEvent } from "react";
import type { z } from "zod";
import { ApiError, errorMessage, fieldErrors } from "@/lib/api/problem";
import { zodFieldErrors } from "@/lib/validation/errors";

export function useApiForm<S extends z.ZodType>(schema: S, submit: (values: z.infer<S>) => Promise<void>) {
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [formError, setFormError] = useState<string | null>(null);
  const [error, setError] = useState<unknown>(null);
  const [pending, startTransition] = useTransition();

  function onSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const parsed = schema.safeParse(Object.fromEntries(new FormData(event.currentTarget)));
    setFormError(null);
    setError(null);
    if (!parsed.success) {
      setErrors(zodFieldErrors(parsed.error));
      return;
    }
    setErrors({});
    startTransition(async () => {
      try {
        await submit(parsed.data);
      } catch (caught) {
        const fields = fieldErrors(caught);
        setErrors(fields);
        setError(caught);
        const onlyFields = caught instanceof ApiError && caught.code === "validation_failed" && Object.keys(fields).length > 0;
        setFormError(onlyFields ? null : errorMessage(caught));
      }
    });
  }

  return { errors, formError, error, pending, onSubmit };
}
