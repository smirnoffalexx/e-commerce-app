import { describe, expect, it } from "vitest";
import { zodFieldErrors } from "./errors";
import { changePasswordSchema, registerSchema } from "./auth";

function errorsOf(result: ReturnType<typeof registerSchema.safeParse>) {
  return result.success ? {} : zodFieldErrors(result.error);
}

describe("registerSchema", () => {
  it("accepts a valid account", () => {
    const result = registerSchema.safeParse({
      email: " Jane@Example.com ",
      password: "correct-horse-battery",
      confirm_password: "correct-horse-battery",
    });
    expect(result.success).toBe(true);
  });

  it("mirrors backend password rules", () => {
    expect(errorsOf(registerSchema.safeParse({ email: "a@b.co", password: "short", confirm_password: "short" }))).toHaveProperty(
      "password",
    );

    const multiByte = "ж".repeat(37);
    expect(errorsOf(registerSchema.safeParse({ email: "a@b.co", password: multiByte, confirm_password: multiByte }))).toEqual({
      password: "Password must be at most 72 bytes.",
    });

    expect(
      errorsOf(registerSchema.safeParse({ email: "jane@example.com", password: "JANE@example.com", confirm_password: "JANE@example.com" })),
    ).toEqual({ password: "Password must not be the same as your email." });
  });

  it("requires the confirmation to match", () => {
    expect(
      errorsOf(registerSchema.safeParse({ email: "a@b.co", password: "correct-horse-battery", confirm_password: "other-horse-battery" })),
    ).toEqual({ confirm_password: "Passwords do not match." });
  });
});

describe("changePasswordSchema", () => {
  it("rejects reusing the current password", () => {
    const result = changePasswordSchema.safeParse({
      current_password: "correct-horse-battery",
      new_password: "correct-horse-battery",
      confirm_password: "correct-horse-battery",
    });
    expect(result.success ? {} : zodFieldErrors(result.error)).toEqual({
      new_password: "New password must differ from the current one.",
    });
  });
});
