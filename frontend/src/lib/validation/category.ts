import { z } from "zod";

export const CATEGORY_NAME_MAX_LENGTH = 64;

export const categorySchema = z.object({
  name: z
    .string()
    .trim()
    .min(1, "Name is required.")
    .max(CATEGORY_NAME_MAX_LENGTH, `Name must be at most ${CATEGORY_NAME_MAX_LENGTH} characters.`)
    .refine((value) => !/[\u0000-\u001f\u007f-\u009f]/.test(value), "Name must not contain control characters.")
    .refine((value) => /[A-Za-z0-9]/.test(value), "Name must contain latin letters or digits."),
});
