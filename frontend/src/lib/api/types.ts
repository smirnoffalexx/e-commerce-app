export type Role = "USER" | "ADMIN" | "SUPERADMIN";

export type Money = string;

export interface User {
  id: string;
  email: string;
  role: Role;
  active: boolean;
  must_change_password: boolean;
  created_at: string;
}

export interface Session {
  token_type: string;
  access_token: string;
  expires_in: number;
  access_token_expires_at: string;
  refresh_token: string;
  refresh_token_expires_at: string;
  password_change_required: boolean;
  user: User;
}

export interface Page<T> {
  items: T[];
  page: number;
  page_size: number;
  total: number;
  total_pages: number;
}

export interface Category {
  id: string;
  name: string;
  slug: string;
  created_at: string;
  updated_at: string;
}

export interface Product {
  id: string;
  sku: string;
  name: string;
  description: string;
  category?: Pick<Category, "id" | "name" | "slug">;
  category_id: string;
  price: Money;
  stock: number;
  reserved: number;
  available: number;
  in_stock: boolean;
  weight_kg: string;
  version: number;
  created_at: string;
  updated_at: string;
}

export type ProductSort = "relevance" | "newest" | "price_asc" | "price_desc" | "name_asc";

export interface ProductQuery {
  q?: string;
  category_id?: string;
  min_price?: string;
  max_price?: string;
  in_stock?: boolean;
  sort?: ProductSort;
  page?: number;
  page_size?: number;
}

export interface ProductInput {
  sku: string;
  name: string;
  description: string;
  category_id: string;
  price: Money;
  stock: number;
  weight_kg: string;
}

export type OrderStatus = "PENDING" | "PAID" | "FAILED";

export interface OrderItem {
  product_id: string;
  sku: string;
  name: string;
  unit_price: Money;
  quantity: number;
  line_total: Money;
}

export interface Order {
  id: string;
  user_id: string;
  customer_email: string;
  status: OrderStatus;
  items: OrderItem[];
  total: Money;
  card_last4: string;
  payment_id?: string;
  failure_code?: string;
  paid_at?: string;
  created_at: string;
  updated_at: string;
}

export interface CheckoutRequest {
  items: { product_id: string; quantity: number; unit_price: Money }[];
  payment: { card_number: string; exp_month: number; exp_year: number; cvc: string };
}

export type CheckoutIssueCode =
  | "price_changed"
  | "out_of_stock"
  | "insufficient_stock"
  | "product_unavailable";

export interface CheckoutIssue {
  product_id: string;
  code: CheckoutIssueCode;
  requested_quantity: number;
  available_quantity?: number;
  expected_price?: Money;
  current_price?: Money;
}

export type ImportMode = "upsert" | "skip" | "fail";

export interface ImportOptions {
  mode: ImportMode;
  dry_run: boolean;
  atomic: boolean;
  create_missing_categories: boolean;
}

export interface ImportReport extends ImportOptions {
  committed: boolean;
  total_rows: number;
  blank_rows: number;
  created: number;
  updated: number;
  skipped: number;
  failed: number;
  created_categories: string[];
  duplicates: { sku: string; lines: number[]; applied_line: number }[];
  errors: { line: number; sku?: string; field?: string; code: string; message: string }[];
}

export interface AdminCredentials {
  user: User;
  temporary_password: string;
  temporary_password_expires_at: string;
}

export interface FieldProblem {
  field: string;
  code: string;
  message: string;
}

export interface Problem {
  type?: string;
  title?: string;
  status: number;
  code: string;
  detail?: string;
  instance?: string;
  request_id?: string;
  errors?: FieldProblem[];
  items?: CheckoutIssue[];
  order_id?: string;
}
