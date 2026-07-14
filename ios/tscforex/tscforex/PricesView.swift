//
//  PricesView.swift
//  MarketsView — Binance-style market list. Main list keeps a fixed, curated
//  order; the Favorites tab is drag-reorderable and persisted. Includes a Top
//  Movers strip, search, live flashing prices and DB-seeded sparklines.
//

import SwiftUI

enum MarketCategory: String, CaseIterable { case favorites, all, gold, currency, crypto, stock }

// Curated fixed order for the main list: gold, then majors, then crypto/stock.
private let fixedOrder: [String: Int] = {
    let codes = [
        // Gold
        "GRAM_GOLD", "CEYREK_GOLD", "YARIM_GOLD", "TAM_GOLD",
        "CUMHURIYET_GOLD", "ATA_GOLD", "GRAM_22K", "GRAM_18K", "GRAM_14K", "ONS_GOLD",
        // Major currencies
        "USDTRY", "EURTRY", "GBPTRY", "CHFTRY", "JPYTRY",
        // Crypto (by market cap)
        "BTC", "ETH", "BNB", "SOL", "XRP", "ADA", "DOGE", "TRX", "AVAX", "DOT",
        "LINK", "MATIC", "LTC", "ATOM", "XLM", "SHIB", "BCH",
        // BIST equities
        "THYAO", "GARAN", "AKBNK", "ISCTR", "YKBNK", "SISE", "EREGL", "KCHOL",
        "SAHOL", "ASELS", "BIMAS", "TUPRS", "FROTO", "TCELL", "PGSUS", "KOZAL", "TOASO", "SASA",
    ]
    var m: [String: Int] = [:]
    for (i, c) in codes.enumerated() { m[c] = i }
    return m
}()

private func kindRank(_ kind: String) -> Int {
    switch kind {
    case "gold": return 0
    case "currency": return 100
    case "crypto": return 400
    case "stock": return 700
    default: return 900
    }
}

private func fixedRank(_ p: InstrumentPrice) -> Int {
    if let r = fixedOrder[p.code] { return r }
    return kindRank(p.kind) + 90 + Int(p.code.hashValue & 0x3F)
}

struct MarketsView: View {
    @EnvironmentObject var prices: PriceStore
    @State private var query = ""
    @State private var category: MarketCategory = .all
    @State private var editMode: EditMode = .inactive
    @State private var path = NavigationPath()

    private var tabs: [(value: MarketCategory, label: String, icon: String)] {
        [(.favorites, "Favoriler", "star.fill"),
         (.all, "Tümü", "square.grid.2x2.fill"),
         (.gold, "Altın", "circle.hexagongrid.fill"),
         (.currency, "Döviz", "banknote.fill"),
         (.crypto, "Kripto", "bitcoinsign.circle.fill"),
         (.stock, "Hisse", "chart.line.uptrend.xyaxis")]
    }

    private func filtered(_ base: [InstrumentPrice]) -> [InstrumentPrice] {
        guard !query.isEmpty else { return base }
        let q = query.lowercased()
        return base.filter { $0.displayName.lowercased().contains(q) || $0.code.lowercased().contains(q) }
    }

    private var mainRows: [InstrumentPrice] {
        var base: [InstrumentPrice]
        switch category {
        case .favorites: base = prices.orderedFavorites()
        case .all:       base = prices.sorted
        case .gold:      base = prices.gold()
        case .currency:  base = prices.currencies()
        case .crypto:    base = prices.crypto()
        case .stock:     base = prices.stocks()
        }
        if category != .favorites { base.sort { fixedRank($0) < fixedRank($1) } }
        return filtered(base)
    }

    var body: some View {
        NavigationStack(path: $path) {
            ZStack {
                Theme.bg.ignoresSafeArea()
                VStack(spacing: 0) {
                    header
                    searchBar
                    categoryStrip
                    columnHeader
                    content
                }
            }
            .navigationDestination(for: String.self) { code in
                AssetDetailView(code: code)
            }
            .toolbar(.hidden, for: .navigationBar)
        }
    }

    // MARK: Header

    private var header: some View {
        HStack(alignment: .center) {
            VStack(alignment: .leading, spacing: 2) {
                Text("Piyasalar").font(.system(size: 28, weight: .bold)).foregroundStyle(Theme.text)
                Text("Altın · Döviz · Kripto · Hisse").font(.system(size: 12)).foregroundStyle(Theme.textDim)
            }
            Spacer()
            if category == .favorites && !prices.orderedFavorites().isEmpty {
                Button {
                    withAnimation { editMode = editMode == .active ? .inactive : .active }
                    Haptics.tap()
                } label: {
                    Text(editMode == .active ? "Bitti" : "Düzenle")
                        .font(.system(size: 14, weight: .semibold)).foregroundStyle(Theme.accent)
                }
            } else {
                ConnectionDot(connected: prices.connected)
            }
        }
        .padding(.horizontal, 16).padding(.top, 6).padding(.bottom, 10)
    }

    private var searchBar: some View {
        HStack(spacing: 8) {
            Image(systemName: "magnifyingglass").foregroundStyle(Theme.textDim)
            TextField("", text: $query, prompt: Text("Ara: Gram Altın, Dolar, Euro…").foregroundColor(Theme.textFaint))
                .foregroundStyle(Theme.text)
                .textInputAutocapitalization(.never)
                .autocorrectionDisabled()
            if !query.isEmpty {
                Button { query = "" } label: {
                    Image(systemName: "xmark.circle.fill").foregroundStyle(Theme.textFaint)
                }
            }
        }
        .padding(.horizontal, 12).padding(.vertical, 11)
        .background(Theme.surface, in: RoundedRectangle(cornerRadius: 12, style: .continuous))
        .padding(.horizontal, 16)
    }

