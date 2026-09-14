import { FormEvent, useState } from "react";
import { Link, Navigate, useLocation, useNavigate } from "react-router-dom";
import { ApiError } from "../api";
import { useAuth } from "../auth";
import { BANK_NAME } from "../brand";

export function LoginPage() {
  const { customer, login } = useAuth();
  const navigate = useNavigate();
  const location = useLocation();
  const from = (location.state as { from?: string } | null)?.from ?? "/";
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [pending, setPending] = useState(false);

  if (customer) return <Navigate to={from} replace />;

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    setError("");
    setPending(true);
    try {
      await login(email, password);
      navigate(from, { replace: true });
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Sign in failed");
    } finally {
      setPending(false);
    }
  }

  return (
    <div className="gate">
      <aside className="gate-brand">
        <p className="eyebrow">Internet banking</p>
        <h1>{BANK_NAME}</h1>
        <p className="lede">
          Demo internet bank. Phase one is identity only; accounts, transfers, and
          double-entry posting come later. No fake balances.
        </p>
      </aside>
      <main className="gate-panel">
        <form className="card" onSubmit={onSubmit}>
          <h2>Sign in</h2>
          <label>
            Email
            <input
              type="email"
              autoComplete="username"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              required
            />
          </label>
          <label>
            Password
            <input
              type="password"
              autoComplete="current-password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              required
              minLength={8}
            />
          </label>
          {error ? <p className="error">{error}</p> : null}
          <button type="submit" disabled={pending}>
            {pending ? "Signing in…" : "Sign in"}
          </button>
          <p className="muted">
            No customer profile yet? <Link to="/register">Register</Link>
          </p>
        </form>
      </main>
    </div>
  );
}

export function RegisterPage() {
  const { customer, register } = useAuth();
  const navigate = useNavigate();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [pending, setPending] = useState(false);

  if (customer) return <Navigate to="/" replace />;

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    setError("");
    setPending(true);
    try {
      await register(email, password);
      navigate("/", { replace: true });
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Registration failed");
    } finally {
      setPending(false);
    }
  }

  return (
    <div className="gate">
      <aside className="gate-brand">
        <p className="eyebrow">Internet banking</p>
        <h1>Create your profile</h1>
        <p className="lede">
          Registration creates a customer identity, not a deposit account. Passwords
          are stored with Argon2id.
        </p>
      </aside>
      <main className="gate-panel">
        <form className="card" onSubmit={onSubmit}>
          <h2>Register</h2>
          <label>
            Email
            <input
              type="email"
              autoComplete="username"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              required
            />
          </label>
          <label>
            Password (at least 8 characters)
            <input
              type="password"
              autoComplete="new-password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              required
              minLength={8}
            />
          </label>
          {error ? <p className="error">{error}</p> : null}
          <button type="submit" disabled={pending}>
            {pending ? "Creating profile…" : "Register"}
          </button>
          <p className="muted">
            Already registered? <Link to="/login">Sign in</Link>
          </p>
        </form>
      </main>
    </div>
  );
}
