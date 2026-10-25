import { z } from "zod";

export const PASSWORD_MIN_LENGTH = 10;
export const PASSWORD_MAX_BYTES = 72;

export const emailField = z
  .string()
  .trim()
  .min(1, "Email is required.")
  .max(254, "Email must be at most 254 characters.")
  .pipe(z.email("Enter a valid email address."));

const newPassword = z
  .string()
  .min(PASSWORD_MIN_LENGTH, `Password must be at least ${PASSWORD_MIN_LENGTH} characters.`)
  .refine((value) => new TextEncoder().encode(value).length <= PASSWORD_MAX_BYTES, {
    message: `Password must be at most ${PASSWORD_MAX_BYTES} bytes.`,
  })
  .refine((value) => value.trim().length > 0, { message: "Password must not be blank." });

export const loginSchema = z.object({
  email: emailField,
  password: z.string().min(1, "Password is required."),
});

export const registerSchema = z
  .object({ email: emailField, password: newPassword, confirm_password: z.string() })
  .refine((data) => data.password.toLowerCase() !== data.email.toLowerCase(), {
    path: ["password"],
    message: "Password must not be the same as your email.",
  })
  .refine((data) => data.password === data.confirm_password, {
    path: ["confirm_password"],
    message: "Passwords do not match.",
  });

export const changePasswordSchema = z
  .object({
    current_password: z.string().min(1, "Current password is required."),
    new_password: newPassword,
    confirm_password: z.string(),
  })
  .refine((data) => data.new_password !== data.current_password, {
    path: ["new_password"],
    message: "New password must differ from the current one.",
  })
  .refine((data) => data.new_password === data.confirm_password, {
    path: ["confirm_password"],
    message: "Passwords do not match.",
  });

export const adminSchema = z.object({ email: emailField });
