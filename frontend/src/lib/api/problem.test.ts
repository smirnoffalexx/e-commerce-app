import { describe, expect, it } from "vitest";
import { ApiError, errorMessage, fieldErrors, toProblem } from "./problem";

describe("toProblem", () => {
  it("keeps the RFC 9457 body and the real status", () => {
    const problem = toProblem(409, { code: "version_conflict", detail: "stale" });
    expect(problem).toMatchObject({ status: 409, code: "version_conflict", detail: "stale" });
  });

  it("maps gateway failures without a problem body to backend_unavailable", () => {
    expect(toProblem(502, "<html>")).toEqual({ status: 502, code: "backend_unavailable" });
  });
});

describe("errorMessage", () => {
  it("translates stable error codes", () => {
    expect(errorMessage(new ApiError({ status: 401, code: "invalid_credentials" }))).toBe("Invalid email or password.");
  });

  it("falls back to the backend detail for unknown codes", () => {
    expect(errorMessage(new ApiError({ status: 400, code: "brand_new", detail: "something specific" }))).toBe(
      "something specific",
    );
  });

  it("treats non-API failures as network errors", () => {
    expect(errorMessage(new TypeError("fetch failed"))).toMatch(/reach the server/);
  });
});

describe("fieldErrors", () => {
  it("returns the first message per field, keeping nested field paths", () => {
    const error = new ApiError({
      status: 422,
      code: "validation_failed",
      errors: [
        { field: "password", code: "same_as_email", message: "password must not be the same as email" },
        { field: "password", code: "too_long", message: "password must be at most 72 bytes" },
        { field: "items[0].quantity", code: "too_large", message: "quantity must be at most 100" },
      ],
    });
    expect(fieldErrors(error)).toEqual({
      password: "Password must not be the same as your email.",
      "items[0].quantity": "Quantity must be at most 100.",
    });
  });
});
