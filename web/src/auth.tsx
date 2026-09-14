import { createContext, useContext, useEffect, useMemo, useState, type ReactNode } from "react";
import {
  login as loginApi,
  logout as logoutApi,
  refreshSession,
  register as registerApi,
  setAccessToken,
  type Customer,
} from "./api";

type AuthState = {
  ready: boolean;
  customer: Customer | null;
  login: (email: string, password: string) => Promise<void>;
  register: (email: string, password: string) => Promise<void>;
  logout: () => Promise<void>;
};

const AuthContext = createContext<AuthState | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [ready, setReady] = useState(false);
  const [customer, setCustomer] = useState<Customer | null>(null);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        const sess = await refreshSession();
        if (cancelled) return;
        setAccessToken(sess.access_token);
        setCustomer(sess.customer);
      } catch {
        if (cancelled) return;
        setAccessToken(null);
        setCustomer(null);
      } finally {
        if (!cancelled) setReady(true);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, []);

  const value = useMemo<AuthState>(
    () => ({
      ready,
      customer,
      login: async (email, password) => {
        const sess = await loginApi(email, password);
        setAccessToken(sess.access_token);
        setCustomer(sess.customer);
      },
      register: async (email, password) => {
        const sess = await registerApi(email, password);
        setAccessToken(sess.access_token);
        setCustomer(sess.customer);
      },
      logout: async () => {
        try {
          await logoutApi();
        } finally {
          setAccessToken(null);
          setCustomer(null);
        }
      },
    }),
    [ready, customer],
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth() {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error("useAuth must be used within AuthProvider");
  return ctx;
}
