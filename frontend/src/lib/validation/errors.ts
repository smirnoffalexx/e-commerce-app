import type { z } from "zod";

export function zodFieldErrors(error: z.ZodError): Record<string, string> {
  const out: Record<string, string> = {};
  for (const issue of error.issues) {
    const field = issue.path.map(String).join(".");
    if (!(field in out)) {
      out[field] = issue.message;
    }
  }
  return out;
}
