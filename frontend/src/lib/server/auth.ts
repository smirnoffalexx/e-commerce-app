import "server-only";
import { NextResponse, type NextRequest } from "next/server";
import type { Session } from "@/lib/api/types";
import { applyCookies, sessionCookies } from "@/lib/session/cookies";
import { callBackend, relay, unavailable } from "./bff";

export async function readJson(request: NextRequest): Promise<unknown> {
  return request.json().catch(() => ({}));
}

export async function establishSession(
  request: NextRequest,
  path: string,
  json: unknown,
  authenticated = false,
): Promise<NextResponse> {
  const upstream = await callBackend(request, path, { method: "POST", json, authenticated });
  if (!upstream) {
    return unavailable();
  }
  if (!upstream.ok) {
    return relay(upstream);
  }
  const session = (await upstream.json()) as Session;
  const response = NextResponse.json({
    user: session.user,
    password_change_required: session.password_change_required,
  });
  applyCookies(response.cookies, sessionCookies(session, new Date()));
  return response;
}
