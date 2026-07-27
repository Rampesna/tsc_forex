import { useState } from "react";
import { keepPreviousData, useQuery } from "@tanstack/react-query";
import { api } from "../api";
import { trDate } from "../format";
import { Icon } from "../Icon";
import type { AuditItem } from "../types";
import { EmptyState, LoadingBlock } from "../UI";

export function Audit() {
  const [page, setPage] = useState(1);
  const { data, isLoading } = useQuery({
    queryKey: ["admin-audit", page],
    queryFn: () => api<{ items: AuditItem[]; total: number; page_size: number }>(`/admin/audit?page=${page}&page_size=30`),
    placeholderData: keepPreviousData,
  });
  if (isLoading) return <LoadingBlock height={500} />;
  return <div className="audit-page">
    <section className="audit-summary">
      <span className="audit-shield"><Icon name="shield" size={23} /></span>
      <div><span className="kicker">DEĞİŞMEZ KAYIT</span><h2>Ayrıcalıklı işlemler izleniyor</h2><p>Girişler, kullanıcı görüntülemeleri, kampanya önizlemeleri, onaylar ve güvenlik değişiklikleri zaman damgasıyla saklanır.</p></div>
      <span className="panel-chip">{data?.total || 0} olay</span>
    </section>
    <section className="panel audit-list">
      <div className="panel-heading"><div><span className="kicker">ZAMAN ÇİZELGESİ</span><h2>Yönetici hareketleri</h2></div></div>
      {!data?.items.length ? <EmptyState title="Denetim kaydı yok" text="İlk yönetici işlemi burada görünecek." /> : data.items.map((item) => <div className="audit-row" key={item.id}>
        <span className={`audit-action ${item.action.includes("failed") ? "failed" : ""}`}><Icon name={item.action.includes("campaign") ? "campaign" : item.action.includes("password") || item.action.includes("login") ? "shield" : "users"} size={16} /></span>
        <span><strong>{humanAction(item.action)}</strong><small>{item.resource_type}{item.resource_id ? ` · ${item.resource_id}` : ""}</small></span>
        <code>{Object.keys(item.metadata || {}).length ? JSON.stringify(item.metadata) : "—"}</code>
        <span><strong>{item.ip_address || "—"}</strong><small>{trimAgent(item.user_agent)}</small></span>
        <time>{trDate(item.created_at)}</time>
      </div>)}
      {(data?.total || 0) > (data?.page_size || 30) && <div className="pagination"><span>{data?.total} olay</span><div><button disabled={page === 1} onClick={() => setPage((x) => x - 1)}>Önceki</button><b>{page}</b><button disabled={page * (data?.page_size || 30) >= (data?.total || 0)} onClick={() => setPage((x) => x + 1)}>Sonraki</button></div></div>}
    </section>
  </div>;
}

function humanAction(value: string) {
  const labels: Record<string, string> = {
    "admin.login": "Yönetici oturumu açıldı",
    "admin.login_failed": "Başarısız yönetici girişi",
    "admin.password_changed": "Yönetici şifresi değiştirildi",
    "admin.user_viewed": "Kullanıcı detayı görüntülendi",
    "campaign.previewed": "Kampanya önizlemesi oluşturuldu",
    "campaign.reopened": "Kampanya taslağı yeniden açıldı",
    "campaign.approved": "Kampanya onaylandı",
    "campaign.cancelled": "Kampanya iptal edildi",
  };
  return labels[value] || value;
}

function trimAgent(value?: string) {
  if (!value) return "İstemci bilinmiyor";
  if (value.length < 52) return value;
  return `${value.slice(0, 49)}…`;
}
