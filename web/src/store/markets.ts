import { create } from "zustand";
import { wsURL } from "../lib/config";
import { api } from "../lib/endpoints";
import type {
  CatalogItem,
  InstrumentPrice,
  InstrumentUpdatePayload,
  NotificationPayload,
  WSEnvelope,
} from "../lib/types";

const SPARK_LEN = 32;

type Flash = "up" | "down" | undefined;

interface MarketsState {
  catalog: CatalogItem[];
  prices: Record<string, InstrumentPrice>;
  spark: Record<string, number[]>;
  flash: Record<string, Flash>;
  connected: boolean;
  order: string[]; // stable catalog order
  bootstrap: () => Promise<void>;
  connect: (token?: string | null) => void;
  disconnect: () => void;
  onNotification?: (n: NotificationPayload) => void;
  setNotificationHandler: (fn: (n: NotificationPayload) => void) => void;
}

let ws: WebSocket | null = null;
let reconnectTimer: ReturnType<typeof setTimeout> | null = null;
let backoff = 1000;
let currentToken: string | null | undefined;
let flashTimers: Record<string, ReturnType<typeof setTimeout>> = {};

export const useMarkets = create<MarketsState>((set, get) => ({
  catalog: [],
  prices: {},
  spark: {},
  flash: {},
  connected: false,
  order: [],

  setNotificationHandler(fn) {
    set({ onNotification: fn });
  },

  async bootstrap() {
    try {
      const [cat, pr, sp] = await Promise.all([
        api.catalog(),
        api.prices().catch(() => ({ prices: [] as InstrumentPrice[], count: 0 })),
        api.sparklines(SPARK_LEN).catch(() => ({ points: {} as Record<string, number[]> })),
      ]);
      const prices: Record<string, InstrumentPrice> = {};
      for (const p of pr.prices) prices[p.code] = p;
      const spark: Record<string, number[]> = {};
      for (const [code, arr] of Object.entries(sp.points)) spark[code] = arr.slice(-SPARK_LEN);
      set({
        catalog: cat.instruments,
        order: cat.instruments.map((c) => c.code),
        prices,
        spark,
      });
    } catch {
      /* offline bootstrap tolerated; WS will fill in */
    }
  },

  connect(token) {
    currentToken = token;
    if (ws && (ws.readyState === WebSocket.OPEN || ws.readyState === WebSocket.CONNECTING)) {
      // Reconnect with the (possibly new) token.
      try {
        ws.close();
      } catch {
        /* ignore */
      }
    }
    openSocket(set, get);
  },

  disconnect() {
    if (reconnectTimer) clearTimeout(reconnectTimer);
    reconnectTimer = null;
    if (ws) {
      ws.onclose = null;
      try {
        ws.close();
      } catch {
        /* ignore */
      }
      ws = null;
    }
    set({ connected: false });
  },
}));

function openSocket(
  set: (partial: Partial<MarketsState>) => void,
  get: () => MarketsState,
) {
  ws = new WebSocket(wsURL(currentToken));

  ws.onopen = () => {
    backoff = 1000;
    set({ connected: true });
  };

  ws.onmessage = (ev) => {
    let msg: WSEnvelope;
    try {
      msg = JSON.parse(ev.data);
    } catch {
      return;
    }
    if (msg.type === "instrument_update") {
      const p = msg.payload as InstrumentUpdatePayload;
      const st = get();
      const prev = st.prices[p.code];
      const dir: Flash = !prev ? undefined : p.price > prev.price ? "up" : p.price < prev.price ? "down" : undefined;
      const sparkArr = (st.spark[p.code] ?? []).concat(p.price).slice(-SPARK_LEN);
      set({
        prices: { ...st.prices, [p.code]: { ...(prev ?? {}), ...p } },
        spark: { ...st.spark, [p.code]: sparkArr },
        flash: dir ? { ...st.flash, [p.code]: dir } : st.flash,
      });
      if (dir) {
        if (flashTimers[p.code]) clearTimeout(flashTimers[p.code]);
        flashTimers[p.code] = setTimeout(() => {
          const s = get();
          set({ flash: { ...s.flash, [p.code]: undefined } });
        }, 650);
      }
    } else if (msg.type === "notification") {
      get().onNotification?.(msg.payload as NotificationPayload);
    }
  };

  ws.onclose = () => {
    set({ connected: false });
    if (reconnectTimer) clearTimeout(reconnectTimer);
    reconnectTimer = setTimeout(() => openSocket(set, get), backoff);
    backoff = Math.min(backoff * 1.8, 15000);
  };

  ws.onerror = () => {
    try {
      ws?.close();
    } catch {
      /* ignore */
    }
  };
}
