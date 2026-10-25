import type { NextRequest } from "next/server";
import { establishSession, readJson } from "@/lib/server/auth";

export async function POST(request: NextRequest) {
  return establishSession(request, "/auth/change-password", await readJson(request), true);
}
