import { FormEvent, useEffect, useState } from "react";
import { ApiError, fundAccount, listAccounts, openAccount, type BankAccount } from "../api";
import { centsToDollars, dollarsToCents } from "../money";

export function AccountsPage() {
  const [accounts, setAccounts] = useState<BankAccount[] | null>(null);
  const [error, setError] = useState("");
  const [pending, setPending] = useState(false);
  const [fundId, setFundId] = useState<string | null>(null);
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

  async function onFund(e: FormEvent) {
    e.preventDefault();
    if (!fundId) return;
    const cents = dollarsToCents(amount);
    if (cents == null || cents <= 0) {
      setError("Enter a dollar amount with at most two decimals");
      return;
    }
    setError("");
    setPending(true);
    try {
      await fundAccount(fundId, cents, crypto.randomUUID());
      setFundId(null);
      await reload();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Funding failed");
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
      <p className="lede">Demand-deposit balances come from the ledger. Add funds is a demo credit from vault cash.</p>
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
                <th></th>
              </tr>
            </thead>
            <tbody>
              {accounts.map((a) => (
                <tr key={a.id}>
                  <td className="money">{a.account_number}</td>
                  <td>{a.status}</td>
                  <td className="money">${centsToDollars(a.balance_cents)}</td>
                  <td>
                    {a.status === "active" ? (
                      <button type="button" className="ghost" onClick={() => setFundId(a.id)}>
                        Add funds
                      </button>
                    ) : null}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
          {fundId ? (
            <form className="card" style={{ marginTop: 24 }} onSubmit={onFund}>
              <h2>Add funds</h2>
              <label>
                Amount (USD)
                <input value={amount} onChange={(e) => setAmount(e.target.value)} required />
              </label>
              <button type="submit" disabled={pending}>
                {pending ? "Posting…" : "Credit account"}
              </button>
            </form>
          ) : null}
        </>
      )}
    </div>
  );
}
