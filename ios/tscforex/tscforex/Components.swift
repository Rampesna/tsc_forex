//
//  Components.swift
//  Reusable, theme-driven building blocks used across the app.
//

import SwiftUI
import Charts

// MARK: - Change pill (e.g. "+1.24%")

struct ChangePill: View {
    let pct: Double
    // Compact by default (market list); `large` for detail headers.
    var large: Bool = false
    var body: some View {
        Text(Fmt.signedPct(pct))
            .font(Theme.number(large ? 13 : 10.5, weight: .bold))
            .foregroundStyle(.white)
            .lineLimit(1)
            .fixedSize()
            .padding(.horizontal, large ? 8 : 5)
            .padding(.vertical, large ? 5 : 2.5)
            .background(Theme.trend(pct), in: RoundedRectangle(cornerRadius: large ? 6 : 5, style: .continuous))
    }
}

// MARK: - Flashing price text (green/red pulse on every change)

struct FlashText: View {
    let value: Double
    var font: Font = Theme.number(16, weight: .semibold)
    var format: (Double) -> String = { Fmt.compactPrice($0) }

    @State private var last: Double = .nan
    @State private var flash: Color = .clear

    var body: some View {
        Text(format(value))
            .font(font)
            .foregroundStyle(Theme.text)
            .lineLimit(1)
            .minimumScaleFactor(0.6)
            .monospacedDigit()
            .padding(.horizontal, 4)
            .background(flash.opacity(0.28), in: RoundedRectangle(cornerRadius: 4))
            .onChange(of: value) { _, newValue in
                guard !last.isNaN else { last = newValue; return }
                if newValue != last {
                    flash = newValue > last ? Theme.up : Theme.down
                    withAnimation(.easeOut(duration: 0.55)) { flash = .clear }
                    last = newValue
                }
            }
            .onAppear { if last.isNaN { last = value } }
    }
}

// MARK: - Sparkline (tiny trend line for list rows)

struct Sparkline: View {
    let points: [Double]
    let color: Color

    var body: some View {
        if points.count < 2 {
            Color.clear
        } else {
            let minV = points.min() ?? 0
            let maxV = points.max() ?? 1
            Chart {
                ForEach(Array(points.enumerated()), id: \.offset) { idx, v in
                    LineMark(x: .value("i", idx), y: .value("v", v))
                        .interpolationMethod(.catmullRom)
                        .foregroundStyle(color)
                        .lineStyle(StrokeStyle(lineWidth: 1.6, lineCap: .round, lineJoin: .round))
                }
            }
            .chartXAxis(.hidden)
            .chartYAxis(.hidden)
            .chartYScale(domain: minV...(maxV == minV ? minV + 0.0001 : maxV))
            .chartLegend(.hidden)
            .chartPlotStyle { $0.padding(.vertical, 3) }
        }
    }
}

// MARK: - Segmented control (timeframes / categories)

struct SegBar<T: Hashable>: View {
    let items: [(value: T, label: String)]
    @Binding var selection: T

    var body: some View {
        HStack(spacing: 4) {
            ForEach(items, id: \.value) { item in
                let active = item.value == selection
                Text(item.label)
                    .font(.system(size: 13, weight: active ? .bold : .medium))
                    .foregroundStyle(active ? Theme.text : Theme.textDim)
                    .frame(maxWidth: .infinity)
                    .padding(.vertical, 7)
                    .background(
                        active ? Theme.surfaceHi : .clear,
                        in: RoundedRectangle(cornerRadius: 8, style: .continuous)
                    )
                    .contentShape(Rectangle())
                    .onTapGesture {
                        withAnimation(.easeInOut(duration: 0.15)) { selection = item.value }
                    }
            }
        }
        .padding(4)
        .background(Theme.bgElevated, in: RoundedRectangle(cornerRadius: 10, style: .continuous))
    }
}

// MARK: - Stat tile (24h high / low / etc.)

struct StatTile: View {
    let label: String
    let value: String
    var tint: Color = Theme.text

