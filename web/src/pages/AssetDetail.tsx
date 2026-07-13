import { useState } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { api } from "../lib/endpoints";
import { useMarkets } from "../store/markets";
import { signed, tl } from "../lib/format";
import { Button, Card, ChangePill } from "../components/common";
import { PriceChart } from "../components/PriceChart";

const INTERVALS = [
  { k: "1m", l: "1DK" },
  { k: "5m", l: "5DK" },
  { k: "15m", l: "15DK" },
  { k: "1h", l: "1SA" },
  { k: "1d", l: "1G" },
];

export function AssetDetail() {
  const { code = "" } = useParams();
  const nav = useNavigate();
  const qc = useQueryClient();
  const live = useMarkets((s) => s.prices[code]);
  const [interval, setInterval] = useState("5m");
  const [alertOpen, setAlertOpen] = useState(false);

  const stats = useQuery({ queryKey: ["stats", code], queryFn: () => api.stats(code), refetchInterval: 15000 });
  const candles = useQuery({ queryKey: ["candles", code, interval], queryFn: () => api.candles(code, interval) });
  const watch = useQuery({ queryKey: ["watchlist"], queryFn: () => api.watchlist() });
  const isFav = watch.data?.codes?.includes(code) ?? false;

  const toggleFav = useMutation({
    mutationFn: () => (isFav ? api.removeWatch(code) : api.addWatch(code)),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["watchlist"] }),
  });

  const p = live?.price ?? stats.data?.last ?? 0;
  const changePct = live?.change_pct ?? stats.data?.change_pct ?? 0;
  const change = live?.change ?? stats.data?.change ?? 0;
  const name = live?.display_name ?? code;

  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 18 }}>
      <button onClick={() => nav(-1)} style={backBtn}>← Geri</button>

      <div style={{ display: "flex", alignItems: "flex-end", justifyContent: "space-between", flexWrap: "wrap", gap: 12 }}>
        <div>
          <h2 style={{ margin: "0 0 6px", fontSize: 22 }}>{name}</h2>
          <div style={{ display: "flex", alignItems: "baseline", gap: 12 }}>
            <span className="mono" style={{ fontSize: 30, fontWeight: 800 }}>{tl(p)}</span>
            <ChangePill pct={changePct} />
            <span className="mono" style={{ color: change >= 0 ? "var(--up)" : "var(--down)", fontSize: 14 }}>
              {signed(change)}
            </span>
          </div>
        </div>
        <div style={{ display: "flex", gap: 10 }}>
          <Button variant={isFav ? "soft" : "ghost"} onClick={() => toggleFav.mutate()}>
            {isFav ? "★ Favoride" : "☆ Favori ekle"}
          </Button>
          <Button variant="primary" onClick={() => setAlertOpen(true)}>🔔 Alarm kur</Button>
        </div>
      </div>

      {/* Stats */}
      <div style={{ display: "grid", gridTemplateColumns: "repeat(auto-fill, minmax(120px,1fr))", gap: 12 }}>
        <Stat label="24s Açılış" value={tl(stats.data?.open ?? 0)} />
        <Stat label="24s Yüksek" value={tl(stats.data?.high ?? 0)} />
        <Stat label="24s Düşük" value={tl(stats.data?.low ?? 0)} />
        <Stat label="Alış" value={tl(live?.bid ?? 0)} />
        <Stat label="Satış" value={tl(live?.ask ?? 0)} />
      </div>

      {/* Chart */}
      <Card style={{ padding: 14 }}>
        <div style={{ display: "flex", gap: 6, marginBottom: 10 }}>
          {INTERVALS.map((iv) => (
            <button
              key={iv.k}
              onClick={() => setInterval(iv.k)}
              style={{
                background: interval === iv.k ? "var(--accent-soft)" : "transparent",
                color: interval === iv.k ? "var(--accent-hi)" : "var(--text-dim)",
                border: "none",
                borderRadius: 8,
                padding: "6px 12px",
                fontSize: 12,
                fontWeight: 700,
                cursor: "pointer",
              }}
            >
              {iv.l}
            </button>
          ))}
        </div>
        {candles.data && candles.data.candles.length > 1 ? (
          <PriceChart candles={candles.data.candles} />
        ) : (
          <div style={{ height: 320, display: "flex", alignItems: "center", justifyContent: "center", color: "var(--text-dim)" }}>
            {candles.isLoading ? "Grafik yükleniyor…" : "Bu aralık için henüz yeterli veri yok"}
          </div>
        )}
      </Card>

      {alertOpen && <AlertModal code={code} price={p} onClose={() => setAlertOpen(false)} />}
    </div>
  );
}

function Stat({ label, value }: { label: string; value: string }) {
  return (
    <Card style={{ padding: "12px 14px" }}>
      <div style={{ fontSize: 11, color: "var(--text-faint)" }}>{label}</div>
      <div className="mono" style={{ fontSize: 15, fontWeight: 700, marginTop: 4 }}>{value}</div>
    </Card>
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
  background: "none",
  border: "none",
  color: "var(--text-dim)",
  fontSize: 14,
  cursor: "pointer",
  padding: 0,
  alignSelf: "flex-start",
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
