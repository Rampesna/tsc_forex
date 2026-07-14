import { useState } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { api } from "../lib/endpoints";
import { useAuth } from "../auth/AuthContext";
import { useMarkets } from "../store/markets";
import { compact, dateShort, price as priceFmt, signed, tl, usd, usdCompact } from "../lib/format";
import { Button, Card, ChangePill } from "../components/common";
import { Icon } from "../components/Icon";
import { PriceChart, type ChartType } from "../components/PriceChart";
import type { InstrumentDetail } from "../lib/types";

const INTERVALS = [
  { k: "1m", l: "1DK" },
  { k: "5m", l: "5DK" },
  { k: "15m", l: "15DK" },
  { k: "1h", l: "1SA" },
  { k: "1d", l: "1G" },
];

const CHART_TYPES: { k: ChartType; l: string }[] = [
  { k: "candles", l: "Mum" },
  { k: "line", l: "Çizgi" },
  { k: "area", l: "Alan" },
];

export function AssetDetail() {
  const { code = "" } = useParams();
  const nav = useNavigate();
  const qc = useQueryClient();
  const { authed } = useAuth();
  const live = useMarkets((s) => s.prices[code]);
  const [interval, setInterval] = useState("5m");
  const [chartType, setChartType] = useState<ChartType>("candles");
  const [alertOpen, setAlertOpen] = useState(false);

  const stats = useQuery({ queryKey: ["stats", code], queryFn: () => api.stats(code), refetchInterval: 15000 });
  const candles = useQuery({ queryKey: ["candles", code, interval], queryFn: () => api.candles(code, interval) });
  const detail = useQuery({ queryKey: ["detail", code], queryFn: () => api.detail(code), refetchInterval: 30000 });
  const watch = useQuery({ queryKey: ["watchlist"], queryFn: () => api.watchlist(), enabled: authed });
  const isFav = watch.data?.codes?.includes(code) ?? false;

  const toggleFav = useMutation({
    mutationFn: () => (isFav ? api.removeWatch(code) : api.addWatch(code)),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["watchlist"] }),
  });

  const p = live?.price ?? stats.data?.last ?? 0;
  const changePct = live?.change_pct ?? stats.data?.change_pct ?? 0;
  const change = live?.change ?? stats.data?.change ?? 0;
  const name = live?.display_name ?? detail.data?.long_name ?? code;

  const requireAuth = (fn: () => void) => () => (authed ? fn() : nav("/login"));

  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 18 }}>
      <button onClick={() => nav(-1)} style={backBtn}>
        <Icon name="arrow-left" size={16} /> Geri
      </button>

      <div style={{ display: "flex", alignItems: "flex-start", justifyContent: "space-between", flexWrap: "wrap", gap: 12 }}>
        <div>
          <h2 style={{ margin: "0 0 6px", fontSize: 20, fontWeight: 600, letterSpacing: "-0.02em" }}>{name}</h2>
          <div style={{ display: "flex", alignItems: "baseline", gap: 12 }}>
            <span className="mono" style={{ fontSize: 28, fontWeight: 700 }}>{tl(p)}</span>
            <ChangePill pct={changePct} />
            <span className="mono" style={{ color: change >= 0 ? "var(--up)" : "var(--down)", fontSize: 13.5 }}>
              {signed(change)}
            </span>
          </div>
        </div>
        <div style={{ display: "flex", gap: 8 }}>
          <button
            onClick={requireAuth(() => toggleFav.mutate())}
            title={isFav ? "Favoriden çıkar" : "Favori ekle"}
            style={{ ...iconBtn, color: isFav ? "var(--accent-hi)" : "var(--text-dim)", borderColor: isFav ? "var(--stroke-strong)" : "var(--stroke)" }}
          >
            <Icon name={isFav ? "star-filled" : "star"} size={18} />
          </button>
          <button onClick={requireAuth(() => setAlertOpen(true))} title="Alarm kur" style={iconBtn}>
            <Icon name="bell" size={18} />
          </button>
        </div>
      </div>

      {/* Chart */}
      <Card style={{ padding: 14 }}>
        <div style={{ display: "flex", justifyContent: "space-between", flexWrap: "wrap", gap: 8, marginBottom: 10 }}>
          <Seg options={INTERVALS.map((i) => ({ k: i.k, l: i.l }))} value={interval} onChange={setInterval} />
          <Seg options={CHART_TYPES.map((t) => ({ k: t.k, l: t.l }))} value={chartType} onChange={(v) => setChartType(v as ChartType)} />
        </div>
        {candles.data && candles.data.candles.length > 1 ? (
          <PriceChart candles={candles.data.candles} type={chartType} />
        ) : (
          <div style={{ height: 340, display: "flex", alignItems: "center", justifyContent: "center", color: "var(--text-dim)", fontSize: 13 }}>
            {candles.isLoading ? "Grafik yükleniyor…" : "Bu aralık için henüz yeterli veri yok"}
          </div>
        )}
      </Card>

      {/* Rich, per-kind detail */}
      <DetailSections d={detail.data} loading={detail.isLoading} live={{ bid: live?.bid, ask: live?.ask }} stats={stats.data} />

      {alertOpen && <AlertModal code={code} price={p} onClose={() => setAlertOpen(false)} />}
    </div>
  );
}

