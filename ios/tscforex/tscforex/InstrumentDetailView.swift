//
//  InstrumentDetailView.swift
//  AssetDetailView — Binance-style asset screen: large live price, 24h stats,
//  interactive Swift Charts (line/candle + timeframes), and alert creation.
//

import SwiftUI

private struct Timeframe: Hashable { let interval: String; let label: String }

struct AssetDetailView: View {
    let code: String
    @EnvironmentObject var auth: AuthStore
    @EnvironmentObject var prices: PriceStore
    @Environment(\.dismiss) private var dismiss

    @State private var candles: [Candle] = []
    @State private var stats: InstrumentStats?
    @State private var mode: ChartMode = .line
    @State private var tf = "5m"
    @State private var loadingChart = false
    @State private var showAlert = false
    @State private var convAmount = "1"

    private let timeframes = [
        Timeframe(interval: "1m", label: "1dk"),
        Timeframe(interval: "5m", label: "5dk"),
        Timeframe(interval: "15m", label: "15dk"),
        Timeframe(interval: "1h", label: "1sa"),
        Timeframe(interval: "1d", label: "1g"),
    ]

    private var price: InstrumentPrice? { prices.prices[code] }
    private var trendUp: Bool { (price?.changePct ?? 0) >= 0 }

    var body: some View {
        ZStack {
            Theme.bg.ignoresSafeArea()
            ScrollView {
                VStack(spacing: 16) {
                    priceHeader
                    actionsRow
                    stats24h
                    chartCard
                    bidAskCard
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
                Button { Haptics.tap(); prices.toggleFavorite(code) } label: {
                    Image(systemName: prices.isFavorite(code) ? "star.fill" : "star")
                        .foregroundStyle(prices.isFavorite(code) ? Theme.accent : Theme.textDim)
                }
            }
        }
        .toolbarBackground(Theme.bg, for: .navigationBar)
        .toolbarBackground(.visible, for: .navigationBar)
        .task(id: tf) { await loadChart() }
        .task { await loadStats() }
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
                          format: { Fmt.tl($0) })
                HStack(spacing: 8) {
                    Text("\(Fmt.signed(p.change)) ₺  \(Fmt.signedPct(p.changePct))")
                        .font(Theme.number(15, weight: .semibold))
                        .foregroundStyle(Theme.trend(p.changePct))
                }
            } else {
                Text("—").font(Theme.number(38, weight: .bold)).foregroundStyle(Theme.textDim)
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
    }

    // MARK: Actions (inline, above the fold — not hidden behind the tab bar)

    private var actionsRow: some View {
        HStack(spacing: 12) {
            Button { Haptics.tap(); prices.toggleFavorite(code) } label: {
                Label(prices.isFavorite(code) ? "Favoride" : "Favori Ekle",
                      systemImage: prices.isFavorite(code) ? "star.fill" : "star")
            }
            .buttonStyle(FilledButtonStyle(color: Theme.surfaceHi, textColor: Theme.text))
            .frame(maxWidth: .infinity)

            Button { Haptics.tap(); showAlert = true } label: {
                Label("Alarm Kur", systemImage: "bell.badge")
            }
            .buttonStyle(FilledButtonStyle())
            .frame(maxWidth: .infinity)
        }
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
        return Fmt.tl(v)
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
                    PriceChart(candles: candles, mode: mode, trendUp: trendUp)
                        .clipShape(RoundedRectangle(cornerRadius: 8, style: .continuous))
                }
                SegBar(items: timeframes.map { ($0.interval, $0.label) }, selection: $tf)
            }
        }
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
        let client = APIClient(token: auth.accessToken)
        if let c = try? await client.candles(code: code, interval: tf) {
            candles = c
        } else {
            candles = []
        }
    }

    private func loadStats() async {
        let client = APIClient(token: auth.accessToken)
        stats = try? await client.stats(code: code)
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
