import type { NextRequest } from "next/server";
import { establishSession, readJson } from "@/lib/server/auth";

export async function POST(request: NextRequest) {
  return establishSession(request, "/auth/login", await readJson(request));
}
