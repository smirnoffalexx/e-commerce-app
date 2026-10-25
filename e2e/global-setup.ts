import { mkdirSync, writeFileSync } from "node:fs";
import path from "node:path";
import { SESSION_FILE, login, superAdminCredentials } from "./support/api";

// One superadmin login for the whole run: specs reuse its tokens to seed data and to sign in.
export default async function globalSetup() {
  const { email, password } = superAdminCredentials();
  const session = await login(email, password);
  mkdirSync(path.dirname(SESSION_FILE), { recursive: true });
  writeFileSync(SESSION_FILE, JSON.stringify(session));
}
