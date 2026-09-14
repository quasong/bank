import type { ReactNode } from "react";
import { Navigate, Route, Routes, useLocation } from "react-router-dom";
import { useAuth } from "./auth";
import { AppShell, ComingSoonPage, OverviewPage } from "./pages/AppShell";
import { LoginPage, RegisterPage } from "./pages/AuthPages";

function Guard({ children }: { children: ReactNode }) {
  const { ready, customer } = useAuth();
  const location = useLocation();
  if (!ready) return <div className="boot">Restoring session…</div>;
  if (!customer) return <Navigate to="/login" replace state={{ from: location.pathname }} />;
  return children;
}

export function App() {
  const { ready } = useAuth();
  if (!ready) return <div className="boot">Restoring session…</div>;

  return (
    <Routes>
      <Route path="/login" element={<LoginPage />} />
      <Route path="/register" element={<RegisterPage />} />
      <Route
        element={
          <Guard>
            <AppShell />
          </Guard>
        }
      >
        <Route path="/" element={<OverviewPage />} />
        <Route
          path="/accounts"
          element={
            <ComingSoonPage
              title="Accounts"
              blurb="Demand-deposit accounts and balances open after the ledger ships. This page will not show invented numbers."
            />
          }
        />
        <Route
          path="/transfers"
          element={
            <ComingSoonPage
              title="Transfers"
              blurb="Transfers need an idempotency key and debit/credit lines in one transaction. Not in phase one."
            />
          }
        />
        <Route
          path="/activity"
          element={
            <ComingSoonPage
              title="Activity"
              blurb="Statements will come from journal lines. Sign-in audit is already stored; money movement is not."
            />
          }
        />
      </Route>
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  );
}
