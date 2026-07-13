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

    // Turkish Lira formatted value (all instrument prices are quoted in TRY).
    static func tl(_ v: Double) -> String {
        "₺" + compactPrice(v)
    }

    static func signedPct(_ v: Double) -> String {
        String(format: "%+.2f%%", v)
    }

    static func signed(_ v: Double) -> String {
        String(format: "%+.2f", v)
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
