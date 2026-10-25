import type { Role } from "@/lib/api/types";

export const STAFF_ROLES: Role[] = ["ADMIN", "SUPERADMIN"];

export function isStaff(role: Role): boolean {
  return STAFF_ROLES.includes(role);
}

export const roleLabels: Record<Role, string> = {
  USER: "Customer",
  ADMIN: "Admin",
  SUPERADMIN: "Super admin",
};

export function canPurchase(user: { role: Role } | null): boolean {
  return !user || user.role === "USER";
}
