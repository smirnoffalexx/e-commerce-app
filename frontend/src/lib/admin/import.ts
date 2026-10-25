import type { ImportMode, ImportOptions, ImportReport } from "@/lib/api/types";

export const MAX_IMPORT_BYTES = 5 * 1024 * 1024;

export const DEFAULT_IMPORT_OPTIONS: Omit<ImportOptions, "dry_run"> = {
  mode: "upsert",
  atomic: false,
  create_missing_categories: false,
};

export const IMPORT_MODES: { value: ImportMode; label: string; description: string }[] = [
  { value: "upsert", label: "Create and update", description: "New SKUs are created, existing ones are updated." },
  { value: "skip", label: "Create only", description: "New SKUs are created, existing ones are left untouched." },
  { value: "fail", label: "Strict", description: "An existing SKU is reported as a row error." },
];

export interface FileInfo {
  name: string;
  size: number;
  lastModified: number;
}

export function validateImportFile(file: FileInfo): string | null {
  if (!file.name.toLowerCase().endsWith(".csv")) {
    return "Choose a .csv file.";
  }
  if (file.size === 0) {
    return "The file is empty.";
  }
  if (file.size > MAX_IMPORT_BYTES) {
    return "The file is larger than 5 MB.";
  }
  return null;
}

export function previewKey(file: FileInfo, options: Omit<ImportOptions, "dry_run">): string {
  return JSON.stringify([file.name, file.size, file.lastModified, options.mode, options.atomic, options.create_missing_categories]);
}

export type ApplyState = { ok: true } | { ok: false; reason: string };

export function applyState(preview: { key: string; report: ImportReport } | null, currentKey: string | null): ApplyState {
  if (!preview || !currentKey) {
    return { ok: false, reason: "Run a preview first." };
  }
  if (preview.key !== currentKey) {
    return { ok: false, reason: "The file or options changed since the preview. Run it again." };
  }
  if (preview.report.atomic && preview.report.failed > 0) {
    return { ok: false, reason: "All-or-nothing mode would save nothing because some rows fail. Fix the file or turn the mode off." };
  }
  if (preview.report.created + preview.report.updated + preview.report.created_categories.length === 0) {
    return { ok: false, reason: "There is nothing to import." };
  }
  return { ok: true };
}

export function formatBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / 1024 / 1024).toFixed(1)} MB`;
}
