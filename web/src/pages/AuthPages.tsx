import { FormEvent, type ReactNode, useState } from "react";
import { Link, Navigate, useLocation, useNavigate } from "react-router-dom";
import { errorMessage } from "../api";
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
  const [showPassword, setShowPassword] = useState(false);

  if (customer) return <Navigate to={from} replace />;

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    setError("");
    setPending(true);
    try {
      await login(email, password);
      navigate(from, { replace: true });
    } catch (err) {
      setError(errorMessage(err, "Sign in failed"));
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
            autoFocus
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            required
          />
        </label>
        <label>
          Password
          <span className="pw">
            <input
              type={showPassword ? "text" : "password"}
              autoComplete="current-password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              required
              minLength={8}
            />
            <button type="button" className="text-link" onClick={() => setShowPassword((v) => !v)}>
              {showPassword ? "Hide" : "Show"}
            </button>
          </span>
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
  const [showPassword, setShowPassword] = useState(false);

  if (customer) return <Navigate to="/" replace />;

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    setError("");
    setPending(true);
    try {
      await register(email, password);
      navigate("/", { replace: true });
    } catch (err) {
      setError(errorMessage(err, "Registration failed"));
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
            autoFocus
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            required
          />
        </label>
        <label>
          Password
          <span className="pw">
            <input
              type={showPassword ? "text" : "password"}
              autoComplete="new-password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              required
              minLength={8}
            />
            <button type="button" className="text-link" onClick={() => setShowPassword((v) => !v)}>
              {showPassword ? "Hide" : "Show"}
            </button>
          </span>
          {password.length > 0 && password.length < 8 ? (
            <span className="avail">{8 - password.length} more characters</span>
          ) : null}
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