    var body: some View {
        VStack(alignment: .leading, spacing: 3) {
            Text(label).font(.system(size: 11)).foregroundStyle(Theme.textDim).lineLimit(1)
            Text(value).font(Theme.number(14, weight: .semibold)).foregroundStyle(tint)
                .lineLimit(1).minimumScaleFactor(0.55)
        }
        .frame(maxWidth: .infinity, alignment: .leading)
    }
}

// MARK: - Asset icon (colored monogram badge)

struct AssetIcon: View {
    let code: String
    let kind: String
    var size: CGFloat = 38

    var body: some View {
        switch kind {
        case "gold":     goldBadge
        case "crypto":   cryptoBadge
        case "stock":    stockBadge
        default:         currencyBadge
        }
    }

    private var goldBadge: some View {
        ZStack {
            Circle().fill(Theme.accent.opacity(0.16))
            Image(systemName: "circle.hexagongrid.fill")
                .font(.system(size: size * 0.44, weight: .semibold))
                .foregroundStyle(LinearGradient(colors: [Color(hex: 0xFFD65A), Color(hex: 0xE0A21E)],
                                                startPoint: .top, endPoint: .bottom))
        }
        .frame(width: size, height: size).clipShape(Circle())
        .overlay(Circle().stroke(Theme.stroke, lineWidth: 1))
    }

    @ViewBuilder
    private var currencyBadge: some View {
        if let flag = Flags.emoji(forCurrencyPair: code) {
            ZStack {
                Circle().fill(Theme.surfaceHi)
                Text(flag).font(.system(size: size * 0.58)).minimumScaleFactor(0.5)
            }
            .frame(width: size, height: size).clipShape(Circle())
            .overlay(Circle().stroke(Theme.stroke, lineWidth: 1))
        } else {
            ZStack {
                Circle().fill(Color(hex: 0x4A90E2).opacity(0.16))
                Image(systemName: "banknote.fill")
                    .font(.system(size: size * 0.42, weight: .semibold))
                    .foregroundStyle(Color(hex: 0x4A90E2))
            }
            .frame(width: size, height: size).clipShape(Circle())
            .overlay(Circle().stroke(Theme.stroke, lineWidth: 1))
        }
    }

    private var cryptoBadge: some View {
        let brand = CryptoBrand.style(for: code)
        return ZStack {
            Circle().fill(brand.color.gradient)
            if let glyph = brand.glyph {
                Text(glyph).font(.system(size: size * 0.5, weight: .bold)).foregroundStyle(.white)
            } else {
                Text(code.prefix(3)).font(.system(size: size * 0.3, weight: .heavy))
                    .foregroundStyle(.white).minimumScaleFactor(0.5)
            }
        }
        .frame(width: size, height: size).clipShape(Circle())
        .overlay(Circle().stroke(Color.white.opacity(0.12), lineWidth: 1))
    }

    private var stockBadge: some View {
        let color = StockColor.color(for: code)
        return ZStack {
            RoundedRectangle(cornerRadius: size * 0.28, style: .continuous)
                .fill(LinearGradient(colors: [color.opacity(0.95), color.opacity(0.6)],
                                     startPoint: .topLeading, endPoint: .bottomTrailing))
            Text(code.prefix(2)).font(.system(size: size * 0.34, weight: .heavy))
                .foregroundStyle(.white)
        }
        .frame(width: size, height: size)
        .clipShape(RoundedRectangle(cornerRadius: size * 0.28, style: .continuous))
        .overlay(RoundedRectangle(cornerRadius: size * 0.28, style: .continuous)
            .stroke(Color.white.opacity(0.12), lineWidth: 1))
    }
}

