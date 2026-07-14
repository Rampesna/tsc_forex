//
//  FavoritesWidget.swift
//  Home-screen widget that shows the top favorites of the signed-in account, in
//  the order the user arranged them in the app. The favorite list + order come
//  from the shared App Group container the app writes to; current prices are
//  refreshed directly from the public prices API on each timeline reload so the
//  widget stays fresh even when the app isn't open.
//

import WidgetKit
import SwiftUI

// MARK: - Shared storage (favorite list + order, written by the app)

enum Shared {
    static let appGroup = "group.talhacan.tscforex"
    static let storageKey = "favorites_widget_v1"

    struct Item: Codable, Hashable {
        let code: String
        let name: String
        let price: Double
        let changePct: Double
    }

    static func load() -> [Item] {
        guard let defaults = UserDefaults(suiteName: appGroup),
              let data = defaults.data(forKey: storageKey),
              let items = try? JSONDecoder().decode([Item].self, from: data)
        else { return [] }
        return items
    }
}

// MARK: - Live prices (public API, no auth)

private enum PriceAPI {
    static let base = "https://tsc-api.talhacan.com/api/v1"

    private struct Row: Decodable {
        let code: String
        let price: Double
        let changePct: Double
        enum CodingKeys: String, CodingKey { case code, price; case changePct = "change_pct" }
    }
    private struct Resp: Decodable { let prices: [Row] }

    /// Returns code → (price, changePct). Empty on any failure (widget then
    /// falls back to the last snapshot the app wrote).
    static func fetch() async -> [String: (price: Double, pct: Double)] {
        guard let url = URL(string: base + "/instruments/prices") else { return [:] }
        var req = URLRequest(url: url, timeoutInterval: 10)
        req.cachePolicy = .reloadIgnoringLocalCacheData
        guard let (data, resp) = try? await URLSession.shared.data(for: req),
              let http = resp as? HTTPURLResponse, (200..<300).contains(http.statusCode),
              let decoded = try? JSONDecoder().decode(Resp.self, from: data)
        else { return [:] }
        var map: [String: (Double, Double)] = [:]
        for r in decoded.prices { map[r.code] = (r.price, r.changePct) }
        return map
    }
}

// MARK: - Palette

private enum WTheme {
    static let text = Color(red: 0.97, green: 0.97, blue: 0.98)
    static let dim = Color(red: 0.62, green: 0.65, blue: 0.70)
    static let faint = Color(red: 0.42, green: 0.45, blue: 0.50)
    static let accent = Color(red: 0.18, green: 0.66, blue: 0.75)
    static let up = Color(red: 0.31, green: 0.80, blue: 0.60)
    static let down = Color(red: 0.96, green: 0.42, blue: 0.49)
    static let upSoft = Color(red: 0.31, green: 0.80, blue: 0.60).opacity(0.14)
    static let downSoft = Color(red: 0.96, green: 0.42, blue: 0.49).opacity(0.14)
    static let bgTop = Color(red: 0.07, green: 0.08, blue: 0.09)
    static let bgBottom = Color(red: 0.04, green: 0.045, blue: 0.05)
    static let hair = Color.white.opacity(0.05)
}

// MARK: - Formatting

private func fmtTRY(_ v: Double) -> String {
    let nf = NumberFormatter()
    nf.numberStyle = .decimal
    nf.locale = Locale(identifier: "tr_TR")
    let frac = (v > 0 && v < 1) ? 4 : 2
    nf.minimumFractionDigits = frac
    nf.maximumFractionDigits = frac
    let s = nf.string(from: NSNumber(value: v)) ?? String(format: "%.2f", v)
    return "₺" + s
}

private func fmtPct(_ v: Double) -> String {
    (v > 0 ? "+" : "") + String(format: "%.2f", v) + "%"
}

private func hhmm(_ d: Date) -> String {
    let f = DateFormatter()
    f.locale = Locale(identifier: "tr_TR")
    f.dateFormat = "HH:mm"
    return f.string(from: d)
}

// MARK: - Timeline

struct FavEntry: TimelineEntry {
    let date: Date
    let items: [Shared.Item]
}

struct FavProvider: TimelineProvider {
    func placeholder(in context: Context) -> FavEntry {
        FavEntry(date: Date(), items: FavProvider.sample)
    }

    func getSnapshot(in context: Context, completion: @escaping (FavEntry) -> Void) {
        let items = Shared.load()
        completion(FavEntry(date: Date(), items: items.isEmpty ? FavProvider.sample : items))
    }

    func getTimeline(in context: Context, completion: @escaping (Timeline<FavEntry>) -> Void) {
        Task {
            var items = Shared.load()
            let live = await PriceAPI.fetch()
            if !live.isEmpty {
                items = items.map { it in
                    if let p = live[it.code] {
                        return Shared.Item(code: it.code, name: it.name, price: p.price, changePct: p.pct)
                    }
                    return it
                }
            }
            let entry = FavEntry(date: Date(), items: items)
            // Ask WidgetKit to refresh ~every 10 min. iOS meters widget refreshes,
            // so this is best-effort; the app also forces a reload when opened.
            let next = Calendar.current.date(byAdding: .minute, value: 10, to: Date())
                ?? Date().addingTimeInterval(600)
            completion(Timeline(entries: [entry], policy: .after(next)))
        }
    }

