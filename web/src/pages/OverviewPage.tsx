import { useState } from "react";
import { Link } from "react-router-dom";
import { errorMessage, openAccount, type ActivityItem } from "../api";
import { CURRENCIES, greeting, statusLabel, todayKicker } from "../format";
import { useAccounts, useActivity, useSelectedAccount, useToast } from "../hooks";
import { MoneySheet } from "../moneyflow";
import {
  AccountHero,
  Banner,
  CurrencyChoices,
  EmptyState,
  IconPlus,
  IconSend,
  IconSwap,
  IconOut,
  Page,
  PageSkeleton,
  Sheet,
  Toast,
  TxnDetail,
  TxnRow,
  TxnSkeleton,
  Wallets,
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
    const first = (accounts?.length ?? 0) === 0;
    try {
      const res = await openAccount(currency);
      await reload();
      if (first) select(res.account.id);
      show(first ? "Account opened" : `${res.account.currency} is ready`);
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
          body="One tap creates a US dollar balance. Add euros and pounds whenever you need them."
          action={
            <button className="btn btn-primary" type="button" disabled={pending} onClick={() => void onOpen()}>
              {pending ? "Opening…" : "Open account"}
            </button>
          }
        />
      ) : selected ? (
        <>
          <AccountHero account={selected} onCopied={() => show("Copied")} />
          {!canMove ? (
            <Banner>
              This account is {statusLabel(selected.status)}.{" "}
              <Link to="/accounts">{selected.status === "frozen" ? "Unfreeze it" : "See details"}</Link> to move money.
            </Banner>
          ) : null}
          <Wallets
            accounts={accounts}
            selectedId={selected.id}
            onSelect={select}
            onAdd={missing.length > 0 ? () => setAdding(true) : undefined}
          />
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
              <h2>{selected.currency} activity</h2>
              <Link to="/activity">See all</Link>
            </div>
            {items == null ? (
              <TxnSkeleton rows={3} />
            ) : recent.length === 0 ? (
              <p className="panel-empty">
                Nothing in {selected.currency} yet.{" "}
                {canMove ? (
                  <>
                    <button type="button" className="text-link" onClick={() => setMoney("fund")}>
                      Add money
                    </button>
                    {" or "}
                    <Link to="/convert">convert</Link>
                  </>
                ) : (
                  "Add money"
                )}{" "}
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
          <p className="sheet-copy">Each balance gets its own local details, like Wise.</p>
          <CurrencyChoices currencies={missing} pending={pending} onPick={(ccy) => void onOpen(ccy)} />
        </Sheet>
      ) : null}
      {openTxn ? <TxnDetail item={openTxn} onClose={() => setOpenTxn(null)} /> : null}
      <Toast text={toast} />
    </Page>
  );
}
