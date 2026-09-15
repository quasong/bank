export const CURRENCIES = ["USD", "EUR", "GBP"] as const;
export const USD_ROUTING = "121000248";

export function usdParts(cents: number): { sign: string; dollars: string; frac: string } {
  const neg = cents < 0;
  const abs = Math.abs(cents);
  return {
    sign: neg ? "-" : "",
    dollars: Math.floor(abs / 100).toLocaleString("en-US"),
    frac: String(abs % 100).padStart(2, "0"),
  };
}

export function currencySymbol(ccy = "USD"): string {
  if (ccy === "EUR") return "€";
  if (ccy === "GBP") return "£";
  return "$";
}

export function formatMoney(cents: number, ccy = "USD"): string {
  const { sign, dollars, frac } = usdParts(cents);
  return `${sign}${currencySymbol(ccy)}${dollars}.${frac}`;
}

export function formatUSD(cents: number): string {
  return formatMoney(cents, "USD");
}

export function currencyName(ccy = "USD"): string {
  if (ccy === "EUR") return "Euro";
  if (ccy === "GBP") return "British pound";
  return "US dollar";
}

export function currencyShortName(ccy = "USD"): string {
  if (ccy === "EUR") return "Euro";
  if (ccy === "GBP") return "Pound";
  return "Dollar";
}

export function formatLocalAccount(n: string): string {
  const d = compactAccountInput(n).replace(/\D/g, "").slice(-8);
  if (d.length !== 8) return n;
  return `${d.slice(0, 4)} ${d.slice(4)}`;
}

export function shortAccountLabel(n: string, _ccy?: string): string {
  const compact = compactAccountInput(n);
  return compact.slice(-4) || compact;
}

export function payeeAccountHint(n: string): string {
  const compact = compactAccountInput(n);
  if (compact.startsWith("GB") || compact.length === 22) return `IBAN · ${compact.slice(-4)}`;
  if (compact.startsWith("040004")) return `04-00-04 · ${compact.slice(-4)}`;
  if (compact.startsWith(USD_ROUTING) || compact.length === 17 || /^\d{8}$/.test(compact)) {
    return formatLocalAccount(compact);
  }
  return formatAccountNumber(n);
}

export function fxPair(description?: string): { from: string; to: string } | null {
  const m = (description ?? "").match(/FX \d+ ([A-Z]{3}) to \d+ ([A-Z]{3})/i);
  if (!m) return null;
  return { from: m[1], to: m[2] };
}

export function compactAccountInput(raw: string, max = 22): string {
  return raw.replace(/[\s·-]/g, "").toUpperCase().slice(0, max);
}

export function accountLooksReady(n: string): boolean {
  const c = compactAccountInput(n);
  const dda = /^\d{8}$/.test(c) && !USD_ROUTING.startsWith(c);
  return dda || new RegExp(`^${USD_ROUTING}\\d{8}$`).test(c) || /^040004\d{8}$/.test(c) || /^GB\d{2}THEB040004\d{8}$/.test(c);
}

export function accountCurrencyOf(n: string): string {
  const c = compactAccountInput(n);
  if (/^GB/i.test(c) || c.length === 22) return "EUR";
  if (c.startsWith("040004") && c.length === 14) return "GBP";
  return "USD";
}

export function formatAccountNumber(n: string): string {
  const compact = compactAccountInput(n);
  if (new RegExp(`^${USD_ROUTING}\\d{8}$`).test(compact)) {
    return `${compact.slice(0, 9)} · ${compact.slice(9, 13)} ${compact.slice(13)}`;
  }
  if (/^040004\d{8}$/.test(compact)) {
    return `${compact.slice(0, 2)}-${compact.slice(2, 4)}-${compact.slice(4, 6)} · ${compact.slice(6, 10)} ${compact.slice(6 + 4)}`;
  }
  if (compact.length >= 4 && compact.startsWith("GB")) {
    return compact.replace(/(.{4})/g, "$1 ").trim();
  }
  return n;
}

