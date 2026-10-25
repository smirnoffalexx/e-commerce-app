import { readFileSync } from "node:fs";
import path from "node:path";
import { randomUUID } from "node:crypto";
import type { BrowserContext } from "@playwright/test";

// Test data is prepared straight through the Go API; the browser only does what a user would do.

export const API_URL = process.env.E2E_API_URL ?? "http://localhost:8080/api/v1";
export const PASSWORD = "correct-horse-battery-staple";
export const SESSION_FILE = path.join(__dirname, "..", ".auth", "superadmin.json");

export interface Session {
  access_token: string;
  refresh_token: string;
  expires_in: number;
  refresh_token_expires_at: string;
  user: { id: string; email: string; role: string };
}

export interface Product {
  id: string;
  sku: string;
  name: string;
  price: string;
  stock: number;
  available: number;
  category_id: string;
  version: number;
}

export function unique(prefix: string): string {
  return `${prefix}-${randomUUID().slice(0, 8)}`;
}

// SUPERADMIN_* come from the environment or from the root .env used by docker compose.
export function superAdminCredentials(): { email: string; password: string } {
  const env = { ...readDotEnv(), ...process.env };
  const email = env.SUPERADMIN_EMAIL;
  const password = env.SUPERADMIN_PASSWORD;
  if (!email || !password) {
    throw new Error("SUPERADMIN_EMAIL and SUPERADMIN_PASSWORD must be set (env or ../.env)");
  }
  return { email, password };
}

function readDotEnv(): Record<string, string> {
  try {
    const raw = readFileSync(path.join(__dirname, "..", "..", ".env"), "utf8");
    return Object.fromEntries(
      raw
        .split("\n")
        .map((line) => line.trim())
        .filter((line) => line && !line.startsWith("#") && line.includes("="))
        .map((line) => {
          const i = line.indexOf("=");
          return [line.slice(0, i).trim(), line.slice(i + 1).trim().replace(/^["']|["']$/g, "")];
        }),
    );
  } catch {
    return {};
  }
}

export async function api<T>(method: string, route: string, body?: unknown, token?: string): Promise<T> {
  const response = await fetch(API_URL + route, {
    method,
    headers: {
      ...(body === undefined ? {} : { "Content-Type": "application/json" }),
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
    },
    body: body === undefined ? undefined : JSON.stringify(body),
  });
  const text = await response.text();
  if (!response.ok) {
    throw new Error(`${method} ${route}: ${response.status} ${text}`);
  }
  return (text ? JSON.parse(text) : undefined) as T;
}

export function login(email: string, password: string): Promise<Session> {
  return api<Session>("POST", "/auth/login", { email, password });
}

export function superAdminSession(): Session {
  return JSON.parse(readFileSync(SESSION_FILE, "utf8")) as Session;
}

export async function registerCustomer(): Promise<{ email: string; session: Session }> {
  const email = `${unique("customer")}@e2e.test`;
  await api("POST", "/auth/register", { email, password: PASSWORD });
  return { email, session: await login(email, PASSWORD) };
}

export async function createProduct(options: { name?: string; price?: string; stock?: number } = {}): Promise<Product> {
  const token = superAdminSession().access_token;
  const category = await api<{ id: string }>("POST", "/categories", { name: unique("E2E Category") }, token);
  return api<Product>(
    "POST",
    "/products",
    {
      sku: unique("E2E").toUpperCase(),
      name: options.name ?? unique("E2E Product"),
      description: "Created by the e2e suite",
      category_id: category.id,
      price: options.price ?? "25.00",
      stock: options.stock ?? 5,
      weight_kg: "0.5",
    },
    token,
  );
}

export async function setPrice(product: Product, price: string): Promise<Product> {
  const current = await getProduct(product.id);
  return api<Product>(
    "PUT",
    `/products/${product.id}`,
    {
      sku: current.sku,
      name: current.name,
      description: "Created by the e2e suite",
      category_id: current.category_id,
      price,
      stock: current.stock,
      weight_kg: "0.5",
      version: current.version,
    },
    superAdminSession().access_token,
  );
}

export function getProduct(id: string): Promise<Product> {
  return api<Product>("GET", `/products/${id}`);
}

// Signs the browser in the way the BFF does: httpOnly session cookies on the frontend origin.
export async function signIn(context: BrowserContext, baseURL: string, session: Session) {
  const url = new URL(baseURL);
  const common = { domain: url.hostname, path: "/", httpOnly: true, secure: false, sameSite: "Lax" as const };
  await context.addCookies([
    { ...common, name: "shop_access", value: session.access_token },
    { ...common, name: "shop_refresh", value: session.refresh_token },
  ]);
}
