import "server-only";
import { NextResponse, type NextRequest } from "next/server";
import { backendHeaders, backendUrl } from "@/lib/backend/http";
import { ACCESS_COOKIE, applyCookies, clearedSessionCookies } from "@/lib/session/cookies";

const RELAYED_RESPONSE_HEADERS = [
  "content-type",
  "content-disposition",
  "idempotent-replayed",
  "retry-after",
  "x-request-id",
  "ratelimit-limit",
  "ratelimit-remaining",
  "ratelimit-reset",
];

export function problem(status: number, code: string, detail?: string) {
  return NextResponse.json(
    { status, code, detail },
    { status, headers: { "content-type": "application/problem+json" } },
  );
}

export async function callBackend(
  request: NextRequest,
  path: string,
  init: { method?: string; body?: BodyInit; json?: unknown; authenticated?: boolean } = {},
): Promise<Response | null> {
  const token = init.authenticated === false ? undefined : request.cookies.get(ACCESS_COOKIE)?.value;
  const headers = backendHeaders(request.headers, token);
  let body = init.body;
  if (init.json !== undefined) {
    headers.set("content-type", "application/json");
    body = JSON.stringify(init.json);
  }
  try {
    return await fetch(backendUrl(path), {
      method: init.method ?? request.method,
      headers,
      body,
      cache: "no-store",
      redirect: "manual",
    });
  } catch {
    return null;
  }
}

export function relay(upstream: Response): NextResponse {
  const headers = new Headers();
  for (const name of RELAYED_RESPONSE_HEADERS) {
    const value = upstream.headers.get(name);
    if (value) {
      headers.set(name, value);
    }
  }
  const body = upstream.status === 204 || upstream.status === 304 ? null : upstream.body;
  const response = new NextResponse(body, { status: upstream.status, headers });
  if (upstream.status === 401) {
    applyCookies(response.cookies, clearedSessionCookies());
  }
  return response;
}

export function unavailable() {
  return problem(502, "backend_unavailable", "backend is unreachable");
}
