import type { Session } from "@/lib/api/types";

export type RefreshResult =
  | { ok: true; session: Session }
  | { ok: false; reason: "invalid" | "unavailable" };

export type RefreshFn = (refreshToken: string) => Promise<RefreshResult>;

interface Entry {
  promise: Promise<RefreshResult>;
  settledAt?: number;
}

export interface RefresherOptions {
  reuseWindowMs?: number;
  now?: () => number;
}

export function createRefresher(refresh: RefreshFn, options: RefresherOptions = {}): RefreshFn {
  const reuseWindowMs = options.reuseWindowMs ?? 15_000;
  const now = options.now ?? Date.now;
  const entries = new Map<string, Entry>();

  function sweep() {
    const limit = now() - reuseWindowMs;
    for (const [token, entry] of entries) {
      if (entry.settledAt !== undefined && entry.settledAt < limit) {
        entries.delete(token);
      }
    }
  }

  return (refreshToken) => {
    sweep();
    const existing = entries.get(refreshToken);
    if (existing) {
      return existing.promise;
    }
    const entry: Entry = {
      promise: refresh(refreshToken)
        .catch((): RefreshResult => ({ ok: false, reason: "unavailable" }))
        .then((result) => {
          if (!result.ok && result.reason === "unavailable") {
            entries.delete(refreshToken);
          } else {
            entry.settledAt = now();
          }
          return result;
        }),
    };
    entries.set(refreshToken, entry);
    return entry.promise;
  };
}
