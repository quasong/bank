export function centsToDollars(cents: number): string {
  const sign = cents < 0 ? "-" : "";
  const abs = Math.abs(cents);
  const d = Math.floor(abs / 100);
  const c = String(abs % 100).padStart(2, "0");
  return `${sign}${d}.${c}`;
}

export function dollarsToCents(input: string): number | null {
  const m = /^(\d+)(?:\.(\d{1,2}))?$/.exec(input.trim());
  if (!m) return null;
  const dollars = Number(m[1]);
  const centsPart = (m[2] ?? "00").padEnd(2, "0");
  const cents = Number(centsPart);
  if (!Number.isInteger(dollars) || !Number.isInteger(cents)) return null;
  return dollars * 100 + cents;
}
