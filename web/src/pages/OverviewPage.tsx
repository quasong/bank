import { useEffect, useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import { ApiError, listActivity, openAccount, type ActivityItem } from "../api";
import { activityHint, activityTitle, greeting, timeLabel } from "../format";
import { useAccounts } from "../hooks";
import { AccountHero, Banner, EmptyState, IconIn, IconOut, IconPlus, IconSend, IconWallet, MoneyText, Page } from "../ui";

export function OverviewPage() {
  const { accounts, error, setError, reload } = useAccounts();
  const [items, setItems] = useState<ActivityItem[]>([]);
  const [pending, setPending] = useState(false);
  const navigate = useNavigate();
  const open = accounts?.[0];

  useEffect(() => {
    if (!open) {
      setItems([]);
      return;
    }
    listActivity(open.id)
      .then((data) => setItems(data.items.slice(0, 6)))
      .catch(() => setItems([]));
  }, [open]);

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

  if (!accounts) {
    return <p className="kicker">Loading…</p>;
  }

  const canMove = open?.status === "active";

  return (
    <Page title={greeting()}>
      {error ? <Banner>{error}</Banner> : null}
      {!open ? (
        <EmptyState
          title="Open your USD account"
          body="One tap creates a demand-deposit account so you can add money and send it."
          action={
            <button className="btn btn-primary" type="button" disabled={pending} onClick={onOpen}>
              {pending ? "Opening…" : "Open account"}
            </button>
          }
        />
      ) : (
        <>
          <AccountHero account={open} />
          <div className="quicks">
            <Link className="quick" to={canMove ? "/transfers" : "/accounts"}>
              <span className="quick-icon">
                <IconSend />
              </span>
              Send
            </Link>
            <button className="quick" type="button" onClick={() => navigate("/accounts?action=fund")}>
              <span className="quick-icon">
                <IconPlus />
              </span>
              Add
            </button>
            <button className="quick" type="button" onClick={() => navigate("/accounts?action=withdraw")}>
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
            {items.length === 0 ? (
              <p className="txn-copy" style={{ padding: "18px" }}>
                <span>Nothing here yet. Add money to get started.</span>
              </p>
            ) : (
              items.map((item) => (
                <div className="txn" key={item.journal_id + item.side}>
                  <span className={`txn-icon ${item.signed_cents >= 0 ? "in" : "out"}`}>
                    {item.signed_cents >= 0 ? <IconIn /> : <IconOut />}
                  </span>
                  <div className="txn-copy">
                    <strong>{activityTitle(item.kind, item.signed_cents)}</strong>
                    <span>
                      {activityHint(item.kind)} · {timeLabel(item.created_at)}
                    </span>
                  </div>
                  <MoneyText cents={item.signed_cents} signed />
                </div>
              ))
            )}
          </section>
        </>
      )}
    </Page>
  );
}
