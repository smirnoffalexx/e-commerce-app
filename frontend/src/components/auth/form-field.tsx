import type { ComponentProps } from "react";
import { Field, FieldDescription, FieldError, FieldLabel } from "@/components/ui/field";
import { Input } from "@/components/ui/input";

interface FormFieldProps extends ComponentProps<typeof Input> {
  name: string;
  label: string;
  error?: string;
  description?: string;
}

export function FormField({ name, label, error, description, ...input }: FormFieldProps) {
  const id = `field-${name}`;
  return (
    <Field data-invalid={error ? true : undefined}>
      <FieldLabel htmlFor={id}>{label}</FieldLabel>
      <Input id={id} name={name} aria-invalid={error ? true : undefined} {...input} />
      {description && !error && <FieldDescription>{description}</FieldDescription>}
      <FieldError>{error}</FieldError>
    </Field>
  );
}