// ---------------------------------------------------------------------------

type Metric = { label: string; value: string; color?: string };

function DetailSections({
  d,
  loading,
  live,
  stats,
}: {
  d?: InstrumentDetail;
  loading: boolean;
  live: { bid?: number; ask?: number };
  stats?: { open: number; high: number; low: number };
}) {
  if (loading && !d) {
    return (
      <MetricGrid title="Piyasa verileri" metrics={Array.from({ length: 6 }).map(() => ({ label: "", value: "" }))} skeleton />
    );
  }
  if (!d) return null;

  const sections: { title: string; metrics: Metric[] }[] = [];

  // 24h / session box shared by all kinds (from live stats where available).
  const session: Metric[] = [];
  if (stats?.open) session.push({ label: "Gün açılış", value: tl(stats.open) });
  if (stats?.high) session.push({ label: "Gün yüksek", value: tl(stats.high) });
  if (stats?.low) session.push({ label: "Gün düşük", value: tl(stats.low) });
  if (live.bid) session.push({ label: "Alış", value: tl(live.bid) });
  if (live.ask) session.push({ label: "Satış", value: tl(live.ask) });
  if (session.length) sections.push({ title: "Seans", metrics: session });

  if (d.kind === "crypto") sections.push({ title: "Kripto piyasası", metrics: cryptoMetrics(d) });
  else if (d.kind === "stock") sections.push({ title: "Hisse istatistikleri", metrics: stockMetrics(d) });
  else if (d.kind === "gold") sections.push({ title: "Altın metrikleri", metrics: goldMetrics(d) });
  else if (d.kind === "currency") sections.push({ title: "Kur detayları", metrics: currencyMetrics(d) });

  return (
    <>
      {sections
        .filter((s) => s.metrics.length > 0)
        .map((s) => (
          <MetricGrid key={s.title} title={s.title} metrics={s.metrics} />
        ))}
    </>
  );
}

function cryptoMetrics(d: InstrumentDetail): Metric[] {
  const m: Metric[] = [];
  if (d.price_usd) m.push({ label: "Fiyat (USD)", value: usd(d.price_usd) });
  if (d.change_pct24h != null) m.push({ label: "24s değişim", value: `${d.change_pct24h.toFixed(2)}%`, color: d.change_pct24h >= 0 ? "var(--up)" : "var(--down)" });
  if (d.high24h) m.push({ label: "24s yüksek", value: tl(d.high24h) });
  if (d.low24h) m.push({ label: "24s düşük", value: tl(d.low24h) });
  if (d.market_cap_usd) m.push({ label: "Piyasa değeri", value: usdCompact(d.market_cap_usd) });
  if (d.rank) m.push({ label: "Sıralama", value: `#${d.rank}` });
  if (d.quote_volume_usd) m.push({ label: "24s hacim", value: usdCompact(d.quote_volume_usd) });
  if (d.dominance_pct) m.push({ label: "Dominance", value: `${d.dominance_pct.toFixed(2)}%` });
  if (d.circulating_supply) m.push({ label: "Dolaşımdaki arz", value: compact(d.circulating_supply) });
  if (d.max_supply) m.push({ label: "Maks. arz", value: compact(d.max_supply) });
  if (d.ath_usd) m.push({ label: `ATH${d.ath_date ? " · " + dateShort(d.ath_date) : ""}`, value: usd(d.ath_usd) });
  if (d.atl_usd) m.push({ label: `ATL${d.atl_date ? " · " + dateShort(d.atl_date) : ""}`, value: usd(d.atl_usd) });
  if (d.trades) m.push({ label: "24s işlem", value: compact(d.trades) });
  return m;
}

