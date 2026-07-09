// Client for the Go auth service behind the nginx load balancer — the single
// entry point of the backend (microservices/auth).

const API_URL = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

export type AuthUser = {
  id: string;
  email: string;
  user_metadata: Record<string, unknown>;
};

// Same shape as a supabase-js session: the auth service creates sessions
// through Supabase Auth, so the tokens can be handed to supabase.auth.setSession.
export type AuthSession = {
  access_token: string;
  token_type: string;
  expires_in: number;
  expires_at: number;
  refresh_token: string;
  user: AuthUser;
};

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`${API_URL}${path}`, {
    ...init,
    headers: { "Content-Type": "application/json", ...init?.headers },
  });

  const body = await res.json().catch(() => null);

  if (!res.ok) {
    throw new Error(body?.error ?? `Request failed (${res.status})`);
  }

  return body as T;
}

export function login(email: string, password: string): Promise<AuthSession> {
  return request<AuthSession>("/api/v1/auth/login", {
    method: "POST",
    body: JSON.stringify({ email, password }),
  });
}

// While email confirmation is pending the returned session has an empty
// access_token; the user object is still populated.
export function signup(name: string, email: string, password: string): Promise<AuthSession> {
  return request<AuthSession>("/api/v1/auth/signup", {
    method: "POST",
    body: JSON.stringify({ name, email, password }),
  });
}
