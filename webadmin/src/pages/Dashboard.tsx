import { useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import type { EChartsOption } from "echarts";
import { api } from "../api";
import { axisStyle, Chart, chartColors } from "../Chart";
import { compact, shortDate, trDate, tryMoney } from "../format";
import { Icon, type IconName } from "../Icon";
import type { DashboardData } from "../types";
import { EmptyState, LoadingBlock, StatusPill } from "../UI";

export function Dashboard() {
  const [days, setDays] = useState(30);
  const { data, isLoading, error } = useQuery({
    queryKey: ["admin-dashboard", days],
    queryFn: () => api<DashboardData>(`/admin/dashboard?days=${days}`),
    refetchInterval: 30_000,
  });

  const revenueOption = useMemo<EChartsOption>(() => ({
    color: [chartColors[0]],
    animationDuration: 500,
    tooltip: {
      trigger: "axis",
      backgroundColor: "#13171d",
      borderColor: "#2a3039",
      textStyle: { color: "#e9edf4", fontSize: 11 },
      formatter: (params: unknown) => {
        const item = (params as { axisValue: string; value: number; data?: { count?: number } }[])[0];
        return `${item.axisValue}<br/><strong>${tryMoney(Number(item.value))}</strong>`;
      },
    },
    grid: { left: 15, right: 16, top: 25, bottom: 22, containLabel: true },
    xAxis: { type: "category", boundaryGap: false, data: data?.revenue.series.map((x) => shortDate(x.time)) || [], ...axisStyle },
    yAxis: { type: "value", axisLabel: { color: "#737b8d", fontSize: 10, formatter: (v: number) => compact(v) }, splitLine: axisStyle.splitLine },
    series: [{
      type: "line", smooth: 0.32, symbol: "none", lineStyle: { width: 2.2 },
      areaStyle: {
        color: {
          type: "linear", x: 0, y: 0, x2: 0, y2: 1,
          colorStops: [{ offset: 0, color: "rgba(57,198,216,.28)" }, { offset: 1, color: "rgba(57,198,216,0)" }],
        },
      },
      data: data?.revenue.series.map((x) => x.value) || [],
    }],
  }), [data]);

  const activityOption = useMemo<EChartsOption>(() => ({
    color: [chartColors[1]],
    tooltip: { trigger: "axis", backgroundColor: "#13171d", borderColor: "#2a3039", textStyle: { color: "#e9edf4" } },
    grid: { left: 8, right: 12, top: 12, bottom: 18, containLabel: true },
    xAxis: { type: "category", boundaryGap: false, data: data?.activity_series.map((x) => shortDate(x.time)) || [], ...axisStyle },
    yAxis: { type: "value", axisLabel: { color: "#737b8d", fontSize: 10 }, splitLine: axisStyle.splitLine },
    series: [{ type: "line", smooth: true, symbol: "none", lineStyle: { width: 2 }, data: data?.activity_series.map((x) => x.value) || [] }],
  }), [data]);

  const donut = (values: { name: string; value: number }[] | undefined): EChartsOption => ({
    color: chartColors,
    tooltip: { trigger: "item", backgroundColor: "#13171d", borderColor: "#2a3039", textStyle: { color: "#e9edf4" } },
    legend: { bottom: 0, icon: "circle", itemWidth: 7, itemHeight: 7, textStyle: { color: "#8e96a8", fontSize: 10 } },
    series: [{
      type: "pie", radius: ["55%", "75%"], center: ["50%", "43%"],
      itemStyle: { borderColor: "#101318", borderWidth: 3, borderRadius: 4 },
      label: { show: false }, data: values || [],
    }],
  });

  if (isLoading) return <LoadingBlock height={560} />;
  if (error || !data) return <EmptyState title="Dashboard verisi alınamadı" text="API ve veritabanı bağlantısını kontrol edin." />;

  const cards: { label: string; value: string; meta: string; icon: IconName; tone: string }[] = [
    { label: "Toplam kullanıcı", value: compact(data.totals.users), meta: `+${data.totals.new_users} bu dönemde`, icon: "users", tone: "cyan" },
    { label: "Anlık çevrimiçi", value: compact(data.totals.online_users), meta: "Tüm podlar", icon: "online", tone: "green" },
    { label: "Aktif abonelik", value: compact(data.totals.active_subscriptions), meta: `${Math.round((data.totals.active_subscriptions / Math.max(data.totals.users, 1)) * 100)}% dönüşüm`, icon: "package", tone: "violet" },
    { label: "30 günlük tahmini brüt", value: tryMoney(data.revenue.monthly.gross_try), meta: `${data.revenue.monthly.sales} doğrulanmış işlem`, icon: "revenue", tone: "amber" },
  ];

  return (
    <div className="dashboard-grid">
      <div className="range-switch">
        {[7, 30, 90, 365].map((value) => <button key={value} className={days === value ? "active" : ""} onClick={() => setDays(value)}>{value === 365 ? "1 yıl" : `${value} gün`}</button>)}
      </div>
      <section className="metric-grid">
        {cards.map((card) => <article className="metric-card" key={card.label}>
          <div className={`metric-icon ${card.tone}`}><Icon name={card.icon} /></div>
          <span>{card.label}</span><strong>{card.value}</strong><small>{card.meta}</small>
          <div className="metric-spark">{[32, 46, 39, 58, 51, 72, 66, 84].map((h, i) => <i key={i} style={{ height: `${h}%` }} />)}</div>
        </article>)}
      </section>
      <section className="panel revenue-main">
        <div className="panel-heading"><div><span className="kicker">GELİR PERFORMANSI</span><h2>Tahmini brüt mağaza geliri</h2></div><span className="panel-chip">TRY · doğrulanmış</span></div>
        <Chart option={revenueOption} height={310} />
        <p className="data-notice">{data.revenue.notice}</p>
      </section>
      <section className="panel distribution-panel">
        <div className="panel-heading"><div><span className="kicker">KULLANICI TABANI</span><h2>Platform dağılımı</h2></div></div>
        <Chart option={donut(data.platform_distribution)} height={290} />
      </section>
      <section className="panel activity-panel">
        <div className="panel-heading"><div><span className="kicker">ÜRÜN ETKİLEŞİMİ</span><h2>Günlük aktivite</h2></div><span className="panel-chip">{data.totals.ai_research_uses} AI araştırması</span></div>
        <Chart option={activityOption} height={260} />
      </section>
      <section className="panel plans-panel">
        <div className="panel-heading"><div><span className="kicker">ABONELİKLER</span><h2>Paket dağılımı</h2></div></div>
        <Chart option={donut(data.plan_distribution)} height={260} />
      </section>
      <section className="panel top-assets">
        <div className="panel-heading"><div><span className="kicker">TAKİP EĞİLİMİ</span><h2>En çok favorilenen varlıklar</h2></div><span className="panel-chip">{data.totals.favorites} favori</span></div>
        <div className="rank-list">
          {data.top_favorite_assets.map((item, index) => <div key={item.name}><span>{String(index + 1).padStart(2, "0")}</span><strong>{item.name}</strong><div><i style={{ width: `${Math.max(8, (item.value / Math.max(data.top_favorite_assets[0]?.value || 1, 1)) * 100)}%` }} /></div><b>{item.value}</b></div>)}
        </div>
      </section>
      <section className="panel online-panel">
        <div className="panel-heading"><div><span className="kicker">CANLI</span><h2>Şu anda uygulamada</h2></div><span className="live-label"><i /> {data.totals.online_users} çevrimiçi</span></div>
        {data.recently_online.length ? <div className="compact-table">
          {data.recently_online.map((user) => <div key={user.user_id}><span className="table-avatar">{(user.display_name || user.email).slice(0, 2).toUpperCase()}</span><span><strong>{user.display_name || "İsimsiz kullanıcı"}</strong><small>{user.email}</small></span><StatusPill value={user.platform.split("+")[0]} /><span className="muted">{user.connections} oturum</span><span className="muted">{trDate(user.last_seen_at)}</span></div>)}
        </div> : <EmptyState icon="online" title="Aktif oturum yok" text="Kullanıcılar bağlandığında burada anlık görünecek." />}
      </section>
    </div>
  );
}
