import { z } from "zod";
import type { ProductInput } from "@/lib/api/types";

export const SKU_MAX_LENGTH = 64;
export const NAME_MAX_LENGTH = 200;
export const DESCRIPTION_MAX_LENGTH = 5000;
export const STOCK_MAX = 2_147_483_647;

const SKU_PATTERN = /^[A-Z0-9](?:[A-Z0-9-]*[A-Z0-9])?$/;
const CONTROL_CHARS = /[\u0000-\u001f\u007f-\u009f]/;
const CONTROL_CHARS_EXCEPT_NEWLINES = /[\u0000-\u0008\u000b\u000c\u000e-\u001f\u007f-\u009f]/;

function decimal(field: string, places: number, max: string) {
  const pattern = new RegExp(`^\\d+(\\.\\d{1,${places}})?$`);
  return z
    .string()
    .trim()
    .min(1, `${field} is required.`)
    .refine((value) => pattern.test(value), `${field} must be a non-negative number with at most ${places} decimals.`)
    .refine((value) => !pattern.test(value) || Number(value) <= Number(max), `${field} must be at most ${max}.`);
}

export const productSchema = z.object({
  sku: z
    .string()
    .trim()
    .toUpperCase()
    .min(1, "SKU is required.")
    .max(SKU_MAX_LENGTH, `SKU must be at most ${SKU_MAX_LENGTH} characters.`)
    .regex(SKU_PATTERN, "SKU may contain only latin letters, digits and inner dashes."),
  name: z
    .string()
    .trim()
    .min(1, "Name is required.")
    .max(NAME_MAX_LENGTH, `Name must be at most ${NAME_MAX_LENGTH} characters.`)
    .refine((value) => !CONTROL_CHARS.test(value), "Name must not contain control characters."),
  description: z
    .string()
    .trim()
    .max(DESCRIPTION_MAX_LENGTH, `Description must be at most ${DESCRIPTION_MAX_LENGTH} characters.`)
    .refine((value) => !CONTROL_CHARS_EXCEPT_NEWLINES.test(value), "Description must not contain control characters."),
  category_id: z.string().min(1, "Choose a category."),
  price: decimal("Price", 2, "9999999999.99"),
  stock: z
    .string()
    .trim()
    .min(1, "Stock is required.")
    .regex(/^\d+$/, "Stock must be a whole non-negative number.")
    .transform(Number)
    .refine((value) => value <= STOCK_MAX, "Stock is too large."),
  weight_kg: decimal("Weight", 3, "9999999.999"),
}) satisfies z.ZodType<ProductInput, unknown>;
