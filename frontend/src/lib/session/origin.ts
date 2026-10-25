const SAFE_METHODS = new Set(["GET", "HEAD", "OPTIONS"]);

export function allowedOrigins(configured: string | undefined, requestOrigin: string): string[] {
  const list = (configured ?? "")
    .split(",")
    .map((origin) => origin.trim().replace(/\/+$/, ""))
    .filter(Boolean);
  return list.length > 0 ? list : [requestOrigin];
}

export function isTrustedRequest(
  method: string,
  headers: Pick<Headers, "get">,
  allowed: string[],
): boolean {
  if (SAFE_METHODS.has(method.toUpperCase())) {
    return true;
  }
  const origin = headers.get("origin");
  if (origin) {
    return allowed.includes(origin);
  }
  return headers.get("sec-fetch-site") === "same-origin";
}
