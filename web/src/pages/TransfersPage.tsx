import { FormEvent, useEffect, useMemo, useState } from "react";
import { Link, useSearchParams } from "react-router-dom";
import { createTransfer, errorMessage } from "../api";
import {
  accountCurrencyOf,
  accountLooksReady,
  compactAccountInput,
  copyText,
  formatAccountNumber,
  formatMoney,
  maskAccountInput,
  payeeAccountHint,
  payeeLabel,
  receiptCode,
  statusLabel,
  USD_ROUTING,
} from "../format";
import { centsToDollars, dollarsToCents } from "../money";
import { useAccounts, usePayees, useSelectedAccount, useToast } from "../hooks";
import { SavePersonSheet } from "../people";
import { AmountField, Banner, CurrencyFlag, EmptyState, IconCheck, Page, PageSkeleton, Toast } from "../ui";

export function TransfersPage() {
  const { accounts, error, setError, reload } = useAccounts();
  const { selected, select } = useSelectedAccount(accounts);
  const { payees, reload: reloadPayees } = usePayees();
  const [searchParams, setSearchParams] = useSearchParams();
  const [toNumber, setToNumber] = useState("");
  const [payeeName, setPayeeName] = useState("");
  const [note, setNote] = useState("");
  const [adding, setAdding] = useState(false);
  const [amount, setAmount] = useState("");
  const [step, setStep] = useState<"edit" | "confirm">("edit");
  const [pending, setPending] = useState(false);
  const [copied, setCopied] = useState(false);
  const { text: toast, show } = useToast();
  const [sent, setSent] = useState<{
    amount: string;
    to: string;
    left: string;
    journalId: string;
    receipt: string;
    note: string;
  } | null>(null);

  const blocked = selected != null && selected.status !== "active";
  const ownNumbers = useMemo(() => (accounts ?? []).map((a) => a.account_number), [accounts]);
  const saved = useMemo(
    () =>
      (payees ?? []).filter(
        (p) => !ownNumbers.includes(p.account_number) && accountCurrencyOf(p.account_number) === (selected?.currency ?? "USD"),
      ),
    [payees, ownNumbers, selected?.currency],
  );
  const cents = dollarsToCents(amount);
  const ownAccount = ownNumbers.includes(toNumber);
  const leftover =
    selected && cents != null && cents > 0 && cents <= selected.balance_cents ? selected.balance_cents - cents : null;
  const dest = payeeLabel(toNumber, payeeName) || formatAccountNumber(toNumber);
  const alreadySaved = saved.some((p) => p.account_number === toNumber);
  const destCcy = toNumber ? accountCurrencyOf(toNumber) : "";
  const destReady = accountLooksReady(toNumber) && destCcy === (selected?.currency ?? "");
  const destMismatch = accountLooksReady(toNumber) && !ownAccount && destCcy !== (selected?.currency ?? "");
  const ready =
    !blocked &&
    cents != null &&
    cents > 0 &&
    destReady &&
    selected != null &&
    !ownAccount &&
    cents <= selected.balance_cents;

  useEffect(() => {
    const from = searchParams.get("from");
    if (from && accounts?.some((a) => a.id === from)) select(from);
    const to = compactAccountInput(searchParams.get("to") ?? "");
    if (!accountLooksReady(to)) return;
    setToNumber(to);
    if (payees == null) return;
    const match = payees.find((p) => p.account_number === to);
    setPayeeName(match && match.display_name !== to ? match.display_name : "");
    setSearchParams({}, { replace: true });
  }, [searchParams, payees, accounts, select, setSearchParams]);

  function pickPayee(accountNumber: string, displayName: string) {
    setToNumber(accountNumber);
    setPayeeName(displayName === accountNumber ? "" : displayName);
  }

  function validate(): string | null {
    if (cents == null || cents <= 0) return "Enter an amount with at most two decimals";
    if (!accountLooksReady(toNumber)) return "Enter a valid account number or IBAN";
    if (selected && accountCurrencyOf(toNumber) !== selected.currency) {
      return "Destination must be the same currency. Convert first, then send.";
    }
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
      const res = await createTransfer(selected.id, toNumber, cents, crypto.randomUUID(), payeeName, note);
      const next = await reload();
      await reloadPayees().catch(() => undefined);
      const left = next.find((a) => a.id === selected.id)?.balance_cents ?? selected.balance_cents - cents;
      setSent({
        amount: formatMoney(cents, selected.currency),
        to: payeeLabel(res.transfer.to_account_number, payeeName) || formatAccountNumber(res.transfer.to_account_number),
        left: formatMoney(left, selected.currency),
        journalId: res.transfer.journal_id,
        receipt: receiptCode(res.transfer.journal_id, res.transfer.receipt),
        note: res.transfer.note || note.trim(),
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
          body="You need a balance before you can send money."
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
            {sent.note ? (
              <>
                <br />
                {sent.note}
              </>
            ) : null}
            <br />
            {sent.left} left in this balance
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
              setNote("");
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
              <strong>{formatMoney(cents, selected.currency)}</strong>
            </div>
            <div>
              <span>To</span>
              <strong>{dest}</strong>
            </div>
            {accountLooksReady(toNumber) && payeeName.trim() && payeeName.trim() !== toNumber ? (
              <div>
                <span>Account</span>
                <strong>{formatAccountNumber(toNumber)}</strong>
              </div>
            ) : null}
            {note.trim() ? (
              <div>
                <span>Note</span>
                <strong>{note.trim()}</strong>
              </div>
            ) : null}
            <div>
              <span>Remaining</span>
              <strong>{formatMoney(selected.balance_cents - cents, selected.currency)}</strong>
            </div>
          </div>
          <button className="btn btn-primary btn-block" type="submit" disabled={pending}>
            {pending ? "Sending…" : `Send ${formatMoney(cents, selected.currency)}`}
          </button>
          <button className="btn btn-secondary btn-block" type="button" disabled={pending} onClick={() => setStep("edit")}>
            Back
          </button>
        </form>
      </Page>
    );
  }

  return (
    <Page title="Send" kicker={selected ? `From ${selected.currency}` : ""}>
      {error ? <Banner>{error}</Banner> : null}
      {blocked ? (
        <Banner>
          This account is {statusLabel(selected?.status ?? "")}. <Link to="/accounts">Unfreeze it</Link> to send.
        </Banner>
      ) : null}
      <form className="send-card" onSubmit={onContinue}>
        {accounts.length > 1 ? (
          <div className="chips" role="group" aria-label="Send from">
            {accounts.map((a) => (
              <button
                key={a.id}
                type="button"
                className={`chip chip-payee ccy-${a.currency}${selected?.id === a.id ? " chip-on" : ""}`}
                onClick={() => {
                  select(a.id);
                  if (toNumber && accountLooksReady(toNumber) && accountCurrencyOf(toNumber) !== a.currency) {
                    setToNumber("");
                    setPayeeName("");
                  }
                }}
              >
                <strong>
                  <CurrencyFlag code={a.currency} />
                  {a.currency}
                </strong>
                <span>{formatMoney(a.balance_cents, a.currency)}</span>
              </button>
            ))}
          </div>
        ) : null}
        <AmountField value={amount} onChange={setAmount} disabled={blocked} autoFocus currency={selected?.currency} />
        <p className="avail">
          Available {formatMoney(selected?.balance_cents ?? 0, selected?.currency)}
          {selected && selected.balance_cents > 0 && !blocked ? (
            <>
              {" · "}
              <button type="button" className="text-link" onClick={() => setAmount(centsToDollars(selected.balance_cents))}>
                Max
              </button>
            </>
          ) : null}
          {leftover != null ? ` · ${formatMoney(leftover, selected?.currency)} after this send` : null}
        </p>
        <label>
          To
          <input
            className={`digits${ownAccount ? " input-warn" : destReady ? " input-ok" : ""}`}
            value={maskAccountInput(toNumber)}
            onChange={(e) => setToNumber(compactAccountInput(e.target.value))}
            disabled={blocked}
            maxLength={29}
            placeholder={selected?.currency === "EUR" ? "GB IBAN" : selected?.currency === "GBP" ? "04-00-04 · account" : `${USD_ROUTING} · account`}
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
        <label>
          Note
          <input
            value={note}
            onChange={(e) => setNote(e.target.value.slice(0, 40))}
            disabled={blocked}
            maxLength={40}
            placeholder="Optional"
            autoComplete="off"
            aria-label="Transfer note"
          />
        </label>
        {saved.length > 0 || !blocked ? (
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
                <span>{payeeAccountHint(p.account_number)}</span>
              </button>
            ))}
            <button type="button" className="chip" onClick={() => setAdding(true)}>
              Add
            </button>
          </div>
        ) : null}
        {destReady && !ownAccount && !alreadySaved ? (
          <p className="avail">
            <button type="button" className="text-link" onClick={() => setAdding(true)}>
              Save this person
            </button>
          </p>
        ) : null}
        <p className={`avail${ownAccount || destMismatch ? " avail-warn" : ""}`}>
          {ownAccount
            ? "That's your own account"
            : destMismatch
              ? `That's ${destCcy}. Convert first, then send ${destCcy}.`
              : toNumber && !destReady
                ? `Enter complete ${selected?.currency ?? ""} details`
                : "Same currency as the balance you send from"}
        </p>
        {destMismatch ? (
          <Link className="text-link" to="/convert">
            Convert to {destCcy}
          </Link>
        ) : null}
        <button className="btn btn-primary btn-block" type="submit" disabled={!ready}>
          {cents != null && cents > 0 ? `Continue · ${formatMoney(cents, selected?.currency)}` : "Continue"}
        </button>
      </form>
      {adding ? (
        <SavePersonSheet
          ownAccountNumbers={ownNumbers}
          initialNumber={toNumber}
          initialName={payeeName}
          onClose={() => setAdding(false)}
          onSaved={async (payee) => {
            await reloadPayees();
            pickPayee(payee.account_number, payee.display_name);
            setAdding(false);
            show("Saved");
          }}
        />
      ) : null}
      <Toast text={toast} />
    </Page>
  );
}
