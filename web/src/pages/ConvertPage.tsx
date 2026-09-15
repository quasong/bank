import { FormEvent, useEffect, useMemo, useState } from "react";
import { Link } from "react-router-dom";
import { convertFX, errorMessage, quoteFX, type FXQuote } from "../api";
import { CURRENCIES, formatMoney } from "../format";
import { centsToDollars, dollarsToCents } from "../money";
import { useAccounts, useSelectedAccount, useToast } from "../hooks";
import { AmountField, Banner, EmptyState, IconCheck, Page, PageSkeleton, Toast } from "../ui";

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

  useEffect(() => {
    if (!targets.some((c) => c === toCcy) && targets[0]) setToCcy(targets[0]);
  }, [fromCcy, targets, toCcy]);

  useEffect(() => {
    if (!selected || !cents || cents <= 0 || fromCcy === toCcy || blocked) {
      setQuote(null);
      return;
    }
    let cancelled = false;
    setQuoting(true);
    const timer = window.setTimeout(() => {
      quoteFX(fromCcy, toCcy, cents)
        .then((res) => {
          if (!cancelled) setQuote(res.quote);
        })
        .catch((err) => {
          if (!cancelled) {
            setQuote(null);
            setError(errorMessage(err, "Could not fetch a live rate"));
          }
        })
        .finally(() => {
          if (!cancelled) setQuoting(false);
        });
    }, 280);
    return () => {
      cancelled = true;
      window.clearTimeout(timer);
    };
  }, [selected, cents, fromCcy, toCcy, blocked, setError]);

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
          <h2>
            {done.from} → {done.to}
          </h2>
          <p>Live rate {done.rate}</p>
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

  return (
    <Page title="Convert" kicker="Live ECB rate">
      {error ? <Banner>{error}</Banner> : null}
      {blocked ? (
        <Banner>
          This account cannot convert right now. <Link to="/accounts">See details</Link>
        </Banner>
      ) : null}
      <form className="send-card" onSubmit={onSubmit}>
        <div className="chips">
          {(accounts ?? []).map((a) => (
            <button
              key={a.id}
              type="button"
              className={`chip${selected?.id === a.id ? " chip-on" : ""}`}
              onClick={() => select(a.id)}
            >
              {a.currency}
            </button>
          ))}
        </div>
        <AmountField value={amount} onChange={setAmount} disabled={blocked} autoFocus currency={fromCcy} />
        <p className="avail">
          Available {formatMoney(selected?.balance_cents ?? 0, fromCcy)}
          {selected && selected.balance_cents > 0 && !blocked ? (
            <>
              {" · "}
              <button type="button" className="text-link" onClick={() => setAmount(centsToDollars(selected.balance_cents))}>
                Max
              </button>
            </>
          ) : null}
        </p>
        <label>
          To
          <select value={toCcy} onChange={(e) => setToCcy(e.target.value)} disabled={blocked}>
            {targets.map((c) => (
              <option key={c} value={c}>
                {c}
              </option>
            ))}
          </select>
        </label>
        <div className="review">
          <div>
            <span>You convert</span>
            <strong>{cents ? formatMoney(cents, fromCcy) : "—"}</strong>
          </div>
          <div>
            <span>They receive</span>
            <strong>{quote?.quote_cents ? formatMoney(quote.quote_cents, toCcy) : quoting ? "…" : "—"}</strong>
          </div>
          <div>
            <span>Rate</span>
            <strong>{quote ? `1 ${fromCcy} = ${quote.rate} ${toCcy}` : "—"}</strong>
          </div>
        </div>
        <button
          className="btn btn-primary btn-block"
          type="submit"
          disabled={blocked || pending || !quote?.quote_cents || !cents || (selected != null && cents > selected.balance_cents)}
        >
          {pending ? "Converting…" : "Convert"}
        </button>
      </form>
      <Toast text={toast} />
    </Page>
  );
}
