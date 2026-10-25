import { ApiError, toProblem } from "./problem";

type Query = Record<string, string | number | boolean | undefined | null>;

export interface RequestOptions {
  method?: string;
  body?: unknown;
  query?: Query;
  headers?: HeadersInit;
  signal?: AbortSignal;
}

export function buildQuery(query?: Query): string {
  if (!query) {
    return "";
  }
  const params = new URLSearchParams();
  for (const [key, value] of Object.entries(query)) {
    if (value !== undefined && value !== null && value !== "") {
      params.set(key, String(value));
    }
  }
  const text = params.toString();
  return text ? `?${text}` : "";
}

export async function api<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const headers = new Headers(options.headers);
  let body: BodyInit | undefined;
  if (options.body instanceof FormData) {
    body = options.body;
  } else if (options.body !== undefined) {
    headers.set("Content-Type", "application/json");
    body = JSON.stringify(options.body);
  }

  let response: Response;
  try {
    response = await fetch(`/api${path}${buildQuery(options.query)}`, {
      method: options.method ?? (body ? "POST" : "GET"),
      headers,
      body,
      signal: options.signal,
      credentials: "same-origin",
    });
  } catch (error) {
    if (error instanceof DOMException && error.name === "AbortError") {
      throw error;
    }
    throw new ApiError({ status: 0, code: "network_error" });
  }

  if (response.status === 204) {
    return undefined as T;
  }
  const data = await response.json().catch(() => undefined);
  if (!response.ok) {
    throw new ApiError(toProblem(response.status, data));
  }
  return data as T;
}
