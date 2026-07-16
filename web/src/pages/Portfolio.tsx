import { useCallback, useEffect, useRef, useState } from "react";
import { useNavigate } from "react-router-dom";
import { useAuth } from "../auth/AuthContext";
import { tokens } from "../lib/api";
import { api } from "../lib/endpoints";
import { signed, signedPct, tl } from "../lib/format";
import type {
  PortfolioOverview,
  PortfolioPerformancePoint,
  PortfolioPosition,
  PortfolioTransaction,
  PortfolioTransactionType,
} from "../lib/types";
import { useMarkets } from "../store/markets";
import { Button, Card, EmptyState } from "../components/common";

const CACHE_KEY = "tsc_portfolio_cache_v2";
const LEGACY_KEY = "tsc_holdings";
const LEGACY_DONE = "tsc_portfolio_legacy_imported_v2";
const OUTBOX_KEY = "tsc_portfolio_outbox_v1";

interface LegacyHolding {
  id: string;
  code: string;
  qty: number;
  buyPrice: number;
  addedAt?: string;
}

type TransactionDraft = Omit<
  PortfolioTransaction,
  "id" | "portfolio_id" | "user_id" | "created_at" | "updated_at"
>;

interface OutboxItem {
  portfolioId: string;
  draft: TransactionDraft;
}

const num = (value: string | number | undefined) => Number(value ?? 0);

function readJSON<T>(key: string, fallback: T): T {
  try {
    return JSON.parse(localStorage.getItem(key) || "") as T;
  } catch {
    return fallback;
  }
}

function jwtUserId() {
  try {
    const payload = tokens.access?.split(".")[1];
    if (!payload) return "";
    let normalized = payload.replace(/-/g, "+").replace(/_/g, "/");
    normalized += "=".repeat((4 - (normalized.length % 4)) % 4);
    return String(JSON.parse(atob(normalized)).uid ?? "");
  } catch {
    return "";
  }
}

function scopedKey(base: string, userId: string) {
  return `${base}:${userId}`;
}

function loadScopedCache(userId: string): PortfolioOverview | null {
  if (!userId) return null;
  const key = scopedKey(CACHE_KEY, userId);
  const current = readJSON<PortfolioOverview | null>(key, null);
  if (current) return current;
  const old = readJSON<PortfolioOverview | null>(CACHE_KEY, null);
  if (old?.portfolio.user_id !== userId) return null;
  localStorage.setItem(key, JSON.stringify(old));
  const oldOutbox = localStorage.getItem(OUTBOX_KEY);
  if (oldOutbox) localStorage.setItem(scopedKey(OUTBOX_KEY, userId), oldOutbox);
  if (localStorage.getItem(LEGACY_DONE)) {
    localStorage.setItem(scopedKey(LEGACY_DONE, userId), "1");
  }
  localStorage.removeItem(CACHE_KEY);
  localStorage.removeItem(OUTBOX_KEY);
  localStorage.removeItem(LEGACY_DONE);
  return old;
}

