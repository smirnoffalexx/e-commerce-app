import "server-only";
import { cookies, headers } from "next/headers";
import { redirect } from "next/navigation";
import { cache } from "react";
import { buildQuery, type RequestOptions } from "@/lib/api/client";
import { ApiError, toProblem } from "@/lib/api/problem";
import type { Role, User } from "@/lib/api/types";
import { backendHeaders, backendUrl } from "@/lib/backend/http";
import { ACCESS_COOKIE } from "@/lib/session/cookies";

export async function serverApi<T>(path: string, options: Omit<RequestOptions, "signal"> = {}): Promise<T> {
  const token = (await cookies()).get(ACCESS_COOKIE)?.value;
  const outgoing = backendHeaders(await headers(), token);
  outgoing.set("accept", "application/json");
  let body: string | undefined;
  if (options.body !== undefined) {
    outgoing.set("content-type", "application/json");
    body = JSON.stringify(options.body);
  }

  let response: Response;
  try {
    response = await fetch(backendUrl(path) + buildQuery(options.query), {
      method: options.method ?? "GET",
      headers: outgoing,
      body,
      cache: "no-store",
    });
  } catch {
    throw new ApiError({ status: 502, code: "backend_unavailable" });
  }
  const data = response.status === 204 ? undefined : await response.json().catch(() => undefined);
  if (!response.ok) {
    throw new ApiError(toProblem(response.status, data));
  }
  return data as T;
}

export const getCurrentUser = cache(async (): Promise<User | null> => {
  if (!(await cookies()).has(ACCESS_COOKIE)) {
    return null;
  }
  try {
    return await serverApi<User>("/auth/me");
  } catch (error) {
    if (error instanceof ApiError && (error.status === 401 || error.status === 403)) {
      return null;
    }
    throw error;
  }
});

export async function requireUser(roles?: Role[], next = "/"): Promise<User> {
  const user = await getCurrentUser();
  if (!user) {
    redirect(`/login?next=${encodeURIComponent(next)}`);
  }
  if (roles && !roles.includes(user.role)) {
    redirect("/");
  }
  return user;
}
