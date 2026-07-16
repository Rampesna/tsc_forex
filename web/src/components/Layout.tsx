import { useEffect, useState } from "react";
import { Link, NavLink, Outlet, useNavigate } from "react-router-dom";
import { useAuth } from "../auth/AuthContext";
import { useMarkets } from "../store/markets";
import type { NotificationPayload } from "../lib/types";
import { ConnectionDot } from "./common";
import { Icon, type IconName } from "./Icon";

const NAV: { to: string; label: string; icon: IconName; end?: boolean; auth?: boolean }[] = [
  { to: "/", label: "Piyasalar", icon: "markets", end: true },
  { to: "/favorites", label: "Favoriler", icon: "star", auth: true },
  { to: "/rewards", label: "Ödüller", icon: "trophy", auth: true },
  { to: "/alerts", label: "Alarmlar", icon: "bell", auth: true },
  { to: "/portfolio", label: "Portföy", icon: "portfolio", auth: true },
  { to: "/notifications", label: "Bildirimler", icon: "inbox", auth: true },
  { to: "/settings", label: "Ayarlar", icon: "settings" },
];

const LEGAL = [
  { to: "/privacy", label: "Gizlilik" },
  { to: "/terms", label: "Kullanım Şartları" },
  { to: "/support", label: "Destek" },
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
          width: 236,
          borderRight: "1px solid var(--stroke)",
          padding: "18px 12px",
          position: "sticky",
          top: 0,
          height: "100vh",
          display: "flex",
          flexDirection: "column",
          gap: 2,
          background: "var(--bg-elev)",
        }}
        className="tsc-sidebar"
      >
        <div style={{ display: "flex", alignItems: "center", gap: 10, padding: "4px 10px 16px" }}>
          <img src="/logo.png" width={28} height={28} style={{ borderRadius: 7 }} alt="" />
          <div style={{ fontWeight: 650, fontSize: 15, letterSpacing: "-0.02em" }}>TSC Forex</div>
        </div>

        {NAV.filter((n) => !n.auth || authed).map((n) => (
          <NavLink
            key={n.to}
            to={n.to}
            end={n.end}
            style={({ isActive }) => ({
              display: "flex",
              alignItems: "center",
              gap: 11,
              padding: "8px 10px",
              borderRadius: 7,
              fontSize: 13.5,
              fontWeight: 500,
              color: isActive ? "var(--text)" : "var(--text-dim)",
              background: isActive ? "var(--surface-hi)" : "transparent",
              transition: "background 0.12s, color 0.12s",
            })}
          >
            {({ isActive }) => (
              <>
                <Icon name={n.icon} size={17} style={{ color: isActive ? "var(--accent-hi)" : "var(--text-faint)" }} />
                {n.label}
              </>
            )}
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
              marginTop: 6,
              display: "flex",
              alignItems: "center",
              justifyContent: "center",
              gap: 7,
              width: "100%",
              background: "transparent",
              border: "none",
              color: "var(--text-faint)",
              fontSize: 12.5,
              cursor: "pointer",
              padding: 7,
            }}
          >
            <Icon name="logout" size={14} /> Çıkış yap
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
            justifyContent: "space-between",
            padding: "0 24px",
            gap: 16,
            position: "sticky",
            top: 0,
            background: "rgba(11,14,17,0.8)",
            backdropFilter: "blur(10px)",
            zIndex: 10,
          }}
        >
          <div style={{ display: "flex", alignItems: "center", gap: 9, color: "var(--text-dim)", fontSize: 12.5 }}>
            <span style={{ color: "var(--text)", fontWeight: 650 }}>TSC</span>
            <span style={{ color: "var(--text-faint)" }}>/</span>
            Piyasa terminali
          </div>
          <ConnectionDot connected={connected} />
        </header>
        <div className="tsc-main-content" style={{ flex: 1, padding: "30px 32px 42px", maxWidth: 1460, width: "100%", margin: "0 auto" }}>
          <Outlet />
        </div>
        <footer
          style={{
            borderTop: "1px solid var(--stroke)",
            padding: "18px 24px",
            display: "flex",
            flexWrap: "wrap",
            alignItems: "center",
            gap: 16,
            fontSize: 12.5,
            color: "var(--text-faint)",
          }}
        >
          <span>© {new Date().getFullYear()} TSC Forex</span>
          <span style={{ color: "var(--stroke-strong)" }}>·</span>
          {LEGAL.map((l) => (
            <Link key={l.to} to={l.to} style={{ color: "var(--text-dim)" }}>
              {l.label}
            </Link>
          ))}
          <span style={{ marginLeft: "auto", color: "var(--text-faint)" }}>
            Fiyatlar bilgilendirme amaçlıdır, yatırım tavsiyesi değildir.
          </span>
        </footer>
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
          .tsc-sidebar { position: fixed !important; z-index: 50; transform: translateX(-100%); transition: transform .2s; }
          .tsc-main-content { padding: 22px 16px 32px !important; }
        }
      `}</style>
    </div>
  );
}
