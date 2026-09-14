import { FormEvent, useEffect, useState } from "react";
import { ApiError, createTransfer, listAccounts, type BankAccount } from "../api";
import { dollarsToCents } from "../money";

export function TransfersPage() {
  const [accounts, setAccounts] = useState<BankAccount[]>([]);
  const [fromId, setFromId] = useState("");
  const [toNumber, setToNumber] = useState("");
  const [amount, setAmount] = useState("10.00");
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");
  const [pending, setPending] = useState(false);

  useEffect(() => {
    listAccounts()
      .then((data) => {
        setAccounts(data.accounts);
        if (data.accounts[0]) setFromId(data.accounts[0].id);
      })
      .catch((err) => setError(err instanceof Error ? err.message : "Failed to load accounts"));
  }, []);

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    const cents = dollarsToCents(amount);
    if (cents == null || cents <= 0) {
      setError("Enter a dollar amount with at most two decimals");
      return;
    }
    if (blocked) {
      setError(`This account cannot send money while ${selected?.status}.`);
      return;
    }
    setError("");
    setNotice("");
    setPending(true);
    try {
      const res = await createTransfer(fromId, toNumber, cents, crypto.randomUUID());
      setNotice(
        res.transfer.replay
          ? "Idempotent replay; balances were not moved again."
          : `Sent. Destination ${res.transfer.to_account_number}.`,
      );
      const data = await listAccounts();
      setAccounts(data.accounts);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Transfer failed");
    } finally {
      setPending(false);
    }
  }

  const selected = accounts.find((a) => a.id === fromId);
  const blocked = selected != null && selected.status !== "active";

  if (accounts.length === 0) {
    return (
      <div>
        <h1>Transfers</h1>
        <p className="lede">Open and fund a deposit account before sending money.</p>
      </div>
    );
  }

  return (
    <div>
      <h1>Transfers</h1>
      <p className="lede">Debit your liability account and credit another customer by account number. Each submit uses a new idempotency key.</p>
      {error ? <p className="error">{error}</p> : null}
      {notice ? <p className="lede">{notice}</p> : null}
      {blocked ? <p className="error">This account cannot send money while {selected?.status}.</p> : null}
      <form className="card" onSubmit={onSubmit}>
        <label>
          From
          <select value={fromId} onChange={(e) => setFromId(e.target.value)}>
            {accounts.map((a) => (
              <option key={a.id} value={a.id}>
                {a.account_number} ({a.status})
              </option>
            ))}
          </select>
        </label>
        <label>
          Destination account number
          <input
            value={toNumber}
            onChange={(e) => setToNumber(e.target.value.replace(/\D/g, "").slice(0, 8))}
            required
            disabled={blocked}
            inputMode="numeric"
            pattern="[0-9]{8}"
            maxLength={8}
            placeholder="00000000"
          />
        </label>
        <label>
          Amount (USD)
          <input value={amount} onChange={(e) => setAmount(e.target.value)} required disabled={blocked} />
        </label>
        <button type="submit" disabled={pending || blocked}>
          {pending ? "Sending…" : "Send"}
        </button>
      </form>
    </div>
  );
}
