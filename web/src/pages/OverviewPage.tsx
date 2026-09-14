import { useState } from "react";
import { Link } from "react-router-dom";
import { errorMessage, openAccount, type ActivityItem } from "../api";
import { greeting, statusLabel, todayKicker } from "../format";
import { useAccounts, useActivity, useToast } from "../hooks";
import { MoneySheet } from "../moneyflow";
import {
  AccountHero,
  Banner,
  EmptyState,
  IconPlus,
  IconSend,
  IconWallet,
  IconOut,
  Page,
  PageSkeleton,
  Toast,
  TxnDetail,
  TxnRow,
} from "../ui";

type MoneyKind = "fund" | "withdraw";

export function OverviewPage() {
  const { accounts, error, setError, reload } = useAccounts();
  const open = accounts?.[0];
  const { items, reload: reloadActivity } = useActivity(open?.id);
  const { text: toast, show } = useToast();
  const [pending, setPending] = useState(false);
  const [money, setMoney] = useState<MoneyKind | null>(null);
  const [openTxn, setOpenTxn] = useState<ActivityItem | null>(null);

  async function onOpen() {
    setError("");
    setPending(true);
    try {
      await openAccount();
      await reload();
      show("Account opened");
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

  const canMove = open?.status === "active";
  const recent = items?.slice(0, 6) ?? [];

  return (
    <Page title={greeting()} kicker={todayKicker()}>
      {error ? <Banner>{error}</Banner> : null}
      {!open ? (
        <EmptyState
          title="Open your USD account"
          body="One tap creates an account so you can add money and send it."
          action={
            <button className="btn btn-primary" type="button" disabled={pending} onClick={onOpen}>
              {pending ? "Opening…" : "Open account"}
            </button>
          }
        />
      ) : (
        <>
          <AccountHero account={open} onCopied={() => show("Copied account number")} />
          {!canMove ? (
            <Banner>
              This account is {statusLabel(open.status)}.{" "}
              <Link to="/accounts">{open.status === "frozen" ? "Unfreeze it" : "See details"}</Link> to move money.
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
            <Link className="quick" to="/accounts">
              <span className="quick-icon">
                <IconWallet />
              </span>
              Details
            </Link>
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
      )}
      {money && open ? (
        <MoneySheet kind={money} account={open} onClose={() => setMoney(null)} onSuccess={onMoneySuccess} />
      ) : null}
      {openTxn ? <TxnDetail item={openTxn} onClose={() => setOpenTxn(null)} /> : null}
      <Toast text={toast} />
    </Page>
  );
}
