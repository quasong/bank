import { useState } from "react";
import { Link } from "react-router-dom";
import { errorMessage, openAccount, type ActivityItem, type BankAccount } from "../api";
import { CURRENCIES, greeting, statusLabel, todayKicker } from "../format";
import { jarsFor, spendAccounts, useAccounts, useActivity, useSelectedAccount, useToast } from "../hooks";
import { AddJarSheet, JarSheet, JarsPanel, createJar } from "../jars";
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
  const [addingJar, setAddingJar] = useState(false);
  const [openJar, setOpenJar] = useState<BankAccount | null>(null);

  const missing = CURRENCIES.filter((c) => !spendAccounts(accounts).some((a) => a.currency === c));
  const wallets = spendAccounts(accounts);

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

  async function onJar(label: string) {
    if (!selected) return;
    setError("");
    setPending(true);
    try {
      const res = await createJar(selected.currency, label);
      await reload();
      show(`${res.account.label || "Jar"} is ready`);
      setAddingJar(false);
    } catch (err) {
      setError(errorMessage(err, "Could not open jar"));
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
          body="One tap creates a US dollar balance. Add euros, pounds, and other currencies whenever you need them."
          action={
            <button className="btn btn-primary" type="button" disabled={pending} onClick={() => void onOpen()}>
              {pending ? "Opening…" : "Open account"}
            </button>
          }
        />
      ) : selected ? (
        <>
          <Wallets
            accounts={wallets}
            selectedId={selected.id}
            onSelect={select}
            onAdd={missing.length > 0 ? () => setAdding(true) : undefined}
          />
          <AccountHero account={selected} onCopied={() => show("Copied")} />
          <JarsPanel
            spend={selected}
            jars={jarsFor(accounts, selected.currency)}
            canMove={canMove}
            onAdd={() => setAddingJar(true)}
            onOpen={setOpenJar}
          />
          {!canMove ? (
            <Banner>
              This account is {statusLabel(selected.status)}.{" "}
              <Link to="/accounts">{selected.status === "frozen" ? "Unfreeze it" : "See details"}</Link> to move money.
            </Banner>
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
              <h2>Recent</h2>
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
      {addingJar && selected ? (
        <AddJarSheet spend={selected} pending={pending} onClose={() => setAddingJar(false)} onCreate={(label) => void onJar(label)} />
      ) : null}
      {openJar && selected ? (
        <JarSheet
          spend={selected}
          jar={openJar}
          pending={pending}
          onClose={() => setOpenJar(null)}
          onReload={async () => {
            const next = await reload();
            const fresh = next.find((a) => a.id === openJar.id);
            if (fresh && fresh.status !== "closed") setOpenJar(fresh);
            else setOpenJar(null);
            await reloadActivity();
          }}
          onToast={show}
          onError={setError}
        />
      ) : null}
      {openTxn ? <TxnDetail item={openTxn} onClose={() => setOpenTxn(null)} /> : null}
      <Toast text={toast} />
    </Page>
  );
}
