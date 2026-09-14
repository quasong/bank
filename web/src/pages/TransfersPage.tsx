import { FormEvent, useState } from "react";
import { Link } from "react-router-dom";
import { createTransfer, errorMessage } from "../api";
import { digitsOnly, formatAccountNumber, formatUSD, maskAccountInput, statusLabel } from "../format";
import { centsToDollars, dollarsToCents } from "../money";
import { useAccounts } from "../hooks";
import { AmountField, Banner, EmptyState, IconCheck, Page, PageSkeleton } from "../ui";

export function TransfersPage() {
  const { accounts, error, setError, reload } = useAccounts();
  const [toNumber, setToNumber] = useState("");
  const [amount, setAmount] = useState("");
  const [step, setStep] = useState<"edit" | "confirm">("edit");
  const [pending, setPending] = useState(false);
  const [sent, setSent] = useState<{ amount: string; to: string; left: string } | null>(null);

  const selected = accounts?.[0];
  const blocked = selected != null && selected.status !== "active";
  const cents = dollarsToCents(amount);
  const ownAccount = selected != null && toNumber === selected.account_number;
  const leftover =
    selected && cents != null && cents > 0 && cents <= selected.balance_cents ? selected.balance_cents - cents : null;
  const ready =
    !blocked &&
    cents != null &&
    cents > 0 &&
    toNumber.length === 8 &&
    selected != null &&
    !ownAccount &&
    cents <= selected.balance_cents;

  function validate(): string | null {
    if (cents == null || cents <= 0) return "Enter an amount with at most two decimals";
    if (toNumber.length !== 8) return "Destination is an 8-digit account number";
    if (ownAccount) return "That's your own account";
    if (selected && cents > selected.balance_cents) return "You don't have that much available";
    if (blocked) return "This account cannot send money right now.";
    return null;
  }

  function onContinue(e: FormEvent) {
    e.preventDefault();
    const issue = validate();
    if (issue) {
      setError(issue);
      return;
    }
    setError("");
    setStep("confirm");
  }

  async function onConfirm(e?: FormEvent) {
    e?.preventDefault();
    if (!selected || !cents) return;
    const issue = validate();
    if (issue) {
      setError(issue);
      setStep("edit");
      return;
    }
    setError("");
    setPending(true);
    try {
      const res = await createTransfer(selected.id, toNumber, cents, crypto.randomUUID());
      const next = await reload();
      const left = next[0]?.balance_cents ?? selected.balance_cents - cents;
      setSent({
        amount: formatUSD(cents),
        to: formatAccountNumber(res.transfer.to_account_number),
        left: formatUSD(left),
      });
    } catch (err) {
      setError(errorMessage(err, "Transfer failed"));
      setStep("edit");
    } finally {
      setPending(false);
    }
  }

  if (!accounts) {
    return <PageSkeleton />;
  }

  if (accounts.length === 0) {
    return (
      <Page title="Send">
        <EmptyState
          title="Open an account first"
          body="You need a USD account before you can send money."
          action={
            <Link className="btn btn-primary" to="/accounts">
              Go to account
            </Link>
          }
        />
      </Page>
    );
  }

  if (sent) {
    return (
      <Page title="Send">
        <div className="send-card success">
          <div className="success-mark">
            <IconCheck />
          </div>
          <h2>You sent {sent.amount}</h2>
          <p>
            To {sent.to}
            <br />
            {sent.left} left in your account
          </p>
          <button
            className="btn btn-primary btn-block"
            type="button"
            onClick={() => {
              setSent(null);
              setToNumber("");
              setAmount("");
              setStep("edit");
            }}
          >
            Send again
          </button>
          <Link className="btn btn-secondary btn-block" to="/">
            Back home
          </Link>
        </div>
      </Page>
    );
  }

  if (step === "confirm" && selected && cents) {
    return (
      <Page title="Send" kicker="Check it">
        {error ? <Banner>{error}</Banner> : null}
        <form className="send-card" onSubmit={onConfirm}>
          <div className="review">
            <div>
              <span>You send</span>
              <strong>{formatUSD(cents)}</strong>
            </div>
            <div>
              <span>To</span>
              <strong>{formatAccountNumber(toNumber)}</strong>
            </div>
            <div>
              <span>Remaining</span>
              <strong>{formatUSD(selected.balance_cents - cents)}</strong>
            </div>
          </div>
          <button className="btn btn-primary btn-block" type="submit" disabled={pending}>
            {pending ? "Sending…" : `Send ${formatUSD(cents)}`}
          </button>
          <button className="btn btn-secondary btn-block" type="button" disabled={pending} onClick={() => setStep("edit")}>
            Back
          </button>
        </form>
      </Page>
    );
  }

  return (
    <Page title="Send" kicker="USD">
      {error ? <Banner>{error}</Banner> : null}
      {blocked ? (
        <Banner>
          This account is {statusLabel(selected?.status ?? "")}. <Link to="/accounts">Unfreeze it</Link> to send.
        </Banner>
      ) : null}
      <form className="send-card" onSubmit={onContinue}>
        <AmountField value={amount} onChange={setAmount} disabled={blocked} autoFocus />
        <p className="avail">
          Available {formatUSD(selected?.balance_cents ?? 0)}
          {selected && selected.balance_cents > 0 && !blocked ? (
            <>
              {" · "}
              <button type="button" className="text-link" onClick={() => setAmount(centsToDollars(selected.balance_cents))}>
                Max
              </button>
            </>
          ) : null}
          {leftover != null ? ` · ${formatUSD(leftover)} after this send` : null}
        </p>
        <label>
          To
          <input
            className={`digits${ownAccount ? " input-warn" : toNumber.length === 8 ? " input-ok" : ""}`}
            value={maskAccountInput(toNumber)}
            onChange={(e) => setToNumber(digitsOnly(e.target.value))}
            disabled={blocked}
            inputMode="numeric"
            maxLength={11}
            placeholder="0000 · 0000"
            autoComplete="off"
            aria-label="Destination account number"
          />
        </label>
        <p className={`avail${ownAccount ? " avail-warn" : ""}`}>
          {ownAccount ? "That's your own account" : `${toNumber.length}/8 digits`}
        </p>
        <button className="btn btn-primary btn-block" type="submit" disabled={!ready}>
          {cents != null && cents > 0 ? `Continue · ${formatUSD(cents)}` : "Continue"}
        </button>
      </form>
    </Page>
  );
}
