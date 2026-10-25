import { describe, expect, it } from "vitest";
import type { Session } from "@/lib/api/types";
import {
  ACCESS_COOKIE,
  ACCESS_EXPIRY_SKEW_SECONDS,
  PASSWORD_CHANGE_COOKIE,
  REFRESH_COOKIE,
  clearedSessionCookies,
  sessionCookies,
} from "./cookies";

const now = new Date("2026-10-25T10:00:00Z");

function makeSession(overrides: Partial<Session> = {}): Session {
  return {
    token_type: "Bearer",
    access_token: "access",
    expires_in: 900,
    access_token_expires_at: "2026-10-25T10:15:00Z",
    refresh_token: "refresh",
    refresh_token_expires_at: "2026-11-01T10:00:00Z",
    password_change_required: false,
    user: {} as Session["user"],
    ...overrides,
  };
}

function byName(cookies: ReturnType<typeof sessionCookies>) {
  return Object.fromEntries(cookies.map((cookie) => [cookie.name, cookie]));
}

describe("sessionCookies", () => {
  it("expires the access cookie before the token so the proxy refreshes in time", () => {
    const cookies = byName(sessionCookies(makeSession(), now, true));
    expect(cookies[ACCESS_COOKIE].options.maxAge).toBe(900 - ACCESS_EXPIRY_SKEW_SECONDS);
    expect(cookies[REFRESH_COOKIE].options.maxAge).toBe(7 * 24 * 3600);
  });

  it("keeps tokens out of reach of scripts and cross-site requests", () => {
    for (const cookie of sessionCookies(makeSession(), now, true)) {
      expect(cookie.options).toMatchObject({ httpOnly: true, secure: true, sameSite: "lax", path: "/" });
    }
  });

  it("marks the session when the password must be changed and clears the mark otherwise", () => {
    const required = byName(sessionCookies(makeSession({ password_change_required: true }), now, false));
    expect(required[PASSWORD_CHANGE_COOKIE]).toMatchObject({ value: "1" });
    expect(required[PASSWORD_CHANGE_COOKIE].options.maxAge).toBeGreaterThan(0);

    const changed = byName(sessionCookies(makeSession(), now, false));
    expect(changed[PASSWORD_CHANGE_COOKIE].options.maxAge).toBe(0);
  });

  it("clears every session cookie on logout", () => {
    const cleared = clearedSessionCookies(false);
    expect(cleared.map((cookie) => cookie.name).sort()).toEqual(
      [ACCESS_COOKIE, PASSWORD_CHANGE_COOKIE, REFRESH_COOKIE].sort(),
    );
    expect(cleared.every((cookie) => cookie.options.maxAge === 0 && cookie.value === "")).toBe(true);
  });
});
