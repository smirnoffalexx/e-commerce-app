"use client";

import { Trash2Icon } from "lucide-react";
import { useRouter } from "next/navigation";
import { toast } from "sonner";
import { adminApi } from "@/lib/api/admin";
import { Button } from "@/components/ui/button";
import { ConfirmButton } from "@/components/admin/confirm-button";

export function DeleteProductButton({ id, name }: { id: string; name: string }) {
  const router = useRouter();
  return (
    <ConfirmButton
      trigger={
        <Button variant="ghost" size="icon-sm" aria-label={`Delete ${name}`} className="text-muted-foreground hover:text-destructive">
          <Trash2Icon />
        </Button>
      }
      title={`Delete ${name}?`}
      description="The product disappears from the catalog and carts. Past orders keep their copy of the name and price."
      confirmLabel="Delete"
      onConfirm={async () => {
        await adminApi.deleteProduct(id);
        toast.success(`${name} deleted`);
        router.refresh();
      }}
    />
  );
}
