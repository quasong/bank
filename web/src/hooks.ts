import { useCallback, useEffect, useState } from "react";
import { listAccounts, type BankAccount } from "./api";

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
