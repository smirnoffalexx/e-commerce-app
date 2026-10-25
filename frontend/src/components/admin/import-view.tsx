"use client";

import { FileSpreadsheetIcon, PlayIcon, ScanEyeIcon, UploadIcon, XIcon } from "lucide-react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useState, useTransition } from "react";
import { toast } from "sonner";
import { DEFAULT_IMPORT_OPTIONS, IMPORT_MODES, applyState, formatBytes, previewKey, validateImportFile } from "@/lib/admin/import";
import { adminApi } from "@/lib/api/admin";
import { errorMessage } from "@/lib/api/problem";
import type { ImportMode, ImportReport } from "@/lib/api/types";
import { cn } from "@/lib/utils";
import { FormAlert } from "@/components/auth/form-alert";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from "@/components/ui/card";
import { Checkbox } from "@/components/ui/checkbox";
import { Field, FieldContent, FieldDescription, FieldLabel, FieldSet, FieldLegend } from "@/components/ui/field";
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group";
import { Spinner } from "@/components/ui/spinner";
import { ImportReportView } from "./import-report";

type Options = typeof DEFAULT_IMPORT_OPTIONS;

export function ImportView() {
  const router = useRouter();
  const [file, setFile] = useState<File | null>(null);
  const [options, setOptions] = useState<Options>(DEFAULT_IMPORT_OPTIONS);
  const [preview, setPreview] = useState<{ key: string; report: ImportReport } | null>(null);
  const [result, setResult] = useState<ImportReport | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [dragging, setDragging] = useState(false);
  const [pending, startTransition] = useTransition();
  const [action, setAction] = useState<"preview" | "import" | null>(null);

  const currentKey = file ? previewKey(file, options) : null;
  const state = applyState(preview, currentKey);

  function chooseFile(next: File | null) {
    setResult(null);
    setError(null);
    if (!next) {
      setFile(null);
      return;
    }
    const problem = validateImportFile(next);
    if (problem) {
      setError(problem);
      return;
    }
    setFile(next);
  }

  function run(dryRun: boolean) {
    if (!file) {
      return;
    }
    setError(null);
    setAction(dryRun ? "preview" : "import");
    const key = previewKey(file, options);
    startTransition(async () => {
      try {
        const report = await adminApi.importProducts(file, { ...options, dry_run: dryRun });
        if (dryRun) {
          setPreview({ key, report });
          setResult(null);
        } else {
          setResult(report);
          setPreview(null);
          if (report.committed) {
            toast.success(`Imported: ${report.created} created, ${report.updated} updated`);
          }
          router.refresh();
        }
      } catch (caught) {
        setError(errorMessage(caught));
      }
    });
  }

  const shown = result ?? (preview?.key === currentKey ? preview.report : null);

  return (
    <div className="flex flex-col gap-6">
      <Card>
        <CardHeader>
          <CardTitle>File</CardTitle>
          <CardDescription>
            CSV with the header <code className="font-mono text-xs">name,sku,description,category,price,stock,weight_kg</code>, up to 5 MB.
          </CardDescription>
        </CardHeader>
        <CardContent className="flex flex-col gap-6">
          {file ? (
            <div className="flex items-center gap-3 rounded-lg border bg-muted/40 p-3">
              <FileSpreadsheetIcon className="size-8 text-primary" />
              <div className="min-w-0 flex-1">
                <p className="truncate font-medium">{file.name}</p>
                <p className="text-sm text-muted-foreground">{formatBytes(file.size)}</p>
              </div>
              <Button variant="ghost" size="icon-sm" aria-label="Remove file" disabled={pending} onClick={() => chooseFile(null)}>
                <XIcon />
              </Button>
            </div>
          ) : (
            <label
              onDragOver={(event) => {
                event.preventDefault();
                setDragging(true);
              }}
              onDragLeave={() => setDragging(false)}
              onDrop={(event) => {
                event.preventDefault();
                setDragging(false);
                chooseFile(event.dataTransfer.files[0] ?? null);
              }}
              className={cn(
                "flex cursor-pointer flex-col items-center justify-center gap-2 rounded-xl border-2 border-dashed p-10 text-center transition-colors hover:border-primary/50 hover:bg-primary/5",
                dragging && "border-primary bg-primary/5",
              )}
            >
              <UploadIcon className="size-8 text-muted-foreground" />
              <span className="font-medium">Drop a CSV file here or click to choose</span>
              <input
                type="file"
                accept=".csv,text/csv"
                className="sr-only"
                onChange={(event) => {
                  chooseFile(event.target.files?.[0] ?? null);
                  event.target.value = "";
                }}
              />
            </label>
          )}

          <FieldSet>
            <FieldLegend variant="label">Existing SKUs</FieldLegend>
            <RadioGroup value={options.mode} onValueChange={(mode) => setOptions({ ...options, mode: mode as ImportMode })} disabled={pending}>
              {IMPORT_MODES.map((mode) => (
                <Field key={mode.value} orientation="horizontal">
                  <RadioGroupItem value={mode.value} id={`mode-${mode.value}`} />
                  <FieldContent>
                    <FieldLabel htmlFor={`mode-${mode.value}`}>{mode.label}</FieldLabel>
                    <FieldDescription>{mode.description}</FieldDescription>
                  </FieldContent>
                </Field>
              ))}
            </RadioGroup>
          </FieldSet>

          <FieldSet>
            <FieldLegend variant="label">Options</FieldLegend>
            <Field orientation="horizontal">
              <Checkbox
                id="atomic"
                checked={options.atomic}
                disabled={pending}
                onCheckedChange={(checked) => setOptions({ ...options, atomic: checked === true })}
              />
              <FieldContent>
                <FieldLabel htmlFor="atomic">All or nothing</FieldLabel>
                <FieldDescription>Save nothing if any row fails. Otherwise valid rows are saved and invalid ones reported.</FieldDescription>
              </FieldContent>
            </Field>
            <Field orientation="horizontal">
              <Checkbox
                id="create-categories"
                checked={options.create_missing_categories}
                disabled={pending}
                onCheckedChange={(checked) => setOptions({ ...options, create_missing_categories: checked === true })}
              />
              <FieldContent>
                <FieldLabel htmlFor="create-categories">Create missing categories</FieldLabel>
                <FieldDescription>Otherwise a row with an unknown category is an error.</FieldDescription>
              </FieldContent>
            </Field>
          </FieldSet>
          <FormAlert message={error} />
        </CardContent>
        <CardFooter className="flex flex-col items-stretch gap-3 border-t sm:flex-row sm:items-center sm:justify-between">
          <p className="text-sm text-muted-foreground">{file && !state.ok && preview ? state.reason : "Preview first: it runs the whole import and rolls it back."}</p>
          <div className="flex gap-2">
            <Button variant="outline" disabled={!file || pending} onClick={() => run(true)}>
              {pending && action === "preview" ? <Spinner /> : <ScanEyeIcon />}
              Preview
            </Button>
            <Button disabled={!state.ok || pending} onClick={() => run(false)}>
              {pending && action === "import" ? <Spinner /> : <PlayIcon />}
              Import
            </Button>
          </div>
        </CardFooter>
      </Card>

      {shown && <ImportReportView report={shown} />}
      {result?.committed && (
        <Button variant="outline" className="self-start" asChild>
          <Link href="/admin/products">Go to products</Link>
        </Button>
      )}
    </div>
  );
}
