const MONEY_PATTERN = /^(\d+)(?:\.(\d{1,2}))?$/;

export function isMoney(value: string): boolean {
  return MONEY_PATTERN.test(value.trim());
}

export function toCents(value: string): number {
  const match = MONEY_PATTERN.exec(value.trim());
  if (!match) {
    throw new Error(`invalid money amount: ${value}`);
  }
  return Number(match[1]) * 100 + Number((match[2] ?? "").padEnd(2, "0"));
}

export function fromCents(cents: number): string {
  const whole = Math.floor(cents / 100);
  const fraction = String(cents % 100).padStart(2, "0");
  return `${whole}.${fraction}`;
}
