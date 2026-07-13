export interface CatalogItem {
  code: string;
  display_name: string;
  kind: string;
  unit: string;
}

export interface InstrumentPrice {
  code: string;
  display_name: string;
  kind: string;
  price: number;
  bid: number;
  ask: number;
  change: number;
  change_pct: number;
  provider?: string;
  timestamp?: string;
}

export interface Stats {
  code: string;
  open: number;
  high: number;
  low: number;
  last: number;
  change: number;
  change_pct: number;
  count: number;
}

export interface Candle {
  bucket: string;
  open: number;
  high: number;
  low: number;
  close: number;
}

export interface Profile {
  id: string;
  email: string;
  display_name: string;
  avatar_url: string;
  provider: string;
  role: string;
  created_at: string;
}

export interface Alert {
  id: string;
  code: string;
  direction: "above" | "below";
  threshold: number;
  active: boolean;
  last_fired_at?: string;
  created_at: string;
}

export interface Notification {
  id: string;
  code: string;
  title: string;
  body: string;
  price: number;
  read_at?: string;
  created_at: string;
}

export interface TokenResponse {
  access_token: string;
  refresh_token: string;
  token_type: string;
}

export type WSMessageType =
  | "price_update"
  | "instrument_update"
  | "notification"
  | "heartbeat"
  | "error"
  | "subscribed";

export interface WSEnvelope<T = unknown> {
  type: WSMessageType;
  payload?: T;
  timestamp: string;
}

export interface InstrumentUpdatePayload {
  code: string;
  display_name: string;
  kind: string;
  price: number;
  bid: number;
  ask: number;
  change: number;
  change_pct: number;
  provider?: string;
}

export interface NotificationPayload {
  id?: string;
  code: string;
  title: string;
  body: string;
  price: number;
}
