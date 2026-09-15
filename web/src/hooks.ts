import { useCallback, useEffect, useRef, useState } from "react";
import { listAccounts, listActivity, listAudit, listPayees, type ActivityItem, type AuditEvent, type BankAccount, type Payee } from "./api";

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
  const [items, setItems] = useState<ActivityItem[] | null>(null);

  const reload = useCallback(async () => {
    if (!accountId) {
      setItems([]);
      return [];
    }
    const data = await listActivity(accountId);
    setItems(data.items);
    return data.items;
  }, [accountId]);

  useEffect(() => {
    reload().catch(() => setItems([]));
  }, [reload]);

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

export function useSelectedAccount(accounts: BankAccount[] | null) {
  const [id, setId] = useState(() => {
    try {
      return sessionStorage.getItem(SELECTED_KEY) ?? "";
    } catch {
      return "";
    }
  });

  const selected = accounts?.find((a) => a.id === id) ?? accounts?.[0];

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
      const next = accounts.find((a) => a.status === "active")?.id ?? accounts[0].id;
      select(next);
    }
  }, [accounts, id, select]);

  return { selected, select };
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
