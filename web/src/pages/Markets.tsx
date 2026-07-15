import { useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import { useMarkets } from "../store/markets";
import { money, tl } from "../lib/format";
import { ChangePill, AssetBadge } from "../components/common";
import { Icon } from "../components/Icon";
import { Sparkline } from "../components/Sparkline";
import type { InstrumentPrice } from "../lib/types";

const CATS: { id: string; label: string; kinds: string[] | null }[] = [
  { id: "all", label: "Tümü", kinds: null },
  { id: "gold", label: "Altın", kinds: ["gold"] },
  { id: "currency", label: "Döviz", kinds: ["currency", "fx"] },
  { id: "crypto", label: "Kripto", kinds: ["crypto"] },
  { id: "stock", label: "Hisse", kinds: ["stock", "equity"] },
];

export function Markets() {
  const order = useMarkets((s) => s.order);
  const prices = useMarkets((s) => s.prices);
  const spark = useMarkets((s) => s.spark);
  const flash = useMarkets((s) => s.flash);
  const [q, setQ] = useState("");
  const [cat, setCat] = useState("all");
  const nav = useNavigate();

  const rows = useMemo(() => {
    let list = order.map((code) => prices[code]).filter(Boolean) as InstrumentPrice[];
    const active = CATS.find((c) => c.id === cat);
    if (active?.kinds) list = list.filter((p) => active.kinds!.includes(p.kind));
    if (q.trim()) {
      const s = q.toLocaleLowerCase("tr");
      list = list.filter(
        (p) => p.display_name.toLocaleLowerCase("tr").includes(s) || p.code.toLowerCase().includes(s),
      );
    }
    return list;
  }, [order, prices, q, cat]);

  const gainers = useMemo(() => {
    const list = Object.values(prices).filter((p) => Number.isFinite(p.change_pct) && p.change_pct > 0);
    return list.sort((a, b) => b.change_pct - a.change_pct).slice(0, 8);
  }, [prices]);

  const loading = order.length === 0;

  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 18 }}>
      <div>
        <h2 style={{ margin: "0 0 4px", fontSize: 20, fontWeight: 600, letterSpacing: "-0.02em" }}>
          Piyasalar
        </h2>
        <div style={{ color: "var(--text-dim)", fontSize: 13 }}>
          Altın, döviz, kripto ve hisse — canlı, ₺ cinsinden
        </div>
      </div>

      {/* Top gainers strip */}
      {gainers.length > 0 && (
        <div>
          <div style={{ fontSize: 11, fontWeight: 600, color: "var(--text-faint)", textTransform: "uppercase", letterSpacing: "0.06em", marginBottom: 8 }}>
            Günün öne çıkanları
          </div>
          <div style={{ display: "flex", gap: 8, overflowX: "auto", paddingBottom: 4 }} className="no-sb">
            {gainers.map((m) => (
              <div
                key={m.code}
                onClick={() => nav(`/asset/${m.code}`)}
                className="row-hover"
                style={{
                  minWidth: 150,
                  border: "1px solid var(--stroke)",
                  borderRadius: 10,
                  padding: "10px 12px",
                  cursor: "pointer",
                  background: "var(--surface)",
                  flexShrink: 0,
                }}
              >
                <div style={{ fontSize: 12, color: "var(--text-dim)", whiteSpace: "nowrap", overflow: "hidden", textOverflow: "ellipsis" }}>
                  {m.display_name}
                </div>
                <div className="mono" style={{ fontSize: 15, fontWeight: 650, margin: "4px 0 2px" }}>{tl(m.price)}</div>
                <ChangePill pct={m.change_pct} small />
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Category filter */}
      <div style={{ display: "flex", gap: 6, flexWrap: "wrap" }}>
        {CATS.map((c) => {
          const on = c.id === cat;
          return (
            <button
              key={c.id}
              onClick={() => setCat(c.id)}
              style={{
                border: "1px solid " + (on ? "var(--stroke-strong)" : "var(--stroke)"),
                background: on ? "var(--surface-hi)" : "transparent",
                color: on ? "var(--text)" : "var(--text-dim)",
                borderRadius: 8,
                padding: "6px 12px",
                fontSize: 13,
                fontWeight: 550,
                cursor: "pointer",
                transition: "background 0.12s, color 0.12s",
              }}
            >
              {c.label}
            </button>
          );
        })}
      </div>

      {/* Search */}
      <div style={{ position: "relative" }}>
        <span style={{ position: "absolute", left: 12, top: "50%", transform: "translateY(-50%)", color: "var(--text-faint)" }}>
          <Icon name="search" size={16} />
        </span>
        <input
          value={q}
          onChange={(e) => setQ(e.target.value)}
          placeholder="Sembol veya isim ara"
          style={{
            width: "100%",
            background: "var(--surface)",
            border: "1px solid var(--stroke)",
            borderRadius: 9,
            padding: "10px 14px 10px 36px",
            color: "var(--text)",
            fontSize: 13.5,
            outline: "none",
          }}
        />
      </div>

      {/* List */}
      <div style={{ border: "1px solid var(--stroke)", borderRadius: "var(--radius)", overflow: "hidden", background: "var(--surface)" }}>
        <div style={{ display: "flex", alignItems: "center", padding: "9px 16px", borderBottom: "1px solid var(--stroke)", fontSize: 11, fontWeight: 600, color: "var(--text-faint)", textTransform: "uppercase", letterSpacing: "0.05em" }}>
          <span style={{ flex: 1 }}>Enstrüman</span>
          <span style={{ width: 88, textAlign: "center" }}>24s</span>
          <span style={{ width: 140, textAlign: "right" }}>Fiyat</span>
        </div>

        {loading &&
          Array.from({ length: 8 }).map((_, i) => (
            <div key={i} style={{ display: "flex", alignItems: "center", gap: 12, padding: "12px 16px", borderTop: i === 0 ? "none" : "1px solid var(--stroke)" }}>
              <div className="skeleton" style={{ width: 32, height: 32, borderRadius: 10 }} />
              <div style={{ flex: 1 }}>
                <div className="skeleton" style={{ width: 120, height: 11, marginBottom: 6 }} />
                <div className="skeleton" style={{ width: 60, height: 9 }} />
              </div>
              <div className="skeleton" style={{ width: 80, height: 24 }} />
              <div className="skeleton" style={{ width: 90, height: 16 }} />
            </div>
          ))}

        {!loading && rows.length === 0 && (
          <div style={{ padding: 40, textAlign: "center", color: "var(--text-dim)", fontSize: 13 }}>
            Sonuç yok
          </div>
        )}

        {rows.map((p, i) => (
          <div
            key={p.code}
            onClick={() => nav(`/asset/${p.code}`)}
            className={"row-hover " + (flash[p.code] === "up" ? "flash-up" : flash[p.code] === "down" ? "flash-down" : "")}
            style={{
              display: "flex",
              alignItems: "center",
              gap: 12,
              padding: "10px 16px",
              borderTop: i === 0 ? "none" : "1px solid var(--stroke)",
              cursor: "pointer",
              height: 60,
            }}
          >
            <AssetBadge code={p.code} size={32} />
            <div style={{ flex: 1, minWidth: 0 }}>
              <div style={{ fontSize: 13.5, fontWeight: 550, whiteSpace: "nowrap", overflow: "hidden", textOverflow: "ellipsis" }}>
                {p.display_name}
              </div>
              <div style={{ fontSize: 11, color: "var(--text-faint)", marginTop: 1 }}>{p.code}</div>
            </div>
            <div style={{ width: 88, display: "flex", justifyContent: "center" }}>
              <Sparkline data={spark[p.code] ?? []} />
            </div>
            <div style={{ width: 140, textAlign: "right" }}>
              <div className="mono" style={{ fontSize: 14.5, fontWeight: 650 }}>{money(p.price, p.currency)}</div>
              {p.currency === "USD" && (p.price_try ?? 0) > 0 && (
                <div className="mono" style={{ fontSize: 10.5, color: "var(--text-faint)", marginTop: 1 }}>≈ {tl(p.price_try as number)}</div>
              )}
              <div style={{ marginTop: 2, display: "flex", justifyContent: "flex-end" }}>
                <ChangePill pct={p.change_pct} small />
              </div>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}
