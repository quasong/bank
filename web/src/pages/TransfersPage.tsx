import { FormEvent, useState } from "react";
import { Link } from "react-router-dom";
import { ApiError, createTransfer } from "../api";
import { formatAccountNumber, formatUSD } from "../format";
import { dollarsToCents } from "../money";
import { useAccounts } from "../hooks";
import { AmountField, Banner, EmptyState, IconCheck, Page } from "../ui";

export function TransfersPage() {
  const { accounts, error, setError, reload } = useAccounts();
  const [toNumber, setToNumber] = useState("");
  const [amount, setAmount] = useState("10.00");
  const [pending, setPending] = useState(false);
  const [sent, setSent] = useState<{ amount: string; to: string } | null>(null);

  const selected = accounts?.[0];
  const blocked = selected != null && selected.status !== "active";

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    if (!selected) return;
    const cents = dollarsToCents(amount);
    if (cents == null || cents <= 0) {
      setError("Enter an amount with at most two decimals");
      return;
    }
    if (toNumber.length !== 8) {
      setError("Destination is an 8-digit account number");
      return;
    }
    if (blocked) {
      setError("This account cannot send money right now.");
      return;
    }
    setError("");
    setPending(true);
    try {
      const res = await createTransfer(selected.id, toNumber, cents, crypto.randomUUID());
      await reload();
      setSent({ amount: formatUSD(cents), to: formatAccountNumber(res.transfer.to_account_number) });
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Transfer failed");
    } finally {
      setPending(false);
    }
  }

  if (!accounts) {
    return <p className="kicker">Loading…</p>;
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
          <p>To {sent.to}</p>
          <button
            className="btn btn-primary btn-block"
            type="button"
            onClick={() => {
              setSent(null);
              setToNumber("");
            }}
          >
            Send again
          </button>
        </div>
      </Page>
    );
  }

  return (
    <Page title="Send" kicker="USD">
      {error ? <Banner>{error}</Banner> : null}
      {blocked ? (
        <Banner>
          This account is {selected?.status}. Unfreeze it in Account to send.
        </Banner>
      ) : null}
      <form className="send-card" onSubmit={onSubmit}>
        <AmountField value={amount} onChange={setAmount} disabled={blocked} />
        <p className="avail">Available {formatUSD(selected?.balance_cents ?? 0)}</p>
        <label>
          To
          <input
            className="digits"
            value={toNumber}
            onChange={(e) => setToNumber(e.target.value.replace(/\D/g, "").slice(0, 8))}
            required
            disabled={blocked}
            inputMode="numeric"
            pattern="[0-9]{8}"
            maxLength={8}
            placeholder="00000000"
            autoComplete="off"
          />
        </label>
        <button className="btn btn-primary btn-block" type="submit" disabled={pending || blocked}>
          {pending ? "Sending…" : "Send"}
        </button>
      </form>
    </Page>
  );
}