export function Portfolio() {
  const { profile } = useAuth();
  const navigate = useNavigate();
  const userId = profile?.id || jwtUserId();
  const cacheKey = scopedKey(CACHE_KEY, userId);
  const outboxKey = scopedKey(OUTBOX_KEY, userId);
  const legacyDoneKey = scopedKey(LEGACY_DONE, userId);
  const prices = useMarkets((state) => state.prices);
  const catalog = useMarkets((state) => state.catalog);
  const pricesRef = useRef(prices);
  pricesRef.current = prices;
  const [overview, setOverview] = useState<PortfolioOverview | null>(() => loadScopedCache(userId));
  const [performance, setPerformance] = useState<PortfolioPerformancePoint[]>([]);
  const [range, setRange] = useState("1y");
  const [loading, setLoading] = useState(!overview);
  const [error, setError] = useState("");
  const [editor, setEditor] = useState<{ code?: string; transaction?: PortfolioTransaction } | null>(null);
  const [selected, setSelected] = useState<{
    position: PortfolioPosition;
    transactions: PortfolioTransaction[];
  } | null>(null);
  const [legacyPending, setLegacyPending] = useState<LegacyHolding[]>([]);

  useEffect(() => {
    setOverview(loadScopedCache(userId));
    setPerformance([]);
    setLegacyPending([]);
  }, [userId]);

  const refresh = useCallback(async () => {
    if (!userId) return;
    setLoading(true);
    setError("");
    try {
      let list = (await api.portfolios()).portfolios;
      if (!list.length) {
        const created = await api.createPortfolio("Ana Portföy");
        list = [await api.portfolio(created.id)];
      }
      let next = list[0];
      const legacy = readJSON<LegacyHolding[]>(LEGACY_KEY, []);
      if (legacy.length && !localStorage.getItem(legacyDoneKey)) {
        if (next.positions.some((position) => num(position.quantity) > 0)) {
          setLegacyPending(legacy);
        } else {
          await importLegacy(next.portfolio.id, legacy, pricesRef.current);
          localStorage.setItem(legacyDoneKey, "1");
          localStorage.removeItem(LEGACY_KEY);
          next = await api.portfolio(next.portfolio.id);
        }
      }
      if (await flushOutbox(outboxKey)) {
        next = await api.portfolio(next.portfolio.id);
      }
      setOverview(next);
      localStorage.setItem(cacheKey, JSON.stringify(next));
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : "Portföy yüklenemedi.");
    } finally {
      setLoading(false);
    }
  }, [cacheKey, legacyDoneKey, outboxKey, userId]);

  useEffect(() => {
    void refresh();
  }, [refresh]);

  useEffect(() => {
    if (!overview) return;
    api.portfolioPerformance(overview.portfolio.id, range, 365)
      .then((result) => setPerformance(result.points))
      .catch(() => setPerformance([]));
  }, [overview?.portfolio.id, overview?.portfolio.version, range]);

  useEffect(() => {
    const online = () => void refresh();
    window.addEventListener("online", online);
    return () => window.removeEventListener("online", online);
  }, [refresh]);

  const openPosition = async (position: PortfolioPosition) => {
    if (!overview) return;
    setSelected({ position, transactions: [] });
    try {
      setSelected(await api.portfolioPosition(overview.portfolio.id, position.code));
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : "Pozisyon detayı yüklenemedi.");
    }
  };

  const saveTransaction = async (draft: TransactionDraft, transactionId?: string) => {
    if (!overview) return;
    let queued = false;
    try {
      if (transactionId) {
        await api.updatePortfolioTransaction(overview.portfolio.id, transactionId, draft);
      } else {
        await api.createPortfolioTransaction(overview.portfolio.id, draft);
      }
    } catch (reason) {
      if (!transactionId && (!navigator.onLine || reason instanceof TypeError)) {
        const outbox = readJSON<OutboxItem[]>(outboxKey, []);
        outbox.push({ portfolioId: overview.portfolio.id, draft });
        localStorage.setItem(outboxKey, JSON.stringify(outbox));
        setError("İşlem çevrimdışı kuyruğa alındı; bağlantı gelince gönderilecek.");
        queued = true;
      } else {
        throw reason;
      }
    }
    setEditor(null);
    setSelected(null);
    if (!queued) await refresh();
  };

  const deleteTransaction = async (transaction: PortfolioTransaction) => {
    if (!overview || !window.confirm("Bu işlemi silmek istediğinize emin misiniz?")) return;
    await api.deletePortfolioTransaction(overview.portfolio.id, transaction.id);
    setSelected(null);
    await refresh();
  };

  const confirmLegacy = async () => {
    if (!overview) return;
    setLoading(true);
    try {
      await importLegacy(overview.portfolio.id, legacyPending, prices);
      localStorage.setItem(legacyDoneKey, "1");
      localStorage.removeItem(LEGACY_KEY);
      setLegacyPending([]);
      await refresh();
    } finally {
      setLoading(false);
    }
  };

  if (loading && !overview) {
    return <Card style={{ padding: 24 }}>Portföy senkronize ediliyor…</Card>;
  }

  const active = overview?.positions.filter((position) => num(position.quantity) > 0) ?? [];
  const total = num(overview?.total_value_try);
  const unrealized = num(overview?.unrealized_pnl_try);
  const realized = num(overview?.realized_pnl_try);
  const day = num(overview?.day_change_try);

  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 18 }}>
      <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", gap: 12 }}>
        <div>
          <h2 style={{ margin: "0 0 4px", fontSize: 22 }}>Portföy</h2>
          <div style={{ color: "var(--text-dim)", fontSize: 13 }}>
            İşlem defteri, bulut senkronu ve TRY bazlı canlı analiz
          </div>
        </div>
        <Button variant="primary" onClick={() => setEditor({})}>+ İşlem ekle</Button>
      </div>

      {error && (
        <Card style={{ padding: 12, color: "var(--down)", fontSize: 13 }}>
          {error}
        </Card>
      )}

      {!!legacyPending.length && (
        <Card style={{ padding: 16, borderColor: "var(--accent)" }}>
          <div style={{ fontWeight: 700 }}>Cihazda {legacyPending.length} eski pozisyon bulundu</div>
          <div style={{ color: "var(--text-dim)", fontSize: 13, margin: "5px 0 12px" }}>
            Sunucudaki portföyde zaten veri var. Yinelenen kayıt oluşmaması için önizleyip içe aktarın.
          </div>
          <div style={{ display: "flex", gap: 6, flexWrap: "wrap", marginBottom: 12 }}>
            {legacyPending.map((holding) => (
              <span key={holding.id} style={{ padding: "4px 7px", borderRadius: 7, background: "var(--surface-hi)", fontSize: 11 }}>
                {holding.code} · {holding.qty.toLocaleString("tr-TR")} adet
              </span>
            ))}
          </div>
          <div style={{ display: "flex", gap: 8 }}>
            <Button variant="primary" onClick={() => void confirmLegacy()}>İçe aktar</Button>
            <Button
              variant="ghost"
              onClick={() => {
                localStorage.setItem(legacyDoneKey, "1");
                setLegacyPending([]);
              }}
            >
              Atla
            </Button>
          </div>
        </Card>
      )}

      <Card style={{ padding: 20 }}>
        <div style={{ display: "grid", gridTemplateColumns: "minmax(220px, 2fr) repeat(3, minmax(120px, 1fr))", gap: 18 }}>
          <Metric label="Toplam Değer" value={tl(total)} large />
          <Metric label="Bugün" value={`${signed(day)} ₺`} tone={day} sub={signedPct(num(overview?.day_change_pct))} />
          <Metric label="Gerçekleşmemiş K/Z" value={`${signed(unrealized)} ₺`} tone={unrealized} />
          <Metric label="Gerçekleşen K/Z" value={`${signed(realized)} ₺`} tone={realized} />
        </div>
      </Card>

      <Card style={{ padding: 18 }}>
        <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: 12 }}>
          <div style={{ fontWeight: 700 }}>Portföy performansı</div>
          <div style={{ display: "flex", gap: 4 }}>
            {["1m", "3m", "1y", "5y"].map((value) => (
              <button key={value} onClick={() => setRange(value)} style={rangeButton(range === value)}>
                {value.toUpperCase()}
              </button>
            ))}
          </div>
        </div>
        <PerformanceChart points={performance} />
      </Card>

      {active.length === 0 ? (
        <EmptyState icon="portfolio" title="Portföyün boş" hint="İlk alış işlemini ekleyerek başlayın" />
      ) : (
        <div style={{ display: "grid", gridTemplateColumns: "minmax(0, 2fr) minmax(240px, 1fr)", gap: 16 }}>
          <Card style={{ overflow: "hidden" }}>
            <div style={{ padding: "14px 16px", fontWeight: 700 }}>Pozisyonlar</div>
            {active.map((position, index) => (
              <button
                key={position.id}
                onClick={() => void openPosition(position)}
                style={{
                  width: "100%",
                  display: "flex",
                  alignItems: "center",
                  gap: 14,
                  padding: "14px 16px",
                  border: "none",
                  borderTop: index ? "1px solid var(--stroke)" : "1px solid var(--stroke)",
                  background: "transparent",
                  color: "inherit",
                  cursor: "pointer",
                  textAlign: "left",
                }}
              >
                <div style={{ flex: 1 }}>
                  <div style={{ fontWeight: 700 }}>{position.display_name}</div>
                  <div style={{ fontSize: 12, color: "var(--text-faint)", marginTop: 3 }}>
                    {num(position.quantity).toLocaleString("tr-TR")} adet · Ort. {position.currency === "USD" ? "$" : "₺"}
                    {num(position.average_cost_native).toLocaleString("tr-TR", { maximumFractionDigits: 4 })}
                  </div>
                </div>
                <div style={{ textAlign: "right" }}>
                  <div className="mono" style={{ fontWeight: 700 }}>{tl(num(position.value_try))}</div>
                  <div className="mono" style={{ color: tone(num(position.unrealized_pnl_try)), fontSize: 12 }}>
                    {signed(num(position.unrealized_pnl_try))} ₺ ({signedPct(num(position.unrealized_pnl_pct))})
                  </div>
                </div>
                <span style={{ color: "var(--text-faint)" }}>›</span>
              </button>
            ))}
          </Card>

          <Card style={{ padding: 16 }}>
            <div style={{ fontWeight: 700, marginBottom: 14 }}>Dağılım</div>
            {active.slice(0, 8).map((position) => (
              <div key={position.code} style={{ marginBottom: 12 }}>
                <div style={{ display: "flex", justifyContent: "space-between", fontSize: 12 }}>
                  <span>{position.code}</span>
                  <span className="mono">%{num(position.allocation_pct).toFixed(1)}</span>
                </div>
                <div style={{ height: 5, background: "var(--surface-hi)", borderRadius: 99, marginTop: 5 }}>
                  <div style={{ height: "100%", width: `${Math.min(100, num(position.allocation_pct))}%`, background: "var(--accent)", borderRadius: 99 }} />
                </div>
              </div>
            ))}
          </Card>
        </div>
      )}

      {editor && (
        <TransactionEditor
          codes={catalog.map((item) => ({
            code: item.code,
            name: item.display_name,
            kind: item.kind,
            currency: item.currency ?? "TRY",
          }))}
          prices={prices}
          initialCode={editor.code}
          transaction={editor.transaction}
          onClose={() => setEditor(null)}
          onSave={(draft) => saveTransaction(draft, editor.transaction?.id)}
        />
      )}

      {selected && (
        <PositionPanel
          detail={selected}
          onClose={() => setSelected(null)}
          onAsset={() => navigate(`/asset/${selected.position.code}`)}
          onAdd={() => {
            setEditor({ code: selected.position.code });
            setSelected(null);
          }}
          onEdit={(transaction) => {
            setEditor({ code: transaction.code, transaction });
            setSelected(null);
          }}
          onDelete={(transaction) => void deleteTransaction(transaction)}
        />
      )}
    </div>
  );
}

