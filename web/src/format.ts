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

export function digitsOnly(value: string, max = 8): string {
  return value.replace(/\D/g, "").slice(0, max);
}

export function maskAccountInput(digits: string): string {
  const d = digitsOnly(digits);
  if (d.length <= 4) return d;
  return `${d.slice(0, 4)} · ${d.slice(4)}`;
}

export function activityHint(kind: string, signedCents = 0): string {
  if (kind === "funding") return "Added instantly";
  if (kind === "withdrawal") return "Cashed out";
  if (kind === "transfer") return signedCents >= 0 ? "From another account" : "To another account";
  return "Payment";
}

export function sanitizeAmount(raw: string): string {
  let v = raw.replace(/[^\d.]/g, "");
  const dot = v.indexOf(".");
  if (dot !== -1) {
    v = `${v.slice(0, dot + 1)}${v.slice(dot + 1).replace(/\./g, "").slice(0, 2)}`;
  }
  const [dollarsRaw = "", frac] = v.split(".");
  const dollars = dollarsRaw === "" && frac != null ? "0" : dollarsRaw.replace(/^0+(\d)/, "$1").slice(0, 9);
  if (frac == null && !v.includes(".")) return dollars;
  return `${dollars}.${frac ?? ""}`;
}

export async function copyText(text: string): Promise<boolean> {
  try {
    await navigator.clipboard.writeText(text);
    return true;
  } catch {
    const el = document.createElement("textarea");
    el.value = text;
    el.setAttribute("readonly", "");
    el.style.position = "fixed";
    el.style.top = "0";
    el.style.left = "-9999px";
    document.body.appendChild(el);
    el.select();
    const ok = document.execCommand("copy");
    document.body.removeChild(el);
    return ok;
  }
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
