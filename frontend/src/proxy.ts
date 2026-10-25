import { NextResponse, type NextRequest } from "next/server";
import { refreshAtBackend } from "@/lib/backend/http";
import {
  ACCESS_COOKIE,
  PASSWORD_CHANGE_COOKIE,
  REFRESH_COOKIE,
  applyCookies,
  clearedSessionCookies,
  sessionCookies,
  type CookieSpec,
} from "@/lib/session/cookies";
import { allowedOrigins, isTrustedRequest } from "@/lib/session/origin";
import { createRefresher } from "@/lib/session/refresh";
import { navigationRedirect } from "@/lib/session/routes";

let refresher: ReturnType<typeof createRefresher> | undefined;

function refresh(token: string, request: NextRequest) {
  refresher ??= createRefresher((value) => refreshAtBackend(value, request.headers));
  return refresher(token);
}

function forbiddenOrigin() {
  return NextResponse.json(
    { status: 403, code: "invalid_origin", title: "Forbidden", detail: "cross-site request rejected" },
    { status: 403, headers: { "content-type": "application/problem+json" } },
  );
}

export async function proxy(request: NextRequest) {
  const { pathname, search } = request.nextUrl;
  const isApi = pathname.startsWith("/api/");

  if (isApi && !isTrustedRequest(request.method, request.headers, allowedOrigins(process.env.APP_ORIGIN, request.nextUrl.origin))) {
    return forbiddenOrigin();
  }

  const refreshToken = request.cookies.get(REFRESH_COOKIE)?.value;
  let authenticated = request.cookies.has(ACCESS_COOKIE);
  let passwordChangeRequired = request.cookies.has(PASSWORD_CHANGE_COOKIE);
  let updates: CookieSpec[] | null = null;

  if (!authenticated && refreshToken) {
    const result = await refresh(refreshToken, request);
    if (result.ok) {
      updates = sessionCookies(result.session, new Date());
      authenticated = true;
      passwordChangeRequired = result.session.password_change_required;
    } else if (result.reason === "invalid") {
      updates = clearedSessionCookies();
      passwordChangeRequired = false;
    }
  }

  if (!isApi) {
    const target = navigationRedirect({ pathname, search, authenticated, passwordChangeRequired });
    if (target) {
      const response = NextResponse.redirect(new URL(target, request.url));
      if (updates) {
        applyCookies(response.cookies, updates);
      }
      return response;
    }
  }

  if (!updates) {
    return NextResponse.next();
  }
  for (const cookie of updates) {
    if (cookie.options.maxAge > 0) {
      request.cookies.set(cookie.name, cookie.value);
    } else {
      request.cookies.delete(cookie.name);
    }
  }
  const response = NextResponse.next({ request: { headers: request.headers } });
  applyCookies(response.cookies, updates);
  return response;
}

export const config = {
  matcher: ["/((?!_next/static|_next/image|favicon.ico|.*\\.[a-zA-Z0-9]+$).*)"],
};