function stockMetrics(d: InstrumentDetail): Metric[] {
  const m: Metric[] = [];
  if (d.day_high) m.push({ label: "Gün yüksek", value: tl(d.day_high) });
  if (d.day_low) m.push({ label: "Gün düşük", value: tl(d.day_low) });
  if (d.week52_high) m.push({ label: "52h yüksek", value: tl(d.week52_high) });
  if (d.week52_low) m.push({ label: "52h düşük", value: tl(d.week52_low) });
  if (d.prev_close) m.push({ label: "Önceki kapanış", value: tl(d.prev_close) });
  if (d.volume) m.push({ label: "Hacim", value: compact(d.volume) });
  if (d.exchange) m.push({ label: "Borsa", value: d.exchange });
  return m;
}

function goldMetrics(d: InstrumentDetail): Metric[] {
  const m: Metric[] = [];
  if (d.gram_has_try) m.push({ label: "Gram has altın", value: tl(d.gram_has_try) });
  if (d.ons_usd) m.push({ label: "Ons (USD)", value: usd(d.ons_usd) });
  if (d.ons_try) m.push({ label: "Ons (₺)", value: tl(d.ons_try) });
  if (d.usd_try) m.push({ label: "USD/TRY", value: priceFmt(d.usd_try) });
  if (d.gram_weight) m.push({ label: "Gram ağırlık", value: `${d.gram_weight} g` });
  if (d.purity_label) m.push({ label: "Ayar", value: d.purity_label });
  return m;
}

function currencyMetrics(d: InstrumentDetail): Metric[] {
  const m: Metric[] = [];
  if (d.iso) m.push({ label: "Kod", value: d.iso });
  if (d.inverse) m.push({ label: "Ters kur (1₺)", value: `${priceFmt(d.inverse)} ${d.iso ?? ""}` });
  if (d.usd_cross) m.push({ label: "USD çapraz", value: usd(d.usd_cross) });
  if (d.usd_try) m.push({ label: "USD/TRY", value: priceFmt(d.usd_try) });
  return m;
}

function MetricGrid({ title, metrics, skeleton }: { title: string; metrics: Metric[]; skeleton?: boolean }) {
  return (
    <div>
      <div style={{ fontSize: 11, fontWeight: 600, color: "var(--text-faint)", textTransform: "uppercase", letterSpacing: "0.06em", marginBottom: 10 }}>
        {title}
      </div>
      <div style={{ display: "grid", gridTemplateColumns: "repeat(auto-fill, minmax(150px,1fr))", gap: 10 }}>
        {metrics.map((m, i) => (
          <Card key={i} style={{ padding: "12px 14px" }}>
            {skeleton ? (
              <>
                <div className="skeleton" style={{ width: 60, height: 9, marginBottom: 8 }} />
                <div className="skeleton" style={{ width: 90, height: 14 }} />
              </>
            ) : (
              <>
                <div style={{ fontSize: 11, color: "var(--text-faint)" }}>{m.label}</div>
                <div className="mono" style={{ fontSize: 15, fontWeight: 650, marginTop: 4, color: m.color }}>{m.value}</div>
              </>
            )}
          </Card>
        ))}
      </div>
    </div>
  );
}

