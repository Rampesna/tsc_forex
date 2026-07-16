import { useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import { useAuth } from "../auth/AuthContext";
import { useMarkets } from "../store/markets";
import { money, tl } from "../lib/format";
import { api } from "../lib/endpoints";
import { ChangePill, AssetBadge } from "../components/common";
import { Icon } from "../components/Icon";
import { Sparkline } from "../components/Sparkline";
import type { InstrumentPrice, MarketInsight } from "../lib/types";

const CATS: { id: string; label: string; kinds: string[] | null }[] = [
  { id: "all", label: "Tüm piyasalar", kinds: null },
  { id: "gold", label: "Altın", kinds: ["gold"] },
  { id: "currency", label: "Döviz", kinds: ["currency", "fx"] },
  { id: "crypto", label: "Kripto", kinds: ["crypto"] },
  { id: "stock", label: "Hisse", kinds: ["stock", "equity"] },
];

function MarketAreaChart({ data }: { data: number[] }) {
  if (data.length < 2) return <div className="market-chart-empty" />;
  const width = 760;
  const height = 238;
  const min = Math.min(...data);
  const max = Math.max(...data);
  const range = max - min || 1;
  const points = data.map((value, index) => {
    const x = (index / (data.length - 1)) * width;
    const y = height - 16 - ((value - min) / range) * (height - 38);
    return [x, y] as const;
  });
  const line = points.map(([x, y], index) => `${index === 0 ? "M" : "L"}${x.toFixed(1)} ${y.toFixed(1)}`).join(" ");
  const area = `${line} L${width} ${height} L0 ${height} Z`;

  return (
    <svg className="market-area-chart" viewBox={`0 0 ${width} ${height}`} preserveAspectRatio="none" aria-hidden>
      <defs>
        <linearGradient id="marketFill" x1="0" y1="0" x2="0" y2="1">
          <stop offset="0%" stopColor="var(--accent)" stopOpacity="0.24" />
          <stop offset="100%" stopColor="var(--accent)" stopOpacity="0" />
        </linearGradient>
      </defs>
      {[0.25, 0.5, 0.75].map((position) => (
        <line key={position} x1="0" y1={height * position} x2={width} y2={height * position} stroke="var(--stroke)" strokeWidth="1" />
      ))}
      <path d={area} fill="url(#marketFill)" />
      <path d={line} fill="none" stroke="var(--accent-hi)" strokeWidth="2.4" vectorEffect="non-scaling-stroke" />
    </svg>
  );
}

function MovementRow({ item, rank, onClick }: { item: InstrumentPrice; rank: number; onClick: () => void }) {
  return (
    <button className="movement-row" onClick={onClick}>
      <span className="movement-rank mono">{String(rank).padStart(2, "0")}</span>
      <AssetBadge code={item.code} size={32} />
      <span className="movement-copy">
        <strong>{item.display_name}</strong>
        <small>{item.code}</small>
      </span>
      <span className="movement-value">
        <span className="mono">{money(item.price, item.currency)}</span>
        <ChangePill pct={item.change_pct} small />
      </span>
    </button>
  );
}

export function Markets() {
  const order = useMarkets((s) => s.order);
  const prices = useMarkets((s) => s.prices);
  const spark = useMarkets((s) => s.spark);
  const flash = useMarkets((s) => s.flash);
  const { authed, profile } = useAuth();
  const [q, setQ] = useState("");
  const [cat, setCat] = useState("all");
  const [aiCode, setAiCode] = useState("");
  const [insight, setInsight] = useState<MarketInsight | null>(null);
  const [aiLoading, setAiLoading] = useState(false);
  const [aiError, setAiError] = useState("");
  const nav = useNavigate();

  const allPrices = useMemo(
    () => order.map((code) => prices[code]).filter(Boolean) as InstrumentPrice[],
    [order, prices],
  );

  const rows = useMemo(() => {
    let list = allPrices;
    const active = CATS.find((c) => c.id === cat);
    if (active?.kinds) list = list.filter((p) => active.kinds!.includes(p.kind));
    if (q.trim()) {
      const search = q.toLocaleLowerCase("tr");
      list = list.filter(
        (p) => p.display_name.toLocaleLowerCase("tr").includes(search) || p.code.toLowerCase().includes(search),
      );
    }
    return list;
  }, [allPrices, q, cat]);

  const movers = useMemo(
    () => [...allPrices].filter((p) => Number.isFinite(p.change_pct)).sort((a, b) => Math.abs(b.change_pct) - Math.abs(a.change_pct)).slice(0, 5),
    [allPrices],
  );
  const positive = allPrices.filter((p) => p.change_pct >= 0).length;
  const breadth = allPrices.length ? Math.round((positive / allPrices.length) * 100) : 50;
  const averageChange = allPrices.length
    ? allPrices.reduce((sum, item) => sum + (Number.isFinite(item.change_pct) ? item.change_pct : 0), 0) / allPrices.length
    : 0;
  const mood = breadth >= 64 ? "Risk iştahı güçlü" : breadth >= 46 ? "Dengeli seyir" : "Temkinli görünüm";
  const defaultAsset = movers[0]?.code || allPrices[0]?.code || "";
  const selectedCode = aiCode || defaultAsset;

  const marketSeries = useMemo(() => {
    const candidates = movers.length ? movers : allPrices.slice(0, 8);
    const longest = Math.max(0, ...candidates.map((item) => spark[item.code]?.length || 0));
    if (longest < 2) return [];
    return Array.from({ length: longest }, (_, index) => {
      const normalized = candidates.flatMap((item) => {
        const series = spark[item.code] || [];
        const value = series[index - (longest - series.length)];
        const first = series[0];
        return value != null && first ? [((value - first) / first) * 100] : [];
      });
      return normalized.length ? normalized.reduce((sum, value) => sum + value, 0) / normalized.length : 0;
    });
  }, [allPrices, movers, spark]);

  async function askAI() {
    if (!authed) {
      nav("/login");
      return;
    }
    if (!selectedCode) return;
    setAiLoading(true);
    setAiError("");
    try {
      setInsight(await api.marketInsight(selectedCode));
    } catch (error) {
      setAiError(error instanceof Error ? error.message : "AI yorumu şu anda alınamadı.");
    } finally {
      setAiLoading(false);
    }
  }

  const loading = order.length === 0;

  return (
    <div className="market-dashboard">
      <section className="dashboard-heading">
        <div>
          <div className="eyebrow"><span className="eyebrow-dot" /> TSC PİYASA TERMİNALİ</div>
          <h1>{profile?.display_name ? `Merhaba, ${profile.display_name.split(" ")[0]}` : "Piyasaya tek bakışta hakim ol."}</h1>
          <p>Canlı fiyatlar, önemli hareketler ve risk odaklı piyasa içgörüleri tek ekranda.</p>
        </div>
        <div className="dashboard-date">
          <Icon name="clock" size={15} />
          {new Intl.DateTimeFormat("tr-TR", { weekday: "long", day: "numeric", month: "long" }).format(new Date())}
        </div>
      </section>

      <section className="market-overview-grid">
        <article className="dashboard-card market-pulse-card">
          <div className="card-topline">
            <div>
              <span className="card-kicker">Piyasa nabzı</span>
              <div className="market-score-line">
                <strong className="mono">{breadth}</strong>
                <span>/ 100</span>
                <span className={`mood-chip ${breadth >= 46 ? "positive" : "negative"}`}>{mood}</span>
              </div>
            </div>
            <div className="pulse-meta">
              <span>Ortalama hareket</span>
              <ChangePill pct={averageChange} />
            </div>
          </div>
          <div className="chart-shell">
            {marketSeries.length > 1 ? <MarketAreaChart data={marketSeries} /> : <div className="chart-skeleton skeleton" />}
            <div className="chart-axis"><span>09:00</span><span>12:00</span><span>15:00</span><span>Şimdi</span></div>
          </div>
          <div className="breadth-footer">
            <div><span className="breadth-dot up-dot" /><strong>{positive}</strong> yükselen</div>
            <div className="breadth-bar"><span style={{ width: `${breadth}%` }} /></div>
            <div><strong>{Math.max(0, allPrices.length - positive)}</strong> düşen <span className="breadth-dot down-dot" /></div>
          </div>
        </article>

        <article className="dashboard-card movers-card">
          <div className="section-title-row">
            <div><span className="card-kicker">Radar</span><h2>En sert hareketler</h2></div>
            <span className="live-badge"><span /> CANLI</span>
          </div>
          <div className="movement-list">
            {movers.map((item, index) => <MovementRow key={item.code} item={item} rank={index + 1} onClick={() => nav(`/asset/${item.code}`)} />)}
            {loading && Array.from({ length: 5 }).map((_, index) => <div key={index} className="movement-loading skeleton" />)}
          </div>
        </article>
      </section>

      <section className="dashboard-card ai-insight-card">
        <div className="ai-visual">
          <span className="ai-orbit orbit-one" />
          <span className="ai-orbit orbit-two" />
          <div className="ai-core"><Icon name="sparkles" size={24} /></div>
        </div>
        <div className="ai-copy">
          <div className="ai-label"><Icon name="sparkles" size={14} /> TSC DERİN ARAŞTIRMA <span>{insight?.model || "GPT-5.6 SOL"}</span></div>
          <h2>{insight?.title || "Piyasayı tek fiyatla değil, bütün bağlantılarıyla araştır"}</h2>
          <p>{insight?.summary || "Çok zaman dilimli TSC verisini; döviz, ons altın ve ilişkili piyasalarla karşılaştırır. Gerektiğinde güncel makroekonomik ve jeopolitik gelişmeleri web üzerinden kaynaklı biçimde araştırır."}</p>
          {insight && (
            <>
              <div className="ai-result-meta">
                <span className={`stance-chip ${insight.stance}`}>{insight.stance_label}</span>
                {insight.time_horizons?.slice(0, 3).map((item) => <span key={item.horizon}>• {item.horizon}: {item.outlook}</span>)}
              </div>
              {!!insight.sources?.length && <div className="ai-source-links">
                {insight.sources.slice(0, 4).map((source) => <a key={source.url} href={source.url} target="_blank" rel="noreferrer">{source.title}</a>)}
              </div>}
            </>
          )}
          {aiError && <div className="ai-error">{aiError}</div>}
        </div>
        <div className="ai-action">
          <label htmlFor="ai-asset">İncelenecek varlık</label>
          <select id="ai-asset" value={selectedCode} onChange={(event) => { setAiCode(event.target.value); setInsight(null); }}>
            {allPrices.map((item) => <option key={item.code} value={item.code}>{item.display_name} · {item.code}</option>)}
          </select>
          <button onClick={askAI} disabled={aiLoading || !selectedCode}>
            {aiLoading ? <><span className="spinner light" /> Kaynaklar araştırılıyor</> : <><Icon name="sparkles" size={16} /> {authed ? "Derin araştırmayı başlat" : "Giriş yap ve dene"}</>}
          </button>
          <small>GPT-5.6 Sol · çok zaman dilimli veri · canlı web araştırması · kaynaklı sonuç</small>
        </div>
      </section>

      {insight && <section className="dashboard-card ai-research-report">
        <div className="section-title-row">
          <div><span className="card-kicker">Kaynaklı rapor</span><h2>{insight.code} için derin piyasa analizi</h2></div>
          <span className={`stance-chip ${insight.stance}`}>{insight.stance_label} · güven {insight.confidence}</span>
        </div>
        <div className="ai-report-grid">
          <article>
            <h3>Zaman dilimleri</h3>
            {insight.time_horizons.map((item) => <div className="ai-report-item" key={item.horizon}>
              <strong>{item.horizon} · {item.outlook}</strong><p>{item.rationale}</p>
            </div>)}
          </article>
          <article><h3>Makro sürücüler</h3>{insight.macro_drivers.map((item) => <p key={item}>• {item}</p>)}</article>
          <article><h3>Çapraz varlık ilişkileri</h3>{insight.cross_asset_analysis.map((item) => <p key={item}>• {item}</p>)}</article>
          <article><h3>Jeopolitik riskler</h3>{insight.geopolitical_risks.map((item) => <p key={item}>• {item}</p>)}</article>
          <article className="ai-scenarios">
            <h3>Senaryolar</h3>
            {insight.scenarios.map((scenario) => <div className="ai-report-item" key={scenario.name}>
              <strong>{scenario.name} · {scenario.likelihood}</strong>
              <p><b>Tetik:</b> {scenario.trigger}</p><p>{scenario.implication}</p>
            </div>)}
          </article>
        </div>
        <div className="ai-report-risk">⚠ {insight.risk_note}<small>{insight.disclaimer}</small></div>
      </section>}

      <section className="dashboard-card instruments-card">
        <div className="instruments-toolbar">
          <div>
            <span className="card-kicker">Canlı liste</span>
            <h2>Piyasalar <span>{rows.length}</span></h2>
          </div>
          <div className="market-search">
            <Icon name="search" size={16} />
            <input value={q} onChange={(event) => setQ(event.target.value)} placeholder="Sembol veya varlık ara" />
            <kbd>⌘ K</kbd>
          </div>
        </div>
        <div className="category-tabs no-sb">
          {CATS.map((category) => (
            <button key={category.id} className={cat === category.id ? "active" : ""} onClick={() => setCat(category.id)}>{category.label}</button>
          ))}
        </div>
        <div className="market-table">
          <div className="market-table-head"><span>Varlık</span><span>Son hareket</span><span>24s</span><span>Fiyat</span></div>
          {loading && Array.from({ length: 7 }).map((_, index) => <div key={index} className="market-row-loading skeleton" />)}
          {!loading && rows.length === 0 && <div className="market-empty">Aramana uyan bir varlık bulunamadı.</div>}
          {rows.map((item) => (
            <button
              key={item.code}
              onClick={() => nav(`/asset/${item.code}`)}
              className={`market-table-row ${flash[item.code] === "up" ? "flash-up" : flash[item.code] === "down" ? "flash-down" : ""}`}
            >
              <span className="asset-cell"><AssetBadge code={item.code} size={34} /><span><strong>{item.display_name}</strong><small>{item.code} · {item.provider || "TSC"}</small></span></span>
              <span className="spark-cell"><Sparkline data={spark[item.code] || []} width={112} height={32} /></span>
              <span><ChangePill pct={item.change_pct} /></span>
              <span className="price-cell"><strong className="mono">{money(item.price, item.currency)}</strong>{item.currency === "USD" && (item.price_try || 0) > 0 && <small className="mono">≈ {tl(item.price_try as number)}</small>}</span>
              <Icon name="chevron-right" size={15} />
            </button>
          ))}
        </div>
      </section>
    </div>
  );
}
