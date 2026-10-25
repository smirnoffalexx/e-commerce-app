import { api } from "./client";
import type { AdminCredentials, Category, ImportOptions, ImportReport, Product, ProductInput, User } from "./types";

export const adminApi = {
  createProduct: (input: ProductInput) => api<Product>("/products", { body: input }),
  updateProduct: (id: string, input: ProductInput, version: number) =>
    api<Product>(`/products/${encodeURIComponent(id)}`, { method: "PUT", body: { ...input, version } }),
  deleteProduct: (id: string) => api<void>(`/products/${encodeURIComponent(id)}`, { method: "DELETE" }),

  createCategory: (name: string) => api<Category>("/categories", { body: { name } }),
  renameCategory: (id: string, name: string) =>
    api<Category>(`/categories/${encodeURIComponent(id)}`, { method: "PUT", body: { name } }),
  deleteCategory: (id: string) => api<void>(`/categories/${encodeURIComponent(id)}`, { method: "DELETE" }),

  importProducts: (file: File, options: ImportOptions) => {
    const body = new FormData();
    body.set("file", file);
    return api<ImportReport>("/products/import", { method: "POST", body, query: { ...options } });
  },

  createAdmin: (email: string) => api<AdminCredentials>("/admins", { body: { email } }),
  resetAdminPassword: (id: string) => api<AdminCredentials>(`/admins/${encodeURIComponent(id)}/reset-password`, { method: "POST" }),
  deactivateAdmin: (id: string) => api<User>(`/admins/${encodeURIComponent(id)}/deactivate`, { method: "POST" }),
  activateAdmin: (id: string) => api<User>(`/admins/${encodeURIComponent(id)}/activate`, { method: "POST" }),
};
