import { useEffect, useState, type ReactNode } from "react";
import { NavLink, useLocation, useNavigate } from "./router";
import { useAuth } from "./auth";
import { Icon } from "./Icon";

const nav = [
  { to: "/", label: "Genel Bakış", icon: "dashboard" as const, end: true },
  { to: "/users", label: "Kullanıcılar", icon: "users" as const },
  { to: "/online", label: "Anlık Kullanıcılar", icon: "online" as const },
  { to: "/revenue", label: "Satış & Gelir", icon: "revenue" as const },
  { to: "/campaigns", label: "Akıllı Bildirimler", icon: "campaign" as const },
  { to: "/audit", label: "Denetim Günlüğü", icon: "audit" as const },
  { to: "/settings", label: "Ayarlar", icon: "settings" as const },
];

const titles: Record<string, { title: string; subtitle: string }> = {
  "/": { title: "Genel Bakış", subtitle: "TSC Forex sistem sağlığı ve büyüme görünümü" },
  "/users": { title: "Kullanıcılar", subtitle: "Hesap, cihaz, paket ve etkileşim yönetimi" },
  "/online": { title: "Anlık Kullanıcılar", subtitle: "Tüm Kubernetes podlarındaki aktif oturumlar" },
  "/revenue": { title: "Satış & Gelir", subtitle: "Doğrulanmış mağaza işlemleri ve paket dağılımı" },
  "/campaigns": { title: "Akıllı Bildirimler", subtitle: "AI destekli, onaylı ve denetlenebilir hedefleme" },
  "/audit": { title: "Denetim Günlüğü", subtitle: "Tüm ayrıcalıklı işlemlerin değişmez zaman çizelgesi" },
  "/settings": { title: "Ayarlar", subtitle: "Yönetici hesabı ve güvenlik ayarları" },
};

export function Layout({ children }: { children: ReactNode }) {
  const { user, signOut } = useAuth();
  const [open, setOpen] = useState(false);
  const location = useLocation();
  const navigate = useNavigate();
  const page =
    location.pathname.startsWith("/users/") ? { title: "Kullanıcı Detayı", subtitle: "Hesabın tüm ürün yolculuğu" } : titles[location.pathname] || titles["/"];

  useEffect(() => {
    const openSearch = (event: KeyboardEvent) => {
      if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === "k") {
        event.preventDefault();
        navigate("/users");
      }
    };
    window.addEventListener("keydown", openSearch);
    return () => window.removeEventListener("keydown", openSearch);
  }, [navigate]);

  async function logout() {
    await signOut();
    navigate("/login", { replace: true });
  }

  return (
    <div className="app-shell">
      <aside className={`sidebar ${open ? "open" : ""}`}>
        <div className="brand">
          <div className="brand-mark"><span>T</span><span>F</span></div>
          <div><strong>TSC Forex</strong><small>Operations</small></div>
          <button className="mobile-close" onClick={() => setOpen(false)} aria-label="Menüyü kapat"><Icon name="close" /></button>
        </div>
        <nav>
          <p>YÖNETİM</p>
          {nav.slice(0, 5).map((item) => (
            <NavLink key={item.to} to={item.to} end={item.end} onClick={() => setOpen(false)}>
              <Icon name={item.icon} /><span>{item.label}</span>
              {item.to === "/online" && <i className="nav-live" />}
            </NavLink>
          ))}
          <p>GÜVENLİK</p>
          {nav.slice(5).map((item) => (
            <NavLink key={item.to} to={item.to} onClick={() => setOpen(false)}>
              <Icon name={item.icon} /><span>{item.label}</span>
            </NavLink>
          ))}
        </nav>
        <div className="sidebar-security">
          <Icon name="shield" size={16} />
          <div><strong>Güvenli yönetim</strong><small>Rol tabanlı erişim · denetimli işlemler</small></div>
        </div>
        <button className="account-card" onClick={logout}>
          <span className="avatar">{(user?.display_name || user?.email || "T").slice(0, 2).toUpperCase()}</span>
          <span><strong>{user?.display_name || "Talha Can"}</strong><small>{user?.email}</small></span>
          <Icon name="logout" size={16} />
        </button>
      </aside>
      {open && <button className="sidebar-scrim" onClick={() => setOpen(false)} aria-label="Menüyü kapat" />}
      <section className="content-shell">
        <header className="topbar">
          <button className="mobile-menu" onClick={() => setOpen(true)} aria-label="Menüyü aç"><Icon name="menu" /></button>
          <button className="top-search" onClick={() => navigate("/users")}><Icon name="search" size={15} /><span>Kullanıcılarda ara…</span><kbd>⌘ K</kbd></button>
          <div className="top-actions">
            <span className="system-state"><i /> Sistem çalışıyor</span>
            <button aria-label="Bildirim kampanyaları" onClick={() => navigate("/campaigns")}><Icon name="bell" size={17} /></button>
            <button className="top-avatar" aria-label="Yönetici ayarları" onClick={() => navigate("/settings")}>{(user?.display_name || "TC").slice(0, 2).toUpperCase()}</button>
          </div>
        </header>
        {user?.must_change_password && location.pathname !== "/settings" && (
          <button className="password-banner" onClick={() => navigate("/settings")}>
            <Icon name="shield" size={16} />
            Başlangıç şifresini değiştirerek yönetici hesabını güvene al.
            <span>Şimdi değiştir <Icon name="arrow" size={14} /></span>
          </button>
        )}
        <main>
          <div className="page-heading">
            <div><h1>{page.title}</h1><p>{page.subtitle}</p></div>
            <span className="heading-date">{new Intl.DateTimeFormat("tr-TR", { dateStyle: "long" }).format(new Date())}</span>
          </div>
          {children}
        </main>
      </section>
    </div>
  );
}
