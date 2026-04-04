export type SessionUser = { id: string; email: string; role: string };

const TOKEN_KEY = "ridesync_token";
const USER_KEY = "ridesync_user";

export function getToken(): string | null {
  if (typeof window === "undefined") return null;
  return localStorage.getItem(TOKEN_KEY);
}

export function setSession(token: string, user: SessionUser) {
  localStorage.setItem(TOKEN_KEY, token);
  localStorage.setItem(USER_KEY, JSON.stringify(user));
}

export function getSession(): { token: string; user: SessionUser } | null {
  const token = getToken();
  const raw = typeof window !== "undefined" ? localStorage.getItem(USER_KEY) : null;
  if (!token || !raw) return null;
  try {
    return { token, user: JSON.parse(raw) as SessionUser };
  } catch {
    return null;
  }
}

export function clearSession() {
  localStorage.removeItem(TOKEN_KEY);
  localStorage.removeItem(USER_KEY);
}