    // MARK: Category strip (horizontally scrollable: Favoriler/Tümü/Altın/Döviz/Kripto/Hisse)

    private var categoryStrip: some View {
        ScrollView(.horizontal, showsIndicators: false) {
            HStack(spacing: 8) {
                ForEach(tabs, id: \.value) { tab in
                    let active = category == tab.value
                    Button {
                        withAnimation(.easeInOut(duration: 0.15)) { category = tab.value }
                        Haptics.tap(); editMode = .inactive
                    } label: {
                        HStack(spacing: 6) {
                            Image(systemName: tab.icon).font(.system(size: 11, weight: .bold))
                            Text(tab.label).font(.system(size: 13, weight: active ? .bold : .semibold))
                        }
                        .foregroundStyle(active ? Color.white : Theme.textDim)
                        .padding(.horizontal, 14).padding(.vertical, 8)
                        .background(active ? Theme.accent : Theme.surface,
                                    in: Capsule())
                        .overlay(Capsule().stroke(active ? .clear : Theme.stroke, lineWidth: 1))
                    }
                    .buttonStyle(.plain)
                }
            }
            .padding(.horizontal, 16)
        }
        .padding(.top, 10).padding(.bottom, 2)
    }

    // MARK: Column header

    private var columnHeader: some View {
        HStack {
            Text("İsim").font(.system(size: 11, weight: .medium)).foregroundStyle(Theme.textDim)
            Spacer()
            Text("Son Fiyat / 24s %").font(.system(size: 11, weight: .medium)).foregroundStyle(Theme.textDim)
        }
        .padding(.horizontal, 16).padding(.top, 14).padding(.bottom, 4)
    }

    // MARK: Content

    @ViewBuilder
    private var content: some View {
        if prices.prices.isEmpty {
            ScrollView { MarketListSkeleton(rows: 12) }
        } else if category == .favorites {
            favoritesList
        } else {
            normalList
        }
    }

    private var normalList: some View {
        ScrollView {
            LazyVStack(spacing: 0) {
                ForEach(mainRows) { p in
                    NavigationLink(value: p.code) {
                        MarketRow(price: p, spark: prices.spark[p.code] ?? [],
                                  isFav: prices.isFavorite(p.code))
                    }
                    .buttonStyle(.plain)
                    Divider().overlay(Theme.stroke).padding(.leading, 62)
                }
            }
            .padding(.bottom, 96)
        }
        .refreshable { await prices.refresh() }
    }

    @ViewBuilder
    private var favoritesList: some View {
        let favs = filtered(prices.orderedFavorites())
        if favs.isEmpty {
            VStack(spacing: 10) {
                Image(systemName: "star").font(.system(size: 34)).foregroundStyle(Theme.textFaint)
                Text("Henüz favori yok").font(.system(size: 16, weight: .semibold)).foregroundStyle(Theme.text)
                Text("Bir varlığın yıldızına dokunarak favorilere ekleyin, sonra buradan sürükleyip sıralayın.")
                    .font(.system(size: 13)).foregroundStyle(Theme.textDim)
                    .multilineTextAlignment(.center).padding(.horizontal, 40)
            }
            .frame(maxWidth: .infinity, maxHeight: .infinity)
        } else {
            List {
                ForEach(favs) { p in
                    MarketRow(price: p, spark: prices.spark[p.code] ?? [],
                              isFav: true)
                        .listRowInsets(EdgeInsets(top: 0, leading: 8, bottom: 0, trailing: 8))
                        .listRowBackground(Theme.bg)
                        .listRowSeparator(.hidden)
                        .contentShape(Rectangle())
                        .onTapGesture { if editMode == .inactive { path.append(p.code) } }
                }
                .onMove { prices.moveFavorite(from: $0, to: $1); Haptics.soft() }
                Color.clear.frame(height: 96)
                    .listRowBackground(Theme.bg).listRowSeparator(.hidden)
            }
            .listStyle(.plain)
            .scrollContentBackground(.hidden)
            .environment(\.editMode, $editMode)
        }
    }
}

struct MarketRow: View {
    let price: InstrumentPrice
    let spark: [Double]
    let isFav: Bool

    var body: some View {
        HStack(spacing: 10) {
            AssetIcon(code: price.code, kind: price.kind, size: 36)
            VStack(alignment: .leading, spacing: 3) {
                HStack(spacing: 5) {
                    Text(price.displayName).font(.system(size: 15, weight: .semibold))
                        .foregroundStyle(Theme.text).lineLimit(1).truncationMode(.tail)
                    if isFav {
                        Image(systemName: "star.fill").font(.system(size: 8)).foregroundStyle(Theme.accent)
                    }
                }
                Text(price.code).font(.system(size: 11)).foregroundStyle(Theme.textDim).lineLimit(1)
            }
            .frame(maxWidth: .infinity, alignment: .leading)
            Sparkline(points: Array(spark.suffix(12)), color: Theme.trend(price.changePct))
                .frame(width: 44, height: 24)
            VStack(alignment: .trailing, spacing: 4) {
                FlashText(value: price.price,
                          font: Theme.number(15, weight: .bold),
                          format: { Fmt.tl($0) })
                ChangePill(pct: price.changePct)
            }
            .frame(width: 96, alignment: .trailing)
        }
        .padding(.horizontal, 16)
        .frame(height: 62)
        .contentShape(Rectangle())
    }
}