// Brand colors + glyphs for major coins (falls back to a hashed hue + ticker).
enum CryptoBrand {
    static func style(for symbol: String) -> (color: Color, glyph: String?) {
        switch symbol.uppercased() {
        case "BTC":  return (Color(hex: 0xF7931A), "₿")
        case "ETH":  return (Color(hex: 0x627EEA), "Ξ")
        case "BNB":  return (Color(hex: 0xF3BA2F), nil)
        case "SOL":  return (Color(hex: 0x9945FF), nil)
        case "XRP":  return (Color(hex: 0x23292F), nil)
        case "ADA":  return (Color(hex: 0x0033AD), nil)
        case "DOGE": return (Color(hex: 0xC2A633), "Ð")
        case "TRX":  return (Color(hex: 0xEF0027), nil)
        case "AVAX": return (Color(hex: 0xE84142), nil)
        case "DOT":  return (Color(hex: 0xE6007A), nil)
        case "LINK": return (Color(hex: 0x2A5ADA), nil)
        case "MATIC": return (Color(hex: 0x8247E5), nil)
        case "LTC":  return (Color(hex: 0x345D9D), "Ł")
        case "ATOM": return (Color(hex: 0x2E3148), nil)
        case "XLM":  return (Color(hex: 0x08B5E5), nil)
        case "SHIB": return (Color(hex: 0xFFA409), nil)
        case "BCH":  return (Color(hex: 0x0AC18E), nil)
        default:     return (hashedColor(symbol), nil)
        }
    }

    private static func hashedColor(_ s: String) -> Color {
        let hue = Double(abs(s.hashValue) % 360) / 360
        return Color(hue: hue, saturation: 0.62, brightness: 0.78)
    }
}

// Deterministic accent colors for BIST equities.
enum StockColor {
    private static let palette: [UInt32] = [
        0x2F6FED, 0x0EA5A5, 0x7C5CFC, 0xE0632A, 0xDB2777, 0x059669, 0x9333EA, 0x0891B2,
    ]
    static func color(for symbol: String) -> Color {
        Color(hex: palette[abs(symbol.hashValue) % palette.count])
    }
}

// Flags maps a currency (or XXXTRY pair) to its country flag emoji. ISO-4217
// codes are built as <ISO-3166 country><currency initial>, so the first two
// letters of the base currency are (almost always) the country code, which we
// convert to regional-indicator symbols. A few non-geographic codes are mapped
// explicitly.
enum Flags {
    private static let overrides: [String: String] = [
        "EUR": "🇪🇺", "XAU": "🥇", "XAG": "🥈", "XDR": "🌐", "XOF": "🌍", "XAF": "🌍",
        "XCD": "🌎", "ANG": "🇳🇱", "TRY": "🇹🇷",
    ]

    /// `pair` is a code like "USDTRY" or a bare currency like "USD".
    static func emoji(forCurrencyPair pair: String) -> String? {
        let up = pair.uppercased()
        var base = up
        if up.count == 6 && up.hasSuffix("TRY") { base = String(up.prefix(3)) }
        if base.count > 3 { base = String(base.prefix(3)) }
        if let o = overrides[base] { return o }
        return regionalIndicators(String(base.prefix(2)))
    }

    private static func regionalIndicators(_ two: String) -> String? {
        let letters = Array(two.uppercased())
        guard letters.count == 2,
              letters.allSatisfy({ $0.isLetter && $0.isASCII }) else { return nil }
        var s = ""
        for ch in letters {
            guard let scalar = ch.unicodeScalars.first,
                  let flagScalar = Unicode.Scalar(0x1F1E6 + (scalar.value - 65)) else { return nil }
            s.unicodeScalars.append(flagScalar)
        }
        return s
    }
}

// MARK: - Connection status dot

struct ConnectionDot: View {
    let connected: Bool
    var body: some View {
        HStack(spacing: 5) {
            Circle().fill(connected ? Theme.up : Theme.down).frame(width: 7, height: 7)
            Text(connected ? "Canlı" : "Kopuk")
                .font(.system(size: 11, weight: .semibold))
                .foregroundStyle(connected ? Theme.up : Theme.down)
        }
        .padding(.horizontal, 8).padding(.vertical, 4)
        .background((connected ? Theme.up : Theme.down).opacity(0.12),
                    in: Capsule())
    }
}

// MARK: - Primary / secondary buttons

struct FilledButtonStyle: ButtonStyle {
    var color: Color = Theme.accent
    var textColor: Color = .black
    func makeBody(configuration: Configuration) -> some View {
        configuration.label
            .font(.system(size: 16, weight: .bold))
            .foregroundStyle(textColor)
            .frame(maxWidth: .infinity)
            .padding(.vertical, 15)
            .background(color.opacity(configuration.isPressed ? 0.8 : 1),
                        in: RoundedRectangle(cornerRadius: 12, style: .continuous))
    }
}
