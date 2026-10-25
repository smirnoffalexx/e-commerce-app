"use client";

import { KeyRoundIcon, PlusIcon, ShieldCheckIcon, ShieldOffIcon } from "lucide-react";
import { useRouter } from "next/navigation";
import { useRef, useState } from "react";
import { toast } from "sonner";
import { useApiForm } from "@/hooks/use-api-form";
import { adminApi } from "@/lib/api/admin";
import type { AdminCredentials, User } from "@/lib/api/types";
import { formatDate } from "@/lib/format";
import { adminSchema } from "@/lib/validation/auth";
import { ConfirmButton } from "@/components/admin/confirm-button";
import { FormAlert } from "@/components/auth/form-alert";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { FieldError } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { Spinner } from "@/components/ui/spinner";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { CredentialsDialog } from "./credentials-dialog";

export function AdminManager({ admins }: { admins: User[] }) {
  const router = useRouter();
  const form = useRef<HTMLFormElement>(null);
  const [credentials, setCredentials] = useState<AdminCredentials | null>(null);
  const { errors, formError, pending, onSubmit } = useApiForm(adminSchema, async ({ email }) => {
    setCredentials(await adminApi.createAdmin(email));
    form.current?.reset();
    router.refresh();
  });

  return (
    <div className="flex flex-col gap-6">
      <form ref={form} noValidate onSubmit={onSubmit} className="flex flex-col gap-2">
        <div className="flex gap-2">
          <Input name="email" type="email" placeholder="new.admin@example.com" aria-label="Administrator email" className="bg-card sm:max-w-sm" aria-invalid={errors.email ? true : undefined} />
          <Button type="submit" disabled={pending}>
            {pending ? <Spinner /> : <PlusIcon />}
            Add administrator
          </Button>
        </div>
        <FieldError>{errors.email}</FieldError>
        <FormAlert message={formError} />
      </form>

      <div className="rounded-xl border bg-card">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead className="pl-4">Email</TableHead>
              <TableHead>Status</TableHead>
              <TableHead className="hidden md:table-cell">Added</TableHead>
              <TableHead className="pr-4 text-right">
                <span className="sr-only">Actions</span>
              </TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {admins.length === 0 && (
              <TableRow>
                <TableCell colSpan={4} className="h-24 text-center text-muted-foreground">
                  No administrators yet.
                </TableCell>
              </TableRow>
            )}
            {admins.map((admin) => (
              <TableRow key={admin.id}>
                <TableCell className="pl-4 font-medium">{admin.email}</TableCell>
                <TableCell>
                  <div className="flex flex-wrap gap-1">
                    {admin.active ? <Badge variant="secondary">Active</Badge> : <Badge variant="outline" className="text-muted-foreground">Deactivated</Badge>}
                    {admin.active && admin.must_change_password && (
                      <Badge variant="outline" className="border-amber-500/40 text-amber-700 dark:text-amber-400">
                        Temporary password
                      </Badge>
                    )}
                  </div>
                </TableCell>
                <TableCell className="hidden text-muted-foreground md:table-cell">{formatDate(admin.created_at)}</TableCell>
                <TableCell className="pr-4">
                  <div className="flex justify-end gap-1">
                    {admin.active && (
                      <ConfirmButton
                        trigger={
                          <Button variant="ghost" size="sm">
                            <KeyRoundIcon />
                            Reset password
                          </Button>
                        }
                        title={`Reset the password of ${admin.email}?`}
                        description="Their sessions are signed out and they get a new temporary password valid for 24 hours."
                        confirmLabel="Reset password"
                        destructive={false}
                        onConfirm={async () => {
                          setCredentials(await adminApi.resetAdminPassword(admin.id));
                          router.refresh();
                        }}
                      />
                    )}
                    {admin.active ? (
                      <ConfirmButton
                        trigger={
                          <Button variant="ghost" size="sm" className="text-muted-foreground hover:text-destructive">
                            <ShieldOffIcon />
                            Deactivate
                          </Button>
                        }
                        title={`Deactivate ${admin.email}?`}
                        description="They are signed out and cannot sign in until you activate the account again."
                        confirmLabel="Deactivate"
                        onConfirm={async () => {
                          await adminApi.deactivateAdmin(admin.id);
                          toast.success(`${admin.email} deactivated`);
                          router.refresh();
                        }}
                      />
                    ) : (
                      <ConfirmButton
                        trigger={
                          <Button variant="ghost" size="sm">
                            <ShieldCheckIcon />
                            Activate
                          </Button>
                        }
                        title={`Activate ${admin.email}?`}
                        description="They can sign in again with their current password."
                        confirmLabel="Activate"
                        destructive={false}
                        onConfirm={async () => {
                          await adminApi.activateAdmin(admin.id);
                          toast.success(`${admin.email} activated`);
                          router.refresh();
                        }}
                      />
                    )}
                  </div>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </div>
      <CredentialsDialog credentials={credentials} onClose={() => setCredentials(null)} />
    </div>
  );
}