function Metric({ label, value, sub, tone: valueTone, large }: { label: string; value: string; sub?: string; tone?: number; large?: boolean }) {
  return (
    <div>
      <div style={{ color: "var(--text-dim)", fontSize: 12 }}>{label}</div>
      <div className="mono" style={{ fontSize: large ? 30 : 17, fontWeight: 800, marginTop: 5, color: valueTone === undefined ? "var(--text)" : tone(valueTone) }}>
        {value}
      </div>
      {sub && <div className="mono" style={{ fontSize: 11, color: tone(valueTone ?? 0), marginTop: 2 }}>{sub}</div>}
    </div>
  );
}

function PerformanceChart({ points }: { points: PortfolioPerformancePoint[] }) {
  const firstActive = points.findIndex(
    (point) => num(point.value_try) !== 0 || num(point.cost_basis_try) !== 0,
  );
  const plotted = firstActive > 0 ? points.slice(firstActive) : points;
  const values = plotted.map((point) => num(point.value_try));
  if (values.length < 2) {
    return <div style={{ height: 160, display: "grid", placeItems: "center", color: "var(--text-faint)", fontSize: 13 }}>Henüz yeterli performans verisi yok</div>;
  }
  const width = 800;
  const height = 170;
  const min = Math.min(...values);
  const max = Math.max(...values);
  const span = Math.max(max - min, max * 0.01, 1);
  const path = values.map((value, index) => {
    const x = (index / (values.length - 1)) * width;
    const y = height - ((value - min) / span) * (height - 18) - 9;
    return `${index ? "L" : "M"}${x.toFixed(1)},${y.toFixed(1)}`;
  }).join(" ");
  return (
    <svg viewBox={`0 0 ${width} ${height}`} preserveAspectRatio="none" style={{ width: "100%", height: 170, display: "block" }}>
      <defs>
        <linearGradient id="portfolio-fill" x1="0" y1="0" x2="0" y2="1">
          <stop offset="0" stopColor="var(--accent)" stopOpacity=".22" />
          <stop offset="1" stopColor="var(--accent)" stopOpacity="0" />
        </linearGradient>
      </defs>
      <path d={`${path} L${width},${height} L0,${height} Z`} fill="url(#portfolio-fill)" />
      <path d={path} fill="none" stroke="var(--accent)" strokeWidth="2.2" vectorEffect="non-scaling-stroke" />
    </svg>
  );
}

