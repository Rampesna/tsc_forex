import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { api } from "../lib/endpoints";
import { timeAgo, tl } from "../lib/format";
import { Button, Card, EmptyState } from "../components/common";

export function Notifications() {
  const qc = useQueryClient();
  const notifs = useQuery({ queryKey: ["notifications"], queryFn: () => api.notifications(), refetchInterval: 20000 });
  const items = notifs.data?.notifications ?? [];

  const inval = () => qc.invalidateQueries({ queryKey: ["notifications"] });
  const del = useMutation({ mutationFn: (id: string) => api.deleteNotification(id), onSuccess: inval });
  const read = useMutation({ mutationFn: (id: string) => api.markRead(id), onSuccess: inval });
  const clear = useMutation({ mutationFn: () => api.clearNotifications(), onSuccess: inval });

  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 18 }}>
      <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center" }}>
        <div>
          <h2 style={{ margin: "0 0 4px", fontSize: 22 }}>Bildirimler</h2>
          <div style={{ color: "var(--text-dim)", fontSize: 13 }}>Alarm tetiklenmeleri ve sistem mesajları</div>
        </div>
        {items.length > 0 && (
          <Button variant="danger" onClick={() => clear.mutate()}>Tümünü temizle</Button>
        )}
      </div>

      {items.length === 0 ? (
        <EmptyState icon="📨" title="Bildirim yok" hint="Alarmların tetiklendiğinde burada görünür" />
      ) : (
        <Card style={{ overflow: "hidden" }}>
          {items.map((n, i) => (
            <div
              key={n.id}
              onClick={() => !n.read_at && read.mutate(n.id)}
              style={{
                display: "flex",
                alignItems: "flex-start",
                gap: 12,
                padding: "14px 16px",
                borderTop: i === 0 ? "none" : "1px solid var(--stroke)",
                background: n.read_at ? "transparent" : "rgba(23,162,184,0.05)",
                cursor: n.read_at ? "default" : "pointer",
              }}
            >
              {!n.read_at && <span style={{ width: 8, height: 8, borderRadius: 4, background: "var(--accent)", marginTop: 6 }} />}
              <div style={{ flex: 1 }}>
                <div style={{ fontSize: 14, fontWeight: 700 }}>{n.title}</div>
                <div style={{ fontSize: 13, color: "var(--text-dim)", marginTop: 2 }}>{n.body}</div>
                <div style={{ fontSize: 11, color: "var(--text-faint)", marginTop: 6 }}>
                  {n.code} • {tl(n.price)} • {timeAgo(n.created_at)}
                </div>
              </div>
              <button onClick={(e) => { e.stopPropagation(); del.mutate(n.id); }} style={{ background: "transparent", border: "none", color: "var(--text-faint)", cursor: "pointer", fontSize: 14 }}>✕</button>
            </div>
          ))}
        </Card>
      )}
    </div>
  );
}
