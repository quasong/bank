import { useCallback, useEffect, useRef, useState } from "react";
import { listAccounts, listActivity, type ActivityItem, type BankAccount } from "./api";

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
