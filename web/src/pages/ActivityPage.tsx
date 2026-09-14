import { useMemo } from "react";
import { Link } from "react-router-dom";
import type { ActivityItem } from "../api";
import { dayLabel } from "../format";
import { useAccounts, useActivity } from "../hooks";
import { Banner, EmptyState, Page, PageSkeleton, TxnRow } from "../ui";

export function ActivityPage() {
  const { accounts, error } = useAccounts();
  const account = accounts?.[0];
  const { items } = useActivity(account?.id);

  const groups = useMemo(() => {
    const map = new Map<string, ActivityItem[]>();
    for (const item of items ?? []) {
      const key = dayLabel(item.created_at);
      const list = map.get(key) ?? [];
      list.push(item);
      map.set(key, list);
    }
    return [...map.entries()];
  }, [items]);

  if (!accounts || items == null) {
    return <PageSkeleton />;
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
                <TxnRow key={item.journal_id + item.side} item={item} />
              ))}
            </div>
          ))}
        </section>
      )}
    </Page>
  );
}
