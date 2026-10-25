import type { Session } from "@/lib/api/types";
import type { RefreshResult } from "@/lib/session/refresh";

const FORWARDED_REQUEST_HEADERS = ["accept", "content-type", "idempotency-key", "user-agent", "x-request-id"];

export function backendUrl(path: string): string {
  const base = (process.env.BACKEND_URL ?? "http://localhost:8080").replace(/\/+$/, "");
  return `${base}/api/v1${path}`;
}

export function clientIp(incoming: Pick<Headers, "get">): string | null {
  return incoming.get("x-forwarded-for");
}

export function backendHeaders(incoming: Pick<Headers, "get">, accessToken?: string): Headers {
  const headers = new Headers();
  for (const name of FORWARDED_REQUEST_HEADERS) {
    const value = incoming.get(name);
    if (value) {
      headers.set(name, value);
    }
  }
  const ip = clientIp(incoming);
  if (ip) {
    headers.set("x-forwarded-for", ip);
  }
  if (accessToken) {
    headers.set("authorization", `Bearer ${accessToken}`);
  }
  return headers;
}

export async function refreshAtBackend(refreshToken: string, incoming: Pick<Headers, "get">): Promise<RefreshResult> {
  const headers = backendHeaders(incoming);
  headers.set("content-type", "application/json");
  headers.set("accept", "application/json");
  const response = await fetch(backendUrl("/auth/refresh"), {
    method: "POST",
    headers,
    body: JSON.stringify({ refresh_token: refreshToken }),
    cache: "no-store",
  });
  if (response.ok) {
    return { ok: true, session: (await response.json()) as Session };
  }
  if (response.status === 401 || response.status === 403 || response.status === 422) {
    return { ok: false, reason: "invalid" };
  }
  return { ok: false, reason: "unavailable" };
}
