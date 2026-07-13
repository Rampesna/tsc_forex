import { useRef, useState } from "react";
import { useAuth } from "../auth/AuthContext";
import { api } from "../lib/endpoints";
import { passkeySupported } from "../auth/passkey";
import { Button, Card } from "../components/common";

export function Profile() {
  const { profile, refreshProfile, registerPasskey, setProfile } = useAuth();
  const [name, setName] = useState(profile?.display_name ?? "");
  const [saving, setSaving] = useState(false);
  const [uploading, setUploading] = useState(false);
  const [pkMsg, setPkMsg] = useState<string | null>(null);
  const fileRef = useRef<HTMLInputElement>(null);

  const saveName = async () => {
    setSaving(true);
    try {
      const p = await api.updateProfile(name);
      setProfile(p);
    } finally {
      setSaving(false);
    }
  };

  const onFile = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const f = e.target.files?.[0];
    if (!f) return;
    setUploading(true);
    try {
      await api.uploadAvatar(f);
      await refreshProfile();
    } finally {
      setUploading(false);
    }
  };

  const addPasskey = async () => {
    setPkMsg(null);
    const ok = await registerPasskey();
    setPkMsg(ok ? "Geçiş anahtarı oluşturuldu ✓" : "Geçiş anahtarı oluşturulamadı");
  };

  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 18, maxWidth: 520 }}>
      <h2 style={{ margin: 0, fontSize: 22 }}>Profil</h2>

      <Card style={{ padding: 20, display: "flex", alignItems: "center", gap: 16 }}>
        {profile?.avatar_url ? (
          <img src={profile.avatar_url} width={72} height={72} style={{ borderRadius: 36, objectFit: "cover" }} alt="" />
        ) : (
          <div style={{ width: 72, height: 72, borderRadius: 36, background: "var(--accent-soft)", color: "var(--accent-hi)", display: "flex", alignItems: "center", justifyContent: "center", fontSize: 28, fontWeight: 700 }}>
            {(profile?.display_name || profile?.email || "?").slice(0, 1).toUpperCase()}
          </div>
        )}
        <div style={{ flex: 1 }}>
          <div style={{ fontSize: 16, fontWeight: 700 }}>{profile?.display_name || "İsimsiz"}</div>
          <div style={{ fontSize: 13, color: "var(--text-dim)" }}>{profile?.email}</div>
        </div>
        <input ref={fileRef} type="file" accept="image/*" hidden onChange={onFile} />
        <Button variant="ghost" onClick={() => fileRef.current?.click()} disabled={uploading}>
          {uploading ? "Yükleniyor…" : "Fotoğraf değiştir"}
        </Button>
      </Card>

      <Card style={{ padding: 20 }}>
        <label style={{ fontSize: 12, color: "var(--text-dim)" }}>Görünen ad</label>
        <div style={{ display: "flex", gap: 10, marginTop: 8 }}>
          <input
            value={name}
            onChange={(e) => setName(e.target.value)}
            style={{ flex: 1, background: "var(--surface-hi)", border: "1px solid var(--stroke)", borderRadius: 10, padding: 12, color: "var(--text)", fontSize: 15 }}
          />
          <Button variant="primary" onClick={saveName} disabled={saving || !name.trim()}>
            {saving ? "…" : "Kaydet"}
          </Button>
        </div>
      </Card>

      {passkeySupported() && (
        <Card style={{ padding: 20 }}>
          <div style={{ fontWeight: 700, marginBottom: 4 }}>Geçiş anahtarı (Passkey)</div>
          <div style={{ fontSize: 13, color: "var(--text-dim)", marginBottom: 12 }}>
            Bu cihaza parolasız, biyometrik giriş için bir geçiş anahtarı ekle.
          </div>
          <Button variant="soft" onClick={addPasskey}>🔑 Geçiş anahtarı oluştur</Button>
          {pkMsg && <div style={{ marginTop: 10, fontSize: 13, color: "var(--text-dim)" }}>{pkMsg}</div>}
        </Card>
      )}

      <Card style={{ padding: 20 }}>
        <Row label="Hesap türü" value={providerLabel(profile?.provider)} />
        <Row label="Rol" value={profile?.role ?? "-"} />
        <Row label="Katılım" value={profile?.created_at ? new Date(profile.created_at).toLocaleDateString("tr-TR") : "-"} />
      </Card>
    </div>
  );
}

function providerLabel(p?: string) {
  switch (p) {
    case "google": return "Google";
    case "apple": return "Apple";
    case "password": return "E-posta";
    default: return p || "-";
  }
}

function Row({ label, value }: { label: string; value: string }) {
  return (
    <div style={{ display: "flex", justifyContent: "space-between", padding: "8px 0", fontSize: 14 }}>
      <span style={{ color: "var(--text-dim)" }}>{label}</span>
      <span style={{ fontWeight: 600 }}>{value}</span>
    </div>
  );
}