function Seg({ options, value, onChange }: { options: { k: string; l: string }[]; value: string; onChange: (v: string) => void }) {
  return (
    <div style={{ display: "flex", gap: 4, background: "var(--surface-hi)", borderRadius: 9, padding: 3 }}>
      {options.map((o) => (
        <button
          key={o.k}
          onClick={() => onChange(o.k)}
          style={{
            background: value === o.k ? "var(--surface)" : "transparent",
            color: value === o.k ? "var(--text)" : "var(--text-dim)",
            border: value === o.k ? "1px solid var(--stroke)" : "1px solid transparent",
            borderRadius: 7,
            padding: "5px 11px",
            fontSize: 12,
            fontWeight: 600,
            cursor: "pointer",
          }}
        >
          {o.l}
        </button>
      ))}
    </div>
  );
}

function AlertModal({ code, price: cur, onClose }: { code: string; price: number; onClose: () => void }) {
  const qc = useQueryClient();
  const [direction, setDirection] = useState<"above" | "below">("above");
  const [threshold, setThreshold] = useState(cur ? cur.toFixed(2) : "");
  const create = useMutation({
    mutationFn: () => api.createAlert(code, direction, parseFloat(threshold)),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["alerts"] });
      onClose();
    },
  });
  return (
    <div onClick={onClose} style={overlay}>
      <div onClick={(e) => e.stopPropagation()} style={modal}>
        <h3 style={{ margin: "0 0 14px" }}>Alarm kur — {code}</h3>
        <div style={{ display: "flex", gap: 8, marginBottom: 12 }}>
          <SegBtn active={direction === "above"} onClick={() => setDirection("above")}>Üstüne çıkınca</SegBtn>
          <SegBtn active={direction === "below"} onClick={() => setDirection("below")}>Altına inince</SegBtn>
        </div>
        <input
          value={threshold}
          onChange={(e) => setThreshold(e.target.value)}
          placeholder="Eşik değeri (₺)"
          inputMode="decimal"
          style={{
            width: "100%",
            background: "var(--surface-hi)",
            border: "1px solid var(--stroke)",
            borderRadius: 10,
            padding: 12,
            color: "var(--text)",
            fontSize: 15,
            marginBottom: 14,
          }}
        />
        {create.isError && (
          <div style={{ color: "var(--down)", fontSize: 13, marginBottom: 10 }}>
            {(create.error as Error).message}
          </div>
        )}
        <div style={{ display: "flex", gap: 10 }}>
          <Button variant="ghost" full onClick={onClose}>Vazgeç</Button>
          <Button variant="primary" full disabled={!threshold || create.isPending} onClick={() => create.mutate()}>
            {create.isPending ? "Kaydediliyor…" : "Oluştur"}
          </Button>
        </div>
      </div>
    </div>
  );
}

function SegBtn({ active, onClick, children }: { active: boolean; onClick: () => void; children: React.ReactNode }) {
  return (
    <button
      onClick={onClick}
      style={{
        flex: 1,
        background: active ? "var(--accent-soft)" : "var(--surface-hi)",
        color: active ? "var(--accent-hi)" : "var(--text-dim)",
        border: "none",
        borderRadius: 10,
        padding: 12,
        fontSize: 13,
        fontWeight: 700,
        cursor: "pointer",
      }}
    >
      {children}
    </button>
  );
}

const backBtn: React.CSSProperties = {
  display: "flex",
  alignItems: "center",
  gap: 6,
  background: "none",
  border: "none",
  color: "var(--text-dim)",
  fontSize: 13.5,
  cursor: "pointer",
  padding: 0,
  alignSelf: "flex-start",
};
const iconBtn: React.CSSProperties = {
  width: 38,
  height: 38,
  display: "flex",
  alignItems: "center",
  justifyContent: "center",
  background: "var(--surface)",
  border: "1px solid var(--stroke)",
  borderRadius: 9,
  color: "var(--text-dim)",
  cursor: "pointer",
};
const overlay: React.CSSProperties = {
  position: "fixed",
  inset: 0,
  background: "rgba(0,0,0,0.6)",
  display: "flex",
  alignItems: "center",
  justifyContent: "center",
  zIndex: 200,
  padding: 20,
};
const modal: React.CSSProperties = {
  background: "var(--surface)",
  border: "1px solid var(--stroke-strong)",
  borderRadius: 16,
  padding: 22,
  width: "100%",
  maxWidth: 380,
};
