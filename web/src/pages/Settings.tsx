import { useAuth } from "../auth/AuthContext";
import { useMarkets } from "../store/markets";
import { Button, Card } from "../components/common";
import { BRAND } from "../lib/config";

export function Settings() {
  const { logout, profile } = useAuth();
  const connected = useMarkets((s) => s.connected);
  const count = useMarkets((s) => s.order.length);

  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 18, maxWidth: 520 }}>
      <h2 style={{ margin: 0, fontSize: 22 }}>Ayarlar</h2>

      <Card style={{ padding: 20 }}>
        <Row label="Bağlantı" value={connected ? "Canlı" : "Bağlanıyor…"} valueColor={connected ? "var(--up)" : undefined} />
        <Row label="Takip edilen enstrüman" value={String(count)} />
        <Row label="Hesap" value={profile?.email ?? "-"} />
      </Card>

      <Card style={{ padding: 20 }}>
        <div style={{ fontWeight: 700, marginBottom: 6 }}>Hakkında</div>
        <div style={{ fontSize: 13, color: "var(--text-dim)" }}>
          {BRAND.name} — canlı altın ve döviz piyasası. Fiyatlar bilgilendirme amaçlıdır,
          yatırım tavsiyesi değildir.
        </div>
        <div style={{ fontSize: 12, color: "var(--text-faint)", marginTop: 10 }}>Sürüm 1.0.0</div>
      </Card>

      <Button variant="danger" onClick={logout}>Çıkış yap</Button>
    </div>
  );
}

function Row({ label, value, valueColor }: { label: string; value: string; valueColor?: string }) {
  return (
    <div style={{ display: "flex", justifyContent: "space-between", padding: "8px 0", fontSize: 14 }}>
      <span style={{ color: "var(--text-dim)" }}>{label}</span>
      <span style={{ fontWeight: 600, color: valueColor }}>{value}</span>
    </div>
  );
}
