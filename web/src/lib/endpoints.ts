import { request, uploadForm } from "./api";
import type {
  Alert,
  Candle,
  CatalogItem,
  GamCheckInResult,
  GamProfile,
  InstrumentDetail,
  InstrumentPrice,
  Leaderboard,
  Notification,
  Profile,
  Stats,
  TokenResponse,
} from "./types";

export const api = {
  // --- auth ---
  register: (email: string, password: string) =>
    request<TokenResponse>("/auth/register", { method: "POST", body: { email, password } }),
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
  candles: (code: string, interval: string, fromMs?: number) => {
    let url = `/instruments/${code}/history?interval=${interval}`;
    if (fromMs && fromMs > 0) url += `&from=${new Date(fromMs).toISOString()}`;
    return request<{ code: string; interval: string; candles: Candle[] }>(url);
  },
  detail: (code: string) => request<InstrumentDetail>(`/instruments/${code}/detail`),

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
};
