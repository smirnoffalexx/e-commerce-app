export const PASSWORD_CHANGE_PATH = "/account/password";

const PROTECTED_PREFIXES = ["/account", "/orders", "/checkout", "/admin"];

export interface NavigationState {
  pathname: string;
  search: string;
  authenticated: boolean;
  passwordChangeRequired: boolean;
}

function matches(pathname: string, prefix: string) {
  return pathname === prefix || pathname.startsWith(`${prefix}/`);
}

export function navigationRedirect(state: NavigationState): string | null {
  const { pathname, search, authenticated, passwordChangeRequired } = state;
  if (!authenticated) {
    if (PROTECTED_PREFIXES.some((prefix) => matches(pathname, prefix))) {
      return `/login?next=${encodeURIComponent(pathname + search)}`;
    }
    return null;
  }
  if (passwordChangeRequired && pathname !== PASSWORD_CHANGE_PATH) {
    return `${PASSWORD_CHANGE_PATH}?required=1`;
  }
  return null;
}

export function safeNextPath(next: string | null | undefined, fallback = "/"): string {
  if (!next || !next.startsWith("/") || next.startsWith("//") || next.startsWith("/\\")) {
    return fallback;
  }
  return next;
}
