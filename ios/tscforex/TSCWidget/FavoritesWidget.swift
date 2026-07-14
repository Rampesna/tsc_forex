//
//  FavoritesWidget.swift
//  Home-screen widget that shows the top favorites of the signed-in account, in
//  the order the user arranged them in the app. Data is read from the shared
//  App Group container the app writes to — no networking or auth in the widget.
//

import WidgetKit
import SwiftUI

// MARK: - Shared storage

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

// MARK: - Palette

private enum WTheme {
    static let bg = Color(red: 0.04, green: 0.05, blue: 0.06)
    static let card = Color(red: 0.09, green: 0.10, blue: 0.11)
    static let text = Color(red: 0.97, green: 0.97, blue: 0.97)
    static let dim = Color(red: 0.61, green: 0.64, blue: 0.69)
    static let accent = Color(red: 0.18, green: 0.66, blue: 0.75)
    static let up = Color(red: 0.31, green: 0.79, blue: 0.60)
    static let down = Color(red: 0.95, green: 0.42, blue: 0.49)
}

// MARK: - Formatting

private func fmtTRY(_ v: Double) -> String {
    let nf = NumberFormatter()
    nf.numberStyle = .decimal
    nf.locale = Locale(identifier: "tr_TR")
    nf.minimumFractionDigits = (v > 0 && v < 1) ? 4 : 2
    nf.maximumFractionDigits = (v > 0 && v < 1) ? 4 : 2
    let s = nf.string(from: NSNumber(value: v)) ?? String(format: "%.2f", v)
    return "₺" + s
}

private func fmtPct(_ v: Double) -> String {
    let sign = v > 0 ? "+" : ""
    return sign + String(format: "%.2f", v) + "%"
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
        let entry = FavEntry(date: Date(), items: Shared.load())
        // Refresh roughly every 15 minutes; the app also pushes fresh data (and
        // forces a reload) whenever it's open, so this is just a safety net.
        let next = Calendar.current.date(byAdding: .minute, value: 15, to: Date()) ?? Date().addingTimeInterval(900)
        completion(Timeline(entries: [entry], policy: .after(next)))
    }

    fileprivate static let sample: [Shared.Item] = [
        .init(code: "GRAMALTIN", name: "Gram Altın", price: 6058.30, changePct: 0.59),
        .init(code: "USDTRY", name: "Dolar", price: 42.18, changePct: -0.21),
        .init(code: "EURTRY", name: "Euro", price: 45.63, changePct: 0.34),
        .init(code: "BTC", name: "Bitcoin", price: 4180500, changePct: 1.82),
    ]
}

// MARK: - Views

struct FavoritesWidgetView: View {
    @Environment(\.widgetFamily) private var family
    let entry: FavEntry

    private var maxRows: Int { family == .systemSmall ? 2 : 4 }

    var body: some View {
        VStack(alignment: .leading, spacing: 0) {
            header
            if entry.items.isEmpty {
                emptyState
            } else {
                VStack(spacing: 0) {
                    ForEach(Array(entry.items.prefix(maxRows).enumerated()), id: \.element.code) { idx, item in
                        row(item)
                        if idx < min(entry.items.count, maxRows) - 1 {
                            Divider().overlay(Color.white.opacity(0.06))
                        }
                    }
                }
                .padding(.top, 6)
                Spacer(minLength: 0)
            }
        }
        .containerBackground(WTheme.bg, for: .widget)
    }

    private var header: some View {
        HStack(spacing: 7) {
            RoundedRectangle(cornerRadius: 5, style: .continuous)
                .fill(WTheme.accent)
                .frame(width: 16, height: 16)
                .overlay(
                    Text("T")
                        .font(.system(size: 10, weight: .black, design: .rounded))
                        .foregroundStyle(.white)
                )
            Text("Favoriler")
                .font(.system(size: 13, weight: .semibold))
                .foregroundStyle(WTheme.text)
            Spacer()
            if family != .systemSmall {
                Text("TSC Forex")
                    .font(.system(size: 10, weight: .medium))
                    .foregroundStyle(WTheme.dim)
            }
        }
    }

    private func row(_ item: Shared.Item) -> some View {
        HStack(spacing: 8) {
            VStack(alignment: .leading, spacing: 1) {
                Text(item.name)
                    .font(.system(size: 12, weight: .semibold))
                    .foregroundStyle(WTheme.text)
                    .lineLimit(1)
                Text(item.code)
                    .font(.system(size: 9, weight: .medium))
                    .foregroundStyle(WTheme.dim)
                    .lineLimit(1)
            }
            Spacer(minLength: 4)
            VStack(alignment: .trailing, spacing: 1) {
                Text(fmtTRY(item.price))
                    .font(.system(size: 12, weight: .bold, design: .rounded))
                    .foregroundStyle(WTheme.text)
                    .lineLimit(1)
                Text(fmtPct(item.changePct))
                    .font(.system(size: 9, weight: .semibold, design: .rounded))
                    .foregroundStyle(item.changePct >= 0 ? WTheme.up : WTheme.down)
            }
        }
        .padding(.vertical, 6)
    }

    private var emptyState: some View {
        VStack(spacing: 4) {
            Spacer()
            Text("Henüz favori yok")
                .font(.system(size: 12, weight: .semibold))
                .foregroundStyle(WTheme.text)
            Text("Uygulamada yıldıza dokunarak\nfavori ekleyin")
                .font(.system(size: 10))
                .foregroundStyle(WTheme.dim)
                .multilineTextAlignment(.center)
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
