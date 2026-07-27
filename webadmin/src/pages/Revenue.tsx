import { useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import type { EChartsOption } from "echarts";
import { api } from "../api";
import { axisStyle, Chart, chartColors } from "../Chart";
import { compact, shortDate, tryMoney } from "../format";
import { Icon } from "../Icon";
import type { DashboardData } from "../types";
import { LoadingBlock } from "../UI";

export function Revenue() {
  const [days, setDays] = useState(90);
  const { data, isLoading } = useQuery({
    queryKey: ["admin-revenue", days],
    queryFn: () => api<DashboardData>(`/admin/dashboard?days=${days}`),
  });
  const line = useMemo<EChartsOption>(() => ({
    color: [chartColors[0], chartColors[1]],
    tooltip: { trigger: "axis", backgroundColor: "#13171d", borderColor: "#2a3039", textStyle: { color: "#e9edf4" } },
    legend: { top: 0, right: 0, textStyle: { color: "#8e96a8" }, data: ["Tahmini brüt", "İşlem"] },
    grid: { left: 12, right: 12, top: 42, bottom: 20, containLabel: true },
    xAxis: { type: "category", data: data?.revenue.series.map((x) => shortDate(x.time)) || [], ...axisStyle },
    yAxis: [
      { type: "value", axisLabel: { color: "#737b8d", formatter: (v: number) => compact(v) }, splitLine: axisStyle.splitLine },
      { type: "value", axisLabel: { color: "#737b8d" }, splitLine: { show: false } },
    ],
    series: [
      { name: "Tahmini brüt", type: "bar", barMaxWidth: 18, itemStyle: { borderRadius: [4, 4, 0, 0] }, data: data?.revenue.series.map((x) => x.value) || [] },
      { name: "İşlem", type: "line", yAxisIndex: 1, smooth: true, symbolSize: 5, data: data?.revenue.series.map((x) => x.count || 0) || [] },
    ],
  }), [data]);
  const sales = useMemo<EChartsOption>(() => ({
    color: chartColors,
    tooltip: { trigger: "axis", axisPointer: { type: "shadow" }, backgroundColor: "#13171d", borderColor: "#2a3039", textStyle: { color: "#e9edf4" } },
    grid: { left: 10, right: 15, top: 10, bottom: 10, containLabel: true },
    xAxis: { type: "value", ...axisStyle },
    yAxis: { type: "category", data: data?.sales_by_plan.map((x) => x.name) || [], ...axisStyle },
    series: [{ type: "bar", barWidth: 17, itemStyle: { borderRadius: [0, 5, 5, 0] }, data: data?.sales_by_plan.map((x, i) => ({ value: x.value, itemStyle: { color: chartColors[i % chartColors.length] } })) || [] }],
  }), [data]);
  if (isLoading || !data) return <LoadingBlock height={580} />;
  const windows = [data.revenue.daily, data.revenue.weekly, data.revenue.monthly];
  return <div className="revenue-page">
    <div className="range-switch">{[30, 90, 365].map((value) => <button className={days === value ? "active" : ""} onClick={() => setDays(value)} key={value}>{value === 365 ? "1 yıl" : `${value} gün`}</button>)}</div>
    <section className="metric-grid revenue-windows">{windows.map((item, index) => <article className="metric-card" key={item.label}><div className={`metric-icon ${["cyan", "violet", "amber"][index]}`}><Icon name="revenue" /></div><span>{item.label}</span><strong>{tryMoney(item.gross_try)}</strong><small>{item.sales} doğrulanmış işlem · tahmini brüt</small></article>)}</section>
    <section className="panel revenue-wide"><div className="panel-heading"><div><span className="kicker">ZAMAN SERİSİ</span><h2>Gelir ve işlem hacmi</h2></div><span className="panel-chip">TRY</span></div><Chart option={line} height={380} /><p className="data-notice">{data.revenue.notice}</p></section>
    <section className="panel revenue-breakdown"><div className="panel-heading"><div><span className="kicker">ÜRÜN KARMASI</span><h2>Satılan paketler</h2></div></div><Chart option={sales} height={300} /></section>
    <section className="panel revenue-explain"><div className="panel-heading"><div><span className="kicker">MUHASEBE NOTU</span><h2>Bu rakam neyi gösteriyor?</h2></div></div><div className="explain-grid"><div><Icon name="check" /><span><strong>Dahil</strong><small>Sunucuda doğrulanmış App Store ve Google Play production işlemleri</small></span></div><div><Icon name="package" /><span><strong>Hesaplama</strong><small>İşlem anındaki paket katalog fiyatlarının TRY toplamı</small></span></div><div><Icon name="alert" /><span><strong>Hariç</strong><small>Vergi, mağaza komisyonu, kur farkı, dönemsel fiyat ve net banka ödemesi</small></span></div></div></section>
  </div>;
}
