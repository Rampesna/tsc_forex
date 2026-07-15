//
//  Theme.swift
//  Central design system — Binance-grade dark palette + Linear-style polish.
//  Every screen pulls colors, typography and spacing from here so the app
//  stays visually consistent as it scales.
//

import SwiftUI

enum Theme {
    // MARK: Palette
    static let bg           = Color(hex: 0x0B0E11) // app background (near-black)
    static let bgElevated   = Color(hex: 0x12161C) // grouped section background
    static let surface      = Color(hex: 0x181C22) // cards / rows
    static let surfaceHi     = Color(hex: 0x21262E) // pressed / highlighted
    static let stroke       = Color.white.opacity(0.06)
    static let strokeStrong = Color.white.opacity(0.12)

    static let text         = Color(hex: 0xEAECEF)
    static let textDim      = Color(hex: 0x8A94A6)
    static let textFaint    = Color(hex: 0x5E6673)

    static let up           = Color(hex: 0x0ECB81) // gain green
    static let down         = Color(hex: 0xF6465D) // loss red
    // Brand palette derived from the logo (deep teal #003D4D). The deep tone is
    // illegible on the near-black background, so interactive elements use a
    // brighter, readable teal while `brandDeep` is reserved for tiles/gradients.
    static let brandDeep    = Color(hex: 0x003D4D) // logo teal (backgrounds/tiles)
    static let accent       = Color(hex: 0x17A2B8) // readable brand teal
    static let accentSoft   = Color(hex: 0x17A2B8).opacity(0.16)

    static func trend(_ v: Double) -> Color { v >= 0 ? up : down }

    // MARK: Typography
    static func number(_ size: CGFloat, weight: Font.Weight = .semibold) -> Font {
        .system(size: size, weight: weight, design: .rounded).monospacedDigit()
    }

    // MARK: Radii / spacing
    static let radius: CGFloat = 16
    static let radiusSm: CGFloat = 10
}

extension Color {
    init(hex: UInt32, alpha: Double = 1) {
        self.init(
            .sRGB,
            red: Double((hex >> 16) & 0xFF) / 255,
            green: Double((hex >> 8) & 0xFF) / 255,
            blue: Double(hex & 0xFF) / 255,
            opacity: alpha
        )
    }
}

// MARK: - Reusable card container

struct Card<Content: View>: View {
    var padding: CGFloat = 16
    @ViewBuilder var content: Content

    var body: some View {
        content
            .padding(padding)
            .background(Theme.surface, in: RoundedRectangle(cornerRadius: Theme.radius, style: .continuous))
            .overlay(
                RoundedRectangle(cornerRadius: Theme.radius, style: .continuous)
                    .stroke(Theme.stroke, lineWidth: 1)
            )
    }
}

// MARK: - Number / price formatting helpers

enum Fmt {
    static func price(_ v: Double, currency: Bool = true) -> String {
        let digits = v >= 1000 ? 2 : (v >= 1 ? 4 : 6)
        let s = grouped(v, digits: v >= 1 ? 2 : digits)
        return currency ? "₺" + s : s
    }

    static func compactPrice(_ v: Double) -> String {
        grouped(v, digits: v >= 1 ? 2 : 4)
    }

    // Turkish Lira formatted value.
    static func tl(_ v: Double) -> String {
        "₺" + compactPrice(v)
    }

    // Native-currency value: "$" for USD-quoted instruments (global equities), "₺" otherwise.
    static func money(_ v: Double, currency: String?) -> String {
        (currency ?? "TRY") == "USD" ? usd(v) : tl(v)
    }

    static func signedPct(_ v: Double) -> String {
        String(format: "%+.2f%%", v)
    }

    static func signed(_ v: Double) -> String {
        String(format: "%+.2f", v)
    }

    // Compact large-number formatting with Turkish scale suffixes
    // (B = bin, Mn = milyon, Mr = milyar, Tn = trilyon).
    static func compactNum(_ v: Double) -> String {
        let a = abs(v)
        switch a {
        case 1e12...: return grouped(v / 1e12, digits: 2) + " Tn"
        case 1e9...:  return grouped(v / 1e9, digits: 2) + " Mr"
        case 1e6...:  return grouped(v / 1e6, digits: 2) + " Mn"
        case 1e3...:  return grouped(v / 1e3, digits: 2) + " B"
        default:      return grouped(v, digits: a >= 1 ? 2 : 4)
        }
    }

    // USD-quoted value ($). `compact` uses the scale suffixes for big figures.
    static func usd(_ v: Double, compact: Bool = false) -> String {
        if compact { return "$" + compactNum(v) }
        let digits = v >= 1 ? 2 : 6
        return "$" + grouped(v, digits: digits)
    }

    // TRY value with compact scale suffix (for large amounts like ons/market cap).
    static func tlCompact(_ v: Double) -> String { "₺" + compactNum(v) }

    // Unsigned percentage, Turkish decimal ("58,78%").
    static func pct(_ v: Double) -> String { grouped(v, digits: 2) + "%" }

    // ISO date (2025-10-06T..) → "06.10.2025".
    static func dateShort(_ s: String?) -> String? {
        guard let s, s.count >= 10 else { return nil }
        let parts = String(s.prefix(10)).split(separator: "-")
        guard parts.count == 3 else { return String(s.prefix(10)) }
        return "\(parts[2]).\(parts[1]).\(parts[0])"
    }

    // Trims trailing zeros (e.g. gram weights: 1.754 → "1,754", 1.0 → "1").
    static func trim(_ v: Double) -> String {
        let f = NumberFormatter()
        f.numberStyle = .decimal
        f.minimumFractionDigits = 0
        f.maximumFractionDigits = 4
        f.groupingSeparator = "."
        f.decimalSeparator = ","
        return f.string(from: NSNumber(value: v)) ?? String(v)
    }

    private static func grouped(_ v: Double, digits: Int) -> String {
        let f = NumberFormatter()
        f.numberStyle = .decimal
        f.minimumFractionDigits = digits
        f.maximumFractionDigits = digits
        f.groupingSeparator = "."
        f.decimalSeparator = ","
        return f.string(from: NSNumber(value: v)) ?? String(format: "%.\(digits)f", v)
    }
}
