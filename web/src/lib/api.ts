import { API_BASE } from "./config";

// --- Token storage (localStorage) ------------------------------------------
const ACCESS = "tsc_access";
const REFRESH = "tsc_refresh";

export const tokens = {
  get access() {
    return localStorage.getItem(ACCESS);
  },
  get refresh() {
    return localStorage.getItem(REFRESH);
  },
  set(access: string, refresh: string) {
    localStorage.setItem(ACCESS, access);
    localStorage.setItem(REFRESH, refresh);
  },
  clear() {
    localStorage.removeItem(ACCESS);
    localStorage.removeItem(REFRESH);
  },
};

// Notify subscribers (AuthContext) when a token refresh yields a new access
// token, so long-lived consumers (WebSocket) can reconnect.
type TokenListener = (access: string | null) => void;
const listeners = new Set<TokenListener>();
export function onTokenChange(fn: TokenListener) {
  listeners.add(fn);
  return () => listeners.delete(fn);
}
function emit(access: string | null) {
  listeners.forEach((l) => l(access));
}

export class ApiError extends Error {
  status: number;
  constructor(status: number, message: string) {
    super(message);
    this.status = status;
  }
}

// Single-flight refresh so concurrent 401s trigger only one refresh call.
let refreshing: Promise<boolean> | null = null;

async function doRefresh(): Promise<boolean> {
  const rt = tokens.refresh;
  if (!rt) return false;
  if (!refreshing) {
    refreshing = (async () => {
      try {
        const res = await fetch(`${API_BASE}/auth/refresh`, {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ refresh_token: rt }),
        });
        if (!res.ok) {
          tokens.clear();
          emit(null);
          return false;
        }
        const data = await res.json();
        tokens.set(data.access_token, data.refresh_token);
        emit(data.access_token);
        return true;
      } catch {
        return false;
      } finally {
        refreshing = null;
      }
    })();
  }
  return refreshing;
}

interface RequestOpts {
  method?: string;
  body?: unknown;
  auth?: boolean;
  raw?: boolean; // return Response instead of parsed JSON
  retry?: boolean; // internal
}

export async function request<T = unknown>(path: string, opts: RequestOpts = {}): Promise<T> {
  const { method = "GET", body, auth = false, retry = true } = opts;
  const headers: Record<string, string> = {};
  if (body !== undefined) headers["Content-Type"] = "application/json";
  if (auth && tokens.access) headers["Authorization"] = `Bearer ${tokens.access}`;

  const res = await fetch(`${API_BASE}${path}`, {
    method,
    headers,
    body: body !== undefined ? JSON.stringify(body) : undefined,
  });

  if (res.status === 401 && auth && retry) {
    const ok = await doRefresh();
    if (ok) return request<T>(path, { ...opts, retry: false });
  }

  if (!res.ok) {
    let msg = `HTTP ${res.status}`;
    try {
      const j = await res.json();
      msg = j.error || j.message || msg;
    } catch {
      /* ignore */
    }
    throw new ApiError(res.status, msg);
  }

  if (res.status === 204) return undefined as T;
  return (await res.json()) as T;
}

// Multipart upload with the same auth/refresh semantics (used for avatars).
export async function uploadForm<T = unknown>(
  path: string,
  form: FormData,
  retry = true,
): Promise<T> {
  const headers: Record<string, string> = {};
  if (tokens.access) headers["Authorization"] = `Bearer ${tokens.access}`;
  const res = await fetch(`${API_BASE}${path}`, { method: "POST", headers, body: form });
  if (res.status === 401 && retry) {
    const ok = await doRefresh();
    if (ok) return uploadForm<T>(path, form, false);
  }
  if (!res.ok) {
    let msg = `HTTP ${res.status}`;
    try {
      msg = (await res.json()).error || msg;
    } catch {
      /* ignore */
    }
    throw new ApiError(res.status, msg);
  }
  return (await res.json()) as T;
}
