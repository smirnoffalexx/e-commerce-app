import type { NextRequest } from "next/server";
import { callBackend, problem, relay, unavailable } from "@/lib/server/bff";

const BLOCKED_PATHS = new Set(["auth/refresh", "auth/login", "auth/register", "auth/logout", "auth/change-password"]);

async function handle(request: NextRequest, context: RouteContext<"/api/[...path]">) {
  const { path } = await context.params;
  const joined = path.map(encodeURIComponent).join("/");
  if (BLOCKED_PATHS.has(joined)) {
    return problem(404, "route_not_found");
  }
  const hasBody = request.method !== "GET" && request.method !== "HEAD";
  const upstream = await callBackend(request, `/${joined}${request.nextUrl.search}`, {
    body: hasBody ? await request.arrayBuffer() : undefined,
  });
  return upstream ? relay(upstream) : unavailable();
}

export { handle as GET, handle as POST, handle as PUT, handle as PATCH, handle as DELETE };