export function maskAccountInput(raw: string): string {
  const compact = compactAccountInput(raw);
  if (!compact) return "";
  if (/^GB/i.test(compact) || /[A-Z]/.test(compact)) {
    return compact.replace(/(.{4})/g, "$1 ").trim();
  }
  if (compact.startsWith("040004")) {
    const d = compact.replace(/\D/g, "").slice(0, 14);
    if (d.length <= 6) {
      if (d.length <= 2) return d;
      if (d.length <= 4) return `${d.slice(0, 2)}-${d.slice(2)}`;
      return `${d.slice(0, 2)}-${d.slice(2, 4)}-${d.slice(4)}`;
    }
    return `${d.slice(0, 2)}-${d.slice(2, 4)}-${d.slice(4, 6)} · ${d.slice(6, 10)}${d.length > 10 ? ` ${d.slice(10)}` : ""}`;
  }
  const d = compact.replace(/\D/g, "").slice(0, 17);
  if (d.startsWith(USD_ROUTING) || d.length > 8) {
    if (d.length <= 9) return d;
    const acct = d.slice(9);
    return `${d.slice(0, 9)} · ${acct.slice(0, 4)}${acct.length > 4 ? ` ${acct.slice(4)}` : ""}`;
  }
  if (d.length <= 4) return d;
  return `${d.slice(0, 4)} ${d.slice(4)}`;
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
  if (kind === "fx") return "Converted";
  return kind;
}

export function activityHint(
  kind: string,
  signedCents = 0,
  counterpartyNumber?: string,
  counterpartyName?: string,
  note?: string,
  description?: string,
): string {
  const memo = (note ?? "").trim();
  if (memo) return memo;
  if (kind === "funding") return "Added instantly";
  if (kind === "withdrawal") return "Cashed out";
  if (kind === "fx") {
    const pair = fxPair(description);
    if (pair) return signedCents >= 0 ? `From ${pair.from}` : `To ${pair.to}`;
    return "Currency conversion";
  }
  if (kind === "transfer") {
    const who = payeeLabel(counterpartyNumber, counterpartyName);
    if (signedCents >= 0) return who ? `From ${who}` : "From another account";
    return who ? `To ${who}` : "To another account";
  }
  return "Payment";
}

export function activityKindLabel(kind: string, signedCents = 0): string {
  if (kind === "funding") return "Added instantly";
  if (kind === "withdrawal") return "Cashed out";
  if (kind === "fx") return "Converted";
  if (kind === "transfer") return signedCents >= 0 ? "Incoming" : "Outgoing";
  return "Payment";
}

export function payeeLabel(number?: string, name?: string): string {
  if (!number) return "";
  const n = (name ?? "").trim();
  if (n && n !== number) return n;
  return formatAccountNumber(number);
}

export function payeeDetail(number?: string, name?: string): string {
  if (!number) return "";
  const formatted = formatAccountNumber(number);
  const n = (name ?? "").trim();
  if (n && n !== number) return `${n} · ${formatted}`;
  return formatted;
}

export function receiptCode(journalId: string, receipt?: string): string {
  if (receipt) return receipt;
  return journalId.replace(/-/g, "").toUpperCase().slice(-8);
}

export function sanitizeAmount(raw: string): string {
  let v = raw.includes(",") && !raw.includes(".") ? raw.replace(",", ".") : raw;
  v = v.replace(/[^\d.]/g, "");
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

export function dateTimeLabel(iso: string): string {
  const d = new Date(iso);
  return `${d.toLocaleDateString("en-GB", { weekday: "short", day: "numeric", month: "short" })} · ${timeLabel(iso)}`;
}

export function recentWhen(iso: string, now = new Date()): string {
  const sec = Math.max(0, Math.round((now.getTime() - new Date(iso).getTime()) / 1000));
  if (sec < 45) return "Just now";
  if (sec < 3600) return `${Math.max(1, Math.round(sec / 60))} min ago`;
  return timeLabel(iso);
}

export function todayKicker(now = new Date()): string {
  return now.toLocaleDateString("en-GB", { weekday: "long", day: "numeric", month: "short" });
}

export function openedLabel(iso: string): string {
  return new Date(iso).toLocaleDateString("en-GB", { day: "numeric", month: "short", year: "numeric" });
}

export function statusLabel(status: string): string {
  if (status === "active") return "Active";
  if (status === "frozen") return "Frozen";
  if (status === "closed") return "Closed";
  return status;
}

export function auditLabel(action: string): string {
  switch (action) {
    case "register":
      return "Created profile";
    case "login_success":
      return "Signed in";
    case "login_failed":
      return "Failed sign-in";
    case "logout":
      return "Signed out";
    case "funding":
      return "Added money";
    case "withdrawal":
      return "Withdrew money";
    case "transfer":
      return "Sent money";
    case "account_freeze":
      return "Froze account";
    case "account_unfreeze":
      return "Unfroze account";
    case "account_close":
      return "Closed account";
    case "fx":
      return "Converted money";
    default:
      return action;
  }
}

export function auditAmount(meta?: Record<string, string>): string {
  const raw = meta?.amount_cents ?? "";
  if (!/^-?\d+$/.test(raw)) return "";
  return formatMoney(Number(raw), meta?.currency || meta?.from_currency || "USD");
}
