import { useCallback, useEffect, useRef, useState } from "react";
import { listAccounts, listActivity, listAudit, listPayees, type ActivityItem, type AuditEvent, type BankAccount, type Payee } from "./api";

export function isSpend(a: { product?: string }) {
  return a.product !== "jar";
}

export function isJar(a: { product?: string }) {
  return a.product === "jar";
}

export function jarName(a: { label?: string }) {
  return (a.label ?? "").trim() || "Jar";
}

export function spendAccounts(accounts: BankAccount[] | null | undefined) {
  return (accounts ?? []).filter(isSpend);
}

export function jarsFor(accounts: BankAccount[] | null | undefined, currency: string) {
  return (accounts ?? []).filter((a) => isJar(a) && a.currency === currency && a.status !== "closed");
}

export function jarsParked(accounts: BankAccount[] | null | undefined, currency: string) {
  return jarsFor(accounts, currency).reduce((sum, j) => sum + j.balance_cents, 0);
}

export function pocketAccountIds(accounts: BankAccount[] | null | undefined, spend?: BankAccount) {
  if (!spend) return [] as string[];
  return [spend.id, ...jarsFor(accounts, spend.currency).map((j) => j.id)];
}

export function collapseMoves(items: ActivityItem[]) {
  const seen = new Set<string>();
  return items.filter((item) => {
    if (item.kind !== "move") return true;
    if (seen.has(item.journal_id)) return false;
    seen.add(item.journal_id);
    return true;
  });
}

export function useAccounts() {
  const [accounts, setAccounts] = useState<BankAccount[] | null>(null);
  const [error, setError] = useState("");

  const reload = useCallback(async () => {
    const data = await listAccounts();
    setAccounts(data.accounts);
    return data.accounts;
  }, []);

  useEffect(() => {
    reload().catch((err) => setError(err instanceof Error ? err.message : "Could not load accounts"));
  }, [reload]);

  return { accounts, error, setError, reload };
}

export function useActivity(accountId: string | undefined) {
  const [state, setState] = useState<{ id?: string; items: ActivityItem[] | null }>({ items: null });

  const reload = useCallback(async () => {
    if (!accountId) {
      setState({ id: undefined, items: [] });
      return [];
    }
    const data = await listActivity(accountId);
    setState({ id: accountId, items: data.items });
    return data.items;
  }, [accountId]);

  useEffect(() => {
    let cancelled = false;
    if (!accountId) {
      setState({ id: undefined, items: [] });
      return;
    }
    setState({ id: accountId, items: null });
    listActivity(accountId)
      .then((data) => {
        if (!cancelled) setState({ id: accountId, items: data.items });
      })
      .catch(() => {
        if (!cancelled) setState({ id: accountId, items: [] });
      });
    return () => {
      cancelled = true;
    };
  }, [accountId]);

  const items = state.id === accountId ? state.items : null;
  return { items, reload };
}

export function usePayees() {
  const [payees, setPayees] = useState<Payee[] | null>(null);

  const reload = useCallback(async () => {
    const data = await listPayees();
    setPayees(data.payees);
    return data.payees;
  }, []);

  useEffect(() => {
    reload().catch(() => setPayees([]));
  }, [reload]);

  return { payees, reload };
}

export function useAudit() {
  const [events, setEvents] = useState<AuditEvent[] | null>(null);

  const reload = useCallback(async () => {
    const data = await listAudit();
    setEvents(data.events);
    return data.events;
  }, []);

  useEffect(() => {
    reload().catch(() => setEvents([]));
  }, [reload]);

  return { events, reload };
}

const SELECTED_KEY = "tb.selected-account";

function preferredAccount(accounts: BankAccount[], id: string) {
  const spend = spendAccounts(accounts);
  const match = spend.find((a) => a.id === id);
  if (match) return match;
  return (
    spend.find((a) => a.status === "active" && a.balance_cents > 0) ??
    spend.find((a) => a.status === "active") ??
    spend[0] ??
    accounts[0]
  );
}

export function useSelectedAccount(accounts: BankAccount[] | null) {
  const [id, setId] = useState(() => {
    try {
      return sessionStorage.getItem(SELECTED_KEY) ?? "";
    } catch {
      return "";
    }
  });

  const selected = accounts?.length ? preferredAccount(accounts, id) : undefined;

  const select = useCallback((next: string) => {
    setId(next);
    try {
      sessionStorage.setItem(SELECTED_KEY, next);
    } catch {
      /* ignore */
    }
  }, []);

  useEffect(() => {
    if (!accounts?.length) return;
    if (!accounts.some((a) => a.id === id)) {
      select(preferredAccount(accounts, "").id);
    }
  }, [accounts, id, select]);

  return { selected, select };
}

function sortActivity(batches: ActivityItem[][]) {
  return batches.flat().sort((a, b) => (a.created_at < b.created_at ? 1 : a.created_at > b.created_at ? -1 : 0));
}

export function useActivityFeed(accountIds: string[], limit = 100) {
  const key = accountIds.join(",");
  const [state, setState] = useState<{ key: string; items: ActivityItem[] | null }>({ key: "", items: null });

  const reload = useCallback(async () => {
    const ids = key ? key.split(",") : [];
    if (!ids.length) {
      setState({ key, items: [] });
      return [] as ActivityItem[];
    }
    const batches = await Promise.all(ids.map((id) => listActivity(id, limit).then((data) => data.items)));
    const items = sortActivity(batches);
    setState({ key, items });
    return items;
  }, [key, limit]);

  useEffect(() => {
    let cancelled = false;
    const ids = key ? key.split(",") : [];
    if (!ids.length) {
      setState({ key, items: [] });
      return;
    }
    setState({ key, items: null });
    Promise.all(ids.map((id) => listActivity(id, limit).then((data) => data.items)))
      .then((batches) => {
        if (cancelled) return;
        setState({ key, items: sortActivity(batches) });
      })
      .catch(() => {
        if (!cancelled) setState({ key, items: [] });
      });
    return () => {
      cancelled = true;
    };
  }, [key, limit]);

  const items = state.key === key ? state.items : null;
  return { items, reload };
}

export function useToast(ms = 2400) {
  const [text, setText] = useState("");
  const timer = useRef(0);
  const show = useCallback(
    (next: string) => {
      setText(next);
      window.clearTimeout(timer.current);
      timer.current = window.setTimeout(() => setText(""), ms);
    },
    [ms],
  );
  useEffect(() => () => window.clearTimeout(timer.current), []);
  return { text, show };
}
