//
//  InstrumentDetailView.swift
//  AssetDetailView — Binance-style asset screen: large live price, 24h stats,
//  interactive Swift Charts (line/candle + timeframes), and alert creation.
//

import SwiftUI

private struct Timeframe: Hashable { let key: String; let interval: String; let label: String; let days: Int }

struct AssetDetailView: View {
    let code: String
    @EnvironmentObject var auth: AuthStore
    @EnvironmentObject var prices: PriceStore
    @EnvironmentObject var billing: BillingStore
    @Environment(\.dismiss) private var dismiss
    @Environment(\.requestAuthentication) private var requestAuthentication

    @State private var candles: [Candle] = []
    @State private var stats: InstrumentStats?
    @State private var detail: InstrumentDetail?
    @State private var mode: ChartMode = .line
    @State private var tf = "1G"
    @State private var loadingChart = false
    @State private var loadingOlder = false
    @State private var loadingNewer = false
    @State private var hasMoreHistory = true
    // True when we've trimmed the most-recent candles off the render window
    // (the user panned back in time). While true, the present isn't on screen,
    // so we must NOT fold the live price into the last visible candle.
    @State private var hasMoreFuture = false
    @State private var chartReset = 0
    // Network/cache window. The custom chart renderer draws only the visible
    // 8...160 bars; these limits only bound the nearby paged cache.
    private let pageSize = 240
    private let maxWindow = 600
    @State private var showAlert = false
    @State private var convAmount = "1"
    @State private var aiInsight: MarketInsight?
    @State private var aiLoading = false
    @State private var aiError: String?
    @State private var showAIDisclosure = false
    @State private var showPaywall = false
    @State private var acceptedAIDisclosure = false

    // Binance-style time ranges. Long ranges use daily/weekly/monthly buckets to
    // match backfilled history; short ranges use intraday buckets from live data.
    private let timeframes = [
        Timeframe(key: "1G", interval: "5m", label: "1G", days: 1),
        Timeframe(key: "1H", interval: "1h", label: "1H", days: 7),
        Timeframe(key: "1A", interval: "1d", label: "1A", days: 30),
        Timeframe(key: "1Y", interval: "1d", label: "1Y", days: 365),
        Timeframe(key: "5Y", interval: "1w", label: "5Y", days: 365 * 5),
        Timeframe(key: "MAX", interval: "1mo", label: "MAX", days: 365 * 40),
    ]

    private var price: InstrumentPrice? { prices.prices[code] }
    private var trendUp: Bool { (price?.changePct ?? 0) >= 0 }

    var body: some View {
        ZStack {
            Theme.bg.ignoresSafeArea()
            ScrollView {
                VStack(spacing: 16) {
                    priceHeader
                    stats24h
                    chartCard
                    aiInsightCard
                    detailSections
                    converterCard
                    Color.clear.frame(height: 96)
                }
                .padding(.horizontal, 16)
                .padding(.top, 8)
            }
        }
        .navigationBarBackButtonHidden(true)
        .toolbar {
            ToolbarItem(placement: .topBarLeading) {
                Button { dismiss() } label: {
                    Image(systemName: "chevron.left").foregroundStyle(Theme.text)
                }
            }
            ToolbarItem(placement: .principal) {
                VStack(spacing: 1) {
                    Text(price?.displayName ?? code).font(.system(size: 15, weight: .bold)).foregroundStyle(Theme.text)
                    Text(code).font(.system(size: 10)).foregroundStyle(Theme.textDim)
                }
            }
            ToolbarItem(placement: .topBarTrailing) {
                HStack(spacing: 18) {
                    Button {
                        Haptics.tap()
                        if auth.isLoggedIn { prices.toggleFavorite(code) }
                        else { requestAuthentication() }
                    } label: {
                        Image(systemName: prices.isFavorite(code) ? "star.fill" : "star")
                            .foregroundStyle(prices.isFavorite(code) ? Theme.accent : Theme.textDim)
                    }
                    Button {
                        Haptics.tap()
                        if auth.isLoggedIn { showAlert = true }
                        else { requestAuthentication() }
                    } label: {
                        Image(systemName: "bell")
                            .foregroundStyle(Theme.textDim)
                    }
                }
            }
        }
        .toolbarBackground(Theme.bg, for: .navigationBar)
        .toolbarBackground(.visible, for: .navigationBar)
        .task(id: tf) { await loadChart() }
        .task { await loadStats() }
        .task { await loadDetail() }
        .task(id: auth.userID) {
            acceptedAIDisclosure = AIResearchConsentStore.isAccepted(for: auth.userID)
        }
        .sheet(isPresented: $showAlert) {
            AlertSheet(code: code, current: price?.price ?? stats?.last ?? 0)
                .environmentObject(auth)
                .presentationDetents([.height(380)])
                .presentationBackground(Theme.bgElevated)
        }
        .sheet(isPresented: $showAIDisclosure) {
            AIResearchDisclosureView {
                guard let userID = auth.userID else {
                    aiError = "AI araştırma onayı hesabınıza kaydedilemedi. Profilinizi yenileyip tekrar deneyin."
                    showAIDisclosure = false
                    return
                }
                AIResearchConsentStore.setAccepted(true, for: userID)
                acceptedAIDisclosure = true
                showAIDisclosure = false
                Task { await loadAIInsight() }
            }
        }
        .sheet(isPresented: $showPaywall) {
            PaywallView()
                .environmentObject(auth)
                .environmentObject(billing)
        }
        .alert("Favoriler", isPresented: Binding(
            get: { prices.favoriteMessage != nil },
            set: { if !$0 { prices.favoriteMessage = nil } }
        )) {
            Button("Tamam") { prices.favoriteMessage = nil }
        } message: {
            Text(prices.favoriteMessage ?? "")
        }
    }

