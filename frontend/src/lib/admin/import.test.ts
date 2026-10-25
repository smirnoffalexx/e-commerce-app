import { describe, expect, it } from "vitest";
import type { ImportReport } from "@/lib/api/types";
import { DEFAULT_IMPORT_OPTIONS, MAX_IMPORT_BYTES, applyState, previewKey, validateImportFile } from "./import";

const file = { name: "catalog.CSV", size: 1200, lastModified: 1 };

function report(overrides: Partial<ImportReport> = {}): ImportReport {
  return {
    mode: "upsert",
    dry_run: true,
    atomic: false,
    create_missing_categories: false,
    committed: false,
    total_rows: 10,
    blank_rows: 0,
    created: 3,
    updated: 5,
    skipped: 0,
    failed: 2,
    created_categories: [],
    duplicates: [],
    errors: [],
    ...overrides,
  };
}

describe("validateImportFile", () => {
  it("checks the extension and the backend size limit before uploading", () => {
    expect(validateImportFile(file)).toBeNull();
    expect(validateImportFile({ ...file, name: "catalog.xlsx" })).toBe("Choose a .csv file.");
    expect(validateImportFile({ ...file, size: 0 })).toBe("The file is empty.");
    expect(validateImportFile({ ...file, size: MAX_IMPORT_BYTES + 1 })).toBe("The file is larger than 5 MB.");
  });
});

describe("applyState", () => {
  const key = previewKey(file, DEFAULT_IMPORT_OPTIONS);

  it("allows applying exactly what was previewed", () => {
    expect(applyState({ key, report: report() }, key)).toEqual({ ok: true });
  });

  it("requires a new preview after the file or any option changed", () => {
    expect(applyState(null, key).ok).toBe(false);
    const changed = previewKey(file, { ...DEFAULT_IMPORT_OPTIONS, mode: "skip" });
    expect(applyState({ key, report: report() }, changed)).toMatchObject({ ok: false, reason: expect.stringContaining("changed") });
    expect(applyState({ key, report: report() }, previewKey({ ...file, lastModified: 2 }, DEFAULT_IMPORT_OPTIONS)).ok).toBe(false);
  });

  it("blocks an all-or-nothing import that is known to roll back", () => {
    expect(applyState({ key, report: report({ atomic: true, failed: 1 }) }, key)).toMatchObject({ ok: false, reason: expect.stringContaining("All-or-nothing") });
  });

  it("blocks an import that would change nothing", () => {
    expect(applyState({ key, report: report({ created: 0, updated: 0, skipped: 10, failed: 0 }) }, key)).toEqual({
      ok: false,
      reason: "There is nothing to import.",
    });
    expect(applyState({ key, report: report({ created: 0, updated: 0, created_categories: ["Toys"] }) }, key).ok).toBe(true);
  });
});
