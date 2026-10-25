import { CircleCheckIcon, CircleXIcon, EyeIcon } from "lucide-react";
import type { ImportReport } from "@/lib/api/types";
import { cn } from "@/lib/utils";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";

const MAX_ERRORS_SHOWN = 200;

export function ImportReportView({ report }: { report: ImportReport }) {
  const stats = [
    { label: report.dry_run ? "Would create" : "Created", value: report.created, tone: "text-emerald-600" },
    { label: report.dry_run ? "Would update" : "Updated", value: report.updated, tone: "text-primary" },
    { label: "Skipped", value: report.skipped, tone: "text-muted-foreground" },
    { label: "Failed", value: report.failed, tone: report.failed > 0 ? "text-destructive" : "text-muted-foreground" },
  ];

  return (
    <div className="flex flex-col gap-4">
      <ReportBanner report={report} />
      <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
        {stats.map((stat) => (
          <div key={stat.label} className="rounded-xl border bg-card p-4">
            <p className="text-sm text-muted-foreground">{stat.label}</p>
            <p className={cn("mt-1 text-2xl font-semibold tabular-nums", stat.tone)}>{stat.value}</p>
          </div>
        ))}
      </div>
      <p className="text-sm text-muted-foreground">
        {report.total_rows} data rows{report.blank_rows > 0 ? `, ${report.blank_rows} blank rows ignored` : ""}.
      </p>
      {report.created_categories.length > 0 && (
        <div className="flex flex-wrap items-center gap-2 text-sm">
          <span className="text-muted-foreground">{report.dry_run ? "New categories:" : "Created categories:"}</span>
          {report.created_categories.map((name) => (
            <Badge key={name} variant="secondary">
              {name}
            </Badge>
          ))}
        </div>
      )}
      {report.duplicates.length > 0 && (
        <Card>
          <CardHeader>
            <CardTitle>Duplicate SKUs</CardTitle>
            <CardDescription>The same SKU appears more than once; only the last row is applied.</CardDescription>
          </CardHeader>
          <CardContent>
            <ul className="flex flex-col gap-1 text-sm">
              {report.duplicates.map((duplicate) => (
                <li key={duplicate.sku}>
                  <span className="font-mono">{duplicate.sku}</span>
                  <span className="text-muted-foreground">
                    {" "}
                    on lines {duplicate.lines.join(", ")}, applied line {duplicate.applied_line}
                  </span>
                </li>
              ))}
            </ul>
          </CardContent>
        </Card>
      )}
      {report.errors.length > 0 && (
        <Card>
          <CardHeader>
            <CardTitle>Row errors</CardTitle>
            <CardDescription>Line numbers count the header as line 1.</CardDescription>
          </CardHeader>
          <CardContent className="px-0">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead className="w-16 pl-6">Line</TableHead>
                  <TableHead>SKU</TableHead>
                  <TableHead>Field</TableHead>
                  <TableHead className="pr-6">Problem</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {report.errors.slice(0, MAX_ERRORS_SHOWN).map((error, index) => (
                  <TableRow key={`${error.line}-${error.field}-${index}`}>
                    <TableCell className="pl-6 tabular-nums">{error.line}</TableCell>
                    <TableCell className="font-mono text-xs">{error.sku}</TableCell>
                    <TableCell className="font-mono text-xs">{error.field}</TableCell>
                    <TableCell className="pr-6 whitespace-normal">{error.message}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
            {report.errors.length > MAX_ERRORS_SHOWN && (
              <p className="px-6 pt-3 text-sm text-muted-foreground">and {report.errors.length - MAX_ERRORS_SHOWN} more</p>
            )}
          </CardContent>
        </Card>
      )}
    </div>
  );
}

function ReportBanner({ report }: { report: ImportReport }) {
  if (report.dry_run) {
    return (
      <Alert>
        <EyeIcon />
        <AlertTitle>Preview: nothing has been saved yet</AlertTitle>
        <AlertDescription>Check the numbers and errors below, then import.</AlertDescription>
      </Alert>
    );
  }
  if (!report.committed) {
    return (
      <Alert variant="destructive">
        <CircleXIcon />
        <AlertTitle>Nothing was saved</AlertTitle>
        <AlertDescription>All-or-nothing mode rolled the import back because some rows failed.</AlertDescription>
      </Alert>
    );
  }
  return (
    <Alert className="border-emerald-500/40 bg-emerald-500/5">
      <CircleCheckIcon className="text-emerald-600" />
      <AlertTitle>Import finished</AlertTitle>
      <AlertDescription>
        {report.failed > 0 ? "Valid rows were saved; rows with errors were skipped." : "All rows were saved."}
      </AlertDescription>
    </Alert>
  );
}
