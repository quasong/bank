import { FormEvent, useEffect, useMemo, useState } from "react";
import { Link } from "react-router-dom";
import { convertFX, errorMessage, quoteFX, type FXQuote } from "../api";
import { CURRENCIES, currencyName, currencySymbol, formatMoney } from "../format";
import { centsToDollars, dollarsToCents } from "../money";
import { useAccounts, useSelectedAccount, useToast } from "../hooks";
import { AmountField, Banner, EmptyState, IconCheck, IconSwap, Page, PageSkeleton, Toast } from "../ui";

function prettyRate(rate: string) {
  const n = Number(rate);
  if (!Number.isFinite(n)) return rate;
  return n.toFixed(5);
}

export function ConvertPage() {
  const { accounts, error, setError, reload } = useAccounts();
  const { selected, select } = useSelectedAccount(accounts);
  const { text: toast, show } = useToast();
  const [toCcy, setToCcy] = useState("EUR");
  const [amount, setAmount] = useState("");
  const [quote, setQuote] = useState<FXQuote | null>(null);
  const [quoting, setQuoting] = useState(false);
  const [pending, setPending] = useState(false);
  const [done, setDone] = useState<{ from: string; to: string; rate: string } | null>(null);

  const cents = dollarsToCents(amount);
  const fromCcy = selected?.currency || "USD";
  const targets = useMemo(() => CURRENCIES.filter((c) => c !== fromCcy), [fromCcy]);
  const blocked = selected != null && selected.status !== "active";
  const destAcct = (accounts ?? []).find((a) => a.currency === toCcy);
  const tooMuch = selected != null && cents != null && cents > selected.balance_cents;

  useEffect(() => {
    if (!targets.some((c) => c === toCcy) && targets[0]) setToCcy(targets[0]);
  }, [fromCcy, targets, toCcy]);

  useEffect(() => {
    if (!selected || !cents || cents <= 0 || fromCcy === toCcy || blocked) {
      setQuote(null);
      setQuoting(false);
      return;
    }
    let cancelled = false;
    setQuoting(true);
    const timer = window.setTimeout(() => {
      quoteFX(fromCcy, toCcy, cents)
        .then((res) => {
          if (cancelled) return;
          setError("");
          setQuote(res.quote);
        })
        .catch((err) => {
          if (cancelled) return;
          setQuote(null);
          setError(errorMessage(err, "Could not fetch a live rate"));
        })
        .finally(() => {
          if (!cancelled) setQuoting(false);
        });
    }, 220);
    return () => {
      cancelled = true;
      window.clearTimeout(timer);
    };
  }, [selected, cents, fromCcy, toCcy, blocked, setError]);

  function swap() {
    if (!destAcct || destAcct.status !== "active") return;
    let nextAmount = quote?.quote_cents ? centsToDollars(quote.quote_cents) : "";
    if (quote?.quote_cents && quote.quote_cents > destAcct.balance_cents) {
      nextAmount = destAcct.balance_cents > 0 ? centsToDollars(destAcct.balance_cents) : "";
    }
    select(destAcct.id);
    setToCcy(fromCcy);
    setAmount(nextAmount);
    setQuote(null);
  }

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    if (!selected || !cents || !quote?.quote_cents) return;
    setError("");
    setPending(true);
    try {
      const res = await convertFX(selected.id, toCcy, cents, crypto.randomUUID());
      await reload();
      select(res.conversion.to_account_id);
      setDone({
        from: formatMoney(res.conversion.amount_cents, res.conversion.from_currency),
        to: formatMoney(res.conversion.quote_cents, res.conversion.to_currency),
        rate: res.conversion.rate,
      });
      show("Converted");
    } catch (err) {
      setError(errorMessage(err, "Conversion failed"));
    } finally {
      setPending(false);
    }
  }

  if (!accounts) return <PageSkeleton />;

  if (accounts.length === 0) {
    return (
      <Page title="Convert">
        <EmptyState
          title="Open an account first"
          body="You need a balance before you can convert currencies."
          action={
            <Link className="btn btn-primary" to="/accounts">
              Go to account
            </Link>
          }
        />
      </Page>
    );
  }

  if (done) {
    return (
      <Page title="Convert">
        <div className="send-card success">
          <div className="success-mark">
            <IconCheck />
          </div>
          <h2>You converted {done.from}</h2>
          <p>
            Into {done.to}
            <br />
            Live rate {done.rate}
          </p>
          <button
            className="btn btn-primary btn-block"
            type="button"
            onClick={() => {
              setDone(null);
              setAmount("");
              setQuote(null);
            }}
          >
            Convert again
          </button>
          <Link className="btn btn-secondary btn-block" to="/">
            Back home
          </Link>
        </div>
        <Toast text={toast} />
      </Page>
    );
  }

  const canConvert = !blocked && !pending && !!quote?.quote_cents && !!cents && !tooMuch;
  const cta = quote?.quote_cents && cents ? `Convert ${formatMoney(cents, fromCcy)} to ${formatMoney(quote.quote_cents, toCcy)}` : "Convert";

  return (
    <Page title="Convert" kicker="Live ECB rate">
      {error ? <Banner>{error}</Banner> : null}
      {blocked ? (
        <Banner>
          This account cannot convert right now. <Link to="/accounts">See details</Link>
        </Banner>
      ) : null}
      {tooMuch ? <Banner>You don't have that much {fromCcy} available.</Banner> : null}
      <form className="send-card fx-card" onSubmit={onSubmit}>
        <div className="fx-leg">
          <div className="fx-leg-h">
            <span>You send</span>
            <label className="fx-ccy">
              <span className="visually-hidden">From currency</span>
              <select
                value={selected?.id ?? ""}
                onChange={(e) => select(e.target.value)}
                disabled={blocked}
                aria-label="From currency"
              >
                {(accounts ?? []).map((a) => (
                  <option key={a.id} value={a.id}>
                    {a.currency}
                  </option>
                ))}
              </select>
            </label>
          </div>
          <AmountField value={amount} onChange={setAmount} disabled={blocked} autoFocus currency={fromCcy} label="" />
          <p className="avail">
            {formatMoney(selected?.balance_cents ?? 0, fromCcy)} available
            {selected && selected.balance_cents > 0 && !blocked ? (
              <>
                {" · "}
                <button type="button" className="text-link" onClick={() => setAmount(centsToDollars(selected.balance_cents))}>
                  Max
                </button>
              </>
            ) : null}
          </p>
        </div>
        <button
          type="button"
          className="fx-swap"
          onClick={swap}
          disabled={!destAcct || destAcct.status !== "active" || blocked}
          aria-label="Swap currencies"
        >
          <IconSwap />
        </button>
        <div className="fx-leg">
          <div className="fx-leg-h">
            <span>You get</span>
            <label className="fx-ccy">
              <span className="visually-hidden">To currency</span>
              <select value={toCcy} onChange={(e) => setToCcy(e.target.value)} disabled={blocked} aria-label="To currency">
                {targets.map((c) => (
                  <option key={c} value={c}>
                    {c}
                    {!accounts.some((a) => a.currency === c) ? " · new" : ""}
                  </option>
                ))}
              </select>
            </label>
          </div>
          <p className="fx-out" aria-live="polite">
            {quote?.quote_cents ? (
              formatMoney(quote.quote_cents, toCcy)
            ) : quoting ? (
              <span className="faint">…</span>
            ) : (
              <span className="faint">{currencySymbol(toCcy)}0.00</span>
            )}
          </p>
          <p className="avail">{destAcct ? `${currencyName(toCcy)} balance` : `We'll open a ${toCcy} balance`}</p>
        </div>
        <p className="fx-rate">
          {quote ? `1 ${fromCcy} = ${prettyRate(quote.rate)} ${toCcy}` : quoting ? "Fetching live rate…" : "Enter an amount for a live rate"}
          {quote?.as_of ? ` · as of ${quote.as_of.slice(0, 10)}` : ""}
        </p>
        <button className="btn btn-primary btn-block" type="submit" disabled={!canConvert}>
          {pending ? "Converting…" : cta}
        </button>
      </form>
      <Toast text={toast} />
    </Page>
  );
}
