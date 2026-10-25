import type { Metadata } from "next";
import { ImportView } from "@/components/admin/import-view";
import { PageHeader } from "@/components/admin/page-header";

export const metadata: Metadata = { title: "Import CSV" };

export default function AdminImportPage() {
  return (
    <>
      <PageHeader title="Import CSV" description="Add or update many products at once" />
      <ImportView />
    </>
  );
}
