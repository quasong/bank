export type Customer = {
  id: string;
  email: string;
  status: string;
  created_at: string;
};

export type Session = {
  access_token: string;
  token_type: string;
  expires_in: number;
  customer: Customer;
};

export class ApiError extends Error {
  code: string;
  status: number;

  constructor(code: string, message: string, status: number) {
    super(message);
    this.code = code;
    this.status = status;
  }
}

let accessToken: string | null = null;

export function setAccessToken(token: string | null) {
  accessToken = token;
}

export function getAccessToken() {
  return accessToken;
}

async function request<T>(path: string, init: RequestInit = {}): Promise<T> {
  const headers = new Headers(init.headers);
  if (init.body && !headers.has("Content-Type")) {
    headers.set("Content-Type", "application/json");
  }
  if (accessToken) {
    headers.set("Authorization", `Bearer ${accessToken}`);
  }
  const res = await fetch(path, { ...init, headers, credentials: "include" });
  if (res.status === 204) {
    return undefined as T;
  }
  const text = await res.text();
  const data = text ? JSON.parse(text) : null;
  if (!res.ok) {
    const code = data?.error?.code ?? "internal";
    const message = data?.error?.message ?? "request failed";
    throw new ApiError(code, message, res.status);
  }
  return data as T;
}

export function register(email: string, password: string) {
  return request<Session>("/api/v1/auth/register", {
    method: "POST",
    body: JSON.stringify({ email, password }),
  });
}

export function login(email: string, password: string) {
  return request<Session>("/api/v1/auth/login", {
    method: "POST",
    body: JSON.stringify({ email, password }),
  });
}

let refreshInFlight: Promise<Session> | null = null;

export function refreshSession() {
  if (!refreshInFlight) {
    refreshInFlight = request<Session>("/api/v1/auth/refresh", { method: "POST" }).finally(() => {
      refreshInFlight = null;
    });
  }
  return refreshInFlight;
}

export function logout() {
  return request<void>("/api/v1/auth/logout", { method: "POST" });
}

export function fetchMe() {
  return request<{ customer: Customer }>("/api/v1/me");
}
