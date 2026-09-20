import { useEffect, useRef, useState, type ReactNode } from "react";
import type { ActivityItem, BankAccount } from "./api";
import {
  activityHint,
  activityKindLabel,
  activityTitle,
  copyText,
  currencyDetailsHint,
  currencyName,
  currencyShortName,
  dateTimeLabel,
  formatAccountNumber,
  formatMoney,
  currencySymbol,
  fxPair,
  payeeDetail,
  receiptCode,
  recentWhen,
  sanitizeAmount,
  statusLabel,
  usdParts,
} from "./format";
import { useHideBalance } from "./prefs";

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

export function IconSwap() {
  return (
    <svg viewBox="0 0 24 24" aria-hidden="true">
      <path d="M7 7h11M15 4l3 3-3 3M17 17H6M9 14l-3 3 3 3" />
    </svg>
  );
}

export function IconChevron() {
  return (
    <svg viewBox="0 0 24 24" aria-hidden="true">
      <path d="M7 10l5 5 5-5" />
    </svg>
  );
}

const FLAG_EMOJI: Record<string, string> = {
  USD: "🇺🇸",
  EUR: "🇪🇺",
  GBP: "🇬🇧",
  AUD: "🇦🇺",
  BGN: "🇧🇬",
  BRL: "🇧🇷",
  CAD: "🇨🇦",
  CHF: "🇨🇭",
  CNY: "🇨🇳",
  CZK: "🇨🇿",
  DKK: "🇩🇰",
  HKD: "🇭🇰",
  ILS: "🇮🇱",
  INR: "🇮🇳",
  MXN: "🇲🇽",
  MYR: "🇲🇾",
  NOK: "🇳🇴",
  NZD: "🇳🇿",
  PHP: "🇵🇭",
  PLN: "🇵🇱",
  RON: "🇷🇴",
  SEK: "🇸🇪",
  SGD: "🇸🇬",
  THB: "🇹🇭",
  TRY: "🇹🇷",
  ZAR: "🇿🇦",
};

export function CurrencyFlag({ code }: { code: string }) {
  return (
    <span className={`ccy-flag ccy-flag-${code}`} aria-hidden="true">
      {FLAG_EMOJI[code] ?? "🏳️"}
    </span>
  );
}

export function IconArrow() {
  return (
    <svg viewBox="0 0 24 24" aria-hidden="true">
      <path d="M5 12h14M13 6l6 6-6 6" />
    </svg>
  );
}

export function IconEye() {
  return (
    <svg viewBox="0 0 24 24" aria-hidden="true">
      <path d="M2.8 12S6.2 6.5 12 6.5 21.2 12 21.2 12 17.8 17.5 12 17.5 2.8 12 2.8 12Z" />
      <circle cx="12" cy="12" r="2.4" />
    </svg>
  );
}

export function IconEyeOff() {
  return (
    <svg viewBox="0 0 24 24" aria-hidden="true">
      <path d="M4 5.2 19.2 20.4M9.4 9.7A3 3 0 0 0 12 15.2M14.7 14.4A3 3 0 0 0 9.8 9.6" />
      <path d="M6.4 8.2C4.4 9.6 2.8 12 2.8 12S6.2 17.5 12 17.5c1.4 0 2.7-.3 3.8-.8M17.7 15.4c1.8-1.3 3.5-3.4 3.5-3.4S17.8 6.5 12 6.5c-.6 0-1.2.1-1.7.2" />
    </svg>
  );
}

export function StatusPill({ status }: { status: string }) {
  return <span className={`pill pill-${status}`}>{statusLabel(status)}</span>;
}

export function MoneyText({
  cents,
  currency = "USD",
  signed = false,
  reveal = false,
}: {
  cents: number;
  currency?: string;
  signed?: boolean;
  reveal?: boolean;
}) {
  const { hidden } = useHideBalance();
  if (hidden && !reveal) return <span className="money amt-mask">••••</span>;
  const cls = signed ? (cents > 0 ? "amt-in" : cents < 0 ? "amt-out" : "amt") : "amt";
  const text = signed && cents > 0 ? `+${formatMoney(cents, currency)}` : formatMoney(cents, currency);
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
      <span className="empty-mark" aria-hidden="true">
        <IconWallet />
      </span>
      <h2>{title}</h2>
      <p>{body}</p>
      {action}
    </div>
  );
}

