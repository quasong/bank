import { useEffect, useState } from "react";
import { listAccounts, listActivity, type ActivityItem, type BankAccount } from "../api";
import { centsToDollars } from "../money";

export function ActivityPage() {
  const [accounts, setAccounts] = useState<BankAccount[]>([]);
  const [accountId, setAccountId] = useState("");
  const [items, setItems] = useState<ActivityItem[]>([]);
  const [error, setError] = useState("");

  useEffect(() => {
    listAccounts()
      .then((data) => {
        setAccounts(data.accounts);
        if (data.accounts[0]) setAccountId(data.accounts[0].id);
      })
      .catch((err) => setError(err instanceof Error ? err.message : "Failed to load accounts"));
  }, []);

  useEffect(() => {
    if (!accountId) {
      setItems([]);
      return;
    }
    listActivity(accountId)
      .then((data) => setItems(data.items))
      .catch((err) => setError(err instanceof Error ? err.message : "Failed to load activity"));
  }, [accountId]);

  if (accounts.length === 0) {
    return (
      <div>
        <h1>Activity</h1>
        <p className="lede">Journal lines appear here after you open an account and post funding or a transfer.</p>
      </div>
    );
  }

  return (
    <div>
      <h1>Activity</h1>
      <p className="lede">Entries are ledger lines for this deposit account, not a separate fake log.</p>
      {error ? <p className="error">{error}</p> : null}
      <label>
        Account
        <select value={accountId} onChange={(e) => setAccountId(e.target.value)}>
          {accounts.map((a) => (
            <option key={a.id} value={a.id}>
              {a.account_number}
            </option>
          ))}
        </select>
      </label>
      {items.length === 0 ? (
        <p className="lede">No journal lines yet.</p>
      ) : (
        <table className="table">
          <thead>
            <tr>
              <th>When</th>
              <th>Kind</th>
              <th>Description</th>
              <th>Amount</th>
            </tr>
          </thead>
          <tbody>
            {items.map((item) => (
              <tr key={item.journal_id + item.side}>
                <td>{new Date(item.created_at).toLocaleString()}</td>
                <td>{item.kind}</td>
                <td>{item.description}</td>
                <td className="money">
                  {item.signed_cents > 0 ? "+" : ""}${centsToDollars(item.signed_cents)}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </div>
  );
}
