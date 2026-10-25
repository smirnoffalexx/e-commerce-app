const dateFormat = new Intl.DateTimeFormat("en-US", { dateStyle: "medium" });
const dateTimeFormat = new Intl.DateTimeFormat("en-US", { dateStyle: "medium", timeStyle: "short" });
const moneyFormat = new Intl.NumberFormat("en-US", { style: "currency", currency: "USD" });

export function formatDate(value: string): string {
  return dateFormat.format(new Date(value));
}

export function formatDateTime(value: string): string {
  return dateTimeFormat.format(new Date(value));
}

export function formatMoney(value: string): string {
  return moneyFormat.format(Number(value));
}
