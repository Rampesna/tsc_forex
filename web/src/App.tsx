import { useEffect } from "react";
import { Navigate, Route, Routes } from "react-router-dom";
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

export default function App() {
  const { ready, authed } = useAuth();
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

  if (!authed) return <Login />;

  return (
    <Routes>
      <Route element={<Layout />}>
        <Route index element={<Markets />} />
        <Route path="asset/:code" element={<AssetDetail />} />
        <Route path="favorites" element={<Favorites />} />
        <Route path="alerts" element={<Alerts />} />
        <Route path="portfolio" element={<Portfolio />} />
        <Route path="notifications" element={<Notifications />} />
        <Route path="profile" element={<Profile />} />
        <Route path="settings" element={<Settings />} />
        <Route path="*" element={<Navigate to="/" replace />} />
      </Route>
    </Routes>
  );
}
