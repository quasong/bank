import { useState } from "react";
import { Link } from "react-router-dom";
import { errorMessage, openAccount, type ActivityItem } from "../api";
import { CURRENCIES, greeting, statusLabel, todayKicker } from "../format";
import { useAccounts, useActivity, useSelectedAccount, useToast } from "../hooks";
import { MoneySheet } from "../moneyflow";
import {
  AccountHero,
  Banner,
  EmptyState,
  IconPlus,
  IconSend,
  IconSwap,
  IconOut,
  MoneyText,
  Page,
  PageSkeleton,
  Sheet,
  Toast,
  TxnDetail,
  TxnRow,
} from "../ui";

type MoneyKind = "fund" | "withdraw";

export function OverviewPage() {
  const { accounts, error, setError, reload } = useAccounts();
  const { selected, select } = useSelectedAccount(accounts);
  const { items, reload: reloadActivity } = useActivity(selected?.id);
  const { text: toast, show } = useToast();
  const [pending, setPending] = useState(false);
  const [money, setMoney] = useState<MoneyKind | null>(null);
  const [openTxn, setOpenTxn] = useState<ActivityItem | null>(null);
  const [adding, setAdding] = useState(false);

  const missing = CURRENCIES.filter((c) => !(accounts ?? []).some((a) => a.currency === c));

  async function onOpen(currency?: string) {
    setError("");
    setPending(true);
    try {
      const res = await openAccount(currency);
      const list = await reload();
      select(res.account.id);
      show(list.length > 1 ? `${res.account.currency} opened` : "Account opened");
      setAdding(false);
    } catch (err) {
      setError(errorMessage(err, "Could not open account"));
    } finally {
      setPending(false);
    }
  }

  async function onMoneySuccess(message: string) {
    setMoney(null);
    show(message);
    await reload();
    await reloadActivity();
  }

  if (!accounts) {
    return <PageSkeleton />;
  }

  const canMove = selected?.status === "active";
  const recent = items?.slice(0, 6) ?? [];

  return (
    <Page title={greeting()} kicker={todayKicker()}>
      {error ? <Banner>{error}</Banner> : null}
      {accounts.length === 0 ? (
        <EmptyState
          title="Open your USD account"
          body="One tap creates a USD balance. You can add euros and pounds after that."
          action={
            <button className="btn btn-primary" type="button" disabled={pending} onClick={() => void onOpen()}>
              {pending ? "Opening…" : "Open account"}
            </button>
          }
        />
      ) : selected ? (
        <>
          <AccountHero account={selected} onCopied={() => show("Copied account number")} />
          {!canMove ? (
            <Banner>
              This account is {statusLabel(selected.status)}.{" "}
              <Link to="/accounts">{selected.status === "frozen" ? "Unfreeze it" : "See details"}</Link> to move money.
            </Banner>
          ) : null}
          {accounts.length > 1 || missing.length > 0 ? (
            <section className="wallets">
              {accounts.map((a) => (
                <button
                  key={a.id}
                  type="button"
                  className={`wallet${a.id === selected.id ? " on" : ""}`}
                  onClick={() => select(a.id)}
                >
                  <span>{a.currency}</span>
                  <MoneyText cents={a.balance_cents} currency={a.currency} />
                </button>
              ))}
              {missing.length > 0 ? (
                <button type="button" className="wallet add" onClick={() => setAdding(true)}>
                  <span>Add</span>
                  <strong>+</strong>
                </button>
              ) : null}
            </section>
          ) : null}
          <div className="quicks">
            {canMove ? (
              <Link className="quick" to="/transfers">
                <span className="quick-icon">
                  <IconSend />
                </span>
                Send
              </Link>
            ) : (
              <button className="quick" type="button" disabled>
                <span className="quick-icon">
                  <IconSend />
                </span>
                Send
              </button>
            )}
            {canMove ? (
              <Link className="quick" to="/convert">
                <span className="quick-icon">
                  <IconSwap />
                </span>
                Convert
              </Link>
            ) : (
              <button className="quick" type="button" disabled>
                <span className="quick-icon">
                  <IconSwap />
                </span>
                Convert
              </button>
            )}
            <button className="quick" type="button" disabled={!canMove} onClick={() => setMoney("fund")}>
              <span className="quick-icon">
                <IconPlus />
              </span>
              Add
            </button>
            <button className="quick" type="button" disabled={!canMove} onClick={() => setMoney("withdraw")}>
              <span className="quick-icon">
                <IconOut />
              </span>
              Withdraw
            </button>
          </div>
          <section className="panel">
            <div className="panel-h">
              <h2>Activity</h2>
              <Link to="/activity">See all</Link>
            </div>
            {items == null ? (
              <div className="txn skel-txn">
                <div className="skel skel-icon" />
                <div className="skel skel-line" />
              </div>
            ) : recent.length === 0 ? (
              <p className="panel-empty">
                Nothing here yet.{" "}
                <button type="button" className="text-link" disabled={!canMove} onClick={() => setMoney("fund")}>
                  Add money
                </button>{" "}
                to get started.
              </p>
            ) : (
              recent.map((item) => (
                <TxnRow key={item.journal_id + item.side} item={item} onOpen={setOpenTxn} />
              ))
            )}
          </section>
        </>
      ) : null}
      {money && selected ? (
        <MoneySheet kind={money} account={selected} onClose={() => setMoney(null)} onSuccess={onMoneySuccess} />
      ) : null}
      {adding ? (
        <Sheet title="Add a currency" onClose={() => setAdding(false)}>
          <p className="sheet-copy">Open another balance. Local details look like a real account in that currency.</p>
          <div className="sheet-actions col">
            {missing.map((ccy) => (
              <button
                key={ccy}
                className="btn btn-primary btn-block"
                type="button"
                disabled={pending}
                onClick={() => void onOpen(ccy)}
              >
                Open {ccy}
              </button>
            ))}
          </div>
        </Sheet>
      ) : null}
      {openTxn ? <TxnDetail item={openTxn} onClose={() => setOpenTxn(null)} /> : null}
      <Toast text={toast} />
    </Page>
  );
}
