import { FormEvent, useMemo, useState } from "react";
import { Link } from "react-router-dom";
import { createTransfer, errorMessage } from "../api";
import { digitsOnly, copyText, formatAccountNumber, formatUSD, maskAccountInput, payeeLabel, receiptCode, statusLabel } from "../format";
import { centsToDollars, dollarsToCents } from "../money";
import { useAccounts, usePayees, useToast } from "../hooks";
import { AmountField, Banner, EmptyState, IconCheck, Page, PageSkeleton, Toast } from "../ui";

export function TransfersPage() {
  const { accounts, error, setError, reload } = useAccounts();
  const { payees, reload: reloadPayees } = usePayees();
  const [toNumber, setToNumber] = useState("");
  const [payeeName, setPayeeName] = useState("");
  const [amount, setAmount] = useState("");
  const [step, setStep] = useState<"edit" | "confirm">("edit");
  const [pending, setPending] = useState(false);
  const [copied, setCopied] = useState(false);
  const { text: toast, show } = useToast();
  const [sent, setSent] = useState<{ amount: string; to: string; left: string; journalId: string; receipt: string } | null>(
    null,
  );

  const selected = accounts?.[0];
  const blocked = selected != null && selected.status !== "active";
  const saved = useMemo(
    () => (payees ?? []).filter((p) => p.account_number !== selected?.account_number),
    [payees, selected?.account_number],
  );
  const cents = dollarsToCents(amount);
  const ownAccount = selected != null && toNumber === selected.account_number;
  const leftover =
    selected && cents != null && cents > 0 && cents <= selected.balance_cents ? selected.balance_cents - cents : null;
  const dest = payeeLabel(toNumber, payeeName) || formatAccountNumber(toNumber);
  const ready =
    !blocked &&
    cents != null &&
    cents > 0 &&
    toNumber.length === 8 &&
    selected != null &&
    !ownAccount &&
    cents <= selected.balance_cents;

  function pickPayee(accountNumber: string, displayName: string) {
    setToNumber(accountNumber);
    setPayeeName(displayName === accountNumber ? "" : displayName);
  }

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
      const res = await createTransfer(selected.id, toNumber, cents, crypto.randomUUID(), payeeName);
      const next = await reload();
      await reloadPayees().catch(() => undefined);
      const left = next[0]?.balance_cents ?? selected.balance_cents - cents;
      setSent({
        amount: formatUSD(cents),
        to: payeeLabel(res.transfer.to_account_number, payeeName) || formatAccountNumber(res.transfer.to_account_number),
        left: formatUSD(left),
        journalId: res.transfer.journal_id,
        receipt: receiptCode(res.transfer.journal_id, res.transfer.receipt),
      });
      setCopied(false);
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
          <div className="review success-receipt">
            <button
              type="button"
              className="review-copy"
              onClick={() => {
                void copyText(sent.journalId).then((ok) => {
                  if (ok) {
                    setCopied(true);
                    show("Copied receipt");
                  }
                });
              }}
            >
              <span>Receipt</span>
              <strong>{copied ? "Copied" : sent.receipt}</strong>
            </button>
          </div>
          <button
            className="btn btn-primary btn-block"
            type="button"
            onClick={() => {
              setSent(null);
              setToNumber("");
              setPayeeName("");
              setAmount("");
              setStep("edit");
              setCopied(false);
            }}
          >
            Send again
          </button>
          <Link className="btn btn-secondary btn-block" to="/">
            Back home
          </Link>
        </div>
        <Toast text={toast} />
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
              <strong>{dest}</strong>
            </div>
            {toNumber.length === 8 && payeeName.trim() && payeeName.trim() !== toNumber ? (
              <div>
                <span>Account</span>
                <strong>{formatAccountNumber(toNumber)}</strong>
              </div>
            ) : null}
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
        <label>
          Name
          <input
            value={payeeName}
            onChange={(e) => setPayeeName(e.target.value.slice(0, 40))}
            disabled={blocked}
            maxLength={40}
            placeholder="Optional"
            autoComplete="off"
            aria-label="Payee name"
          />
        </label>
        {saved.length > 0 ? (
          <div className="chips">
            {saved.map((p) => (
              <button
                key={p.id}
                type="button"
                className={`chip chip-payee${toNumber === p.account_number ? " chip-on" : ""}`}
                disabled={blocked}
                onClick={() => pickPayee(p.account_number, p.display_name)}
              >
                {p.display_name !== p.account_number ? <strong>{p.display_name}</strong> : null}
                <span>{formatAccountNumber(p.account_number)}</span>
              </button>
            ))}
          </div>
        ) : null}
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
