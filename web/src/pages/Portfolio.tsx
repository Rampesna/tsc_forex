import { useMemo, useState } from "react";
import { useMarkets } from "../store/markets";
import { signed, signedPct, tl } from "../lib/format";
import { Button, Card, EmptyState } from "../components/common";

interface Holding {
  id: string;
  code: string;
  qty: number;
  buyPrice: number;
  addedAt: string;
}

const KEY = "tsc_holdings";
function load(): Holding[] {
  try {
    return JSON.parse(localStorage.getItem(KEY) || "[]");
  } catch {
    return [];
  }
}
function save(h: Holding[]) {
  localStorage.setItem(KEY, JSON.stringify(h));
}

export function Portfolio() {
  const prices = useMarkets((s) => s.prices);
  const catalog = useMarkets((s) => s.catalog);
  const [holdings, setHoldings] = useState<Holding[]>(load);
  const [adding, setAdding] = useState(false);

  const rows = useMemo(
    () =>
      holdings.map((h) => {
        const cur = prices[h.code]?.price ?? h.buyPrice;
        const value = cur * h.qty;
        const cost = h.buyPrice * h.qty;
        const pl = value - cost;
        const plPct = cost > 0 ? (pl / cost) * 100 : 0;
        return { ...h, cur, value, cost, pl, plPct };
      }),
    [holdings, prices],
  );

  const totals = useMemo(() => {
    const value = rows.reduce((s, r) => s + r.value, 0);
    const cost = rows.reduce((s, r) => s + r.cost, 0);
    const pl = value - cost;
    const plPct = cost > 0 ? (pl / cost) * 100 : 0;
    return { value, cost, pl, plPct };
  }, [rows]);

  const remove = (id: string) => {
    const next = holdings.filter((h) => h.id !== id);
    setHoldings(next);
    save(next);
  };
  const add = (h: Holding) => {
    const next = [...holdings, h];
    setHoldings(next);
    save(next);
    setAdding(false);
  };

  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 18 }}>
      <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center" }}>
        <div>
          <h2 style={{ margin: "0 0 4px", fontSize: 22 }}>Portföy</h2>
          <div style={{ color: "var(--text-dim)", fontSize: 13 }}>Varlıklarının canlı değeri ve kâr/zarar analizi</div>
        </div>
        <Button variant="primary" onClick={() => setAdding(true)}>+ Varlık ekle</Button>
      </div>

      <Card style={{ padding: 20 }}>
        <div style={{ fontSize: 12, color: "var(--text-dim)" }}>Toplam Değer</div>
        <div className="mono" style={{ fontSize: 30, fontWeight: 800, margin: "6px 0" }}>{tl(totals.value)}</div>
        <div className="mono" style={{ color: totals.pl >= 0 ? "var(--up)" : "var(--down)", fontSize: 14, fontWeight: 600 }}>
          {signed(totals.pl)} ({signedPct(totals.plPct)})
        </div>
      </Card>

      {rows.length === 0 ? (
        <EmptyState icon="portfolio" title="Portföyün boş" hint="Bir varlık ekleyerek başla" />
      ) : (
        <Card style={{ overflow: "hidden" }}>
          {rows.map((r, i) => (
            <div key={r.id} style={{ display: "flex", alignItems: "center", gap: 12, padding: "14px 16px", borderTop: i === 0 ? "none" : "1px solid var(--stroke)" }}>
              <div style={{ flex: 1 }}>
                <div style={{ fontSize: 14, fontWeight: 600 }}>{prices[r.code]?.display_name ?? r.code}</div>
                <div style={{ fontSize: 12, color: "var(--text-faint)" }}>
                  {r.qty} adet • Alış {tl(r.buyPrice)}
                </div>
              </div>
              <div style={{ textAlign: "right" }}>
                <div className="mono" style={{ fontSize: 15, fontWeight: 700 }}>{tl(r.value)}</div>
                <div className="mono" style={{ fontSize: 12, color: r.pl >= 0 ? "var(--up)" : "var(--down)" }}>
                  {signed(r.pl)} ({signedPct(r.plPct)})
                </div>
              </div>
              <button onClick={() => remove(r.id)} style={{ background: "transparent", border: "none", color: "var(--text-faint)", cursor: "pointer", fontSize: 14 }}>✕</button>
            </div>
          ))}
        </Card>
      )}

      {adding && (
        <AddHolding
          codes={catalog.map((c) => ({ code: c.code, name: c.display_name }))}
          priceFor={(c) => prices[c]?.price ?? 0}
          onAdd={add}
          onClose={() => setAdding(false)}
        />
      )}
    </div>
  );
}

function AddHolding({
  codes,
  priceFor,
  onAdd,
  onClose,
}: {
  codes: { code: string; name: string }[];
  priceFor: (code: string) => number;
  onAdd: (h: Holding) => void;
  onClose: () => void;
}) {
  const [code, setCode] = useState(codes[0]?.code ?? "");
  const [qty, setQty] = useState("1");
  const [buyPrice, setBuyPrice] = useState(() => (codes[0] ? priceFor(codes[0].code).toFixed(2) : ""));

  return (
    <div onClick={onClose} style={overlay}>
      <div onClick={(e) => e.stopPropagation()} style={modal}>
        <h3 style={{ margin: "0 0 14px" }}>Varlık ekle</h3>
        <label style={lbl}>Varlık</label>
        <select
          value={code}
          onChange={(e) => {
            setCode(e.target.value);
            setBuyPrice(priceFor(e.target.value).toFixed(2));
          }}
          style={field}
        >
          {codes.map((c) => (
            <option key={c.code} value={c.code}>{c.name}</option>
          ))}
        </select>
        <label style={lbl}>Miktar</label>
        <input value={qty} onChange={(e) => setQty(e.target.value)} inputMode="decimal" style={field} />
        <label style={lbl}>Alış Fiyatı (₺)</label>
        <input value={buyPrice} onChange={(e) => setBuyPrice(e.target.value)} inputMode="decimal" style={field} />
        <div style={{ display: "flex", gap: 10, marginTop: 8 }}>
          <Button variant="ghost" full onClick={onClose}>Vazgeç</Button>
          <Button
            variant="primary"
            full
            disabled={!code || !qty || !buyPrice}
            onClick={() =>
              onAdd({
                id: crypto.randomUUID(),
                code,
                qty: parseFloat(qty),
                buyPrice: parseFloat(buyPrice),
                addedAt: new Date().toISOString(),
              })
            }
          >
            Ekle
          </Button>
        </div>
      </div>
    </div>
  );
}

const overlay: React.CSSProperties = {
  position: "fixed", inset: 0, background: "rgba(0,0,0,0.6)",
  display: "flex", alignItems: "center", justifyContent: "center", zIndex: 200, padding: 20,
};
const modal: React.CSSProperties = {
  background: "var(--surface)", border: "1px solid var(--stroke-strong)", borderRadius: 16, padding: 22, width: "100%", maxWidth: 380,
};
const lbl: React.CSSProperties = { display: "block", fontSize: 12, color: "var(--text-dim)", margin: "10px 0 4px" };
const field: React.CSSProperties = {
  width: "100%", background: "var(--surface-hi)", border: "1px solid var(--stroke)", borderRadius: 10, padding: 12, color: "var(--text)", fontSize: 15,
};
