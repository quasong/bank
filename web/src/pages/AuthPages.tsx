import { FormEvent, type ReactNode, useState } from "react";
import { Link, Navigate, useLocation, useNavigate } from "react-router-dom";
import { ApiError } from "../api";
import { useAuth } from "../auth";
import { BANK_NAME } from "../brand";

function AuthFrame({
  title,
  lede,
  children,
}: {
  title: string;
  lede: string;
  children: ReactNode;
}) {
  return (
    <div className="auth">
      <div className="auth-inner">
        <div className="auth-brand">
          <span className="logo-mark">B</span>
          <strong>{BANK_NAME}</strong>
        </div>
        <h1>{title}</h1>
        <p className="lede">{lede}</p>
        {children}
      </div>
    </div>
  );
}

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
    <AuthFrame title="Welcome back" lede="Sign in to send money, add funds, and check your activity.">
      <form className="stack" onSubmit={onSubmit}>
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
        {error ? <p className="banner banner-error">{error}</p> : null}
        <button className="btn btn-primary btn-block" type="submit" disabled={pending}>
          {pending ? "Signing in…" : "Sign in"}
        </button>
        <p className="muted">
          New here? <Link to="/register">Create a profile</Link>
        </p>
      </form>
    </AuthFrame>
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
    <AuthFrame title="Create your profile" lede="Use an email and a password with at least 8 characters.">
      <form className="stack" onSubmit={onSubmit}>
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
            autoComplete="new-password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            required
            minLength={8}
          />
        </label>
        {error ? <p className="banner banner-error">{error}</p> : null}
        <button className="btn btn-primary btn-block" type="submit" disabled={pending}>
          {pending ? "Creating profile…" : "Continue"}
        </button>
        <p className="muted">
          Already registered? <Link to="/login">Sign in</Link>
        </p>
      </form>
    </AuthFrame>
  );
}
