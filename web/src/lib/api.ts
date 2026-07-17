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
  code?: string;
  feature?: string;
  limit?: number;
  used?: number;

  constructor(status: number, message: string, details: ApiErrorDetails = {}) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.code = details.code;
    this.feature = details.feature;
    this.limit = details.limit;
    this.used = details.used;
  }
}

interface ApiErrorDetails {
  code?: string;
  feature?: string;
  limit?: number;
  used?: number;
}

interface ApiErrorBody extends ApiErrorDetails {
  error?: string;
  message?: string;
}

async function responseError(response: Response): Promise<ApiError> {
  let body: ApiErrorBody = {};
  try {
    body = (await response.json()) as ApiErrorBody;
  } catch {
    // Some upstream/proxy errors intentionally have no JSON response body.
  }
  return new ApiError(
    response.status,
    body.error || body.message || `HTTP ${response.status}`,
    {
      code: typeof body.code === "string" ? body.code : undefined,
      feature: typeof body.feature === "string" ? body.feature : undefined,
      limit: typeof body.limit === "number" ? body.limit : undefined,
      used: typeof body.used === "number" ? body.used : undefined,
    },
  );
}

const FEATURE_LABELS: Record<string, string> = {
  ai_research: "Kaynaklı analiz",
  active_alerts: "Aktif alarm",
  watchlist: "Favori",
  portfolios: "Portföy",
  portfolio_health: "Portföy sağlık analizi",
  exports: "Portföy dışa aktarma",
};

export function describeApiError(error: unknown, fallback: string): string {
  if (!(error instanceof ApiError)) return error instanceof Error ? error.message : fallback;
  if (error.status === 429) {
    return "Çok fazla istek gönderildi. Kısa bir süre bekleyip aynı işlemi yeniden dene; hakkın yeniden tüketilmez.";
  }
  if (error.status === 409 && error.code === "operation_in_progress") {
    return "İşlem sunucuda devam ediyor. Biraz sonra aynı işlem üzerinden yeniden dene.";
  }
  if (error.status !== 402) return error.message || fallback;
  const label = FEATURE_LABELS[error.feature || ""] || "Bu özellik";
  if (error.code === "subscription_required") {
    return `${label} için ücretli plan gerekir. Plan ayrıntılarını Ayarlar'dan inceleyebilirsin.`;
  }
  if (error.code === "quota_exhausted") {
    const usage = typeof error.limit === "number" && typeof error.used === "number"
      ? ` (${error.used}/${error.limit})`
      : "";
    return `${label} hakkın doldu${usage}. Planını ve kota dönemini Ayarlar'dan inceleyebilirsin.`;
  }
  return error.message || fallback;
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
  const { method = "GET", body, auth = false, raw = false, retry = true } = opts;
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
    throw await responseError(res);
  }

  if (raw) return res as T;
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
    throw await responseError(res);
  }
  return (await res.json()) as T;
}
