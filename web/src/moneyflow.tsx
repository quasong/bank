import { FormEvent, useState } from "react";
import { errorMessage, fundAccount, withdrawAccount, type BankAccount } from "./api";
import { formatMoney } from "./format";
import { centsToDollars, dollarsToCents } from "./money";
import { AmountChips, AmountField, Banner, Sheet } from "./ui";

const ADD_PRESETS = [1000, 2000, 5000, 10000];

export function MoneySheet({
  kind,
  account,
  onClose,
  onSuccess,
}: {
  kind: "fund" | "withdraw";
  account: BankAccount;
  onClose: () => void;
  onSuccess: (message: string) => Promise<void> | void;
}) {
  const [amount, setAmount] = useState("");
  const [pending, setPending] = useState(false);
  const [error, setError] = useState("");
  const cents = dollarsToCents(amount);
  const withdraw = kind === "withdraw";
  const tooMuch = withdraw && cents != null && cents > account.balance_cents;
  const ready = cents != null && cents > 0 && !tooMuch;
  const label = withdraw ? "Withdraw" : "Add";
  const chips = withdraw ? ADD_PRESETS.filter((v) => v <= account.balance_cents) : ADD_PRESETS;

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    if (cents == null || cents <= 0) {
      setError("Enter an amount with at most two decimals");
      return;
    }
    if (tooMuch) {
      setError("You don't have that much available");
      return;
    }
    setError("");
    setPending(true);
    try {
      if (withdraw) {
        await withdrawAccount(account.id, cents, crypto.randomUUID());
        await onSuccess(`Withdrew ${formatMoney(cents, account.currency)}`);
      } else {
        await fundAccount(account.id, cents, crypto.randomUUID());
        await onSuccess(`Added ${formatMoney(cents, account.currency)}`);
      }
    } catch (err) {
      setError(errorMessage(err, "Could not complete that"));
    } finally {
      setPending(false);
    }
  }

  return (
    <Sheet title={withdraw ? "Withdraw" : "Add money"} onClose={onClose}>
      <form className="stack" onSubmit={onSubmit}>
        {error ? <Banner>{error}</Banner> : null}
        <AmountField value={amount} onChange={setAmount} autoFocus currency={account.currency} />
        <AmountChips values={chips} onPick={(v) => setAmount(centsToDollars(v))} disabled={pending} currency={account.currency} />
        <p className="avail">
          Available {formatMoney(account.balance_cents, account.currency)}
          {withdraw && account.balance_cents > 0 ? (
            <>
              {" · "}
              <button type="button" className="text-link" onClick={() => setAmount(centsToDollars(account.balance_cents))}>
                Max
              </button>
            </>
          ) : null}
        </p>
        <button className="btn btn-primary btn-block" type="submit" disabled={pending || !ready}>
          {pending ? (withdraw ? "Withdrawing…" : "Adding…") : cents != null && ready ? `${label} ${formatMoney(cents, account.currency)}` : label}
        </button>
      </form>
    </Sheet>
  );
}