function PositionPanel({
  detail,
  onClose,
  onAsset,
  onAdd,
  onEdit,
  onDelete,
}: {
  detail: { position: PortfolioPosition; transactions: PortfolioTransaction[] };
  onClose: () => void;
  onAsset: () => void;
  onAdd: () => void;
  onEdit: (transaction: PortfolioTransaction) => void;
  onDelete: (transaction: PortfolioTransaction) => void;
}) {
  const position = detail.position;
  return (
    <div onClick={onClose} style={overlay}>
      <div onClick={(event) => event.stopPropagation()} style={{ ...modal, maxWidth: 680 }}>
        <div style={{ display: "flex", justifyContent: "space-between", gap: 12 }}>
          <div>
            <h3 style={{ margin: 0 }}>{position.display_name}</h3>
            <div style={{ color: "var(--text-dim)", fontSize: 12, marginTop: 4 }}>{position.code} · %{num(position.allocation_pct).toFixed(1)} portföy payı</div>
          </div>
          <button onClick={onClose} style={closeButton}>×</button>
        </div>
        <div style={{ display: "grid", gridTemplateColumns: "repeat(3, 1fr)", gap: 10, margin: "18px 0" }}>
          <Metric label="Miktar" value={num(position.quantity).toLocaleString("tr-TR")} />
          <Metric
            label="Ortalama Maliyet"
            value={`${position.currency === "USD" ? "$" : "₺"}${num(position.average_cost_native).toLocaleString("tr-TR", { maximumFractionDigits: 4 })}`}
          />
          <Metric
            label="Native Değer"
            value={`${position.currency === "USD" ? "$" : "₺"}${num(position.value_native).toLocaleString("tr-TR", { maximumFractionDigits: 2 })}`}
          />
          <Metric label="TRY Değeri" value={tl(num(position.value_try))} />
          <Metric
            label="Günlük K/Z"
            value={`${signed(num(position.day_change_try))} ₺`}
            sub={signedPct(num(position.day_change_pct))}
            tone={num(position.day_change_try)}
          />
          <Metric
            label="Toplam K/Z"
            value={`${signed(num(position.unrealized_pnl_try))} ₺`}
            sub={signedPct(num(position.unrealized_pnl_pct))}
            tone={num(position.unrealized_pnl_try)}
          />
        </div>
        <div style={{ display: "flex", gap: 8, marginBottom: 18 }}>
          <Button variant="primary" onClick={onAdd}>İşlem ekle</Button>
          <Button variant="ghost" onClick={onAsset}>Varlık detayına git</Button>
        </div>
        <div style={{ fontWeight: 700, marginBottom: 8 }}>İşlem geçmişi</div>
        <div style={{ maxHeight: 300, overflow: "auto" }}>
          {detail.transactions.map((transaction) => (
            <div key={transaction.id} style={{ display: "flex", gap: 10, alignItems: "center", padding: "10px 0", borderTop: "1px solid var(--stroke)" }}>
              <div style={{ flex: 1 }}>
                <div style={{ fontWeight: 600 }}>{transactionLabel(transaction.type)}</div>
                <div style={{ fontSize: 11, color: "var(--text-faint)" }}>
                  {new Date(transaction.occurred_at).toLocaleDateString("tr-TR")} · {num(transaction.quantity) ? `${num(transaction.quantity)} adet` : `${transaction.currency === "USD" ? "$" : "₺"}${num(transaction.amount_native)}`}
                </div>
              </div>
              <button onClick={() => onEdit(transaction)} style={actionButton}>Düzenle</button>
              <button onClick={() => onDelete(transaction)} style={{ ...actionButton, color: "var(--down)" }}>Sil</button>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}

function TransactionEditor({
  codes,
  prices,
  initialCode,
  transaction,
  onClose,
  onSave,
}: {
  codes: { code: string; name: string; kind: string; currency: string }[];
  prices: Record<string, { price: number; currency?: string }>;
  initialCode?: string;
  transaction?: PortfolioTransaction;
  onClose: () => void;
  onSave: (draft: TransactionDraft) => Promise<void>;
}) {
  const initialAsset = codes.find((item) => item.code === (transaction?.code || initialCode));
  const [type, setType] = useState<PortfolioTransactionType>(transaction?.type ?? "buy");
  const [kind, setKind] = useState(initialAsset?.kind ?? "");
  const [kindSearch, setKindSearch] = useState(initialAsset ? assetKindLabel(initialAsset.kind) : "");
  const [kindOpen, setKindOpen] = useState(false);
  const [code, setCode] = useState(transaction?.code || initialCode || "");
  const [assetSearch, setAssetSearch] = useState(initialAsset?.name ?? "");
  const [assetOpen, setAssetOpen] = useState(false);
  const [quantity, setQuantity] = useState(transaction ? String(transaction.quantity) : "");
  const [unitPrice, setUnitPrice] = useState(transaction ? String(transaction.unit_price_native) : String(prices[code]?.price ?? ""));
  const [amount, setAmount] = useState(transaction ? String(transaction.amount_native) : "");
  const [fee, setFee] = useState(transaction ? String(transaction.fee_native) : "0");
  const [note, setNote] = useState(transaction?.note ?? "");
  const [occurredDate, setOccurredDate] = useState(
    (transaction?.occurred_at ?? new Date().toISOString()).slice(0, 10),
  );
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const assetType = !["deposit", "withdraw"].includes(type);
  const tradeType = type === "buy" || type === "sell";
  const matchingKinds = ASSET_KINDS.filter((item) =>
    item.label.toLocaleLowerCase("tr-TR").includes(kindSearch.toLocaleLowerCase("tr-TR")),
  );
  const matchingAssets = codes
    .filter((item) => item.kind === kind)
    .filter((item) =>
      !assetSearch ||
      item.name.toLocaleLowerCase("tr-TR").includes(assetSearch.toLocaleLowerCase("tr-TR")) ||
      item.code.toLocaleLowerCase("tr-TR").includes(assetSearch.toLocaleLowerCase("tr-TR")),
    )
    .sort((a, b) => a.name.localeCompare(b.name, "tr"))
    .slice(0, 10);
  const decimal = (value: string) => Number(value.replace(",", "."));
  const valid =
    Boolean(occurredDate) &&
    occurredDate <= new Date().toISOString().slice(0, 10) &&
    (!assetType || Boolean(code)) &&
    (tradeType
      ? decimal(quantity) > 0 && decimal(unitPrice) > 0 && decimal(fee || "0") >= 0
      : decimal(amount) > 0);

  const submit = async () => {
    setSaving(true);
    setError("");
    try {
      const selected = codes.find((item) => item.code === code);
      await onSave({
        client_id: transaction?.client_id ?? crypto.randomUUID(),
        code: assetType ? code : "",
        type,
        quantity: tradeType ? quantity || "0" : "0",
        unit_price_native: tradeType ? unitPrice || "0" : "0",
        amount_native: tradeType ? "0" : amount || "0",
        fee_native: tradeType ? fee || "0" : "0",
        currency: assetType ? selected?.currency ?? prices[code]?.currency ?? "TRY" : "TRY",
        usdtry: "0",
        occurred_at:
          transaction && transaction.occurred_at.slice(0, 10) === occurredDate
            ? transaction.occurred_at
            : `${occurredDate}T12:00:00Z`,
        note,
      });
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : "İşlem kaydedilemedi.");
    } finally {
      setSaving(false);
    }
  };

  return (
    <div onClick={onClose} style={overlay}>
      <div onClick={(event) => event.stopPropagation()} style={modal}>
        <h3 style={{ margin: "0 0 14px" }}>{transaction ? "İşlemi düzenle" : "Yeni işlem"}</h3>
        <label style={label}>İşlem türü</label>
        <select value={type} onChange={(event) => setType(event.target.value as PortfolioTransactionType)} style={field}>
          {(["buy", "sell", "dividend", "fee", "deposit", "withdraw"] as PortfolioTransactionType[]).map((value) => (
            <option key={value} value={value}>{transactionLabel(value)}</option>
          ))}
        </select>
        {assetType && (
          <>
            <label style={label}>Varlık türü</label>
            <div style={{ position: "relative" }}>
              <input
                value={kindSearch}
                onFocus={() => setKindOpen(true)}
                onBlur={() => setTimeout(() => setKindOpen(false), 120)}
                onChange={(event) => {
                  setKindSearch(event.target.value);
                  setKind("");
                  setCode("");
                  setAssetSearch("");
                  setKindOpen(true);
                }}
                placeholder="Varlık türü ara"
                style={field}
              />
              {kindOpen && (
                <div style={pickerMenu}>
                  {matchingKinds.map((item) => (
                    <button
                      key={item.id}
                      type="button"
                      onMouseDown={(event) => event.preventDefault()}
                      onClick={() => {
                        setKind(item.id);
                        setKindSearch(item.label);
                        setCode("");
                        setAssetSearch("");
                        setKindOpen(false);
                      }}
                      style={pickerOption}
                    >
                      <span><b>{item.label}</b><small style={{ display: "block", color: "var(--text-dim)" }}>{item.subtitle}</small></span>
                      <span style={{ color: "var(--text-dim)" }}>{codes.filter((codeItem) => codeItem.kind === item.id).length}</span>
                    </button>
                  ))}
                </div>
              )}
            </div>
            <label style={label}>Varlık</label>
            <div style={{ position: "relative" }}>
              <input
                value={assetSearch}
                disabled={!kind}
                onFocus={() => setAssetOpen(true)}
                onBlur={() => setTimeout(() => setAssetOpen(false), 120)}
                onChange={(event) => {
                  setAssetSearch(event.target.value);
                  setCode("");
                  setAssetOpen(true);
                }}
                placeholder={kind ? `${assetKindLabel(kind)} ara` : "Önce varlık türünü seç"}
                style={field}
              />
              {assetOpen && kind && (
                <div style={pickerMenu}>
                  {matchingAssets.map((item) => (
                    <button
                      key={item.code}
                      type="button"
                      onMouseDown={(event) => event.preventDefault()}
                      onClick={() => {
                        setCode(item.code);
                        setAssetSearch(item.name);
                        setUnitPrice(String(prices[item.code]?.price ?? ""));
                        setAssetOpen(false);
                      }}
                      style={pickerOption}
                    >
                      <b>{item.name}</b>
                      {code === item.code && <span style={{ color: "var(--up)" }}>✓</span>}
                    </button>
                  ))}
                  {!matchingAssets.length && <div style={{ padding: 12, color: "var(--text-dim)" }}>Varlık bulunamadı.</div>}
                </div>
              )}
            </div>
          </>
        )}
        {tradeType ? (
          <>
            <label style={label}>Miktar</label>
            <input value={quantity} onChange={(event) => setQuantity(event.target.value)} inputMode="decimal" style={field} />
            <label style={label}>Birim fiyatı ({prices[code]?.currency === "USD" ? "$" : "₺"})</label>
            <input value={unitPrice} onChange={(event) => setUnitPrice(event.target.value)} inputMode="decimal" style={field} />
            <label style={label}>Ücret/komisyon</label>
            <input value={fee} onChange={(event) => setFee(event.target.value)} inputMode="decimal" style={field} />
          </>
        ) : (
          <>
            <label style={label}>Tutar</label>
            <input value={amount} onChange={(event) => setAmount(event.target.value)} inputMode="decimal" style={field} />
          </>
        )}
        <label style={label}>İşlem tarihi</label>
        <input
          type="date"
          value={occurredDate}
          max={new Date().toISOString().slice(0, 10)}
          onChange={(event) => setOccurredDate(event.target.value)}
          style={field}
        />
        <label style={label}>Not</label>
        <input value={note} onChange={(event) => setNote(event.target.value)} maxLength={500} style={field} />
        {error && <div style={{ color: "var(--down)", fontSize: 12, marginTop: 8 }}>{error}</div>}
        <div style={{ display: "flex", gap: 10, marginTop: 16 }}>
          <Button variant="ghost" full onClick={onClose}>Vazgeç</Button>
          <Button variant="primary" full disabled={saving || !valid} onClick={() => void submit()}>
            {saving ? "Kaydediliyor…" : "Kaydet"}
          </Button>
        </div>
      </div>
    </div>
  );
}

async function importLegacy(
  portfolioId: string,
  holdings: LegacyHolding[],
  prices: Record<string, { price: number; price_try?: number; currency?: string }>,
) {
  const usdtry = prices.USDTRY?.price || 1;
  for (const holding of holdings) {
    const price = prices[holding.code];
    const isUSD = price?.currency === "USD";
    const nativeCost = holding.buyPrice > 0
      ? (isUSD ? holding.buyPrice / usdtry : holding.buyPrice)
      : price?.price || 0;
    if (!(holding.qty > 0 && nativeCost > 0)) continue;
    await api.createPortfolioTransaction(portfolioId, {
      client_id: `web-legacy-v2-${holding.id}`,
      code: holding.code,
      type: "buy",
      quantity: String(holding.qty),
      unit_price_native: String(nativeCost),
      amount_native: "0",
      fee_native: "0",
      currency: isUSD ? "USD" : "TRY",
      usdtry: isUSD ? String(usdtry) : "1",
      occurred_at: holding.addedAt || new Date().toISOString(),
      note: "Eski cihaz portföyünden aktarıldı",
    });
  }
}

async function flushOutbox(key: string) {
  if (!navigator.onLine) return false;
  const outbox = readJSON<OutboxItem[]>(key, []);
  if (!outbox.length) return false;
  const remaining: OutboxItem[] = [];
  let sent = false;
  for (const item of outbox) {
    try {
      await api.createPortfolioTransaction(item.portfolioId, item.draft);
      sent = true;
    } catch {
      remaining.push(item);
    }
  }
  localStorage.setItem(key, JSON.stringify(remaining));
  return sent;
}

function transactionLabel(type: PortfolioTransactionType) {
  return {
    buy: "Alış",
    sell: "Satış",
    dividend: "Temettü",
    fee: "Ücret",
    deposit: "Para yatırma",
    withdraw: "Para çekme",
  }[type];
}

const ASSET_KINDS = [
  { id: "gold", label: "Altın", subtitle: "Gram, çeyrek, tam ve diğer altınlar" },
  { id: "currency", label: "Döviz", subtitle: "Dolar, euro ve diğer para birimleri" },
  { id: "crypto", label: "Kripto Para", subtitle: "Bitcoin, Ethereum ve diğer kriptolar" },
  { id: "stock", label: "Hisse Senedi", subtitle: "BIST ve global şirket hisseleri" },
];

function assetKindLabel(kind: string) {
  return ASSET_KINDS.find((item) => item.id === kind)?.label ?? "";
}

const tone = (value: number) => value >= 0 ? "var(--up)" : "var(--down)";
const rangeButton = (active: boolean): React.CSSProperties => ({
  border: "1px solid " + (active ? "var(--accent)" : "var(--stroke)"),
  background: active ? "color-mix(in srgb, var(--accent) 15%, transparent)" : "transparent",
  color: active ? "var(--accent)" : "var(--text-dim)",
  borderRadius: 7,
  padding: "5px 8px",
  cursor: "pointer",
  fontSize: 11,
});
const overlay: React.CSSProperties = {
  position: "fixed",
  inset: 0,
  background: "rgba(0,0,0,.68)",
  display: "grid",
  placeItems: "center",
  zIndex: 300,
  padding: 20,
};
const modal: React.CSSProperties = {
  width: "100%",
  maxWidth: 430,
  maxHeight: "90vh",
  overflow: "auto",
  background: "var(--surface)",
  border: "1px solid var(--stroke-strong)",
  borderRadius: 16,
  padding: 22,
  boxShadow: "0 24px 80px rgba(0,0,0,.45)",
};
const label: React.CSSProperties = {
  display: "block",
  color: "var(--text-dim)",
  fontSize: 12,
  margin: "10px 0 4px",
};
const field: React.CSSProperties = {
  width: "100%",
  boxSizing: "border-box",
  color: "var(--text)",
  background: "var(--surface-hi)",
  border: "1px solid var(--stroke)",
  borderRadius: 9,
  padding: 11,
};
const pickerMenu: React.CSSProperties = {
  position: "absolute",
  zIndex: 20,
  left: 0,
  right: 0,
  top: "calc(100% + 4px)",
  maxHeight: 260,
  overflowY: "auto",
  padding: 5,
  borderRadius: 10,
  border: "1px solid var(--stroke-strong)",
  background: "var(--surface)",
  boxShadow: "0 16px 36px rgba(0,0,0,.38)",
};
const pickerOption: React.CSSProperties = {
  width: "100%",
  display: "flex",
  alignItems: "center",
  justifyContent: "space-between",
  gap: 12,
  border: 0,
  borderRadius: 8,
  padding: "9px 10px",
  textAlign: "left",
  color: "var(--text)",
  background: "transparent",
  cursor: "pointer",
};
const closeButton: React.CSSProperties = {
  border: "none",
  background: "transparent",
  color: "var(--text-dim)",
  fontSize: 24,
  cursor: "pointer",
};
const actionButton: React.CSSProperties = {
  border: "none",
  background: "transparent",
  color: "var(--text-dim)",
  cursor: "pointer",
  fontSize: 12,
};
