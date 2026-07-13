import { useMemo } from "react";
import { useNavigate } from "react-router-dom";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { api } from "../lib/endpoints";
import { useMarkets } from "../store/markets";
import { tl } from "../lib/format";
import { Card, ChangePill, EmptyState } from "../components/common";

const ORDER_KEY = "tsc_fav_order";

function loadOrder(): string[] {
  try {
    return JSON.parse(localStorage.getItem(ORDER_KEY) || "[]");
  } catch {
    return [];
  }
}
function saveOrder(o: string[]) {
  localStorage.setItem(ORDER_KEY, JSON.stringify(o));
}

export function Favorites() {
  const nav = useNavigate();
  const qc = useQueryClient();
  const prices = useMarkets((s) => s.prices);
  const watch = useQuery({ queryKey: ["watchlist"], queryFn: () => api.watchlist() });

  const remove = useMutation({
    mutationFn: (code: string) => api.removeWatch(code),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["watchlist"] }),
  });

  const ordered = useMemo(() => {
    const codes = watch.data?.codes ?? [];
    const saved = loadOrder();
    const rank = (c: string) => {
      const i = saved.indexOf(c);
      return i === -1 ? 9999 : i;
    };
    return [...codes].sort((a, b) => rank(a) - rank(b));
  }, [watch.data]);

  const move = (idx: number, dir: -1 | 1) => {
    const arr = [...ordered];
    const j = idx + dir;
    if (j < 0 || j >= arr.length) return;
    [arr[idx], arr[j]] = [arr[j], arr[idx]];
    saveOrder(arr);
    qc.invalidateQueries({ queryKey: ["watchlist"] });
  };

  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 18 }}>
      <div>
        <h2 style={{ margin: "0 0 4px", fontSize: 22 }}>Favoriler</h2>
        <div style={{ color: "var(--text-dim)", fontSize: 13 }}>Sıralamayı yukarı/aşağı taşıyarak özelleştir</div>
      </div>

      {ordered.length === 0 ? (
        <EmptyState icon="⭐" title="Henüz favori yok" hint="Bir varlık detayından ‘Favori ekle’ ile ekleyin" />
      ) : (
        <Card style={{ overflow: "hidden" }}>
          {ordered.map((code, i) => {
            const p = prices[code];
            return (
              <div
                key={code}
                style={{
                  display: "flex",
                  alignItems: "center",
                  gap: 12,
                  padding: "12px 16px",
                  borderTop: i === 0 ? "none" : "1px solid var(--stroke)",
                }}
              >
                <div style={{ display: "flex", flexDirection: "column", gap: 2 }}>
                  <button onClick={() => move(i, -1)} style={arrowBtn}>▲</button>
                  <button onClick={() => move(i, 1)} style={arrowBtn}>▼</button>
                </div>
                <div style={{ flex: 1, minWidth: 0, cursor: "pointer" }} onClick={() => nav(`/asset/${code}`)}>
                  <div style={{ fontSize: 14, fontWeight: 600 }}>{p?.display_name ?? code}</div>
                  <div style={{ fontSize: 11, color: "var(--text-faint)" }}>{code}</div>
                </div>
                <div style={{ textAlign: "right", width: 120 }}>
                  <div className="mono" style={{ fontSize: 15, fontWeight: 700 }}>{tl(p?.price ?? 0)}</div>
                  <div style={{ marginTop: 3 }}><ChangePill pct={p?.change_pct ?? 0} small /></div>
                </div>
                <button onClick={() => remove.mutate(code)} style={removeBtn} title="Kaldır">✕</button>
              </div>
            );
          })}
        </Card>
      )}
    </div>
  );
}

const arrowBtn: React.CSSProperties = {
  background: "var(--surface-hi)",
  border: "none",
  color: "var(--text-dim)",
  borderRadius: 6,
  width: 24,
  height: 18,
  fontSize: 9,
  cursor: "pointer",
  lineHeight: 1,
};
const removeBtn: React.CSSProperties = {
  background: "transparent",
  border: "none",
  color: "var(--text-faint)",
  fontSize: 14,
  cursor: "pointer",
};
