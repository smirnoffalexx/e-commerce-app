import { describe, expect, it } from "vitest";
import { navigationRedirect, safeNextPath } from "./routes";

const guest = { search: "", authenticated: false, passwordChangeRequired: false };

describe("navigationRedirect", () => {
  it("sends guests from protected pages to login and remembers where they were going", () => {
    expect(navigationRedirect({ ...guest, pathname: "/checkout", search: "?step=pay" })).toBe(
      "/login?next=%2Fcheckout%3Fstep%3Dpay",
    );
    expect(navigationRedirect({ ...guest, pathname: "/admin/products" })).toBe("/login?next=%2Fadmin%2Fproducts");
  });

  it("leaves public pages open to guests", () => {
    expect(navigationRedirect({ ...guest, pathname: "/" })).toBeNull();
    expect(navigationRedirect({ ...guest, pathname: "/products/123" })).toBeNull();
    expect(navigationRedirect({ ...guest, pathname: "/administrator" })).toBeNull();
  });

  it("keeps a user with a temporary password on the password page", () => {
    const state = { ...guest, authenticated: true, passwordChangeRequired: true };
    expect(navigationRedirect({ ...state, pathname: "/admin" })).toBe("/account/password?required=1");
    expect(navigationRedirect({ ...state, pathname: "/" })).toBe("/account/password?required=1");
    expect(navigationRedirect({ ...state, pathname: "/account/password" })).toBeNull();
  });
});

describe("safeNextPath", () => {
  it("allows only same-site relative paths", () => {
    expect(safeNextPath("/orders?page=2")).toBe("/orders?page=2");
    expect(safeNextPath("https://evil.example")).toBe("/");
    expect(safeNextPath("//evil.example")).toBe("/");
    expect(safeNextPath("/\\evil.example")).toBe("/");
    expect(safeNextPath(undefined, "/admin")).toBe("/admin");
  });
});
