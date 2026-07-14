import { useEffect, type ReactElement } from "react";
import { Navigate, Route, Routes, useNavigate } from "react-router-dom";
import { useAuth } from "./auth/AuthContext";
import { useMarkets } from "./store/markets";
import { Layout } from "./components/Layout";
import { Login } from "./pages/Login";
import { Markets } from "./pages/Markets";
import { AssetDetail } from "./pages/AssetDetail";
import { Favorites } from "./pages/Favorites";
import { Alerts } from "./pages/Alerts";
import { Portfolio } from "./pages/Portfolio";
import { Notifications } from "./pages/Notifications";
import { Profile } from "./pages/Profile";
import { Settings } from "./pages/Settings";
import { Rewards } from "./pages/Rewards";
import { Privacy, Support, Terms } from "./pages/Legal";

// Market data (list + detail) is public — no login required to browse. Personal
// features (favorites, portfolio, alerts, rewards, profile) are gated behind a
// friendly sign-in prompt, Binance-style.
function RequireAuth({ children, title }: { children: ReactElement; title: string }) {
  const { authed } = useAuth();
  const nav = useNavigate();
  if (authed) return children;
  return (
    <div style={{ maxWidth: 460, margin: "8vh auto 0", textAlign: "center" }}>
      <div
        style={{
          background: "var(--surface)",
          border: "1px solid var(--stroke)",
          borderRadius: "var(--radius)",
          padding: "40px 28px",
        }}
      >
        <div style={{ fontSize: 42, marginBottom: 14 }}>🔐</div>
        <h2 style={{ fontSize: 20, fontWeight: 800, margin: "0 0 8px" }}>{title}</h2>
        <p style={{ color: "var(--text-dim)", fontSize: 14, margin: "0 0 22px", lineHeight: 1.5 }}>
          Bu özelliği kullanmak için ücretsiz TSC Forex hesabınla giriş yap. Piyasaları giriş
          yapmadan da izleyebilirsin.
        </p>
        <button
          onClick={() => nav("/login")}
          style={{
            background: "var(--accent)",
            color: "#04222a",
            border: "none",
            borderRadius: 12,
            padding: "13px 28px",
            fontSize: 15,
            fontWeight: 800,
            cursor: "pointer",
          }}
        >
          Giriş yap / Kayıt ol
        </button>
      </div>
    </div>
  );
}

export default function App() {
  const { ready } = useAuth();
  const bootstrap = useMarkets((s) => s.bootstrap);

  useEffect(() => {
    bootstrap();
  }, [bootstrap]);

  if (!ready) {
    return (
      <div style={{ height: "100vh", display: "flex", alignItems: "center", justifyContent: "center" }}>
        <span className="spinner light" style={{ width: 28, height: 28 }} />
      </div>
    );
  }

  return (
    <Routes>
      <Route path="/login" element={<Login />} />
      <Route path="/privacy" element={<Privacy />} />
      <Route path="/terms" element={<Terms />} />
      <Route path="/support" element={<Support />} />
      <Route element={<Layout />}>
        <Route index element={<Markets />} />
        <Route path="asset/:code" element={<AssetDetail />} />
        <Route path="rewards" element={<RequireAuth title="Ödüller ve seriler"><Rewards /></RequireAuth>} />
        <Route path="favorites" element={<RequireAuth title="Favorilerin"><Favorites /></RequireAuth>} />
        <Route path="alerts" element={<RequireAuth title="Fiyat alarmların"><Alerts /></RequireAuth>} />
        <Route path="portfolio" element={<RequireAuth title="Portföyün"><Portfolio /></RequireAuth>} />
        <Route path="notifications" element={<RequireAuth title="Bildirimlerin"><Notifications /></RequireAuth>} />
        <Route path="profile" element={<RequireAuth title="Profilin"><Profile /></RequireAuth>} />
        <Route path="settings" element={<Settings />} />
        <Route path="*" element={<Navigate to="/" replace />} />
      </Route>
    </Routes>
  );
}
