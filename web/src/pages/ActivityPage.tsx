import { useMemo, useState } from "react";
import { Link } from "react-router-dom";
import type { ActivityItem } from "../api";
import { dayLabel } from "../format";
import { useAccounts, useActivity, useSelectedAccount } from "../hooks";
import { Banner, EmptyState, Page, PageSkeleton, TxnDetail, TxnRow, TxnSkeleton, Wallets } from "../ui";

type Filter = "all" | "in" | "out";

export function ActivityPage() {
  const { accounts, error } = useAccounts();
  const { selected, select } = useSelectedAccount(accounts);
  const { items } = useActivity(selected?.id);
  const [filter, setFilter] = useState<Filter>("all");
  const [openTxn, setOpenTxn] = useState<ActivityItem | null>(null);

  const visible = useMemo(() => {
    const list = items ?? [];
    if (filter === "in") return list.filter((item) => item.signed_cents >= 0);
    if (filter === "out") return list.filter((item) => item.signed_cents < 0);
    return list;
  }, [items, filter]);

  const groups = useMemo(() => {
    const map = new Map<string, ActivityItem[]>();
    for (const item of visible) {
      const key = dayLabel(item.created_at);
      const list = map.get(key) ?? [];
      list.push(item);
      map.set(key, list);
    }
    return [...map.entries()];
  }, [visible]);

  if (!accounts) {
    return <PageSkeleton />;
  }

  if (!selected) {
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
    <Page title="Activity" kicker={selected.currency}>
      {error ? <Banner>{error}</Banner> : null}
      <Wallets accounts={accounts} selectedId={selected.id} onSelect={select} />
      {items == null ? (
        <section className="panel">
          <TxnSkeleton rows={5} />
        </section>
      ) : items.length === 0 ? (
        <EmptyState
          title="Nothing here yet"
          body={`No ${selected.currency} payments yet. Add money or convert into this balance.`}
          action={
            <Link className="btn btn-primary" to="/accounts?action=fund">
              Add money
            </Link>
          }
        />
      ) : (
        <>
          <div className="filters" role="tablist" aria-label="Filter activity">
            {(
              [
                ["all", "All"],
                ["in", "In"],
                ["out", "Out"],
              ] as const
            ).map(([id, label]) => (
              <button
                key={id}
                type="button"
                role="tab"
                aria-selected={filter === id}
                className={`chip${filter === id ? " chip-on" : ""}`}
                onClick={() => setFilter(id)}
              >
                {label}
              </button>
            ))}
          </div>
          {visible.length === 0 ? (
            <EmptyState title="No matching payments" body="Try another filter to see the rest of your activity." />
          ) : (
            <section className="panel">
              {groups.map(([day, rows]) => (
                <div className="day-group" key={day}>
                  <p className="day-label">{day}</p>
                  {rows.map((item) => (
                    <TxnRow key={item.journal_id + item.side} item={item} onOpen={setOpenTxn} />
                  ))}
                </div>
              ))}
            </section>
          )}
        </>
      )}
      {openTxn ? <TxnDetail item={openTxn} onClose={() => setOpenTxn(null)} /> : null}
    </Page>
  );
}
