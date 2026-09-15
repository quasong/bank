import { NavLink, Outlet, useNavigate } from "react-router-dom";
import { useAuth } from "../auth";
import { BANK_NAME } from "../brand";
import { IconHome, IconList, IconSend, IconSwap, IconWallet } from "../ui";

const links = [
  { to: "/", label: "Home", icon: <IconHome />, end: true },
  { to: "/accounts", label: "Account", icon: <IconWallet />, end: false },
  { to: "/convert", label: "Convert", icon: <IconSwap />, end: false },
  { to: "/transfers", label: "Send", icon: <IconSend />, end: false },
  { to: "/activity", label: "Activity", icon: <IconList />, end: false },
];

export function AppShell() {
  const { customer, logout } = useAuth();
  const navigate = useNavigate();

  async function onLogout() {
    await logout();
    navigate("/login", { replace: true });
  }

  const navItems = () =>
    links.map((l) => (
      <NavLink key={l.to} to={l.to} end={l.end} className={({ isActive }) => (isActive ? "nav-item active" : "nav-item")}>
        {l.icon}
        {l.label}
      </NavLink>
    ));

  return (
    <div className="shell">
      <aside className="sidebar">
        <div className="brand">
          <span className="logo-mark">B</span>
          <div>
            <strong>{BANK_NAME}</strong>
            <em>Personal</em>
          </div>
        </div>
        <nav>{navItems()}</nav>
        <div className="sidebar-foot">
          <span className="who">{customer?.email}</span>
          <button type="button" className="btn btn-quiet" onClick={onLogout}>
            Sign out
          </button>
        </div>
      </aside>
      <div className="main">
        <header className="mobile-head">
          <span className="who">{customer?.email}</span>
          <button type="button" className="btn btn-quiet" onClick={onLogout}>
            Sign out
          </button>
        </header>
        <section className="content">
          <Outlet />
        </section>
      </div>
      <nav className="tabbar">{navItems()}</nav>
    </div>
  );
}
