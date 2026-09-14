import type { ReactNode } from "react";
import type { BankAccount } from "./api";
import { formatAccountNumber, formatUSD, statusLabel } from "./format";

export function IconHome() {
  return (
    <svg viewBox="0 0 24 24" aria-hidden="true">
      <path d="M4.5 10.8 12 4.5l7.5 6.3V20a.9.9 0 0 1-.9.9h-4.7v-5.4H10.1V20.9H5.4a.9.9 0 0 1-.9-.9v-9.2Z" />
    </svg>
  );
}

export function IconWallet() {
  return (
    <svg viewBox="0 0 24 24" aria-hidden="true">
      <path d="M4.6 7.2h14.8A1.6 1.6 0 0 1 21 8.8v9.6a1.6 1.6 0 0 1-1.6 1.6H4.6A1.6 1.6 0 0 1 3 18.4V8.8a1.6 1.6 0 0 1 1.6-1.6Zm0 0V6.4A1.9 1.9 0 0 1 6.5 4.5h9.2" />
      <circle cx="16.6" cy="13.6" r="1" fill="currentColor" stroke="none" />
    </svg>
  );
}

export function IconSend() {
  return (
    <svg viewBox="0 0 24 24" aria-hidden="true">
      <path d="M4.4 12.1 19.2 5.4 13 19.6l-1.8-5.4-5.8-2.1Z" />
    </svg>
  );
}

export function IconList() {
  return (
    <svg viewBox="0 0 24 24" aria-hidden="true">
      <path d="M8 7h11M8 12h11M8 17h11" />
      <circle cx="5" cy="7" r="1" fill="currentColor" stroke="none" />
      <circle cx="5" cy="12" r="1" fill="currentColor" stroke="none" />
      <circle cx="5" cy="17" r="1" fill="currentColor" stroke="none" />
    </svg>
  );
}

export function IconPlus() {
  return (
    <svg viewBox="0 0 24 24" aria-hidden="true">
      <path d="M12 6v12M6 12h12" />
    </svg>
  );
}

export function IconMinus() {
  return (
    <svg viewBox="0 0 24 24" aria-hidden="true">
      <path d="M6 12h12" />
    </svg>
  );
}

export function IconIn() {
  return (
    <svg viewBox="0 0 24 24" aria-hidden="true">
      <path d="M12 4v12M7 11l5 5 5-5" />
    </svg>
  );
}

export function IconOut() {
  return (
    <svg viewBox="0 0 24 24" aria-hidden="true">
      <path d="M12 20V8M7 13l5-5 5 5" />
    </svg>
  );
}

export function IconCheck() {
  return (
    <svg viewBox="0 0 24 24" aria-hidden="true">
      <path d="m5 12.5 4.5 4.5L19 7.5" />
    </svg>
  );
}

export function IconClose() {
  return (
    <svg viewBox="0 0 24 24" aria-hidden="true">
      <path d="M7 7l10 10M17 7 7 17" />
    </svg>
  );
}

export function IconFreeze() {
  return (
    <svg viewBox="0 0 24 24" aria-hidden="true">
      <path d="M12 3v18M5.5 6.5 18.5 17.5M18.5 6.5 5.5 17.5" />
    </svg>
  );
}

export function IconArrow() {
  return (
    <svg viewBox="0 0 24 24" aria-hidden="true">
      <path d="M5 12h14M13 6l6 6-6 6" />
    </svg>
  );
}

export function StatusPill({ status }: { status: string }) {
  return <span className={`pill pill-${status}`}>{statusLabel(status)}</span>;
}

export function MoneyText({ cents, signed = false }: { cents: number; signed?: boolean }) {
  const cls = signed ? (cents > 0 ? "amt-in" : cents < 0 ? "amt-out" : "amt") : "amt";
  const text = signed && cents > 0 ? `+${formatUSD(cents)}` : formatUSD(cents);
  return <span className={`money ${cls}`}>{text}</span>;
}

export function Page({ title, kicker, children }: { title?: string; kicker?: string; children: ReactNode }) {
  return (
    <div className="page">
      {title ? (
        <header className="page-head">
          {kicker ? <p className="kicker">{kicker}</p> : null}
          <h1>{title}</h1>
        </header>
      ) : null}
      {children}
    </div>
  );
}

export function EmptyState({ title, body, action }: { title: string; body: string; action?: ReactNode }) {
  return (
    <div className="empty">
      <h2>{title}</h2>
      <p>{body}</p>
      {action}
    </div>
  );
}

export function Banner({ kind = "error", children }: { kind?: "error" | "ok"; children: ReactNode }) {
  return <p className={kind === "ok" ? "banner banner-ok" : "banner banner-error"}>{children}</p>;
}

export function AccountHero({ account }: { account: BankAccount }) {
  return (
    <article className="hero-card">
      <div className="hero-top">
        <span>USD · {formatAccountNumber(account.account_number)}</span>
        <StatusPill status={account.status} />
      </div>
      <p className="hero-label">Balance</p>
      <p className="hero-balance">{formatUSD(account.balance_cents)}</p>
    </article>
  );
}

export function Sheet({
  title,
  onClose,
  children,
}: {
  title: string;
  onClose: () => void;
  children: ReactNode;
}) {
  return (
    <div className="sheet-back" onClick={onClose} role="presentation">
      <div
        className="sheet"
        role="dialog"
        aria-modal="true"
        aria-labelledby="sheet-title"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="sheet-head">
          <h2 id="sheet-title">{title}</h2>
          <button type="button" className="icon-x" onClick={onClose} aria-label="Close">
            <IconClose />
          </button>
        </div>
        {children}
      </div>
    </div>
  );
}

export function AmountField({
  value,
  onChange,
  disabled,
}: {
  value: string;
  onChange: (v: string) => void;
  disabled?: boolean;
}) {
  return (
    <label className="amount-field">
      <span>Amount</span>
      <div className="amount-row">
        <span className="amount-ccy">$</span>
        <input
          value={value}
          onChange={(e) => onChange(e.target.value)}
          inputMode="decimal"
          placeholder="0.00"
          disabled={disabled}
          required
        />
      </div>
    </label>
  );
}
