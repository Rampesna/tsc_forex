import { useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import { useAuth } from "../auth/AuthContext";
import { useMarkets } from "../store/markets";
import { Button, Card } from "../components/common";
import { api } from "../lib/endpoints";
import { BRAND } from "../lib/config";

export function Settings() {
  const { logout, profile, authed } = useAuth();
  const connected = useMarkets((s) => s.connected);
  const count = useMarkets((s) => s.order.length);
  const nav = useNavigate();
  const [deleting, setDeleting] = useState(false);

  const deleteAccount = async () => {
    if (!window.confirm("Hesabın ve tüm verilerin (favoriler, alarmlar, portföy, ödüller) kalıcı olarak silinecek. Bu işlem geri alınamaz. Devam edilsin mi?")) {
      return;
    }
    setDeleting(true);
    try {
      await api.deleteAccount();
    } catch {
      /* ignore — proceed to sign out regardless */
    }
    logout();
    setDeleting(false);
    nav("/");
  };

  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 18, maxWidth: 520 }}>
      <h2 style={{ margin: 0, fontSize: 22 }}>Ayarlar</h2>

      <Card style={{ padding: 20 }}>
        <Row label="Takip edilen enstrüman" value={String(count)} />
        <Row label="Canlı bağlantı" value={connected ? "Aktif" : "Bağlanıyor…"} valueColor={connected ? "var(--up)" : undefined} />
        {authed && <Row label="Hesap" value={profile?.email ?? "-"} />}
      </Card>

      <Card style={{ padding: 20 }}>
        <div style={{ fontWeight: 700, marginBottom: 6 }}>Hakkında</div>
        <div style={{ fontSize: 13, color: "var(--text-dim)" }}>
          {BRAND.name} — canlı döviz, altın, kripto ve hisse piyasası. Fiyatlar bilgilendirme
          amaçlıdır, yatırım tavsiyesi değildir.
        </div>
        <div style={{ display: "flex", gap: 16, marginTop: 12, fontSize: 13 }}>
          <Link to="/privacy" style={{ color: "var(--accent-hi)" }}>Gizlilik</Link>
          <Link to="/terms" style={{ color: "var(--accent-hi)" }}>Kullanım Şartları</Link>
          <Link to="/support" style={{ color: "var(--accent-hi)" }}>Destek</Link>
        </div>
        <div style={{ fontSize: 12, color: "var(--text-faint)", marginTop: 10 }}>Sürüm 1.0.0</div>
      </Card>

      {authed ? (
        <>
          <Button variant="ghost" onClick={logout}>Çıkış yap</Button>
          <button
            onClick={deleteAccount}
            disabled={deleting}
            style={{
              background: "transparent",
              border: "none",
              color: "var(--down)",
              fontSize: 13,
              fontWeight: 700,
              cursor: deleting ? "not-allowed" : "pointer",
              padding: 8,
              opacity: deleting ? 0.6 : 1,
            }}
          >
            {deleting ? "Siliniyor…" : "Hesabı kalıcı olarak sil"}
          </button>
        </>
      ) : (
        <Button variant="primary" onClick={() => nav("/login")}>Giriş yap / Kayıt ol</Button>
      )}
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
