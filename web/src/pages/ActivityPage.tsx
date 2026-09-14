import { useEffect, useMemo, useState } from "react";
import { Link } from "react-router-dom";
import { listActivity, type ActivityItem } from "../api";
import { activityHint, activityTitle, dayLabel, timeLabel } from "../format";
import { useAccounts } from "../hooks";
import { Banner, EmptyState, IconIn, IconOut, MoneyText, Page } from "../ui";

export function ActivityPage() {
  const { accounts, error, setError } = useAccounts();
  const [items, setItems] = useState<ActivityItem[]>([]);
  const account = accounts?.[0];

  useEffect(() => {
    if (!account) {
      setItems([]);
      return;
    }
    listActivity(account.id)
      .then((data) => setItems(data.items))
      .catch((err) => setError(err instanceof Error ? err.message : "Could not load activity"));
  }, [account, setError]);

  const groups = useMemo(() => {
    const map = new Map<string, ActivityItem[]>();
    for (const item of items) {
      const key = dayLabel(item.created_at);
      const list = map.get(key) ?? [];
      list.push(item);
      map.set(key, list);
    }
    return [...map.entries()];
  }, [items]);

  if (!accounts) {
    return <p className="kicker">Loading…</p>;
  }

  if (!account) {
    return (
      <Page title="Activity">
        <EmptyState
          title="No activity yet"
          body="Open an account and add money to see payments here."
          action={
            <Link className="btn btn-primary" to="/accounts">
              Go to account
            </Link>
          }
        />
      </Page>
    );
  }

  return (
    <Page title="Activity">
      {error ? <Banner>{error}</Banner> : null}
      {items.length === 0 ? (
        <EmptyState title="Nothing here yet" body="Add money or send a payment and it will show up in this list." />
      ) : (
        <section className="panel">
          {groups.map(([day, rows]) => (
            <div className="day-group" key={day}>
              <p className="day-label">{day}</p>
              {rows.map((item) => (
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
              ))}
            </div>
          ))}
        </section>
      )}
    </Page>
  );
}
