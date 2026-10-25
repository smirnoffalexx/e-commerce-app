"use client";

import { CheckIcon, PencilIcon, PlusIcon, Trash2Icon, XIcon } from "lucide-react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useRef, useState } from "react";
import { toast } from "sonner";
import { useApiForm } from "@/hooks/use-api-form";
import { adminApi } from "@/lib/api/admin";
import type { Category } from "@/lib/api/types";
import { formatDate } from "@/lib/format";
import { CATEGORY_NAME_MAX_LENGTH, categorySchema } from "@/lib/validation/category";
import { ConfirmButton } from "@/components/admin/confirm-button";
import { FormAlert } from "@/components/auth/form-alert";
import { Button } from "@/components/ui/button";
import { FieldError } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { Spinner } from "@/components/ui/spinner";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";

export function CategoryManager({ categories }: { categories: Category[] }) {
  const [editing, setEditing] = useState<string | null>(null);
  return (
    <div className="flex flex-col gap-6">
      <CreateCategory />
      <div className="rounded-xl border bg-card">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead className="pl-4">Name</TableHead>
              <TableHead className="hidden sm:table-cell">Slug</TableHead>
              <TableHead className="hidden md:table-cell">Created</TableHead>
              <TableHead className="pr-4 text-right">
                <span className="sr-only">Actions</span>
              </TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {categories.length === 0 && (
              <TableRow>
                <TableCell colSpan={4} className="h-24 text-center text-muted-foreground">
                  No categories yet.
                </TableCell>
              </TableRow>
            )}
            {categories.map((category) =>
              editing === category.id ? (
                <RenameRow key={category.id} category={category} onDone={() => setEditing(null)} />
              ) : (
                <TableRow key={category.id}>
                  <TableCell className="pl-4 font-medium">
                    <Link href={`/admin/products?category_id=${category.id}`} className="hover:underline">
                      {category.name}
                    </Link>
                  </TableCell>
                  <TableCell className="hidden font-mono text-xs text-muted-foreground sm:table-cell">{category.slug}</TableCell>
                  <TableCell className="hidden text-muted-foreground md:table-cell">{formatDate(category.created_at)}</TableCell>
                  <TableCell className="pr-4">
                    <div className="flex justify-end gap-1">
                      <Button variant="ghost" size="icon-sm" aria-label={`Rename ${category.name}`} onClick={() => setEditing(category.id)}>
                        <PencilIcon />
                      </Button>
                      <DeleteCategory category={category} />
                    </div>
                  </TableCell>
                </TableRow>
              ),
            )}
          </TableBody>
        </Table>
      </div>
    </div>
  );
}

function CreateCategory() {
  const router = useRouter();
  const form = useRef<HTMLFormElement>(null);
  const { errors, formError, pending, onSubmit } = useApiForm(categorySchema, async ({ name }) => {
    await adminApi.createCategory(name);
    form.current?.reset();
    toast.success(`Category ${name} created`);
    router.refresh();
  });
  return (
    <form ref={form} noValidate onSubmit={onSubmit} className="flex flex-col gap-2">
      <div className="flex gap-2">
        <Input name="name" placeholder="New category name" aria-label="New category name" maxLength={CATEGORY_NAME_MAX_LENGTH} className="bg-card sm:max-w-sm" aria-invalid={errors.name ? true : undefined} />
        <Button type="submit" disabled={pending}>
          {pending ? <Spinner /> : <PlusIcon />}
          Add
        </Button>
      </div>
      <FieldError>{errors.name}</FieldError>
      <FormAlert message={formError} />
    </form>
  );
}

function RenameRow({ category, onDone }: { category: Category; onDone: () => void }) {
  const router = useRouter();
  const { errors, formError, pending, onSubmit } = useApiForm(categorySchema, async ({ name }) => {
    await adminApi.renameCategory(category.id, name);
    toast.success("Category renamed");
    onDone();
    router.refresh();
  });
  return (
    <TableRow>
      <TableCell colSpan={4} className="px-4">
        <form noValidate onSubmit={onSubmit} className="flex flex-col gap-2">
          <div className="flex items-center gap-2">
            <Input
              name="name"
              defaultValue={category.name}
              aria-label="Category name"
              maxLength={CATEGORY_NAME_MAX_LENGTH}
              autoFocus
              className="sm:max-w-sm"
              aria-invalid={errors.name ? true : undefined}
              onKeyDown={(event) => event.key === "Escape" && onDone()}
            />
            <Button type="submit" size="icon-sm" aria-label="Save" disabled={pending}>
              {pending ? <Spinner /> : <CheckIcon />}
            </Button>
            <Button type="button" variant="ghost" size="icon-sm" aria-label="Cancel" onClick={onDone}>
              <XIcon />
            </Button>
          </div>
          <FieldError>{errors.name ?? formError}</FieldError>
        </form>
      </TableCell>
    </TableRow>
  );
}

function DeleteCategory({ category }: { category: Category }) {
  const router = useRouter();
  return (
    <ConfirmButton
      trigger={
        <Button variant="ghost" size="icon-sm" aria-label={`Delete ${category.name}`} className="text-muted-foreground hover:text-destructive">
          <Trash2Icon />
        </Button>
      }
      title={`Delete ${category.name}?`}
      description="Only empty categories can be deleted. Move or delete its products first."
      confirmLabel="Delete"
      onConfirm={async () => {
        await adminApi.deleteCategory(category.id);
        toast.success(`Category ${category.name} deleted`);
        router.refresh();
      }}
    />
  );
}
