import { useQuery } from "@tanstack/react-query";
import { useNavigate } from "../router";
import { api } from "../api";
import { platformName, trDate } from "../format";
import { Icon } from "../Icon";
import type { OnlineUser } from "../types";
import { EmptyState, LoadingBlock, StatusPill } from "../UI";

export function Online() {
  const navigate = useNavigate();
  const { data, isLoading, dataUpdatedAt } = useQuery({
    queryKey: ["admin-online"],
    queryFn: () => api<{ items: OnlineUser[]; count: number }>("/admin/online"),
    refetchInterval: 10_000,
  });
  if (isLoading) return <LoadingBlock height={420} />;
  return (
    <div className="online-page">
      <section className="online-summary">
        <div className="presence-radar"><i /><i /><span>{data?.count || 0}</span></div>
        <div><span className="kicker">GLOBAL PRESENCE</span><h2>{data?.count || 0} kullanıcı çevrimiçi</h2><p>Redis kalp atışları sayesinde tüm Kubernetes podları tek görünümde birleştirilir.</p></div>
        <span className="last-refresh"><i /> Son yenileme {trDate(new Date(dataUpdatedAt).toISOString())}</span>
      </section>
      <section className="panel table-panel">
        <div className="panel-heading"><div><span className="kicker">AKTİF OTURUMLAR</span><h2>Canlı kullanıcı listesi</h2></div><span className="live-label"><i /> 10 sn'de bir yenilenir</span></div>
        {!data?.items.length ? <EmptyState icon="online" title="Şu anda çevrimiçi kullanıcı yok" text="Yeni WebSocket bağlantıları anında burada görünür." /> : <div className="data-table online-table">
          <div className="table-head"><span>Kullanıcı</span><span>Platform</span><span>Bağlantı</span><span>Bağlandığı an</span><span>Son kalp atışı</span><span /></div>
          {data.items.map((user) => <button key={user.user_id} className="table-row" onClick={() => navigate(`/users/${user.user_id}`)}>
            <span className="user-cell"><span className="table-avatar live">{(user.display_name || user.email).slice(0, 2).toUpperCase()}</span><span><strong>{user.display_name || "İsimsiz kullanıcı"}<i className="online-dot" /></strong><small>{user.email}</small></span></span>
            <span><StatusPill value={user.platform.split("+")[0]} /><small className="sub-value">{platformName(user.platform)}</small></span>
            <span><strong>{user.connections}</strong><small>aktif kanal</small></span>
            <span><strong>{trDate(user.connected_at)}</strong></span>
            <span><strong>{trDate(user.last_seen_at)}</strong></span>
            <span className="row-arrow"><Icon name="arrow" size={16} /></span>
          </button>)}
        </div>}
      </section>
    </div>
  );
}
