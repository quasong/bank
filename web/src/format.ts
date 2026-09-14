export function formatUSD(cents: number): string {
  const neg = cents < 0;
  const abs = Math.abs(cents);
  const dollars = Math.floor(abs / 100);
  const frac = String(abs % 100).padStart(2, "0");
  return `${neg ? "-" : ""}$${dollars.toLocaleString("en-US")}.${frac}`;
}

export function formatAccountNumber(n: string): string {
  if (n.length === 8) return `${n.slice(0, 4)} · ${n.slice(4)}`;
  return n;
}

export function greeting(now = new Date()): string {
  const h = now.getHours();
  if (h < 12) return "Good morning";
  if (h < 18) return "Good afternoon";
  return "Good evening";
}

export function displayName(email: string): string {
  const local = email.split("@")[0] ?? email;
  return local.charAt(0).toUpperCase() + local.slice(1);
}

export function activityTitle(kind: string, signedCents: number): string {
  if (kind === "funding") return "Added money";
  if (kind === "withdrawal") return "Withdrew money";
  if (kind === "transfer") return signedCents >= 0 ? "Received" : "Sent";
  return kind;
}

export function activityHint(kind: string): string {
  if (kind === "funding") return "Added from vault";
  if (kind === "withdrawal") return "Sent to vault";
  if (kind === "transfer") return "Between accounts";
  return "Ledger entry";
}

export function dayLabel(iso: string, now = new Date()): string {
  const d = new Date(iso);
  const today = new Date(now.getFullYear(), now.getMonth(), now.getDate());
  const that = new Date(d.getFullYear(), d.getMonth(), d.getDate());
  const diff = (today.getTime() - that.getTime()) / 86400000;
  if (diff === 0) return "Today";
  if (diff === 1) return "Yesterday";
  return d.toLocaleDateString("en-GB", { day: "numeric", month: "short", year: "numeric" });
}

export function timeLabel(iso: string): string {
  return new Date(iso).toLocaleTimeString("en-GB", { hour: "2-digit", minute: "2-digit" });
}

export function statusLabel(status: string): string {
  if (status === "active") return "Active";
  if (status === "frozen") return "Frozen";
  if (status === "closed") return "Closed";
  return status;
}
