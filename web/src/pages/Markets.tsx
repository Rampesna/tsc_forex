import { useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import { useMarkets } from "../store/markets";
import { tl } from "../lib/format";
import { ChangePill, Card } from "../components/common";
import { Sparkline } from "../components/Sparkline";
import type { InstrumentPrice } from "../lib/types";

export function Markets() {
  const order = useMarkets((s) => s.order);
  const prices = useMarkets((s) => s.prices);
  const spark = useMarkets((s) => s.spark);
  const flash = useMarkets((s) => s.flash);
  const [q, setQ] = useState("");
  const nav = useNavigate();

  const rows = useMemo(() => {
    const list = order.map((code) => prices[code]).filter(Boolean) as InstrumentPrice[];
    if (!q.trim()) return list;
    const s = q.toLocaleLowerCase("tr");
    return list.filter(
      (p) => p.display_name.toLocaleLowerCase("tr").includes(s) || p.code.toLowerCase().includes(s),
    );
  }, [order, prices, q]);

  const movers = useMemo(() => {
    const list = Object.values(prices).filter((p) => Number.isFinite(p.change_pct));
    return [...list].sort((a, b) => Math.abs(b.change_pct) - Math.abs(a.change_pct)).slice(0, 4);
  }, [prices]);

  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 20 }}>
      <div>
        <h2 style={{ margin: "0 0 4px", fontSize: 22 }}>Piyasalar</h2>
        <div style={{ color: "var(--text-dim)", fontSize: 13 }}>
          Tüm altın ürünleri ve dünya para birimleri — canlı, ₺ cinsinden
        </div>
      </div>

      {/* Top movers */}
      {movers.length > 0 && (
        <div style={{ display: "grid", gridTemplateColumns: "repeat(auto-fill, minmax(200px, 1fr))", gap: 12 }}>
          {movers.map((m) => (
            <Card key={m.code} onClick={() => nav(`/asset/${m.code}`)} style={{ padding: 14, cursor: "pointer" }}>
              <div style={{ fontSize: 12, color: "var(--text-dim)", marginBottom: 6, whiteSpace: "nowrap", overflow: "hidden", textOverflow: "ellipsis" }}>
                {m.display_name}
              </div>
              <div className="mono" style={{ fontSize: 18, fontWeight: 700 }}>{tl(m.price)}</div>
              <div style={{ marginTop: 6 }}>
                <ChangePill pct={m.change_pct} small />
              </div>
            </Card>
          ))}
        </div>
      )}

      {/* Search */}
      <input
        value={q}
        onChange={(e) => setQ(e.target.value)}
        placeholder="Altın veya döviz ara (ör. Gram Altın, Dolar, EUR)"
        style={{
          background: "var(--surface)",
          border: "1px solid var(--stroke)",
          borderRadius: 12,
          padding: "12px 14px",
          color: "var(--text)",
          fontSize: 14,
          outline: "none",
        }}
      />

      {/* List */}
      <Card style={{ overflow: "hidden" }}>
        {rows.length === 0 && (
          <div style={{ padding: 40, textAlign: "center", color: "var(--text-dim)" }}>
            {order.length === 0 ? "Yükleniyor…" : "Sonuç yok"}
          </div>
        )}
        {rows.map((p, i) => (
          <div
            key={p.code}
            onClick={() => nav(`/asset/${p.code}`)}
            className={flash[p.code] === "up" ? "flash-up" : flash[p.code] === "down" ? "flash-down" : ""}
            style={{
              display: "flex",
              alignItems: "center",
              gap: 14,
              padding: "12px 16px",
              borderTop: i === 0 ? "none" : "1px solid var(--stroke)",
              cursor: "pointer",
              height: 62,
            }}
          >
            <div style={{ flex: 1, minWidth: 0 }}>
              <div style={{ fontSize: 14, fontWeight: 600, whiteSpace: "nowrap", overflow: "hidden", textOverflow: "ellipsis" }}>
                {p.display_name}
              </div>
              <div style={{ fontSize: 11, color: "var(--text-faint)" }}>{p.code}</div>
            </div>
            <Sparkline data={spark[p.code] ?? []} />
            <div style={{ width: 130, textAlign: "right" }}>
              <div className="mono" style={{ fontSize: 15, fontWeight: 700 }}>{tl(p.price)}</div>
              <div style={{ marginTop: 3 }}>
                <ChangePill pct={p.change_pct} small />
              </div>
            </div>
          </div>
        ))}
      </Card>
    </div>
  );
}
