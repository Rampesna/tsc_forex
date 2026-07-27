import { useDeferredValue, useState } from "react";
import { keepPreviousData, useQuery } from "@tanstack/react-query";
import { useNavigate } from "../router";
import { api } from "../api";
import { planName, platformName, trDate } from "../format";
import { Icon } from "../Icon";
import type { UsersPage } from "../types";
import { EmptyState, LoadingBlock, StatusPill } from "../UI";

export function Users() {
  const navigate = useNavigate();
  const [search, setSearch] = useState("");
  const deferredSearch = useDeferredValue(search);
  const [plan, setPlan] = useState("");
  const [platform, setPlatform] = useState("");
  const [page, setPage] = useState(1);
  const params = new URLSearchParams({
    page: String(page), page_size: "20", search: deferredSearch, plan, platform,
  });
  const { data, isLoading } = useQuery({
    queryKey: ["admin-users", deferredSearch, plan, platform, page],
    queryFn: () => api<UsersPage>(`/admin/users?${params}`),
    placeholderData: keepPreviousData,
  });

  return (
    <section className="panel table-panel">
      <div className="table-toolbar">
        <label className="search-field"><Icon name="search" size={16} /><input autoFocus value={search} onChange={(e) => { setSearch(e.target.value); setPage(1); }} placeholder="İsim veya e-posta ile ara…" /></label>
        <div className="filter-row">
          <select value={platform} onChange={(e) => { setPlatform(e.target.value); setPage(1); }}><option value="">Tüm platformlar</option><option value="ios">iOS</option><option value="android">Android</option></select>
          <select value={plan} onChange={(e) => { setPlan(e.target.value); setPage(1); }}><option value="">Tüm paketler</option><option value="free">Ücretsiz</option><option value="plus">Başlangıç</option><option value="pro">Pro</option><option value="expert">Uzman</option><option value="elite">Elite</option></select>
          <span className="result-count">{data?.total ?? 0} kullanıcı</span>
        </div>
      </div>
      {isLoading ? <LoadingBlock height={420} /> : !data?.items.length ? <EmptyState icon="users" title="Kullanıcı bulunamadı" text="Arama veya filtreleri değiştirerek tekrar deneyin." /> : (
        <>
          <div className="data-table users-table">
            <div className="table-head"><span>Kullanıcı</span><span>Platform</span><span>Paket & mağaza</span><span>Ürün kullanımı</span><span>Son aktivite</span><span /></div>
            {data.items.map((user) => (
              <button className="table-row" key={user.id} onClick={() => navigate(`/users/${user.id}`)}>
                <span className="user-cell"><span className="table-avatar">{(user.display_name || user.email).slice(0, 2).toUpperCase()}</span><span><strong>{user.display_name || "İsimsiz kullanıcı"}{user.online && <i className="online-dot" />}</strong><small>{user.email}</small></span></span>
                <span><StatusPill value={user.device_platform} /><small className="sub-value">{platformName(user.device_platform)}</small></span>
                <span><strong>{planName(user.plan_code)}</strong><small>{user.purchase_platform ? platformName(user.purchase_platform) : "Satın alım yok"}</small></span>
                <span className="usage-mini"><b>{user.favorites}<small>favori</small></b><b>{user.active_alerts}<small>alarm</small></b><b>{user.badges}<small>rozet</small></b><b>{user.ai_research_uses}<small>AI</small></b></span>
                <span><strong>{trDate(user.last_active_at)}</strong><small>Kayıt: {trDate(user.created_at)}</small></span>
                <span className="row-arrow"><Icon name="arrow" size={16} /></span>
              </button>
            ))}
          </div>
          <div className="pagination"><span>{data.total} kayıttan {(page - 1) * data.page_size + 1}–{Math.min(page * data.page_size, data.total)}</span><div><button disabled={page === 1} onClick={() => setPage((x) => x - 1)}>Önceki</button><b>{page}</b><button disabled={page * data.page_size >= data.total} onClick={() => setPage((x) => x + 1)}>Sonraki</button></div></div>
        </>
      )}
    </section>
  );
}
