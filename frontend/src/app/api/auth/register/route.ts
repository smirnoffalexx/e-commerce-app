import type { NextRequest } from "next/server";
import { establishSession, readJson } from "@/lib/server/auth";
import { callBackend, relay, unavailable } from "@/lib/server/bff";

export async function POST(request: NextRequest) {
  const credentials = await readJson(request);
  const upstream = await callBackend(request, "/auth/register", {
    method: "POST",
    json: credentials,
    authenticated: false,
  });
  if (!upstream) {
    return unavailable();
  }
  if (!upstream.ok) {
    return relay(upstream);
  }
  return establishSession(request, "/auth/login", credentials);
}
