import type { SessionResponse } from "./types";

const API_BASE = (import.meta.env.VITE_API_BASE_URL as string | undefined)?.replace(/\/$/, "") || "/api/v1";

let accessToken = "";
let refreshPromise: Promise<SessionResponse> | null = null;
let sessionExpiredHandler: (() => void) | null = null;

export class ApiError extends Error {
  status: number;

  constructor(message: string, status: number) {
    super(message);
    this.status = status;
  }
}

export function setAccessToken(value: string) {
  accessToken = value;
}

export function setSessionExpiredHandler(handler: (() => void) | null) {
  sessionExpiredHandler = handler;
}

async function parseResponse<T>(response: Response): Promise<T> {
  if (response.status === 204) return undefined as T;
  const body = await response.json().catch(() => ({}));
  if (!response.ok) {
    throw new ApiError(body.error || "İşlem tamamlanamadı", response.status);
  }
  return body as T;
}

export async function refreshSession(): Promise<SessionResponse> {
  if (!refreshPromise) {
    refreshPromise = fetch(`${API_BASE}/admin/auth/refresh`, {
      method: "POST",
      credentials: "include",
      headers: { "Content-Type": "application/json" },
    })
      .then(parseResponse<SessionResponse>)
      .then((session) => {
        setAccessToken(session.access_token);
        return session;
      })
      .finally(() => {
        refreshPromise = null;
      });
  }
  return refreshPromise;
}

export async function api<T>(
  path: string,
  init: RequestInit = {},
  retry = true,
): Promise<T> {
  const headers = new Headers(init.headers);
  if (!headers.has("Content-Type") && init.body) headers.set("Content-Type", "application/json");
  if (accessToken) headers.set("Authorization", `Bearer ${accessToken}`);
  const response = await fetch(`${API_BASE}${path}`, {
    ...init,
    headers,
    credentials: "include",
  });
  const publicAuthEndpoint = ["/admin/auth/login", "/admin/auth/refresh", "/admin/auth/logout"].includes(path);
  if (response.status === 401 && !publicAuthEndpoint) {
    if (retry) {
      try {
        await refreshSession();
      } catch (error) {
        setAccessToken("");
        sessionExpiredHandler?.();
        throw error;
      }
      return api<T>(path, init, false);
    }
    if (!retry) {
      setAccessToken("");
      sessionExpiredHandler?.();
    }
  }
  return parseResponse<T>(response);
}

export async function login(email: string, password: string) {
  const session = await api<SessionResponse>("/admin/auth/login", {
    method: "POST",
    body: JSON.stringify({ email, password }),
  });
  setAccessToken(session.access_token);
  return session;
}

export async function logout() {
  try {
    await api<void>("/admin/auth/logout", { method: "POST" }, false);
  } finally {
    setAccessToken("");
  }
}
