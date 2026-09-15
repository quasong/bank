import type { ReactNode } from "react";
import { Navigate, Route, Routes, useLocation } from "react-router-dom";
import { useAuth } from "./auth";
import { AppShell } from "./pages/AppShell";
import { OverviewPage } from "./pages/OverviewPage";
import { AccountsPage } from "./pages/AccountsPage";
import { TransfersPage } from "./pages/TransfersPage";
import { ConvertPage } from "./pages/ConvertPage";
import { ActivityPage } from "./pages/ActivityPage";
import { LoginPage, RegisterPage } from "./pages/AuthPages";
import { BootScreen } from "./ui";

function Guard({ children }: { children: ReactNode }) {
  const { ready, customer } = useAuth();
  const location = useLocation();
  if (!ready) return <BootScreen />;
  if (!customer) return <Navigate to="/login" replace state={{ from: location.pathname }} />;
  return children;
}

export function App() {
  const { ready } = useAuth();
  if (!ready) return <BootScreen />;

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
        <Route path="/accounts" element={<AccountsPage />} />
        <Route path="/transfers" element={<TransfersPage />} />
        <Route path="/convert" element={<ConvertPage />} />
        <Route path="/activity" element={<ActivityPage />} />
      </Route>
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  );
}