export function Banner({ kind = "error", children }: { kind?: "error" | "ok"; children: ReactNode }) {
  return <p className={kind === "ok" ? "banner banner-ok" : "banner banner-error"}>{children}</p>;
}

export function AccountHero({
  account,
  parkedCents = 0,
  onCopied,
}: {
  account: BankAccount;
  parkedCents?: number;
  onCopied?: () => void;
}) {
  const [copied, setCopied] = useState(false);
  const { hidden, toggle } = useHideBalance();
  const parts = usdParts(account.balance_cents);
  const ccy = account.currency || "USD";
  const formatted = account.account_number_formatted || formatAccountNumber(account.account_number);

  async function copy() {
    void copyText(account.account_number);
    setCopied(true);
    onCopied?.();
    window.setTimeout(() => setCopied(false), 1600);
  }

  return (
    <article className={`hero-card${account.status !== "active" ? ` is-${account.status}` : ""}`}>
      <div className="hero-top">
        <button
          type="button"
          className={`hero-copy${copied ? " copied" : ""}`}
          onClick={copy}
          aria-label={`Copy account number ${formatted}`}
        >
          <span className="hero-id">{formatted}</span>
          <em>{copied ? "Copied" : "Copy"}</em>
        </button>
        <StatusPill status={account.status} />
      </div>
      <div className="hero-label-row">
        <p className="hero-label">{currencyName(ccy)}</p>
        <button
          type="button"
          className="hero-eye"
          onClick={toggle}
          aria-label={hidden ? "Show balance" : "Hide balance"}
        >
          {hidden ? <IconEyeOff /> : <IconEye />}
        </button>
      </div>
      <p className="hero-balance" aria-hidden={hidden}>
        {hidden ? (
          <span className="hero-dots">••••</span>
        ) : (
          <>
            {parts.sign}
            {currencySymbol(ccy)}
            {parts.dollars}
            <span className="hero-cents">.{parts.frac}</span>
          </>
        )}
      </p>
      {!hidden && parkedCents > 0 ? <p className="hero-aside">{formatMoney(parkedCents, ccy)} in jars</p> : null}
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
  const panelRef = useRef<HTMLDivElement>(null);
  const onCloseRef = useRef(onClose);
  onCloseRef.current = onClose;

  useEffect(() => {
    const root = panelRef.current;
    const prev = document.activeElement as HTMLElement | null;
    const prevOverflow = document.body.style.overflow;
    document.body.style.overflow = "hidden";

    const focusables = () =>
      root
        ? [...root.querySelectorAll<HTMLElement>('button:not([disabled]), input:not([disabled]), [href], select, textarea, [tabindex]:not([tabindex="-1"])')]
        : [];
    const items = focusables();
    const autofocus = root?.querySelector<HTMLElement>("input[autofocus], input:not([disabled])");
    (autofocus ?? items[0])?.focus();

    function onKey(e: KeyboardEvent) {
      if (e.key === "Escape") {
        onCloseRef.current();
        return;
      }
      if (e.key !== "Tab") return;
      const list = focusables();
      if (list.length === 0) return;
      const first = list[0];
      const last = list[list.length - 1];
      if (e.shiftKey && document.activeElement === first) {
        e.preventDefault();
        last.focus();
      } else if (!e.shiftKey && document.activeElement === last) {
        e.preventDefault();
        first.focus();
      }
    }
    document.addEventListener("keydown", onKey);
    return () => {
      document.removeEventListener("keydown", onKey);
      document.body.style.overflow = prevOverflow;
      prev?.focus();
    };
  }, []);

  return (
    <div className="sheet-back" onClick={onClose} role="presentation">
      <div
        ref={panelRef}
        className="sheet"
        role="dialog"
        aria-modal="true"
        aria-labelledby="sheet-title"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="sheet-grab" aria-hidden="true" />
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

export function Wallets({
  accounts,
  selectedId,
  onSelect,
  onAdd,
}: {
  accounts: BankAccount[];
  selectedId?: string;
  onSelect: (id: string) => void;
  onAdd?: () => void;
}) {
  const scroller = useRef<HTMLElement>(null);
  useEffect(() => {
    const root = scroller.current;
    const on = root?.querySelector<HTMLElement>(".wallet.on");
    if (!root || !on) return;
    const onRect = on.getBoundingClientRect();
    const rootRect = root.getBoundingClientRect();
    if (onRect.left < rootRect.left + 8) {
      root.scrollBy({ left: onRect.left - rootRect.left - 8, behavior: "smooth" });
    } else if (onRect.right > rootRect.right - 8) {
      root.scrollBy({ left: onRect.right - rootRect.right + 8, behavior: "smooth" });
    }
  }, [selectedId]);

  if (accounts.length < 2 && !onAdd) return null;
  return (
    <section className="wallets" aria-label="Balances" ref={scroller}>
      {accounts.map((a) => (
        <button
          key={a.id}
          type="button"
          className={`wallet ccy-${a.currency}${a.id === selectedId ? " on" : ""}`}
          onClick={() => onSelect(a.id)}
        >
          <span className="wallet-id">
            <CurrencyFlag code={a.currency} />
            <span>
              <strong>{currencyShortName(a.currency)}</strong>
              <em>{a.currency}</em>
            </span>
          </span>
          <MoneyText cents={a.balance_cents} currency={a.currency} />
        </button>
      ))}
      {onAdd ? (
        <button type="button" className="wallet add" onClick={onAdd}>
          <span className="wallet-id">
            <span>
              <strong>Add</strong>
              <em>New balance</em>
            </span>
          </span>
          <strong>+</strong>
        </button>
      ) : null}
    </section>
  );
}

export function CurrencyChoices({
  currencies,
  pending,
  onPick,
}: {
  currencies: readonly string[];
  pending?: boolean;
  onPick: (ccy: string) => void;
}) {
  const [query, setQuery] = useState("");
  const needle = query.trim().toLowerCase();
  const shown = currencies.filter((ccy) => {
    if (!needle) return true;
    return (
      ccy.toLowerCase().includes(needle) ||
      currencyName(ccy).toLowerCase().includes(needle) ||
      currencyShortName(ccy).toLowerCase().includes(needle) ||
      currencySymbol(ccy).toLowerCase().includes(needle)
    );
  });
  return (
    <div className="sheet-actions col">
      {currencies.length > 6 ? (
        <label className="finder">
          <span className="sr-only">Search currencies</span>
          <input
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="Search currencies"
            autoComplete="off"
            autoFocus
          />
        </label>
      ) : null}
      {shown.map((ccy) => (
        <button
          key={ccy}
          className={`currency-pick ccy-${ccy}`}
          type="button"
          disabled={pending}
          onClick={() => onPick(ccy)}
        >
          <CurrencyFlag code={ccy} />
          <span>
            <strong>
              {currencySymbol(ccy)} {currencyName(ccy)}
            </strong>
            <em>{currencyDetailsHint(ccy)}</em>
          </span>
        </button>
      ))}
      {shown.length === 0 ? <p className="panel-empty">No currency matches that search.</p> : null}
    </div>
  );
}

export function ChoiceMenu({
  label,
  value,
  options,
  onChange,
  disabled,
}: {
  label: string;
  value: string;
  options: { value: string; code: string; name: string; amount: string }[];
  onChange: (value: string) => void;
  disabled?: boolean;
}) {
  const [open, setOpen] = useState(false);
  const [query, setQuery] = useState("");
  const rootRef = useRef<HTMLDivElement>(null);
  const current = options.find((o) => o.value === value) ?? options[0];
  const searchable = options.length > 8;
  const needle = query.trim().toLowerCase();
  const shown = options.filter((o) => {
    if (!needle) return true;
    return `${o.code} ${o.name} ${o.amount} ${currencyName(o.code)}`.toLowerCase().includes(needle);
  });

  useEffect(() => {
    if (!open) return;
    function onDoc(e: MouseEvent) {
      if (!rootRef.current?.contains(e.target as Node)) setOpen(false);
    }
    function onKey(e: KeyboardEvent) {
      if (e.key === "Escape") setOpen(false);
    }
    document.addEventListener("mousedown", onDoc);
    document.addEventListener("keydown", onKey);
    return () => {
      document.removeEventListener("mousedown", onDoc);
      document.removeEventListener("keydown", onKey);
    };
  }, [open]);

  useEffect(() => {
    if (disabled) setOpen(false);
  }, [disabled]);

  useEffect(() => {
    if (open) setQuery("");
  }, [open]);

  return (
    <div
      className={`choice${open ? " is-open" : ""}`}
      ref={rootRef}
      onBlur={(e) => {
        if (!rootRef.current?.contains(e.relatedTarget as Node)) setOpen(false);
      }}
    >
      <button
        type="button"
        className={`choice-btn ccy-${current?.code ?? ""}`}
        aria-label={label}
        aria-haspopup="listbox"
        aria-expanded={open}
        disabled={disabled}
        onClick={() => setOpen((v) => !v)}
      >
        <CurrencyFlag code={current?.code ?? "USD"} />
        {current?.code ?? value}
        <span className="choice-caret" aria-hidden="true">
          <IconChevron />
        </span>
      </button>
      {open ? (
        <ul className="choice-menu" role="listbox" aria-label={label}>
          {searchable ? (
            <li className="choice-find" onMouseDown={(e) => e.preventDefault()}>
              <input
                value={query}
                onChange={(e) => setQuery(e.target.value)}
                placeholder="Search"
                aria-label="Search currencies"
                autoComplete="off"
                autoFocus
              />
            </li>
          ) : null}
          {shown.map((o) => {
            const on = o.value === value;
            return (
              <li key={o.value}>
                <button
                  type="button"
                  role="option"
                  aria-selected={on}
                  className={`choice-opt ccy-${o.code}${on ? " on" : ""}`}
                  onClick={() => {
                    onChange(o.value);
                    setOpen(false);
                  }}
                >
                  <CurrencyFlag code={o.code} />
                  <span className="choice-copy">
                    <strong>{o.name}</strong>
                    <em>{o.code}</em>
                  </span>
                  <b>{o.amount}</b>
                  <span className={`choice-check${on ? " is-on" : ""}`}>{on ? <IconCheck /> : null}</span>
                </button>
              </li>
            );
          })}
          {shown.length === 0 ? (
            <li className="choice-empty">No match</li>
          ) : null}
        </ul>
      ) : null}
    </div>
  );
}

export function AmountField({
  value,
  onChange,
  disabled,
  autoFocus,
  currency = "USD",
  label = "Amount",
}: {
  value: string;
  onChange: (v: string) => void;
  disabled?: boolean;
  autoFocus?: boolean;
  currency?: string;
  label?: string;
}) {
  return (
    <label className="amount-field">
      {label ? <span>{label}</span> : null}
      <div className="amount-row">
        <span className="amount-ccy">{currencySymbol(currency)}</span>
        <input
          value={value}
          onChange={(e) => onChange(sanitizeAmount(e.target.value))}
          onFocus={(e) => e.currentTarget.select()}
          inputMode="decimal"
          placeholder="0.00"
          disabled={disabled}
          autoFocus={autoFocus}
          aria-label={label || "Amount"}
        />
      </div>
    </label>
  );
}

export function Toast({ text }: { text: string }) {
  if (!text) return null;
  return (
    <div className="toast" role="status" aria-live="polite">
      {text}
    </div>
  );
}

export function PageSkeleton() {
  return (
    <div className="page">
      <div className="skel skel-title" />
      <div className="skel skel-hero" />
      <div className="skel skel-row" />
    </div>
  );
}

export function TxnSkeleton({ rows = 2 }: { rows?: number }) {
  return (
    <>
      {Array.from({ length: rows }, (_, i) => (
        <div className="txn skel-txn" key={i}>
          <div className="skel skel-icon" />
          <div className="skel skel-line" />
        </div>
      ))}
    </>
  );
}

export function TxnRow({
  item,
  onOpen,
  showCurrency,
}: {
  item: ActivityItem;
  onOpen?: (item: ActivityItem) => void;
  showCurrency?: boolean;
}) {
  const hint = activityHint(
    item.kind,
    item.signed_cents,
    item.counterparty_account_number,
    item.counterparty_name,
    item.note,
    item.description,
  );
  const body = (
    <>
      <span className={`txn-icon ${item.signed_cents >= 0 ? "in" : "out"}`}>
        {item.signed_cents >= 0 ? <IconIn /> : <IconOut />}
      </span>
      <div className="txn-copy">
        <strong>{activityTitle(item.kind, item.signed_cents)}</strong>
        <span>
          {hint}
          {showCurrency && item.currency ? ` · ${item.currency}` : ""} · {recentWhen(item.created_at)}
        </span>
      </div>
      <MoneyText cents={item.signed_cents} currency={item.currency} signed />
    </>
  );
  if (onOpen) {
    return (
      <button type="button" className="txn txn-btn" onClick={() => onOpen(item)}>
        {body}
      </button>
    );
  }
  return <div className="txn">{body}</div>;
}

export function TxnDetail({ item, onClose }: { item: ActivityItem; onClose: () => void }) {
  const [copied, setCopied] = useState(false);
  const receipt = receiptCode(item.journal_id, item.receipt);
  const pair = item.kind === "fx" ? fxPair(item.description) : null;
  const counterparty = pair ? `${pair.from} → ${pair.to}` : payeeDetail(item.counterparty_account_number, item.counterparty_name);

  async function copyReceipt() {
    const ok = await copyText(item.journal_id);
    if (!ok) return;
    setCopied(true);
    window.setTimeout(() => setCopied(false), 1600);
  }

  return (
    <Sheet title={activityTitle(item.kind, item.signed_cents)} onClose={onClose}>
      <p className="detail-amt">
        <MoneyText cents={item.signed_cents} currency={item.currency} signed reveal />
      </p>
      <div className="review">
        <div>
          <span>When</span>
          <strong>{dateTimeLabel(item.created_at)}</strong>
        </div>
        <div>
          <span>Type</span>
          <strong>{activityKindLabel(item.kind, item.signed_cents)}</strong>
        </div>
        {item.note ? (
          <div>
            <span>Note</span>
            <strong>{item.note}</strong>
          </div>
        ) : null}
        {counterparty ? (
          <div>
            <span>{item.kind === "fx" ? "Pair" : item.signed_cents >= 0 ? "From" : "To"}</span>
            <strong>{counterparty}</strong>
          </div>
        ) : null}
        <button type="button" className="review-copy" onClick={() => void copyReceipt()}>
          <span>Receipt</span>
          <strong>{copied ? "Copied" : receipt}</strong>
        </button>
      </div>
    </Sheet>
  );
}

export function AmountChips({
  values,
  onPick,
  disabled,
  currency = "USD",
}: {
  values: number[];
  onPick: (cents: number) => void;
  disabled?: boolean;
  currency?: string;
}) {
  if (values.length === 0) return null;
  return (
    <div className="chips">
      {values.map((cents) => (
        <button key={cents} type="button" className="chip" disabled={disabled} onClick={() => onPick(cents)}>
          {cents % 100 === 0 ? `${currencySymbol(currency)}${usdParts(cents).dollars}` : formatMoney(cents, currency)}
        </button>
      ))}
    </div>
  );
}

export function BootScreen() {
  return (
    <div className="boot">
      <span className="logo-mark">B</span>
      <p>Just a moment…</p>
    </div>
  );
}
