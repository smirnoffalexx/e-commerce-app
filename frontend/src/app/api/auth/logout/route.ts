import { NextResponse, type NextRequest } from "next/server";
import { applyCookies, clearedSessionCookies, REFRESH_COOKIE } from "@/lib/session/cookies";
import { callBackend } from "@/lib/server/bff";

export async function POST(request: NextRequest) {
  const refreshToken = request.cookies.get(REFRESH_COOKIE)?.value;
  if (refreshToken) {
    await callBackend(request, "/auth/logout", {
      method: "POST",
      json: { refresh_token: refreshToken },
      authenticated: false,
    });
  }
  const response = new NextResponse(null, { status: 204 });
  applyCookies(response.cookies, clearedSessionCookies());
  return response;
}
