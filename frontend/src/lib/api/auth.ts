import { api } from "./client";
import type { User } from "./types";

export interface AuthResult {
  user: User;
  password_change_required: boolean;
}

export const authApi = {
  login: (body: { email: string; password: string }) => api<AuthResult>("/auth/login", { body }),
  register: (body: { email: string; password: string }) => api<AuthResult>("/auth/register", { body }),
  changePassword: (body: { current_password: string; new_password: string }) =>
    api<AuthResult>("/auth/change-password", { body }),
  logout: () => api<void>("/auth/logout", { method: "POST" }),
};
