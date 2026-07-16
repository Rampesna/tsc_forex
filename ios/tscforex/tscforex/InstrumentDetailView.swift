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
    @Environment(\.dismiss) private var dismiss

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
    // Sliding window: fetch older/newer pages of this size and keep at most
    // `maxWindow` candles rendered so Swift Charts stays smooth no matter how
    // far the user scrolls into history.
    private let pageSize = 300
    private let maxWindow = 900
    @State private var showAlert = false
    @State private var convAmount = "1"

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

    // Fold the live WS price into the last candle so the chart moves in real time
    // between fetches (Binance-style) without refetching history.
    private var displayCandles: [Candle] {
        // Only fold the live price in when the present is actually on screen.
        guard !hasMoreFuture, let p = price?.price, p > 0, let last = candles.last else { return candles }
        var arr = candles
        arr[arr.count - 1] = Candle(
            bucket: last.bucket,
            open: last.open,
            high: max(last.high, p),
            low: min(last.low, p),
            close: p,
            count: last.count
        )
        return arr
    }

    var body: some View {
        ZStack {
            Theme.bg.ignoresSafeArea()
            ScrollView {
                VStack(spacing: 16) {
                    priceHeader
                    stats24h
                    chartCard
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
                    Button { Haptics.tap(); prices.toggleFavorite(code) } label: {
                        Image(systemName: prices.isFavorite(code) ? "star.fill" : "star")
                            .foregroundStyle(prices.isFavorite(code) ? Theme.accent : Theme.textDim)
                    }
                    Button { Haptics.tap(); showAlert = true } label: {
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
        .sheet(isPresented: $showAlert) {
            AlertSheet(code: code, current: price?.price ?? stats?.last ?? 0)
                .environmentObject(auth)
                .presentationDetents([.height(380)])
                .presentationBackground(Theme.bgElevated)
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
                    PriceChart(candles: displayCandles, mode: mode, trendUp: trendUp,
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
        let client = APIClient(token: auth.accessToken)
        let frame = timeframes.first { $0.key == tf } ?? timeframes[0]
        let from = Calendar.current.date(byAdding: .day, value: -frame.days, to: Date())
        if let c = try? await client.candles(code: code, interval: frame.interval, from: from) {
            // Keep the render window bounded even on the initial fetch.
            candles = Array(c.suffix(maxWindow))
            if c.count > maxWindow { hasMoreHistory = true }
        } else {
            candles = []
        }
        // Re-center the visible window on the freshly loaded interval.
        chartReset += 1
    }

    // Lazy-load an older page when the user pans to the start of the chart, then
    // drop the newest candles that scrolled off-screen so the rendered set stays
    // bounded (sliding window). Each page is an indexed range read on the compact
    // candles table (no perf hit); scroll position is preserved because the user
    // is anchored to the OLD edge we're keeping.
    private func loadOlder() async {
        guard !loadingOlder, !loadingNewer, hasMoreHistory, let oldest = candles.first?.bucket else { return }
        loadingOlder = true; defer { loadingOlder = false }
        let client = APIClient(token: auth.accessToken)
        let frame = timeframes.first { $0.key == tf } ?? timeframes[0]
        let to = oldest.addingTimeInterval(-1)
        guard let older = try? await client.candles(code: code, interval: frame.interval, to: to, limit: pageSize),
              !older.isEmpty else {
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
        let client = APIClient(token: auth.accessToken)
        let frame = timeframes.first { $0.key == tf } ?? timeframes[0]
        let from = newest.addingTimeInterval(1)
        guard let newer = try? await client.candles(code: code, interval: frame.interval, from: from, limit: pageSize),
              !newer.isEmpty else {
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
        let step = stepSeconds(frame.interval)
        if newer.count < pageSize || (merged.last?.bucket ?? .distantPast) >= Date().addingTimeInterval(-step * 2) {
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
            Text("\(code) fiyatı eşiği geçtiğinde bildirim al.")
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
            .disabled(Double(threshold.replacingOccurrences(of: ",", with: ".")) == nil || busy)

            Spacer(minLength: 0)
        }
        .padding(20)
    }

    private func create() async {
        guard let value = Double(threshold.replacingOccurrences(of: ",", with: ".")) else { return }
        busy = true; defer { busy = false }
        do {
            try await APIClient(token: auth.accessToken)
                .createAlert(code: code, direction: direction, threshold: value)
            status = "Alarm oluşturuldu ✅"
            try? await Task.sleep(nanoseconds: 700_000_000)
            dismiss()
        } catch {
            status = error.localizedDescription
        }
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
