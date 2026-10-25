"use client";

import { CheckIcon, CopyIcon, KeyRoundIcon } from "lucide-react";
import { useState } from "react";
import type { AdminCredentials } from "@/lib/api/types";
import { formatDateTime } from "@/lib/format";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";

export function CredentialsDialog({ credentials, onClose }: { credentials: AdminCredentials | null; onClose: () => void }) {
  const [copied, setCopied] = useState(false);
  return (
    <Dialog
      open={credentials !== null}
      onOpenChange={(open) => {
        if (!open) {
          setCopied(false);
          onClose();
        }
      }}
    >
      {credentials && (
        <DialogContent onInteractOutside={(event) => event.preventDefault()}>
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2">
              <KeyRoundIcon className="size-5" />
              Temporary password
            </DialogTitle>
            <DialogDescription>
              For <span className="font-medium text-foreground">{credentials.user.email}</span>. Share it through a secure channel; the administrator must change it at first sign-in.
            </DialogDescription>
          </DialogHeader>
          <div className="flex items-center gap-2 rounded-lg border bg-muted/50 p-3">
            <code className="flex-1 font-mono text-lg tracking-wider break-all select-all">{credentials.temporary_password}</code>
            <Button
              variant="outline"
              size="sm"
              onClick={async () => {
                await navigator.clipboard.writeText(credentials.temporary_password);
                setCopied(true);
              }}
            >
              {copied ? <CheckIcon /> : <CopyIcon />}
              {copied ? "Copied" : "Copy"}
            </Button>
          </div>
          <Alert>
            <AlertDescription>
              This password is shown only once and expires {formatDateTime(credentials.temporary_password_expires_at)}.
            </AlertDescription>
          </Alert>
          <DialogFooter>
            <Button onClick={onClose}>Done</Button>
          </DialogFooter>
        </DialogContent>
      )}
    </Dialog>
  );
}
