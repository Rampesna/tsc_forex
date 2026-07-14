import { useEffect, useState } from "react";
import { NavLink, Outlet, useNavigate } from "react-router-dom";
import { useAuth } from "../auth/AuthContext";
import { useMarkets } from "../store/markets";
import type { NotificationPayload } from "../lib/types";
import { ConnectionDot } from "./common";

const NAV = [
  { to: "/", label: "Piyasalar", icon: "📈", end: true },
  { to: "/favorites", label: "Favoriler", icon: "⭐" },
  { to: "/rewards", label: "Ödüller", icon: "🏆" },
  { to: "/alerts", label: "Alarmlar", icon: "🔔" },
  { to: "/portfolio", label: "Portföy", icon: "💼" },
  { to: "/notifications", label: "Bildirimler", icon: "📨" },
  { to: "/settings", label: "Ayarlar", icon: "⚙️" },
];

interface Toast extends NotificationPayload {
  key: number;
}

export function Layout() {
  const { profile, authed, logout } = useAuth();
  const connected = useMarkets((s) => s.connected);
  const setNotificationHandler = useMarkets((s) => s.setNotificationHandler);
  const [toasts, setToasts] = useState<Toast[]>([]);
  const nav = useNavigate();

  useEffect(() => {
    setNotificationHandler((n) => {
      const key = Date.now() + Math.random();
      setToasts((t) => [...t, { ...n, key }]);
      setTimeout(() => setToasts((t) => t.filter((x) => x.key !== key)), 6000);
    });
  }, [setNotificationHandler]);

  return (
    <div style={{ display: "flex", minHeight: "100vh" }}>
      {/* Sidebar */}
      <aside
        style={{
          width: 232,
          borderRight: "1px solid var(--stroke)",
          padding: "20px 14px",
          position: "sticky",
          top: 0,
          height: "100vh",
          display: "flex",
          flexDirection: "column",
          gap: 4,
          background: "var(--bg-elev)",
        }}
        className="tsc-sidebar"
      >
        <div style={{ display: "flex", alignItems: "center", gap: 10, padding: "4px 8px 18px" }}>
          <img src="/logo.png" width={34} height={34} style={{ borderRadius: 9 }} alt="" />
          <div>
            <div style={{ fontWeight: 800, fontSize: 16 }}>TSC Forex</div>
            <div style={{ fontSize: 11, color: "var(--text-faint)" }}>Canlı Piyasa</div>
          </div>
        </div>

        {NAV.map((n) => (
          <NavLink
            key={n.to}
            to={n.to}
            end={n.end}
            style={({ isActive }) => ({
              display: "flex",
              alignItems: "center",
              gap: 12,
              padding: "11px 12px",
              borderRadius: 10,
              fontSize: 14,
              fontWeight: 600,
              color: isActive ? "var(--accent-hi)" : "var(--text-dim)",
              background: isActive ? "var(--accent-soft)" : "transparent",
            })}
          >
            <span style={{ fontSize: 16 }}>{n.icon}</span>
            {n.label}
          </NavLink>
        ))}

        <div style={{ marginTop: "auto" }}>
          {!authed ? (
            <button
              onClick={() => nav("/login")}
              style={{
                width: "100%",
                background: "var(--accent)",
                color: "#04222a",
                border: "none",
                borderRadius: 12,
                padding: "12px 14px",
                fontSize: 14,
                fontWeight: 800,
                cursor: "pointer",
              }}
            >
              Giriş yap / Kayıt ol
            </button>
          ) : (
          <>
          <button
            onClick={() => nav("/profile")}
            style={{
              display: "flex",
              alignItems: "center",
              gap: 10,
              width: "100%",
              background: "var(--surface)",
              border: "1px solid var(--stroke)",
              borderRadius: 12,
              padding: 10,
              cursor: "pointer",
              color: "var(--text)",
              textAlign: "left",
            }}
          >
            {profile?.avatar_url ? (
              <img
                src={profile.avatar_url}
                width={32}
                height={32}
                style={{ borderRadius: 16, objectFit: "cover" }}
                alt=""
              />
            ) : (
              <div
                style={{
                  width: 32,
                  height: 32,
                  borderRadius: 16,
                  background: "var(--accent-soft)",
                  color: "var(--accent-hi)",
                  display: "flex",
                  alignItems: "center",
                  justifyContent: "center",
                  fontWeight: 700,
                }}
              >
                {(profile?.display_name || profile?.email || "?").slice(0, 1).toUpperCase()}
              </div>
            )}
            <div style={{ overflow: "hidden" }}>
              <div style={{ fontSize: 13, fontWeight: 600, whiteSpace: "nowrap", textOverflow: "ellipsis", overflow: "hidden" }}>
                {profile?.display_name || "Profil"}
              </div>
              <div style={{ fontSize: 11, color: "var(--text-faint)", whiteSpace: "nowrap", textOverflow: "ellipsis", overflow: "hidden" }}>
                {profile?.email}
              </div>
            </div>
          </button>
          <button
            onClick={logout}
            style={{
              marginTop: 8,
              width: "100%",
              background: "transparent",
              border: "none",
              color: "var(--text-faint)",
              fontSize: 12,
              cursor: "pointer",
              padding: 6,
            }}
          >
            Çıkış yap
          </button>
          </>
          )}
        </div>
      </aside>

      {/* Main */}
      <main style={{ flex: 1, minWidth: 0, display: "flex", flexDirection: "column" }}>
        <header
          style={{
            height: 56,
            borderBottom: "1px solid var(--stroke)",
            display: "flex",
            alignItems: "center",
            justifyContent: "flex-end",
            padding: "0 24px",
            gap: 16,
            position: "sticky",
            top: 0,
            background: "rgba(11,14,17,0.8)",
            backdropFilter: "blur(10px)",
            zIndex: 10,
          }}
        >
          <ConnectionDot connected={connected} />
        </header>
        <div style={{ flex: 1, padding: 24, maxWidth: 1100, width: "100%", margin: "0 auto" }}>
          <Outlet />
        </div>
      </main>

      {/* Toasts */}
      <div style={{ position: "fixed", right: 20, bottom: 20, display: "flex", flexDirection: "column", gap: 10, zIndex: 100 }}>
        {toasts.map((t) => (
          <div
            key={t.key}
            onClick={() => nav("/notifications")}
            style={{
              background: "var(--surface-hi)",
              border: "1px solid var(--stroke-strong)",
              borderRadius: 12,
              padding: "12px 16px",
              minWidth: 260,
              maxWidth: 340,
              boxShadow: "var(--shadow)",
              cursor: "pointer",
            }}
          >
            <div style={{ fontWeight: 700, fontSize: 14, color: "var(--accent-hi)" }}>{t.title}</div>
            <div style={{ fontSize: 13, color: "var(--text-dim)", marginTop: 2 }}>{t.body}</div>
          </div>
        ))}
      </div>

      <style>{`
        @media (max-width: 820px) {
          .tsc-sidebar { position: fixed; z-index: 50; transform: translateX(-100%); transition: transform .2s; }
        }
      `}</style>
    </div>
  );
}
