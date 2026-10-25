import type { Session } from "@/lib/api/types";

export const ACCESS_COOKIE = "shop_access";
export const REFRESH_COOKIE = "shop_refresh";
export const PASSWORD_CHANGE_COOKIE = "shop_password_change";

export const ACCESS_EXPIRY_SKEW_SECONDS = 30;

export interface CookieOptions {
  httpOnly: boolean;
  secure: boolean;
  sameSite: "lax" | "strict";
  path: string;
  maxAge: number;
}

export interface CookieSpec {
  name: string;
  value: string;
  options: CookieOptions;
}

export function cookieSecure(): boolean {
  const flag = process.env.COOKIE_SECURE;
  if (flag === "true" || flag === "false") {
    return flag === "true";
  }
  return process.env.NODE_ENV === "production";
}

function spec(name: string, value: string, maxAge: number, secure: boolean): CookieSpec {
  return {
    name,
    value: maxAge > 0 ? value : "",
    options: { httpOnly: true, secure, sameSite: "lax", path: "/", maxAge: Math.max(0, maxAge) },
  };
}

export function sessionCookies(session: Session, now: Date, secure = cookieSecure()): CookieSpec[] {
  const accessMaxAge = Math.max(1, session.expires_in - ACCESS_EXPIRY_SKEW_SECONDS);
  const refreshMaxAge = Math.floor((Date.parse(session.refresh_token_expires_at) - now.getTime()) / 1000);
  return [
    spec(ACCESS_COOKIE, session.access_token, accessMaxAge, secure),
    spec(REFRESH_COOKIE, session.refresh_token, refreshMaxAge, secure),
    spec(PASSWORD_CHANGE_COOKIE, "1", session.password_change_required ? refreshMaxAge : 0, secure),
  ];
}

export function clearedSessionCookies(secure = cookieSecure()): CookieSpec[] {
  return [ACCESS_COOKIE, REFRESH_COOKIE, PASSWORD_CHANGE_COOKIE].map((name) => spec(name, "", 0, secure));
}

interface CookieJar {
  set(name: string, value: string, options: CookieOptions): unknown;
}

export function applyCookies(jar: CookieJar, cookies: CookieSpec[]) {
  for (const cookie of cookies) {
    jar.set(cookie.name, cookie.value, cookie.options);
  }
}