    // MARK: Header

    private var priceHeader: some View {
        VStack(alignment: .leading, spacing: 6) {
            if let p = price {
                FlashText(value: p.price,
                          font: Theme.number(38, weight: .bold),
                          format: { Fmt.money($0, currency: p.currency) })
                if p.isUSD {
                    Text("≈ \(Fmt.tl(p.tryValue))")
                        .font(Theme.number(14, weight: .medium))
                        .foregroundStyle(Theme.textDim)
                }
                HStack(spacing: 8) {
                    Text("\(Fmt.signed(p.change)) \(p.isUSD ? "$" : "₺")  \(Fmt.signedPct(p.changePct))")
                        .font(Theme.number(15, weight: .semibold))
                        .foregroundStyle(Theme.trend(p.changePct))
                }
            } else {
                Text("—").font(Theme.number(38, weight: .bold)).foregroundStyle(Theme.textDim)
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
    }

    // MARK: 24h stats

    private var stats24h: some View {
        Card(padding: 14) {
            if stats == nil && price == nil {
                HStack(spacing: 10) {
                    ForEach(0..<4, id: \.self) { _ in
                        VStack(alignment: .leading, spacing: 6) {
                            SkeletonBlock(width: 48, height: 10)
                            SkeletonBlock(width: 62, height: 14)
                        }.frame(maxWidth: .infinity, alignment: .leading)
                    }
                }
                .shimmer()
            } else {
                HStack(spacing: 10) {
                    StatTile(label: "24s Yüksek", value: statVal(stats?.high), tint: Theme.up)
                    StatTile(label: "24s Düşük", value: statVal(stats?.low), tint: Theme.down)
                    StatTile(label: "24s Açılış", value: statVal(stats?.open))
                    StatTile(label: "24s Son", value: statVal(stats?.last ?? price?.price))
                }
            }
        }
    }

    private func statVal(_ v: Double?) -> String {
        guard let v, v > 0 else { return "—" }
        return Fmt.money(v, currency: price?.currency)
    }

    // MARK: Chart

    private var chartCard: some View {
        Card(padding: 14) {
            VStack(spacing: 12) {
                HStack {
                    SegBar(items: [(ChartMode.line, "Çizgi"), (ChartMode.candle, "Mum")],
                           selection: $mode)
                        .frame(width: 150)
                    Spacer()
                    if loadingChart { ProgressView().tint(Theme.accent).scaleEffect(0.8) }
                }
                if candles.isEmpty && loadingChart {
                    SkeletonBlock(height: 300, corner: 10).shimmer()
                } else {
                    PriceChart(candles: candles, mode: mode, trendUp: trendUp,
                               livePrice: hasMoreFuture ? nil : price?.price,
                               onReachStart: { Task { await loadOlder() } },
                               onReachEnd: { Task { await loadNewer() } },
                               canLoadOlder: hasMoreHistory,
                               canLoadNewer: hasMoreFuture,
                               resetToken: chartReset, currency: price?.currency)
                        .clipShape(RoundedRectangle(cornerRadius: 8, style: .continuous))
                        .overlay(alignment: .topLeading) {
                            if loadingOlder || loadingNewer {
                                Text(loadingOlder ? "Geçmiş yükleniyor…" : "Güncel yükleniyor…")
                                    .font(.system(size: 10)).foregroundStyle(Theme.textDim)
                                    .padding(.horizontal, 8).padding(.vertical, 3)
                                    .background(Theme.bgElevated.opacity(0.85), in: Capsule())
                                    .padding(8)
                            }
                        }
                }
                SegBar(items: timeframes.map { ($0.key, $0.label) }, selection: $tf)
            }
        }
    }

    // MARK: AI insight

    private var aiInsightCard: some View {
        Card(padding: 16) {
            VStack(alignment: .leading, spacing: 12) {
                HStack(spacing: 7) {
                    Image(systemName: "sparkles")
                        .foregroundStyle(Theme.accent)
                    Text("AI PİYASA ARAŞTIRMASI")
                        .font(.system(size: 11, weight: .bold))
                        .foregroundStyle(Theme.accent)
                        .tracking(1)
                    Spacer()
                    if let insight = aiInsight {
                        Text(insight.stanceLabel)
                            .font(.system(size: 10, weight: .bold))
                            .foregroundStyle(aiStanceColor(insight.stance))
                            .padding(.horizontal, 8).padding(.vertical, 4)
                            .background(aiStanceColor(insight.stance).opacity(0.12), in: Capsule())
                    }
                }

                if let insight = aiInsight {
                    Text(insight.title)
                        .font(.system(size: 17, weight: .bold))
                        .foregroundStyle(Theme.text)
                    Text(insight.summary)
                        .font(.system(size: 13))
                        .foregroundStyle(Theme.textDim)
                        .fixedSize(horizontal: false, vertical: true)
                    VStack(alignment: .leading, spacing: 5) {
                        ForEach(insight.factors, id: \.self) { factor in
                            Text("• \(factor)")
                                .font(.system(size: 12, weight: .medium))
                                .foregroundStyle(Theme.text)
                        }
                    }

                    if let horizons = insight.timeHorizons, !horizons.isEmpty {
                        aiSectionTitle("ZAMAN DİLİMLERİ", icon: "clock.arrow.circlepath")
                        VStack(spacing: 8) {
                            ForEach(horizons, id: \.self) { item in
                                HStack(alignment: .top, spacing: 10) {
                                    Text(item.horizon)
                                        .font(.system(size: 10, weight: .bold))
                                        .foregroundStyle(Theme.accent)
                                        .frame(width: 58, alignment: .leading)
                                    VStack(alignment: .leading, spacing: 2) {
                                        Text(item.outlook)
                                            .font(.system(size: 12, weight: .bold))
                                            .foregroundStyle(Theme.text)
                                        Text(item.rationale)
                                            .font(.system(size: 11))
                                            .foregroundStyle(Theme.textDim)
                                            .fixedSize(horizontal: false, vertical: true)
                                    }
                                    Spacer(minLength: 0)
                                }
                                .padding(10)
                                .background(Theme.bgElevated, in: RoundedRectangle(cornerRadius: 9))
                            }
                        }
                    }

                    if let macro = insight.macroDrivers, !macro.isEmpty {
                        aiBulletSection("MAKRO SÜRÜCÜLER", icon: "chart.line.uptrend.xyaxis", items: macro)
                    }
                    if let crossAsset = insight.crossAssetAnalysis, !crossAsset.isEmpty {
                        aiBulletSection("ÇAPRAZ VARLIK İLİŞKİLERİ", icon: "arrow.triangle.branch", items: crossAsset)
                    }
                    if let geopolitical = insight.geopoliticalRisks, !geopolitical.isEmpty {
                        aiBulletSection("JEOPOLİTİK RİSKLER", icon: "globe.europe.africa", items: geopolitical)
                    }

                    if let scenarios = insight.scenarios, !scenarios.isEmpty {
                        aiSectionTitle("SENARYOLAR", icon: "point.3.connected.trianglepath.dotted")
                        VStack(spacing: 8) {
                            ForEach(scenarios, id: \.self) { scenario in
                                VStack(alignment: .leading, spacing: 5) {
                                    HStack {
                                        Text(scenario.name)
                                            .font(.system(size: 12, weight: .bold))
                                            .foregroundStyle(Theme.text)
                                        Spacer()
                                        Text(aiLikelihoodLabel(scenario.likelihood))
                                            .font(.system(size: 9, weight: .bold))
                                            .foregroundStyle(Theme.accent)
                                    }
                                    Text("Tetik: \(scenario.trigger)")
                                        .font(.system(size: 10, weight: .semibold))
                                        .foregroundStyle(Theme.textDim)
                                    Text(scenario.implication)
                                        .font(.system(size: 11))
                                        .foregroundStyle(Theme.textDim)
                                        .fixedSize(horizontal: false, vertical: true)
                                }
                                .padding(10)
                                .background(Theme.bgElevated, in: RoundedRectangle(cornerRadius: 9))
                            }
                        }
                    }

                    if let playbook = insight.whatIWouldDo {
                        aiSectionTitle("BEN OLSAM NE YAPARDIM", icon: "person.crop.circle.badge.questionmark")
                        VStack(alignment: .leading, spacing: 8) {
                            Text(playbook.intro)
                                .font(.system(size: 12, weight: .semibold))
                                .foregroundStyle(Theme.text)
                                .fixedSize(horizontal: false, vertical: true)
                            aiPlaybookRow(title: "Kısa vade", body: playbook.shortTerm)
                            aiPlaybookRow(title: "Orta vade", body: playbook.mediumTerm)
                            aiPlaybookRow(title: "Uzun vade", body: playbook.longTerm)
                        }
                    }

                    Label(insight.riskNote, systemImage: "exclamationmark.triangle")
                        .font(.system(size: 11))
                        .foregroundStyle(Theme.textDim)

                    if let sources = insight.sources, !sources.isEmpty {
                        aiSectionTitle("ARAŞTIRMA KAYNAKLARI", icon: "link")
                        VStack(alignment: .leading, spacing: 7) {
                            ForEach(Array(sources.prefix(8)), id: \.self) { source in
                                if let url = URL(string: source.url) {
                                    Link(destination: url) {
                                        HStack(spacing: 7) {
                                            Image(systemName: "arrow.up.right.square")
                                            Text(source.title)
                                                .lineLimit(2)
                                            Spacer(minLength: 0)
                                        }
                                        .font(.system(size: 10, weight: .semibold))
                                        .foregroundStyle(Theme.accent)
                                    }
                                }
                            }
                        }
                    }
                    Text(insight.disclaimer)
                        .font(.system(size: 9))
                        .foregroundStyle(Theme.textFaint)
                } else {
                    Text("Çok zaman dilimli piyasa verilerini ilişkili varlıklarla karşılaştırır; güncel makro ve jeopolitik gelişmeleri kaynaklarıyla araştırır. Sonuçlar hatalı veya eksik olabilir ve yatırım tavsiyesi değildir.")
                        .font(.system(size: 13))
                        .foregroundStyle(Theme.textDim)
                        .fixedSize(horizontal: false, vertical: true)
                }

                if let aiError {
                    Text(aiError)
                        .font(.system(size: 11))
                        .foregroundStyle(Theme.down)
                }

                Button {
                    Haptics.tap()
                    if !auth.isLoggedIn { requestAuthentication() }
                    else if acceptedAIDisclosure { Task { await loadAIInsight() } }
                    else { showAIDisclosure = true }
                } label: {
                    HStack(spacing: 8) {
                        if aiLoading { ProgressView().tint(Theme.bg).scaleEffect(0.75) }
                        Image(systemName: aiLoading ? "hourglass" : "sparkles")
                        Text(auth.isLoggedIn ? (aiLoading ? "Kaynaklar araştırılıyor…" : "Derin araştırmayı başlat") : "Araştırma için giriş yap")
                    }
                    .font(.system(size: 13, weight: .bold))
                    .foregroundStyle(Theme.bg)
                    .frame(maxWidth: .infinity)
                    .padding(.vertical, 11)
                    .background(Theme.accent, in: RoundedRectangle(cornerRadius: 10, style: .continuous))
                }
                .buttonStyle(.plain)
                .disabled(aiLoading)
                .opacity(aiLoading ? 0.72 : 1)
            }
        }
    }

    private func aiStanceColor(_ stance: String) -> Color {
        switch stance {
        case "positive": return Theme.up
        case "cautious": return Color(hex: 0xE6B565)
        default: return Theme.accent
        }
    }

    private func aiSectionTitle(_ title: String, icon: String) -> some View {
        Label(title, systemImage: icon)
            .font(.system(size: 9, weight: .bold))
            .foregroundStyle(Theme.textFaint)
            .tracking(0.8)
            .padding(.top, 4)
    }

    private func aiPlaybookRow(title: String, body: String) -> some View {
        VStack(alignment: .leading, spacing: 2) {
            Text(title)
                .font(.system(size: 10, weight: .bold))
                .foregroundStyle(Theme.accent)
            Text(body)
                .font(.system(size: 12))
                .foregroundStyle(Theme.text)
                .fixedSize(horizontal: false, vertical: true)
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .padding(10)
        .background(Theme.bgElevated, in: RoundedRectangle(cornerRadius: 9))
    }

    private func aiBulletSection(_ title: String, icon: String, items: [String]) -> some View {
        VStack(alignment: .leading, spacing: 6) {
            aiSectionTitle(title, icon: icon)
            ForEach(items, id: \.self) { item in
                HStack(alignment: .top, spacing: 7) {
                    Circle().fill(Theme.accent).frame(width: 4, height: 4).padding(.top, 5)
                    Text(item)
                        .font(.system(size: 11))
                        .foregroundStyle(Theme.textDim)
                        .fixedSize(horizontal: false, vertical: true)
                }
            }
        }
    }

    private func aiLikelihoodLabel(_ likelihood: String) -> String {
        switch likelihood {
        case "high": return "YÜKSEK OLASILIK"
        case "low": return "DÜŞÜK OLASILIK"
        default: return "ORTA OLASILIK"
        }
    }

    // MARK: Kind-specific rich detail (Binance-style)

    @ViewBuilder private var detailSections: some View {
        if let d = detail {
            switch d.kind {
            case "crypto":   cryptoSections(d)
            case "stock":    stockSections(d)
            case "gold":     goldSections(d)
            case "currency": currencySections(d)
            default:         EmptyView()
            }
        } else {
            MetricGrid(title: "Piyasa Verileri", metrics: Metric.placeholders(6), loading: true)
        }
    }

    // Crypto: market cap / supply / dominance + 24h stats + ATH/ATL records.
    @ViewBuilder private func cryptoSections(_ d: InstrumentDetail) -> some View {
        let market = cryptoMarket(d)
        let stats = cryptoStats(d)
        let records = cryptoRecords(d)
        VStack(spacing: 16) {
            if !market.isEmpty { MetricGrid(title: "Piyasa Verileri", metrics: market) }
            if !stats.isEmpty { MetricGrid(title: "24s İstatistikler", metrics: stats) }
            if !records.isEmpty { MetricGrid(title: "Rekorlar", metrics: records) }
        }
    }

    // Stock: day & 52-week ranges, volume, exchange.
    @ViewBuilder private func stockSections(_ d: InstrumentDetail) -> some View {
        let stats = stockStats(d)
        let info = stockInfo(d)
        VStack(spacing: 16) {
            if !stats.isEmpty { MetricGrid(title: "İstatistikler", metrics: stats) }
            if !info.isEmpty { MetricGrid(title: "Şirket Bilgisi", metrics: info, columns: 1) }
        }
    }

    // Gold: ounce (USD/TRY), gram-has, purity, product weight.
    @ViewBuilder private func goldSections(_ d: InstrumentDetail) -> some View {
        let m = goldMetrics(d)
        if !m.isEmpty { MetricGrid(title: "Altın Verileri", metrics: m) }
    }

    // Currency: parity, inverse, USD cross.
    @ViewBuilder private func currencySections(_ d: InstrumentDetail) -> some View {
        let m = currencyMetrics(d)
        if !m.isEmpty { MetricGrid(title: "Kur Verileri", metrics: m) }
    }

    // MARK: Metric builders (plain — kept out of @ViewBuilder bodies)

    private func cryptoMarket(_ d: InstrumentDetail) -> [Metric] {
        var m: [Metric] = []
        if let v = d.marketCapUsd { m.append(Metric("Piyasa Değeri", Fmt.usd(v, compact: true))) }
        if let v = d.rank { m.append(Metric("Sıralama", "#\(v)")) }
        if let v = d.dominancePct { m.append(Metric("Dominans", Fmt.pct(v))) }
        if let v = d.quoteVolumeUsd { m.append(Metric("24s Hacim", Fmt.usd(v, compact: true))) }
        if let v = d.circulatingSupply { m.append(Metric("Dolaşan Arz", "\(Fmt.compactNum(v)) \(code)")) }
        if let v = d.maxSupply {
            m.append(Metric("Maks. Arz", "\(Fmt.compactNum(v)) \(code)"))
        } else if let v = d.totalSupply {
            m.append(Metric("Toplam Arz", "\(Fmt.compactNum(v)) \(code)"))
        }
        if let v = d.fdvUsd { m.append(Metric("FDV", Fmt.usd(v, compact: true))) }
        if let v = d.trades { m.append(Metric("İşlem (24s)", Fmt.compactNum(Double(v)))) }
        return m
    }

    private func cryptoStats(_ d: InstrumentDetail) -> [Metric] {
        var m: [Metric] = []
        if let v = d.high24h, v > 0 { m.append(Metric("24s Yüksek", Fmt.tl(v), tint: Theme.up)) }
        if let v = d.low24h, v > 0 { m.append(Metric("24s Düşük", Fmt.tl(v), tint: Theme.down)) }
        if let v = d.open24h, v > 0 { m.append(Metric("24s Açılış", Fmt.tl(v))) }
        if let v = d.weightedAvg, v > 0 { m.append(Metric("Ort. Fiyat", Fmt.tl(v))) }
        if let v = d.changePct24h { m.append(Metric("24s Değişim", Fmt.signedPct(v), tint: Theme.trend(v))) }
        if let v = d.priceUsd, v > 0 { m.append(Metric("USD Fiyat", Fmt.usd(v))) }
        return m
    }

    private func cryptoRecords(_ d: InstrumentDetail) -> [Metric] {
        var m: [Metric] = []
        if let v = d.athUsd, v > 0 { m.append(Metric("En Yüksek (ATH)", Fmt.usd(v), sub: Fmt.dateShort(d.athDate))) }
        if let v = d.atlUsd, v > 0 { m.append(Metric("En Düşük (ATL)", Fmt.usd(v), sub: Fmt.dateShort(d.atlDate))) }
        return m
    }

    private func stockStats(_ d: InstrumentDetail) -> [Metric] {
        var m: [Metric] = []
        let c = d.currency
        if c == "USD", let v = d.priceTry, v > 0 { m.append(Metric("Fiyat (₺)", Fmt.tl(v))) }
        if let v = d.dayHigh, v > 0 { m.append(Metric("Gün Yüksek", Fmt.money(v, currency: c), tint: Theme.up)) }
        if let v = d.dayLow, v > 0 { m.append(Metric("Gün Düşük", Fmt.money(v, currency: c), tint: Theme.down)) }
        if let v = d.week52High, v > 0 { m.append(Metric("52H Yüksek", Fmt.money(v, currency: c))) }
        if let v = d.week52Low, v > 0 { m.append(Metric("52H Düşük", Fmt.money(v, currency: c))) }
        if let v = d.prevClose, v > 0 { m.append(Metric("Önceki Kapanış", Fmt.money(v, currency: c))) }
        if let v = d.volume, v > 0 { m.append(Metric("Hacim (Lot)", Fmt.compactNum(v))) }
        return m
    }

    private func stockInfo(_ d: InstrumentDetail) -> [Metric] {
        var m: [Metric] = []
        if let v = d.exchange { m.append(Metric("Borsa", v == "IST" ? "BIST" : v)) }
        if let v = d.longName { m.append(Metric("Şirket", v)) }
        return m
    }

    private func goldMetrics(_ d: InstrumentDetail) -> [Metric] {
        var m: [Metric] = []
        if let v = d.onsUsd, v > 0 { m.append(Metric("Ons (XAU/USD)", Fmt.usd(v))) }
        if let v = d.onsTry, v > 0 { m.append(Metric("Ons (₺)", Fmt.tl(v))) }
        if let v = d.gramHasTry, v > 0 { m.append(Metric("Gram Has Altın", Fmt.tl(v))) }
        if let v = d.usdTry, v > 0 { m.append(Metric("USD/TRY", Fmt.tl(v))) }
        if let v = d.purityLabel { m.append(Metric("Saflık", v)) }
        if let v = d.gramWeight, v > 0 { m.append(Metric("Gram Ağırlığı", "\(Fmt.trim(v)) gr")) }
        return m
    }

    private func currencyMetrics(_ d: InstrumentDetail) -> [Metric] {
        let iso = d.iso ?? code.replacingOccurrences(of: "TRY", with: "")
        var m: [Metric] = []
        if let v = d.priceTry, v > 0 { m.append(Metric("1 \(iso)", Fmt.tl(v))) }
        if let v = d.inverse, v > 0 { m.append(Metric("1 ₺", "\(Fmt.compactPrice(v)) \(iso)")) }
        if let v = d.usdCross, v > 0 { m.append(Metric("\(iso)/USD", Fmt.usd(v))) }
        if let v = d.usdTry, v > 0 { m.append(Metric("USD/TRY", Fmt.tl(v))) }
        return m
    }

    // MARK: Bid / Ask

    private var bidAskCard: some View {
        Card(padding: 14) {
            HStack {
                VStack(alignment: .leading, spacing: 4) {
                    Text("Alış").font(.system(size: 11)).foregroundStyle(Theme.textDim)
                    Text(price.map { Fmt.tl($0.bid) } ?? "—")
                        .font(Theme.number(16)).foregroundStyle(Theme.up)
                }
                Spacer()
                VStack(alignment: .trailing, spacing: 4) {
                    Text("Satış").font(.system(size: 11)).foregroundStyle(Theme.textDim)
                    Text(price.map { Fmt.tl($0.ask) } ?? "—")
                        .font(Theme.number(16)).foregroundStyle(Theme.down)
                }
            }
        }
    }

    // MARK: Converter / calculator

    private var converterCard: some View {
        let unit = Double(convAmount.replacingOccurrences(of: ",", with: ".")) ?? 0
        let result = unit * (price?.price ?? 0)
        return Card(padding: 14) {
            VStack(alignment: .leading, spacing: 10) {
                Text("HESAPLA").font(.system(size: 11, weight: .bold)).foregroundStyle(Theme.textDim).tracking(1)
                HStack(spacing: 10) {
                    TextField("", text: $convAmount,
                              prompt: Text("Miktar").foregroundColor(Theme.textFaint))
                        .keyboardType(.decimalPad)
                        .font(Theme.number(18)).foregroundStyle(Theme.text)
                        .padding(10)
                        .background(Theme.bgElevated, in: RoundedRectangle(cornerRadius: 8))
                        .frame(width: 120)
                    Text("birim").font(.system(size: 13)).foregroundStyle(Theme.textDim)
                    Spacer()
                    Text("₺\(Fmt.compactPrice(result))")
                        .font(Theme.number(18, weight: .bold)).foregroundStyle(Theme.accent)
                }
                HStack(spacing: 8) {
                    ForEach([0.25, 0.5, 1.0, 5.0, 10.0], id: \.self) { q in
                        let label = q == floor(q) ? String(Int(q)) : String(q)
                        Button { convAmount = label } label: {
                            Text(label)
                                .font(.system(size: 12, weight: .semibold)).foregroundStyle(Theme.text)
                                .padding(.horizontal, 12).padding(.vertical, 6)
                                .background(Theme.surfaceHi, in: Capsule())
                        }
                    }
                }
            }
        }
    }

    // MARK: Data

    private func loadChart() async {
        loadingChart = true; defer { loadingChart = false }
        hasMoreHistory = true
        hasMoreFuture = false
        let requestedTimeframe = tf
        let client = APIClient(token: auth.accessToken)
        let frame = timeframes.first { $0.key == tf } ?? timeframes[0]
        let from = Calendar.current.date(byAdding: .day, value: -frame.days, to: Date())
        if let c = try? await client.candles(
            code: code,
            interval: frame.interval,
            from: from,
            limit: maxWindow
        ) {
            guard tf == requestedTimeframe else { return }
            // Keep only a nearby cache. The renderer further virtualizes this to
            // at most 160 visible bars.
            candles = Array(c.suffix(maxWindow))
        } else {
            guard tf == requestedTimeframe else { return }
            candles = []
        }
        // Re-center the visible window on the freshly loaded interval.
        chartReset += 1
    }

    private func loadAIInsight() async {
        guard auth.isLoggedIn else {
            aiError = "Bu özellik için hesabınızla giriş yapın."
            return
        }
        if auth.userID == nil { await auth.loadProfile() }
        guard let userID = auth.userID else {
            aiError = "Hesap bilgileri doğrulanamadı. Profilinizi yenileyip tekrar deneyin."
            return
        }
        if billing.serverStatus?.plan.code == "free",
           billing.serverStatus?.freeResearchAvailable == true,
           !billing.freeTrialReady {
            aiError = "Ücretsiz araştırma için Apple cihaz doğrulama hizmeti şu anda hazır değil. Daha sonra tekrar deneyin."
            return
        }
        aiLoading = true
        aiError = nil
        defer { aiLoading = false }
        do {
            aiInsight = try await APIClient(token: auth.accessToken).marketInsight(
                code: code, userID: userID
            )
        } catch let error as APIError where error.statusCode == 402 {
            aiError = "Bu dönem için AI araştırma hakkınız doldu. Bir plan seçerek devam edebilirsiniz."
            showPaywall = true
        } catch let error as APIError where error.statusCode == 429 {
            aiError = "Çok sık istek gönderildi. Kısa bir süre bekleyip aynı araştırmayı yeniden deneyin."
        } catch let error as APIError where
            error.statusCode == 409 && error.serverCode == "operation_in_progress" {
            aiError = "Araştırma işlemi halen tamamlanıyor. Biraz sonra tekrar deneyin; mevcut hakkınız yeniden kullanılmayacak."
        } catch {
            aiError = error.localizedDescription
        }
    }

    // Lazy-load an older page when the user pans to the start of the chart, then
    // drop the newest candles that scrolled off-screen so the rendered set stays
    // bounded (sliding window). Each page is an indexed range read on the compact
    // candles table (no perf hit); scroll position is preserved because the user
    // is anchored to the OLD edge we're keeping.
    private func loadOlder() async {
        guard !loadingOlder, !loadingNewer, hasMoreHistory, let oldest = candles.first?.bucket else { return }
        loadingOlder = true; defer { loadingOlder = false }
        let requestedTimeframe = tf
        let client = APIClient(token: auth.accessToken)
        let frame = timeframes.first { $0.key == tf } ?? timeframes[0]
        let to = oldest.addingTimeInterval(-1)
        guard let older = try? await client.candles(
            code: code,
            interval: frame.interval,
            to: to,
            limit: pageSize
        ) else {
            // A transient network failure must remain retryable.
            return
        }
        guard tf == requestedTimeframe else { return }
        guard !older.isEmpty else {
            hasMoreHistory = false
            return
        }
        var byBucket: [Date: Candle] = [:]
        for c in older { byBucket[c.bucket] = c }
        for c in candles { byBucket[c.bucket] = c }
        var merged = byBucket.values.sorted { $0.bucket < $1.bucket }
        if merged.count > maxWindow {
            // Keep the oldest window slice (user is scrolling back); drop the newest.
            merged = Array(merged.prefix(maxWindow))
            hasMoreFuture = true
        }
        candles = merged
        if older.count < pageSize { hasMoreHistory = false }
    }

    // Lazy-load a newer page when the user pans back toward the present, dropping
    // the oldest off-screen candles. Once we reach the live edge we re-enable
    // folding the live price into the last candle.
    private func loadNewer() async {
        guard !loadingNewer, !loadingOlder, hasMoreFuture, let newest = candles.last?.bucket else { return }
        loadingNewer = true; defer { loadingNewer = false }
        let requestedTimeframe = tf
        let client = APIClient(token: auth.accessToken)
        let frame = timeframes.first { $0.key == tf } ?? timeframes[0]
        let from = newest.addingTimeInterval(1)
        // The backend returns the newest `limit` rows inside a range. Bound the
        // upper edge so forward paging fetches the adjacent interval instead of
        // jumping straight from old history to today's latest page.
        let step = stepSeconds(frame.interval)
        let to = from.addingTimeInterval(step * Double(max(pageSize - 1, 1)))
        guard let newer = try? await client.candles(
            code: code,
            interval: frame.interval,
            from: from,
            to: to,
            limit: pageSize
        ) else {
            return
        }
        guard tf == requestedTimeframe else { return }
        guard !newer.isEmpty else {
            hasMoreFuture = false
            return
        }
        var byBucket: [Date: Candle] = [:]
        for c in candles { byBucket[c.bucket] = c }
        for c in newer { byBucket[c.bucket] = c }
        var merged = byBucket.values.sorted { $0.bucket < $1.bucket }
        if merged.count > maxWindow {
            // Keep the newest window slice (user is scrolling forward); drop the oldest.
            merged = Array(merged.suffix(maxWindow))
            hasMoreHistory = true
        }
        candles = merged
        // Reached the live edge? Stop paging forward and resume live folding.
        let latest = merged.last?.bucket ?? .distantPast
        let reachedLiveEdge = latest >= Date().addingTimeInterval(-step * 2)
            || (to >= Date() && newer.count < pageSize)
        if reachedLiveEdge {
            hasMoreFuture = false
        }
    }

    private func stepSeconds(_ interval: String) -> TimeInterval {
        switch interval {
        case "5m":  return 300
        case "1h":  return 3_600
        case "1d":  return 86_400
        case "1w":  return 604_800
        case "1mo": return 2_629_800
        default:    return 300
        }
    }

    private func loadStats() async {
        let client = APIClient(token: auth.accessToken)
        stats = try? await client.stats(code: code)
    }

    private func loadDetail() async {
        let client = APIClient(token: auth.accessToken)
        detail = try? await client.detail(code: code)
    }
}

private struct AIResearchDisclosureView: View {
    @Environment(\.dismiss) private var dismiss
    let onAccept: () -> Void

    var body: some View {
        NavigationStack {
            VStack(alignment: .leading, spacing: 18) {
                Image(systemName: "sparkles")
                    .font(.system(size: 36)).foregroundStyle(Theme.accent)
                Text("AI araştırması hakkında")
                    .font(.system(size: 23, weight: .bold)).foregroundStyle(Theme.text)
                disclosure("Kaynaklı araştırma", "Seçtiğiniz varlık, ilgili piyasa verileri, güncel açık web kaynakları ve hesabınızdan tek yönlü türetilen takma adlı güvenlik kimliği analiz ile kötüye kullanım önleme için OpenAI'a gönderilir. Adınız, e-postanız, profil fotoğrafınız, ödeme bilgileriniz veya kişisel portföy pozisyonlarınız gönderilmez.")
                disclosure("Hata payı", "Yapay zekâ yanlış, güncelliğini yitirmiş veya eksik bilgi üretebilir. Önemli kararlar öncesinde kaynakları ve verileri doğrulayın.")
                disclosure("Yatırım tavsiyesi değildir", "İçerik yalnızca eğitim ve karar desteği amaçlıdır; alım, satım veya getiri garantisi sunmaz.")
                Spacer()
                HStack(spacing: 16) {
                    Link("Gizlilik", destination: URL(string: "https://tsc.talhacan.com/privacy")!)
                    Link("Kullanım Şartları", destination: URL(string: "https://tsc.talhacan.com/terms")!)
                }
                .font(.system(size: 12, weight: .semibold)).foregroundStyle(Theme.accent)
                Button("Anladım ve Devam Et") { onAccept() }
                    .buttonStyle(FilledButtonStyle())
            }
            .padding(22)
            .background(Theme.bg.ignoresSafeArea())
            .toolbar {
                ToolbarItem(placement: .topBarTrailing) {
                    Button("Kapat") { dismiss() }.foregroundStyle(Theme.accent)
                }
            }
        }
        .preferredColorScheme(.dark)
    }

    private func disclosure(_ title: String, _ text: String) -> some View {
        VStack(alignment: .leading, spacing: 4) {
            Text(title).font(.system(size: 14, weight: .bold)).foregroundStyle(Theme.text)
            Text(text).font(.system(size: 12)).foregroundStyle(Theme.textDim)
                .fixedSize(horizontal: false, vertical: true)
        }
    }
}

// MARK: - Alert creation sheet

struct AlertSheet: View {
    let code: String
    let current: Double
    @EnvironmentObject var auth: AuthStore
    @Environment(\.dismiss) private var dismiss

    @State private var direction = "above"
    @State private var threshold = ""
    @State private var status: String?
    @State private var busy = false

    var body: some View {
        VStack(alignment: .leading, spacing: 18) {
            Text("Alarm Kur").font(.system(size: 20, weight: .bold)).foregroundStyle(Theme.text)
            Text("\(code) fiyatı eşiği geçtiğinde bir kez bildirim al. Tetiklenen alarm otomatik kaldırılır.")
                .font(.system(size: 13)).foregroundStyle(Theme.textDim)

            SegBar(items: [("above", "Üstüne çıkınca"), ("below", "Altına inince")],
                   selection: $direction)

            HStack {
                Text("₺").foregroundStyle(Theme.textDim)
                TextField("", text: $threshold,
                          prompt: Text(Fmt.compactPrice(current)).foregroundColor(Theme.textFaint))
                    .keyboardType(.decimalPad)
                    .font(Theme.number(20))
                    .foregroundStyle(Theme.text)
            }
            .padding(14)
            .background(Theme.surface, in: RoundedRectangle(cornerRadius: 10, style: .continuous))

            if let status {
                Text(status).font(.system(size: 13)).foregroundStyle(Theme.textDim)
            }

            Button {
                Task { await create() }
            } label: {
                if busy { ProgressView().tint(.black) } else { Text("Alarmı Oluştur") }
            }
            .buttonStyle(FilledButtonStyle())
            .disabled(parsedThreshold == nil || busy)

            Spacer(minLength: 0)
        }
        .padding(20)
    }

    private func create() async {
        guard let value = parsedThreshold else { return }
        busy = true; defer { busy = false }
        do {
            try await APIClient(token: auth.accessToken)
                .createAlert(code: code, direction: direction, threshold: value)
            PushManager.shared.requestPermission()
            status = "Alarm oluşturuldu ✅"
            try? await Task.sleep(nanoseconds: 700_000_000)
            dismiss()
        } catch {
            status = error.localizedDescription
        }
    }

    private var parsedThreshold: Double? {
        guard let value = Double(threshold.replacingOccurrences(of: ",", with: ".")),
              value > 0, value.isFinite else { return nil }
        return value
    }
}

// MARK: - Metric grid (Binance-style labelled stat cards)

struct Metric: Identifiable {
    let id = UUID()
    let label: String
    let value: String
    var tint: Color?
    var sub: String?

    init(_ label: String, _ value: String, tint: Color? = nil, sub: String? = nil) {
        self.label = label
        self.value = value
        self.tint = tint
        self.sub = sub
    }

    static func placeholders(_ n: Int) -> [Metric] {
        (0..<n).map { _ in Metric("", "") }
    }
}

struct MetricGrid: View {
    var title: String?
    let metrics: [Metric]
    var columns: Int = 2
    var loading: Bool = false

    var body: some View {
        Card(padding: 14) {
            VStack(alignment: .leading, spacing: 14) {
                if let title {
                    Text(title)
                        .font(.system(size: 13, weight: .bold))
                        .foregroundStyle(Theme.text)
                }
                LazyVGrid(
                    columns: Array(repeating: GridItem(.flexible(), alignment: .topLeading), count: columns),
                    alignment: .leading, spacing: 16
                ) {
                    ForEach(metrics) { m in
                        VStack(alignment: .leading, spacing: 3) {
                            if loading {
                                SkeletonBlock(width: 52, height: 9)
                                SkeletonBlock(width: 74, height: 15)
                            } else {
                                Text(m.label)
                                    .font(.system(size: 11))
                                    .foregroundStyle(Theme.textDim)
                                    .lineLimit(1)
                                Text(m.value)
                                    .font(Theme.number(15, weight: .semibold))
                                    .foregroundStyle(m.tint ?? Theme.text)
                                    .lineLimit(1)
                                    .minimumScaleFactor(0.6)
                                if let sub = m.sub, !sub.isEmpty {
                                    Text(sub)
                                        .font(.system(size: 10))
                                        .foregroundStyle(Theme.textFaint)
                                }
                            }
                        }
                        .frame(maxWidth: .infinity, alignment: .leading)
                    }
                }
                .shimmerIf(loading)
            }
        }
    }
}