    static let sample: [Shared.Item] = [
        .init(code: "GRAM_GOLD", name: "Gram Altın", price: 6060.30, changePct: 0.59),
        .init(code: "CEYREK_GOLD", name: "Çeyrek Altın", price: 9785.00, changePct: 0.59),
        .init(code: "USDTRY", name: "Dolar", price: 42.18, changePct: -0.21),
        .init(code: "EURTRY", name: "Euro", price: 45.63, changePct: 0.34),
    ]
}

// MARK: - Views

struct FavoritesWidgetView: View {
    @Environment(\.widgetFamily) private var family
    let entry: FavEntry

    private var maxRows: Int { family == .systemSmall ? 3 : 4 }

    var body: some View {
        VStack(alignment: .leading, spacing: 0) {
            header
            if entry.items.isEmpty {
                emptyState
            } else {
                VStack(spacing: family == .systemSmall ? 7 : 9) {
                    ForEach(Array(entry.items.prefix(maxRows)), id: \.code) { item in
                        row(item)
                    }
                }
                .padding(.top, family == .systemSmall ? 8 : 10)
                Spacer(minLength: 0)
            }
        }
        .containerBackground(for: .widget) {
            LinearGradient(colors: [WTheme.bgTop, WTheme.bgBottom],
                           startPoint: .top, endPoint: .bottom)
        }
    }

    private var header: some View {
        HStack(spacing: 7) {
            RoundedRectangle(cornerRadius: 6, style: .continuous)
                .fill(WTheme.accent)
                .frame(width: 18, height: 18)
                .overlay(
                    Image(systemName: "star.fill")
                        .font(.system(size: 9, weight: .black))
                        .foregroundStyle(.white)
                )
            Text("Favoriler")
                .font(.system(size: 13, weight: .bold))
                .foregroundStyle(WTheme.text)
            Spacer(minLength: 0)
            HStack(spacing: 3) {
                Circle().fill(WTheme.up).frame(width: 5, height: 5)
                Text(hhmm(entry.date))
                    .font(.system(size: 10, weight: .medium, design: .rounded))
                    .foregroundStyle(WTheme.faint)
            }
        }
    }

    private func row(_ item: Shared.Item) -> some View {
        HStack(spacing: 8) {
            Text(item.name)
                .font(.system(size: 13, weight: .semibold))
                .foregroundStyle(WTheme.text)
                .lineLimit(1)
                .minimumScaleFactor(0.8)
                .layoutPriority(1)
            Spacer(minLength: 6)
            VStack(alignment: .trailing, spacing: 2) {
                Text(fmtTRY(item.price))
                    .font(.system(size: 13, weight: .bold, design: .rounded))
                    .foregroundStyle(WTheme.text)
                    .lineLimit(1)
                    .minimumScaleFactor(0.75)
                changeChip(item.changePct)
            }
            .fixedSize(horizontal: true, vertical: false)
        }
    }

    private func changeChip(_ pct: Double) -> some View {
        let positive = pct >= 0
        return HStack(spacing: 2) {
            Image(systemName: positive ? "arrowtriangle.up.fill" : "arrowtriangle.down.fill")
                .font(.system(size: 7, weight: .bold))
            Text(fmtPct(pct))
                .font(.system(size: 10, weight: .bold, design: .rounded))
        }
        .foregroundStyle(positive ? WTheme.up : WTheme.down)
        .padding(.horizontal, 6)
        .padding(.vertical, 2)
        .background(positive ? WTheme.upSoft : WTheme.downSoft,
                    in: Capsule())
    }

    private var emptyState: some View {
        VStack(spacing: 5) {
            Spacer()
            Image(systemName: "star")
                .font(.system(size: 22))
                .foregroundStyle(WTheme.faint)
            Text("Henüz favori yok")
                .font(.system(size: 12, weight: .semibold))
                .foregroundStyle(WTheme.text)
            Text("Uygulamada yıldıza dokunun")
                .font(.system(size: 10))
                .foregroundStyle(WTheme.dim)
            Spacer()
        }
        .frame(maxWidth: .infinity)
    }
}

struct FavoritesWidget: Widget {
    let kind = "FavoritesWidget"

    var body: some WidgetConfiguration {
        StaticConfiguration(kind: kind, provider: FavProvider()) { entry in
            FavoritesWidgetView(entry: entry)
        }
        .configurationDisplayName("Favoriler")
        .description("Favori listenizin en üstteki varlıklarını canlı fiyatlarıyla gösterir.")
        .supportedFamilies([.systemSmall, .systemMedium])
    }
}
