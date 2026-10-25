"use client";

import { RefreshCwIcon, TriangleAlertIcon } from "lucide-react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { toast } from "sonner";
import { useApiForm } from "@/hooks/use-api-form";
import { adminApi } from "@/lib/api/admin";
import { ApiError } from "@/lib/api/problem";
import type { Category, Product } from "@/lib/api/types";
import { DESCRIPTION_MAX_LENGTH, productSchema } from "@/lib/validation/product";
import { FormAlert } from "@/components/auth/form-alert";
import { FormField } from "@/components/auth/form-field";
import { Alert, AlertAction, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardFooter } from "@/components/ui/card";
import { Field, FieldDescription, FieldError, FieldGroup, FieldLabel } from "@/components/ui/field";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Spinner } from "@/components/ui/spinner";
import { Textarea } from "@/components/ui/textarea";

export function ProductForm({ categories, product }: { categories: Category[]; product?: Product }) {
  const router = useRouter();
  const { errors, formError, error, pending, onSubmit } = useApiForm(productSchema, async (values) => {
    if (product) {
      await adminApi.updateProduct(product.id, values, product.version);
      toast.success(`${values.name} saved`);
    } else {
      await adminApi.createProduct(values);
      toast.success(`${values.name} created`);
    }
    router.push("/admin/products");
    router.refresh();
  });
  const conflict = error instanceof ApiError && error.code === "version_conflict";

  return (
    <form onSubmit={onSubmit} noValidate>
      <Card>
        <CardContent>
          <FieldGroup>
            {conflict ? (
              <Alert className="border-amber-500/40 bg-amber-500/5">
                <TriangleAlertIcon className="text-amber-600" />
                <AlertTitle>This product was changed by someone else</AlertTitle>
                <AlertDescription>Reload to see the latest version. Your unsaved changes will be lost.</AlertDescription>
                <AlertAction>
                  <Button type="button" size="sm" variant="outline" onClick={() => router.refresh()}>
                    <RefreshCwIcon />
                    Reload
                  </Button>
                </AlertAction>
              </Alert>
            ) : (
              <FormAlert message={formError} />
            )}
            <div className="grid gap-6 sm:grid-cols-[1fr_2fr]">
              <FormField name="sku" label="SKU" defaultValue={product?.sku} error={errors.sku} className="font-mono uppercase" autoFocus={!product} />
              <FormField name="name" label="Name" defaultValue={product?.name} error={errors.name} />
            </div>
            <Field data-invalid={errors.description ? true : undefined}>
              <FieldLabel htmlFor="field-description">Description</FieldLabel>
              <Textarea
                id="field-description"
                name="description"
                rows={4}
                maxLength={DESCRIPTION_MAX_LENGTH}
                defaultValue={product?.description}
                aria-invalid={errors.description ? true : undefined}
              />
              <FieldError>{errors.description}</FieldError>
            </Field>
            <Field data-invalid={errors.category_id ? true : undefined}>
              <FieldLabel>Category</FieldLabel>
              <Select name="category_id" defaultValue={product?.category_id}>
                <SelectTrigger className="w-full sm:w-72" aria-invalid={errors.category_id ? true : undefined}>
                  <SelectValue placeholder="Choose a category" />
                </SelectTrigger>
                <SelectContent>
                  {categories.map((category) => (
                    <SelectItem key={category.id} value={category.id}>
                      {category.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              {categories.length === 0 && (
                <FieldDescription>
                  No categories yet. <Link href="/admin/categories" className="text-primary underline-offset-4 hover:underline">Create one first</Link>.
                </FieldDescription>
              )}
              <FieldError>{errors.category_id}</FieldError>
            </Field>
            <div className="grid gap-6 sm:grid-cols-3">
              <FormField name="price" label="Price, $" inputMode="decimal" defaultValue={product?.price} error={errors.price} />
              <FormField name="stock" label="Stock" inputMode="numeric" defaultValue={product?.stock.toString()} error={errors.stock} />
              <FormField name="weight_kg" label="Weight, kg" inputMode="decimal" defaultValue={product?.weight_kg} error={errors.weight_kg} />
            </div>
          </FieldGroup>
        </CardContent>
        <CardFooter className="justify-end gap-2 border-t">
          <Button type="button" variant="ghost" asChild>
            <Link href="/admin/products">Cancel</Link>
          </Button>
          <Button type="submit" disabled={pending || conflict}>
            {pending && <Spinner />}
            {product ? "Save changes" : "Create product"}
          </Button>
        </CardFooter>
      </Card>
    </form>
  );
}
