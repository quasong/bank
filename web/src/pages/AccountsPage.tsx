import { FormEvent, useEffect, useState } from "react";
import {
  ApiError,
  closeAccount,
  freezeAccount,
  fundAccount,
  listAccounts,
  openAccount,
  unfreezeAccount,
  withdrawAccount,
  type BankAccount,
} from "../api";
import { centsToDollars, dollarsToCents } from "../money";

type MoneyForm = { id: string; kind: "fund" | "withdraw" };

export function AccountsPage() {
  const [accounts, setAccounts] = useState<BankAccount[] | null>(null);
  const [error, setError] = useState("");
  const [pending, setPending] = useState(false);
  const [form, setForm] = useState<MoneyForm | null>(null);
  const [amount, setAmount] = useState("100.00");

  async function reload() {
    const data = await listAccounts();
    setAccounts(data.accounts);
  }

  useEffect(() => {
    reload().catch((err) => setError(err instanceof Error ? err.message : "Failed to load accounts"));
  }, []);

  async function onOpen() {
    setError("");
    setPending(true);
    try {
      await openAccount();
      await reload();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Could not open account");
    } finally {
      setPending(false);
    }
  }

  async function onMoney(e: FormEvent) {
    e.preventDefault();
    if (!form) return;
    const cents = dollarsToCents(amount);
    if (cents == null || cents <= 0) {
      setError("Enter a dollar amount with at most two decimals");
      return;
    }
    setError("");
    setPending(true);
    try {
      if (form.kind === "fund") {
        await fundAccount(form.id, cents, crypto.randomUUID());
      } else {
        await withdrawAccount(form.id, cents, crypto.randomUUID());
      }
      setForm(null);
      await reload();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Posting failed");
    } finally {
      setPending(false);
    }
  }

  async function runStatus(action: () => Promise<unknown>) {
    setError("");
    setPending(true);
    try {
      await action();
      setForm(null);
      await reload();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Could not update account");
    } finally {
      setPending(false);
    }
  }

  if (!accounts) {
    return <p className="lede">Loading accounts…</p>;
  }

  return (
    <div>
      <h1>Accounts</h1>
      <p className="lede">
        Demand-deposit balances come from the ledger. Add funds credits vault cash; withdraw is the reverse.
        Freeze stops money movement. Close requires a zero balance.
      </p>
      {error ? <p className="error">{error}</p> : null}
      {accounts.length === 0 ? (
        <div className="tile">
          <p>No deposit account yet. Opening one creates a liability ledger account with a zero balance.</p>
          <button type="button" disabled={pending} onClick={onOpen}>
            {pending ? "Opening…" : "Open account"}
          </button>
        </div>
      ) : (
        <>
          <table className="table">
            <thead>
              <tr>
                <th>Number</th>
                <th>Status</th>
                <th>Balance</th>
              </tr>
            </thead>
            <tbody>
              {accounts.map((a) => (
                <tr key={a.id}>
                  <td className="money">{a.account_number}</td>
                  <td>{a.status}</td>
                  <td className="money">${centsToDollars(a.balance_cents)}</td>
                </tr>
              ))}
            </tbody>
          </table>
          {accounts.map((a) =>
            a.status === "closed" ? null : (
              <div key={a.id + "-actions"} className="row-actions" style={{ marginTop: 16 }}>
                {a.status === "active" ? (
                  <>
                    <button type="button" className="ghost" onClick={() => setForm({ id: a.id, kind: "fund" })}>
                      Add funds
                    </button>
                    <button type="button" className="ghost" onClick={() => setForm({ id: a.id, kind: "withdraw" })}>
                      Withdraw
                    </button>
                    <button
                      type="button"
                      className="ghost"
                      disabled={pending}
                      onClick={() => runStatus(() => freezeAccount(a.id))}
                    >
                      Freeze
                    </button>
                  </>
                ) : null}
                {a.status === "frozen" ? (
                  <button
                    type="button"
                    className="ghost"
                    disabled={pending}
                    onClick={() => runStatus(() => unfreezeAccount(a.id))}
                  >
                    Unfreeze
                  </button>
                ) : null}
                <button
                  type="button"
                  className="ghost"
                  disabled={pending}
                  onClick={() => {
                    if (!window.confirm("Close this account? This cannot be undone.")) return;
                    void runStatus(() => closeAccount(a.id));
                  }}
                >
                  Close
                </button>
              </div>
            ),
          )}
          {form ? (
            <form className="card" style={{ marginTop: 24 }} onSubmit={onMoney}>
              <h2>{form.kind === "fund" ? "Add funds" : "Withdraw"}</h2>
              <label>
                Amount (USD)
                <input value={amount} onChange={(e) => setAmount(e.target.value)} required />
              </label>
              <button type="submit" disabled={pending}>
                {pending ? "Posting…" : form.kind === "fund" ? "Credit account" : "Debit account"}
              </button>
            </form>
          ) : null}
        </>
      )}
    </div>
  );
}
