import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { api } from "../lib/endpoints";
import { useMarkets } from "../store/markets";
import { tl } from "../lib/format";
import { Card, EmptyState } from "../components/common";

export function Alerts() {
  const qc = useQueryClient();
  const prices = useMarkets((s) => s.prices);
  const alerts = useQuery({ queryKey: ["alerts"], queryFn: () => api.alerts() });
  const del = useMutation({
    mutationFn: (id: string) => api.deleteAlert(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["alerts"] }),
  });

  const items = alerts.data?.alerts ?? [];

  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 18 }}>
      <div>
        <h2 style={{ margin: "0 0 4px", fontSize: 22 }}>Alarmlar</h2>
        <div style={{ color: "var(--text-dim)", fontSize: 13 }}>Eşik değeri aşıldığında anlık bildirim alırsın</div>
      </div>

      {items.length === 0 ? (
        <EmptyState icon="🔔" title="Aktif alarm yok" hint="Bir varlık detayından ‘Alarm kur’ ile ekleyin" />
      ) : (
        <Card style={{ overflow: "hidden" }}>
          {items.map((a, i) => (
            <div
              key={a.id}
              style={{
                display: "flex",
                alignItems: "center",
                gap: 12,
                padding: "14px 16px",
                borderTop: i === 0 ? "none" : "1px solid var(--stroke)",
              }}
            >
              <div
                style={{
                  width: 38,
                  height: 38,
                  borderRadius: 10,
                  background: a.direction === "above" ? "rgba(14,203,129,0.14)" : "rgba(246,70,93,0.14)",
                  color: a.direction === "above" ? "var(--up)" : "var(--down)",
                  display: "flex",
                  alignItems: "center",
                  justifyContent: "center",
                  fontSize: 18,
                }}
              >
                {a.direction === "above" ? "↑" : "↓"}
              </div>
              <div style={{ flex: 1 }}>
                <div style={{ fontSize: 14, fontWeight: 600 }}>
                  {prices[a.code]?.display_name ?? a.code}
                </div>
                <div style={{ fontSize: 12, color: "var(--text-dim)" }}>
                  {a.direction === "above" ? "Üstüne çıkınca" : "Altına inince"} • {a.active ? "Aktif" : "Tetiklendi"}
                </div>
              </div>
              <div className="mono" style={{ fontWeight: 700 }}>{tl(a.threshold)}</div>
              <button onClick={() => del.mutate(a.id)} style={{ background: "transparent", border: "none", color: "var(--text-faint)", cursor: "pointer", fontSize: 14 }}>✕</button>
            </div>
          ))}
        </Card>
      )}
    </div>
  );
}
