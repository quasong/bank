import { useEffect, useState } from "react";
import { listAccounts, type BankAccount } from "../api";
import { useAuth } from "../auth";
import { centsToDollars } from "../money";

export function OverviewPage() {
  const { customer } = useAuth();
  const [accounts, setAccounts] = useState<BankAccount[] | null>(null);

  useEffect(() => {
    listAccounts()
      .then((data) => setAccounts(data.accounts))
      .catch(() => setAccounts([]));
  }, []);

  const open = accounts?.[0];

  return (
    <div>
      <h1>Welcome</h1>
      {open ? (
        <p className="lede">
          Signed in as {customer?.email}. Demand deposit {open.account_number} holds $
          {centsToDollars(open.balance_cents)}.
        </p>
      ) : (
        <p className="lede">
          Signed in as {customer?.email}. Your customer profile exists
          {accounts ? "; open a deposit account to hold funds." : "."}
        </p>
      )}
      <div className="grid">
        <article className="tile">
          <h3>Accounts and ledger</h3>
          <p>
            Customer deposits are bank liabilities. Demo funding debits vault cash and credits your
            deposit account. Amounts are integer cents.
          </p>
        </article>
        <article className="tile">
          <h3>Transfers and activity</h3>
          <p>
            A transfer writes two journal lines in one transaction, with an idempotency key. Activity
            is those lines, not a second log.
          </p>
        </article>
      </div>
    </div>
  );
}
