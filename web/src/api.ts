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

const FRIENDLY: Record<string, string> = {
  account_frozen: "This account is frozen",
  account_closed: "This account is closed",
  account_has_balance: "Withdraw the remaining balance before closing",
  insufficient_funds: "You don't have that much available",
  email_taken: "That email is already registered",
  invalid_credentials: "Incorrect email or password",
  rate_limited: "Too many attempts. Try again in a moment",
  account_locked: "This profile is locked",
  account_exists: "You already have a USD account",
  own_account: "That's your own account",
  payee_not_found: "Payee not found",
};

export function errorMessage(err: unknown, fallback: string): string {
  if (err instanceof ApiError) {
    return FRIENDLY[err.code] ?? err.message.replace(/^\w/, (c) => c.toUpperCase());
  }
  return fallback;
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

export type BankAccount = {
  id: string;
  account_number: string;
  status: string;
  balance_cents: number;
  opened_at: string;
};

export type ActivityItem = {
  journal_id: string;
  receipt?: string;
  created_at: string;
  kind: string;
  description: string;
  side: string;
  amount_cents: number;
  signed_cents: number;
  counterparty_account_number?: string;
  counterparty_name?: string;
  note?: string;
};

export type AuditEvent = {
  id: string;
  action: string;
  created_at: string;
  metadata: Record<string, string>;
};

export type Payee = {
  id: string;
  account_number: string;
  display_name: string;
  created_at: string;
  last_used_at: string;
};

export function listAccounts() {
  return request<{ accounts: BankAccount[] }>("/api/v1/accounts");
}

export function openAccount() {
  return request<{ account: BankAccount }>("/api/v1/accounts", { method: "POST" });
}

export function fundAccount(id: string, amountCents: number, idempotencyKey: string) {
  return request<{ account: BankAccount; replay: boolean }>(`/api/v1/accounts/${id}/funding`, {
    method: "POST",
    body: JSON.stringify({ amount_cents: amountCents, idempotency_key: idempotencyKey }),
  });
}

export function withdrawAccount(id: string, amountCents: number, idempotencyKey: string) {
  return request<{ account: BankAccount; replay: boolean }>(`/api/v1/accounts/${id}/withdrawals`, {
    method: "POST",
    body: JSON.stringify({ amount_cents: amountCents, idempotency_key: idempotencyKey }),
  });
}

export function freezeAccount(id: string) {
  return request<{ account: BankAccount }>(`/api/v1/accounts/${id}/freeze`, { method: "POST" });
}

export function unfreezeAccount(id: string) {
  return request<{ account: BankAccount }>(`/api/v1/accounts/${id}/unfreeze`, { method: "POST" });
}

export function closeAccount(id: string) {
  return request<{ account: BankAccount }>(`/api/v1/accounts/${id}/close`, { method: "POST" });
}

export function createTransfer(
  fromAccountId: string,
  toAccountNumber: string,
  amountCents: number,
  idempotencyKey: string,
  payeeName?: string,
  note?: string,
) {
  const body: Record<string, unknown> = {
    from_account_id: fromAccountId,
    to_account_number: toAccountNumber,
    amount_cents: amountCents,
    idempotency_key: idempotencyKey,
  };
  const name = payeeName?.trim();
  if (name) {
    body.payee_name = name.slice(0, 40);
  }
  const memo = note?.trim();
  if (memo) {
    body.note = memo.slice(0, 40);
  }
  return request<{
    transfer: {
      journal_id: string;
      receipt: string;
      replay: boolean;
      from_balance_cents: number;
      to_account_number: string;
      note?: string;
    };
  }>("/api/v1/transfers", {
    method: "POST",
    body: JSON.stringify(body),
  });
}

export function listPayees() {
  return request<{ payees: Payee[] }>("/api/v1/payees");
}

export function listActivity(accountId: string) {
  return request<{ items: ActivityItem[] }>(`/api/v1/accounts/${accountId}/activity`);
}

export function listAudit() {
  return request<{ events: AuditEvent[] }>("/api/v1/audit");
}
