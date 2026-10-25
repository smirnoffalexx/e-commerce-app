import { CalendarIcon, KeyRoundIcon, MailIcon, ReceiptIcon, ShieldIcon } from "lucide-react";
import type { Metadata } from "next";
import Link from "next/link";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from "@/components/ui/card";
import { formatDate } from "@/lib/format";
import { roleLabels } from "@/lib/roles";
import { requireUser } from "@/lib/server/api";

export const metadata: Metadata = { title: "Account" };

export default async function AccountPage() {
  const user = await requireUser(undefined, "/account");
  return (
    <div className="mx-auto w-full max-w-3xl px-4 py-10 sm:px-6">
      <h1 className="text-3xl font-semibold tracking-tight">Account</h1>
      <p className="mt-1 text-muted-foreground">Your profile and security settings</p>
      <Card className="mt-8">
        <CardHeader>
          <CardTitle>Profile</CardTitle>
          <CardDescription>Signed in with email and password</CardDescription>
        </CardHeader>
        <CardContent>
          <dl className="grid gap-4 sm:grid-cols-3">
            <Detail icon={<MailIcon />} label="Email" value={user.email} />
            <Detail icon={<ShieldIcon />} label="Role" value={<Badge variant="secondary">{roleLabels[user.role]}</Badge>} />
            <Detail icon={<CalendarIcon />} label="Member since" value={formatDate(user.created_at)} />
          </dl>
        </CardContent>
        <CardFooter className="gap-2 border-t">
          {user.role === "USER" && (
            <Button variant="outline" asChild>
              <Link href="/orders">
                <ReceiptIcon />
                Order history
              </Link>
            </Button>
          )}
          <Button variant="outline" asChild>
            <Link href="/account/password">
              <KeyRoundIcon />
              Change password
            </Link>
          </Button>
        </CardFooter>
      </Card>
    </div>
  );
}

function Detail({ icon, label, value }: { icon: React.ReactNode; label: string; value: React.ReactNode }) {
  return (
    <div className="flex flex-col gap-1.5">
      <dt className="flex items-center gap-1.5 text-sm text-muted-foreground [&_svg]:size-4">
        {icon}
        {label}
      </dt>
      <dd className="truncate font-medium">{value}</dd>
    </div>
  );
}
