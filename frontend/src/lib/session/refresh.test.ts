import { describe, expect, it, vi } from "vitest";
import type { Session } from "@/lib/api/types";
import { createRefresher, type RefreshResult } from "./refresh";

const session = { access_token: "new-access", refresh_token: "new-refresh" } as Session;

function deferred() {
  let resolve!: (value: RefreshResult) => void;
  const promise = new Promise<RefreshResult>((r) => (resolve = r));
  return { promise, resolve };
}

describe("createRefresher", () => {
  it("sends one backend refresh for concurrent requests with the same rotating token", async () => {
    const pending = deferred();
    const backend = vi.fn(() => pending.promise);
    const refresh = createRefresher(backend);

    const first = refresh("old");
    const second = refresh("old");
    pending.resolve({ ok: true, session });

    await expect(first).resolves.toEqual({ ok: true, session });
    await expect(second).resolves.toEqual({ ok: true, session });
    expect(backend).toHaveBeenCalledTimes(1);
  });

  it("reuses the rotated session for late requests that still carry the old token", async () => {
    let now = 0;
    const backend = vi.fn(async (): Promise<RefreshResult> => ({ ok: true, session }));
    const refresh = createRefresher(backend, { reuseWindowMs: 10_000, now: () => now });

    await refresh("old");
    now = 9_000;
    await refresh("old");
    expect(backend).toHaveBeenCalledTimes(1);

    now = 20_000;
    await refresh("old");
    expect(backend).toHaveBeenCalledTimes(2);
  });

  it("remembers a rejected token so a replay does not hit the backend again", async () => {
    const backend = vi.fn(async (): Promise<RefreshResult> => ({ ok: false, reason: "invalid" }));
    const refresh = createRefresher(backend);

    await refresh("stolen");
    await expect(refresh("stolen")).resolves.toEqual({ ok: false, reason: "invalid" });
    expect(backend).toHaveBeenCalledTimes(1);
  });

  it("retries after the backend was unavailable instead of caching the failure", async () => {
    const backend = vi
      .fn<(token: string) => Promise<RefreshResult>>()
      .mockRejectedValueOnce(new Error("ECONNREFUSED"))
      .mockResolvedValueOnce({ ok: true, session });
    const refresh = createRefresher(backend);

    await expect(refresh("old")).resolves.toEqual({ ok: false, reason: "unavailable" });
    await expect(refresh("old")).resolves.toEqual({ ok: true, session });
    expect(backend).toHaveBeenCalledTimes(2);
  });
});
