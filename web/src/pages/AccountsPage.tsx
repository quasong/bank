import { FormEvent, useEffect, useState } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import {
  ApiError,
  closeAccount,
  freezeAccount,
  fundAccount,
  openAccount,
  unfreezeAccount,
  withdrawAccount,
} from "../api";
import { dollarsToCents } from "../money";
import { formatUSD } from "../format";
import { useAccounts } from "../hooks";
import { AccountHero, AmountField, Banner, EmptyState, IconArrow, IconFreeze, IconMinus, IconPlus, Page, Sheet } from "../ui";

type MoneyKind = "fund" | "withdraw";

export function AccountsPage() {
  const { accounts, error, setError, reload } = useAccounts();
  const [pending, setPending] = useState(false);
  const [form, setForm] = useState<MoneyKind | null>(null);
  const [amount, setAmount] = useState("100.00");
  const [closing, setClosing] = useState(false);
  const navigate = useNavigate();
  const [searchParams, setSearchParams] = useSearchParams();

  useEffect(() => {
    const action = searchParams.get("action");
    if (action === "fund" || action === "withdraw") {
      setForm(action);
      setSearchParams({}, { replace: true });
    }
  }, [searchParams, setSearchParams]);

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
    const acct = accounts?.[0];
    if (!acct || !form) return;
    const cents = dollarsToCents(amount);
    if (cents == null || cents <= 0) {
      setError("Enter an amount with at most two decimals");
      return;
    }
    setError("");
    setPending(true);
    try {
      if (form === "fund") {
        await fundAccount(acct.id, cents, crypto.randomUUID());
      } else {
        await withdrawAccount(acct.id, cents, crypto.randomUUID());
      }
      setForm(null);
      await reload();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Could not post the amount");
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
      setClosing(false);
      await reload();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Could not update account");
    } finally {
      setPending(false);
    }
  }

  if (!accounts) {
    return <p className="kicker">Loading…</p>;
  }

  const acct = accounts[0];

  return (
    <Page title="Account" kicker="USD">
      {error ? <Banner>{error}</Banner> : null}
      {!acct ? (
        <EmptyState
          title="No account yet"
          body="Open a USD demand-deposit account to hold a balance."
          action={
            <button className="btn btn-primary" type="button" disabled={pending} onClick={onOpen}>
              {pending ? "Opening…" : "Open account"}
            </button>
          }
        />
      ) : (
        <>
          <AccountHero account={acct} />
          {acct.status === "active" ? (
            <div className="quicks">
              <button className="quick" type="button" onClick={() => setForm("fund")}>
                <span className="quick-icon">
                  <IconPlus />
                </span>
                Add
              </button>
              <button className="quick" type="button" onClick={() => setForm("withdraw")}>
                <span className="quick-icon">
                  <IconMinus />
                </span>
                Withdraw
              </button>
              <button className="quick" type="button" onClick={() => navigate("/transfers")}>
                <span className="quick-icon">
                  <IconArrow />
                </span>
                Send
              </button>
              <button
                className="quick"
                type="button"
                disabled={pending}
                onClick={() => runStatus(() => freezeAccount(acct.id))}
              >
                <span className="quick-icon">
                  <IconFreeze />
                </span>
                Freeze
              </button>
            </div>
          ) : null}
          {acct.status === "frozen" ? (
            <div className="manage">
              <button
                className="btn btn-primary"
                type="button"
                disabled={pending}
                onClick={() => runStatus(() => unfreezeAccount(acct.id))}
              >
                Unfreeze
              </button>
              <button className="btn btn-danger" type="button" disabled={pending} onClick={() => setClosing(true)}>
                Close account
              </button>
            </div>
          ) : null}
          {acct.status === "active" ? (
            <div className="manage">
              <button className="btn btn-quiet" type="button" disabled={pending} onClick={() => setClosing(true)}>
                Close account
              </button>
            </div>
          ) : null}
          {acct.status === "closed" ? <p className="kicker">This account is closed and cannot move money.</p> : null}
        </>
      )}

      {form && acct ? (
        <Sheet title={form === "fund" ? "Add money" : "Withdraw"} onClose={() => setForm(null)}>
          <form className="stack" onSubmit={onMoney}>
            <AmountField value={amount} onChange={setAmount} />
            <p className="avail">Available {formatUSD(acct.balance_cents)}</p>
            <button className="btn btn-primary btn-block" type="submit" disabled={pending}>
              {pending ? "Working…" : form === "fund" ? "Add money" : "Withdraw"}
            </button>
          </form>
        </Sheet>
      ) : null}

      {closing && acct ? (
        <Sheet title="Close this account?" onClose={() => setClosing(false)}>
          <p className="lede" style={{ marginBottom: 16 }}>
            You can only close it when the balance is zero. This cannot be undone.
          </p>
          <div className="manage">
            <button className="btn btn-secondary" type="button" onClick={() => setClosing(false)}>
              Keep it
            </button>
            <button
              className="btn btn-danger"
              type="button"
              disabled={pending}
              onClick={() => runStatus(() => closeAccount(acct.id))}
            >
              Close account
            </button>
          </div>
        </Sheet>
      ) : null}
    </Page>
  );
}
