import { NavLink, Outlet, useNavigate } from "react-router-dom";
import { useAuth } from "../auth";
import { BANK_NAME } from "../brand";

export function AppShell() {
  const { customer, logout } = useAuth();
  const navigate = useNavigate();

  async function onLogout() {
    await logout();
    navigate("/login", { replace: true });
  }

  return (
    <div className="shell">
      <aside className="sidebar">
        <div className="brand">
          <span className="mark" />
          <div>
            <strong>{BANK_NAME}</strong>
            <em>Internet banking</em>
          </div>
        </div>
        <nav>
          <NavLink to="/" end>
            Overview
          </NavLink>
          <NavLink to="/accounts">Accounts</NavLink>
          <NavLink to="/transfers">Transfers</NavLink>
          <NavLink to="/activity">Activity</NavLink>
        </nav>
      </aside>
      <div className="main">
        <header className="topbar">
          <span>{customer?.email}</span>
          <button type="button" className="ghost" onClick={onLogout}>
            Sign out
          </button>
        </header>
        <section className="content">
          <Outlet />
        </section>
      </div>
    </div>
  );
}

export function OverviewPage() {
  const { customer } = useAuth();
  return (
    <div>
      <h1>Welcome</h1>
      <p className="lede">
        Signed in as {customer?.email}. Your customer profile exists; deposit accounts
        are not open yet.
      </p>
      <div className="grid">
        <article className="tile">
          <h3>Next: accounts and ledger</h3>
          <p>
            Open a demand-deposit account. Balances come from journal totals. Amounts
            are stored as integer cents, never float64.
          </p>
        </article>
        <article className="tile">
          <h3>Then: transfers and activity</h3>
          <p>
            Write debit and credit lines in one transaction, with an idempotency key.
            Activity is read from the ledger, not a second fake log.
          </p>
        </article>
      </div>
    </div>
  );
}

export function ComingSoonPage({ title, blurb }: { title: string; blurb: string }) {
  return (
    <div className="soon">
      <p className="eyebrow">Coming soon</p>
      <h1>{title}</h1>
      <p className="lede">{blurb}</p>
    </div>
  );
}
