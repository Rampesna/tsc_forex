import { ApiError, request, uploadForm } from "./api";
import { WEB_ABUSE_ID_KEY } from "./privacy";
import type {
  Alert,
  BillingPlansResponse,
  BillingStatus,
  Candle,
  CatalogItem,
  GamCheckInResult,
  GamificationExperience,
  GamificationPrivacy,
  GamProfile,
  InstrumentDetail,
  InstrumentPrice,
  MarketInsight,
  Leaderboard,
  Notification,
  PortfolioOverview,
  PortfolioHealth,
  PortfolioPerformancePoint,
  PortfolioTransaction,
  PaperTrade,
  Profile,
  Stats,
  StudyGroup,
  TokenResponse,
  QuizAttemptResult,
} from "./types";

export const api = {
  // --- auth ---
  register: async (email: string, password: string) => {
    await request("/auth/register", { method: "POST", body: { email, password } });
    return request<TokenResponse>("/auth/login", { method: "POST", body: { email, password } });
  },
  login: (email: string, password: string) =>
    request<TokenResponse>("/auth/login", { method: "POST", body: { email, password } }),
  google: (idToken: string) =>
    request<TokenResponse>("/auth/google", { method: "POST", body: { id_token: idToken } }),
  apple: (idToken: string, fullName = "") =>
    request<TokenResponse>("/auth/apple", {
      method: "POST",
      body: { id_token: idToken, full_name: fullName },
    }),
  logout: (refresh: string) =>
    request("/auth/logout", { method: "POST", body: { refresh_token: refresh } }),

  // --- market data ---
  catalog: () => request<{ count: number; instruments: CatalogItem[] }>("/instruments"),
  prices: () => request<{ count: number; prices: InstrumentPrice[] }>("/instruments/prices"),
  sparklines: (points = 24) =>
    request<{ points: Record<string, number[]> }>(`/instruments/sparklines?points=${points}`),
  instrument: (code: string) => request<InstrumentPrice>(`/instruments/${code}`),
  stats: (code: string) => request<Stats>(`/instruments/${code}/stats`),
  candles: (code: string, interval: string, opts?: { fromMs?: number; toMs?: number; limit?: number }) => {
    let url = `/instruments/${code}/history?interval=${interval}`;
    if (opts?.fromMs && opts.fromMs > 0) url += `&from=${new Date(opts.fromMs).toISOString()}`;
    if (opts?.toMs && opts.toMs > 0) url += `&to=${new Date(opts.toMs).toISOString()}`;
    if (opts?.limit && opts.limit > 0) url += `&limit=${opts.limit}`;
    return request<{ code: string; interval: string; candles: Candle[] }>(url);
  },
  detail: (code: string) => request<InstrumentDetail>(`/instruments/${code}/detail`),
  marketInsight: async (code: string) => {
    const deviceId = stableDeviceID();
    const pendingKey = `ai-research:${code}`;
    const requestId = pendingRequestID(pendingKey);
    let jobId = "";
    try {
      for (let attempt = 0; attempt < 120; attempt += 1) {
        const result = await request<MarketInsight | { status: string; job_id: string }>("/me/ai/market-insight", {
          method: "POST",
          auth: true,
          body: jobId
            ? { code, job_id: jobId, device_id: deviceId, request_id: requestId }
            : { code, device_id: deviceId, request_id: requestId },
        });
        if (!("job_id" in result)) {
          clearPendingRequestID(pendingKey);
          return result;
        }
        jobId = result.job_id;
        await new Promise((resolve) => window.setTimeout(resolve, 3000));
      }
    } catch (error) {
      if (isTerminalOperationError(error)) clearPendingRequestID(pendingKey);
      throw error;
    }
    throw new Error("Derin araştırma beklenenden uzun sürdü; biraz sonra yeniden deneyin.");
  },

  // --- server-verified App Store entitlement (web is read-only) ---
  billingStatus: () => request<BillingStatus>("/me/billing/status", { auth: true }),
  billingPlans: () => request<BillingPlansResponse>("/billing/plans"),
  researchHistory: () => request<{ items: Array<{ id: string; code: string; payload: MarketInsight; generated_at: string; expires_at: string }> }>("/me/ai/research-history", { auth: true }),

  // --- profile ---
  profile: () => request<Profile>("/me/profile", { auth: true }),
  updateProfile: (displayName: string) =>
    request<Profile>("/me/profile", {
      method: "PATCH",
      auth: true,
      body: { display_name: displayName },
    }),
  uploadAvatar: (file: File) => {
    const fd = new FormData();
    fd.append("avatar", file);
    return uploadForm<{ avatar_url: string }>("/me/avatar", fd);
  },

  // --- watchlist ---
  watchlist: () => request<{ count: number; codes: string[] }>("/me/watchlist", { auth: true }),
  addWatch: (code: string) =>
    request("/me/watchlist", { method: "POST", auth: true, body: { code } }),
  removeWatch: (code: string) =>
    request(`/me/watchlist/${code}`, { method: "DELETE", auth: true }),

  // --- alerts ---
  alerts: () => request<{ count: number; alerts: Alert[] }>("/me/alerts", { auth: true }),
  createAlert: (code: string, direction: "above" | "below", threshold: number) =>
    request<Alert>("/me/alerts", { method: "POST", auth: true, body: { code, direction, threshold } }),
  deleteAlert: (id: string) => request(`/me/alerts/${id}`, { method: "DELETE", auth: true }),

  // --- notifications ---
  notifications: () =>
    request<{ count: number; notifications: Notification[] }>("/me/notifications", { auth: true }),
  markRead: (id: string) => request(`/me/notifications/${id}/read`, { method: "POST", auth: true }),
  deleteNotification: (id: string) =>
    request(`/me/notifications/${id}`, { method: "DELETE", auth: true }),
  clearNotifications: () => request("/me/notifications", { method: "DELETE", auth: true }),

  // --- account ---
  deleteAccount: () => request<{ deleted: boolean }>("/me/account", { method: "DELETE", auth: true }),

  // --- gamification ---
  gamification: () => request<GamProfile>("/me/gamification", { auth: true }),
  checkIn: () =>
    request<GamCheckInResult>("/me/gamification/checkin", { method: "POST", auth: true }),
  leaderboard: (limit = 20) =>
    request<Leaderboard>(`/me/gamification/leaderboard?limit=${limit}`, { auth: true }),
  gamificationExperience: () =>
    request<GamificationExperience>("/me/gamification/experience", { auth: true }),
  saveDailyPlan: (body: {
    risk_appetite: "low" | "moderate" | "high";
    watch_codes: string[];
    learning_goal: string;
    reflection: string;
  }) =>
    request<GamCheckInResult>("/me/gamification/daily-plan", {
      method: "POST",
      auth: true,
      body,
    }),
  submitQuiz: (body: {
    lesson_code: string;
    question_code: string;
    answer: number;
    idempotency_key: string;
  }) =>
    request<QuizAttemptResult>("/me/gamification/quiz-attempts", {
      method: "POST",
      auth: true,
      body,
    }),
  completeDiscipline: (code: string, note: string) =>
    request<{ awarded: boolean; xp_gained: number; rule_version: string }>(
      `/me/gamification/discipline/${code}`,
      { method: "POST", auth: true, body: { note } },
    ),
  season: () => request<GamificationExperience["season"]>("/me/gamification/season", { auth: true }),
  createPaperTrade: (body: Omit<PaperTrade, "id" | "followed_plan" | "retrospective" | "process_score" | "reviewed_at" | "created_at" | "updated_at">) =>
    request<{ created: boolean; trade: PaperTrade }>("/me/gamification/paper-trades", {
      method: "POST",
      auth: true,
      body,
    }),
  reviewPaperTrade: (id: string, followedPlan: boolean, retrospective: string) =>
    request<{ trade: PaperTrade; xp_gained: number }>(
      `/me/gamification/paper-trades/${id}/review`,
      { method: "PATCH", auth: true, body: { followed_plan: followedPlan, retrospective } },
    ),
  saveGamificationPrivacy: (body: GamificationPrivacy) =>
    request<GamificationPrivacy>("/me/gamification/privacy", {
      method: "PATCH",
      auth: true,
      body,
    }),
  createStudyGroup: (name: string) =>
    request<StudyGroup>("/me/gamification/study-groups", {
      method: "POST",
      auth: true,
      body: { name },
    }),
  joinStudyGroup: (inviteCode: string) =>
    request<{ joined: boolean }>("/me/gamification/study-groups/join", {
      method: "POST",
      auth: true,
      body: { invite_code: inviteCode },
    }),

  // --- portfolio ledger ---
  portfolios: () =>
    request<{ count: number; portfolios: PortfolioOverview[] }>("/me/portfolios", { auth: true }),
  createPortfolio: (name: string) =>
    request<PortfolioOverview["portfolio"]>("/me/portfolios", {
      method: "POST",
      auth: true,
      body: { name },
    }),
  portfolio: (id: string) =>
    request<PortfolioOverview>(`/me/portfolios/${id}`, { auth: true }),
  updatePortfolio: (id: string, name: string) =>
    request<PortfolioOverview["portfolio"]>(`/me/portfolios/${id}`, {
      method: "PATCH",
      auth: true,
      body: { name },
    }),
  deletePortfolio: (id: string) =>
    request(`/me/portfolios/${id}`, { method: "DELETE", auth: true }),
  portfolioPosition: (id: string, code: string) =>
    request<{ position: PortfolioOverview["positions"][number]; transactions: PortfolioTransaction[] }>(
      `/me/portfolios/${id}/positions/${encodeURIComponent(code)}`,
      { auth: true },
    ),
  portfolioTransactions: (id: string, limit = 100) =>
    request<{ count: number; transactions: PortfolioTransaction[] }>(
      `/me/portfolios/${id}/transactions?limit=${limit}`,
      { auth: true },
    ),
  createPortfolioTransaction: (
    id: string,
    body: Omit<PortfolioTransaction, "id" | "portfolio_id" | "user_id" | "created_at" | "updated_at">,
  ) =>
    request<{ created: boolean; transaction: PortfolioTransaction }>(
      `/me/portfolios/${id}/transactions`,
      { method: "POST", auth: true, body },
    ),
  updatePortfolioTransaction: (
    id: string,
    transactionId: string,
    body: Omit<PortfolioTransaction, "id" | "portfolio_id" | "user_id" | "created_at" | "updated_at">,
  ) =>
    request<PortfolioTransaction>(`/me/portfolios/${id}/transactions/${transactionId}`, {
      method: "PATCH",
      auth: true,
      body,
    }),
  deletePortfolioTransaction: (id: string, transactionId: string) =>
    request(`/me/portfolios/${id}/transactions/${transactionId}`, {
      method: "DELETE",
      auth: true,
    }),
  portfolioPerformance: (id: string, range = "1y", points = 365) =>
    request<{ count: number; points: PortfolioPerformancePoint[] }>(
      `/me/portfolios/${id}/performance?range=${range}&points=${points}`,
      { auth: true },
    ),
  portfolioHealth: async (id: string) => {
    const pendingKey = `portfolio-health:${id}`;
    try {
      const result = await request<PortfolioHealth>(`/me/portfolios/${id}/health-check`, {
        method: "POST", auth: true, body: { request_id: pendingRequestID(pendingKey) },
      });
      clearPendingRequestID(pendingKey);
      return result;
    } catch (error) {
      if (isTerminalOperationError(error)) clearPendingRequestID(pendingKey);
      throw error;
    }
  },
  exportPortfolio: async (id: string) => {
    const pendingKey = `portfolio-export:${id}`;
    try {
      const result = await request<Response>(`/me/portfolios/${id}/export`, {
        method: "POST", auth: true, raw: true, body: { request_id: pendingRequestID(pendingKey) },
      });
      clearPendingRequestID(pendingKey);
      return result;
    } catch (error) {
      if (isTerminalOperationError(error)) clearPendingRequestID(pendingKey);
      throw error;
    }
  },
};

function makeRequestID(): string {
  return globalThis.crypto?.randomUUID?.() ?? `web-${Date.now()}-${Math.random().toString(36).slice(2)}`;
}

const PENDING_REQUEST_PREFIX = "tsc_pending_operation:";

function pendingRequestID(scope: string): string {
  const key = `${PENDING_REQUEST_PREFIX}${scope}`;
  const existing = localStorage.getItem(key);
  if (existing) return existing;
  const created = makeRequestID();
  localStorage.setItem(key, created);
  return created;
}

function clearPendingRequestID(scope: string) {
  localStorage.removeItem(`${PENDING_REQUEST_PREFIX}${scope}`);
}

function isTerminalOperationError(error: unknown): boolean {
  if (!(error instanceof ApiError)) return false;
  if (error.status === 402) return true;
  if ([400, 401, 403, 404].includes(error.status)) return true;
  return error.code === "quota_exhausted" || error.code === "subscription_required";
}

function stableDeviceID(): string {
  const existing = localStorage.getItem(WEB_ABUSE_ID_KEY);
  if (existing) return existing;
  const created = `web-${makeRequestID()}`;
  localStorage.setItem(WEB_ABUSE_ID_KEY, created);
  return created;
}
