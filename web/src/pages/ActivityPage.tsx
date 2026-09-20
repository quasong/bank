import { useMemo, useState } from "react";
import { Link } from "react-router-dom";
import type { ActivityItem } from "../api";
import { activityHint, activityKindLabel, activityTitle, currencyName, dayLabel } from "../format";
import { useAccounts, useActivityFeed, useSelectedAccount } from "../hooks";
import { Banner, EmptyState, Page, PageSkeleton, TxnDetail, TxnRow, TxnSkeleton, Wallets } from "../ui";

type Filter = "all" | "in" | "out";
type Scope = "wallet" | "all";

function matchesActivity(item: ActivityItem, needle: string) {
  if (!needle) return true;
  const hay = [
    item.kind,
    item.description,
    item.note,
    item.counterparty_name,
    item.counterparty_account_number,
    item.receipt,
    item.journal_id,
    item.currency,
    item.currency ? currencyName(item.currency) : "",
    activityTitle(item.kind, item.signed_cents),
    activityKindLabel(item.kind, item.signed_cents),
    activityHint(
      item.kind,
      item.signed_cents,
      item.counterparty_account_number,
      item.counterparty_name,
      item.note,
      item.description,
    ),
  ]
    .join(" ")
    .toLowerCase();
  return hay.includes(needle);
}

export function ActivityPage() {
  const { accounts, error } = useAccounts();
  const { selected, select } = useSelectedAccount(accounts);
  const [scope, setScope] = useState<Scope>("wallet");
  const [filter, setFilter] = useState<Filter>("all");
  const [query, setQuery] = useState("");
  const [openTxn, setOpenTxn] = useState<ActivityItem | null>(null);

  const accountIds = useMemo(() => {
    if (!accounts?.length || !selected) return [] as string[];
    if (scope === "all") return accounts.map((a) => a.id);
    return [selected.id];
  }, [accounts, selected, scope]);

  const { items } = useActivityFeed(accountIds);

  const visible = useMemo(() => {
    const needle = query.trim().toLowerCase();
    return (items ?? []).filter((item) => {
      if (filter === "in" && item.signed_cents < 0) return false;
      if (filter === "out" && item.signed_cents >= 0) return false;
      return matchesActivity(item, needle);
    });
  }, [items, filter, query]);

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

  const searching = query.trim().length > 0;
  const allBalances = scope === "all";

  return (
    <Page title="Activity" kicker={allBalances ? "All balances" : currencyName(selected.currency)}>
      {error ? <Banner>{error}</Banner> : null}
      <Wallets
        accounts={accounts}
        selectedId={selected.id}
        onSelect={(id) => {
          select(id);
          setScope("wallet");
        }}
      />
      <label className="finder activity-find">
        <span className="sr-only">Search activity</span>
        <input
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          placeholder="Search activity"
          autoComplete="off"
        />
      </label>
      <div className="filters" role="tablist" aria-label="Activity scope">
        {(
          [
            ["wallet", "This wallet"],
            ["all", "All balances"],
          ] as const
        ).map(([id, label]) => (
          <button
            key={id}
            type="button"
            role="tab"
            aria-selected={scope === id}
            className={`chip${scope === id ? " chip-on" : ""}`}
            onClick={() => setScope(id)}
          >
            {label}
          </button>
        ))}
      </div>
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
      {items == null ? (
        <section className="panel">
          <TxnSkeleton rows={5} />
        </section>
      ) : items.length === 0 && !searching ? (
        <EmptyState
          title="Nothing here yet"
          body={
            allBalances
              ? "No payments on any balance yet. Add money or convert to get started."
              : `No ${selected.currency} payments yet. Add money or convert into this balance.`
          }
          action={
            <Link className="btn btn-primary" to="/accounts?action=fund">
              Add money
            </Link>
          }
        />
      ) : visible.length === 0 ? (
        <EmptyState
          title="No matching payments"
          body={searching ? "Try a name, note, currency, or receipt detail." : "Try another filter to see the rest of your activity."}
        />
      ) : (
        <section className="panel">
          {groups.map(([day, rows]) => (
            <div className="day-group" key={day}>
              <p className="day-label">{day}</p>
              {rows.map((item) => (
                <TxnRow key={item.journal_id + item.side} item={item} onOpen={setOpenTxn} showCurrency={allBalances} />
              ))}
            </div>
          ))}
        </section>
      )}
      {openTxn ? <TxnDetail item={openTxn} onClose={() => setOpenTxn(null)} /> : null}
    </Page>
  );
}
