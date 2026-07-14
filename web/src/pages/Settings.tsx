import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { useAuth } from "../auth/AuthContext";
import { Button, Card } from "../components/common";
import { api } from "../lib/endpoints";
import { BRAND } from "../lib/config";

export function Settings() {
  const { logout, profile, authed } = useAuth();
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
      <h2 style={{ margin: 0, fontSize: 20, fontWeight: 600, letterSpacing: "-0.02em" }}>Ayarlar</h2>

      {authed && (
        <Card style={{ padding: 20 }}>
          <Row label="Hesap" value={profile?.email ?? "-"} />
          <Row label="Sürüm" value="1.0.0" />
        </Card>
      )}

      <Card style={{ padding: 20 }}>
        <div style={{ fontWeight: 600, marginBottom: 6 }}>Hakkında</div>
        <div style={{ fontSize: 13, color: "var(--text-dim)", lineHeight: 1.5 }}>
          {BRAND.name} — canlı döviz, altın, kripto ve hisse piyasası. Fiyatlar bilgilendirme
          amaçlıdır, yatırım tavsiyesi değildir.
        </div>
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
