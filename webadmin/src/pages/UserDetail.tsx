import { useState, type ReactNode } from "react";
import { useQuery } from "@tanstack/react-query";
import { useNavigate, useParams } from "../router";
import { api } from "../api";
import { planName, platformName, trDate } from "../format";
import { Icon, type IconName } from "../Icon";
import type { UserDetail } from "../types";
import { EmptyState, LoadingBlock, StatusPill } from "../UI";

const tabs = [
  ["overview", "Genel Bakış"], ["favorites", "Favoriler"], ["alerts", "Alarmlar"],
  ["badges", "Rozetler"], ["tasks", "Görevler & Kullanım"], ["portfolio", "Portföy"], ["activity", "Aktivite"],
] as const;

export function UserDetailPage() {
  const { id = "" } = useParams();
  const navigate = useNavigate();
  const [tab, setTab] = useState<(typeof tabs)[number][0]>("overview");
  const { data, isLoading } = useQuery({
    queryKey: ["admin-user", id],
    queryFn: () => api<UserDetail>(`/admin/users/${id}`),
    enabled: Boolean(id),
  });
  if (isLoading) return <LoadingBlock height={520} />;
  if (!data) return <EmptyState icon="users" title="Kullanıcı bulunamadı" text="Bu hesap silinmiş veya erişilebilir değil." />;
  const user = data.user;
  const learningProgress = data.learning_progress ?? [];
  const disciplineReviews = data.discipline_reviews ?? [];

  return (
    <div className="user-detail-page">
      <section className="user-hero panel">
        <button className="back-button" onClick={() => navigate("/users")}>←</button>
        <span className="profile-avatar">{(user.display_name || user.email).slice(0, 2).toUpperCase()}</span>
        <div className="profile-copy"><div><h2>{user.display_name || "İsimsiz kullanıcı"}</h2>{user.online && <span className="live-label"><i /> Çevrimiçi</span>}</div><p>{user.email}</p><div><StatusPill value={user.device_platform} /><StatusPill value={user.plan_code} /><span className="tag">{user.provider}</span></div></div>
        <div className="hero-facts"><span><small>Kayıt tarihi</small><strong>{trDate(user.created_at)}</strong></span><span><small>Son aktivite</small><strong>{trDate(user.last_active_at)}</strong></span><span><small>Satın alım kanalı</small><strong>{user.purchase_platform ? platformName(user.purchase_platform) : "Yok"}</strong></span></div>
      </section>
      <div className="detail-tabs">{tabs.map(([value, label]) => <button className={tab === value ? "active" : ""} onClick={() => setTab(value)} key={value}>{label}</button>)}</div>

      {tab === "overview" && <div className="detail-overview">
        <section className="metric-grid compact">
          {([
            ["Favori", user.favorites, "favorite"],
            ["Aktif alarm", user.active_alerts, "alert"],
            ["Rozet", user.badges, "badge"],
            ["AI araştırması", user.ai_research_uses, "sparkles"],
          ] as [string, number, IconName][]).map(([label, value, icon]) => <article className="metric-card" key={label}><div className="metric-icon cyan"><Icon name={icon} /></div><span>{label}</span><strong>{value}</strong><small>Tüm zamanlar</small></article>)}
        </section>
        <section className="panel account-overview">
          <div className="panel-heading"><div><span className="kicker">HESAP ÖZETİ</span><h2>Ürün yolculuğu</h2></div></div>
          <div className="info-grid">
            <div><span>Paket</span><strong>{planName(user.plan_code)}</strong><small>{user.subscription_expiry ? `${trDate(user.subscription_expiry)} tarihine kadar` : "Aktif ücretli paket yok"}</small></div>
            <div><span>Cihaz erişimi</span><strong>{platformName(user.device_platform)}</strong><small>{data.devices.length} kayıtlı bildirim cihazı</small></div>
            <div><span>Seviye & XP</span><strong>Seviye {data.gamification?.level || 1}</strong><small>{data.gamification?.xp || 0} toplam XP</small></div>
            <div><span>Günlük seri</span><strong>{data.gamification?.current_streak || 0} gün</strong><small>En uzun: {data.gamification?.longest_streak || 0} gün</small></div>
          </div>
        </section>
        <section className="panel">
          <div className="panel-heading"><div><span className="kicker">CİHAZLAR</span><h2>Bildirim uçları</h2></div></div>
          {!data.devices.length ? <EmptyState title="Kayıtlı cihaz yok" text="Kullanıcı bildirim izni verdiğinde cihazı burada görünür." /> : <div className="card-list">{data.devices.map((device) => <div key={device.id}><span className="list-icon"><Icon name="device" /></span><span><strong>{device.device_model || platformName(device.platform)}</strong><small>{device.app_version ? `v${device.app_version}` : "Sürüm bilinmiyor"} · {device.locale || "dil bilinmiyor"}</small></span><StatusPill value={device.platform} /><small>{trDate(device.last_seen_at)}</small></div>)}</div>}
        </section>
      </div>}

      {tab === "favorites" && <DataSection title="Favori varlıklar" kicker={`${data.favorites.length} KAYIT`} empty="Henüz favori varlık yok">{data.favorites.map((item) => <div className="detail-row" key={item.id}><span className="list-icon favorite"><Icon name="favorite" /></span><span><strong>{item.code}</strong><small>Takip listesine eklendi</small></span><time>{trDate(item.created_at)}</time></div>)}</DataSection>}
      {tab === "alerts" && <DataSection title="Fiyat alarmları" kicker={`${data.alerts.length} KAYIT`} empty="Henüz fiyat alarmı yok">{data.alerts.map((item) => <div className="detail-row" key={item.id}><span className="list-icon amber"><Icon name="alert" /></span><span><strong>{item.code} · {item.direction === "above" ? "Üzerine çıkınca" : "Altına inince"}</strong><small>Eşik: {item.threshold.toLocaleString("tr-TR")}</small></span><StatusPill value={item.active ? "active" : "completed"} /><time>{trDate(item.created_at)}</time></div>)}</DataSection>}
      {tab === "badges" && <DataSection title="Kazanılan rozetler" kicker={`${data.badges.length} ROZET`} empty="Henüz kazanılmış rozet yok">{data.badges.map((item) => <div className="detail-row" key={item.id}><span className="list-icon violet"><Icon name="badge" /></span><span><strong>{item.code.replaceAll("_", " ")}</strong><small>Başarı rozeti</small></span><time>{trDate(item.earned_at)}</time></div>)}</DataSection>}
      {tab === "tasks" && <div className="two-column">
        <DataSection title="Günlük plan geçmişi" kicker={`${data.daily_plans.length} PLAN`} empty="Plan kaydı yok">{data.daily_plans.map((item) => <div className="detail-row" key={item.id}><span><strong>{item.day_key} · {item.risk_appetite}</strong><small>{item.learning_goal}{item.reflection ? ` — ${item.reflection}` : ""}</small></span><time>{trDate(item.created_at)}</time></div>)}</DataSection>
        <DataSection title="Özellik kullanımı" kicker={`${data.feature_usage.length} İŞLEM`} empty="Kullanım kaydı yok">{data.feature_usage.slice(0, 50).map((item) => <div className="detail-row" key={item.id}><span><strong>{item.feature.replaceAll("_", " ")}</strong><small>{item.scope} · {item.period_key}</small></span><b>×{item.quantity}</b></div>)}</DataSection>
        <DataSection title="Öğrenme ilerlemesi" kicker={`${learningProgress.length} MODÜL`} empty="Öğrenme ilerlemesi yok">{learningProgress.map((item) => <div className="detail-row" key={item.id}><span><strong>{item.lesson_code.replaceAll("_", " ")}</strong><small>{item.correct}/{item.attempts} doğru · {item.completed_at ? "Tamamlandı" : "Devam ediyor"}</small></span><b>%{Math.max(0, Math.min(100, item.mastery))}</b></div>)}</DataSection>
        <DataSection title="Karar günlüğü" kicker={`${disciplineReviews.length} KAYIT`} empty="Karar günlüğü kaydı yok">{disciplineReviews.map((item) => <div className="detail-row" key={item.id}><span><strong>{item.code.replaceAll("_", " ")}</strong><small>{item.note}</small></span><time>{item.day_key}</time></div>)}</DataSection>
      </div>}
      {tab === "portfolio" && <div className="two-column">
        <DataSection title="Portföyler" kicker={`${data.portfolios.length} PORTFÖY`} empty="Portföy bulunmuyor">{data.portfolios.map((item) => <div className="detail-row" key={item.id}><span className="list-icon"><Icon name="portfolio" /></span><span><strong>{item.name}</strong><small>Baz para: {item.base_currency}</small></span><time>{trDate(item.updated_at)}</time></div>)}</DataSection>
        <DataSection title="Pozisyonlar" kicker={`${data.positions.length} VARLIK`} empty="Açık pozisyon yok">{data.positions.map((item) => <div className="detail-row" key={`${item.portfolio_id}-${item.code}`}><span><strong>{item.code}</strong><small>{String(item.quantity)} adet · Ort. ₺{String(item.average_cost_try)}</small></span><b>₺{String(item.realized_pnl_try)}</b></div>)}</DataSection>
      </div>}
      {tab === "activity" && <DataSection title="Son kullanıcı aktiviteleri" kicker={`${data.activity.length} OLAY`} empty="Aktivite kaydı yok">{data.activity.map((item, index) => <div className="timeline-row" key={`${item.created_at}-${index}`}><i /><span><strong>{item.label}</strong><small>{item.reference || item.type}</small></span><time>{trDate(item.created_at)}</time></div>)}</DataSection>}
    </div>
  );
}

function DataSection({ title, kicker, empty, children }: { title: string; kicker: string; empty: string; children: ReactNode }) {
  const hasChildren = Array.isArray(children) ? children.length > 0 : Boolean(children);
  return <section className="panel detail-section"><div className="panel-heading"><div><span className="kicker">{kicker}</span><h2>{title}</h2></div></div>{hasChildren ? <div className="detail-list">{children}</div> : <EmptyState title={empty} text="Yeni kayıtlar oluştuğunda burada görünecek." />}</section>;
}
