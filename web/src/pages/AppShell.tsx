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
