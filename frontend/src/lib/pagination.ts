export function pageWindow(current: number, total: number, size = 5): (number | "gap")[] {
  if (total <= size + 2) {
    return Array.from({ length: total }, (_, i) => i + 1);
  }
  const half = Math.floor(size / 2);
  const start = Math.max(2, Math.min(current - half, total - size));
  const end = Math.min(total - 1, start + size - 1);
  const pages: (number | "gap")[] = [1];
  if (start > 2) pages.push("gap");
  for (let page = start; page <= end; page++) pages.push(page);
  if (end < total - 1) pages.push("gap");
  pages.push(total);
  return pages;
}
