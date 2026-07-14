//
//  ChartView.swift
//  Binance-grade interactive price chart built on Swift Charts:
//    • drag to pan (native horizontal scroll)
//    • pinch to zoom (adjusts the visible time window)
//    • tap to drop a crosshair with an OHLC tooltip + trailing price bubble
//    • line/area and candlestick modes with an MA(7) overlay and a tick-activity
//      histogram underneath (like Binance's volume pane).
//

import SwiftUI
import Charts

enum ChartMode: String, CaseIterable { case line, candle }

struct PriceChart: View {
    let candles: [Candle]
    let mode: ChartMode
    let trendUp: Bool

    var body: some View {
        if candles.count < 2 {
            emptyState
        } else {
            InteractiveChart(candles: candles, mode: mode, trendUp: trendUp)
        }
    }

    private var emptyState: some View {
        VStack(spacing: 8) {
            Image(systemName: "chart.xyaxis.line")
                .font(.system(size: 28)).foregroundStyle(Theme.textFaint)
            Text("Grafik verisi birikiyor…")
                .font(.system(size: 13)).foregroundStyle(Theme.textDim)
            Text("Backend her fiyatı kaydediyor; kısa süre içinde mum çubukları oluşacak.")
                .font(.system(size: 11)).foregroundStyle(Theme.textFaint)
                .multilineTextAlignment(.center)
        }
        .frame(maxWidth: .infinity, minHeight: 240)
    }
}

// MARK: - Interactive chart

private struct InteractiveChart: View {
    let candles: [Candle]
    let mode: ChartMode
    let trendUp: Bool

    // Visible window (seconds). Smaller = more zoomed in.
    @State private var visibleLen: TimeInterval = 0
    @State private var zoomAnchor: TimeInterval?
    @State private var scrollX: Date = .now
    @State private var selectedDate: Date?

    private var lineColor: Color { trendUp ? Theme.up : Theme.down }
    private var step: TimeInterval { candleStep(candles) }
    private var fullSpan: TimeInterval {
        guard let first = candles.first?.bucket, let last = candles.last?.bucket else { return 1 }
        return max(last.timeIntervalSince(first), step)
    }
    private var minLen: TimeInterval { step * 8 }            // most-zoomed: ~8 candles
    private var maxLen: TimeInterval { fullSpan + step }     // whole series
    private var maSeries: [(date: Date, value: Double)] { movingAverage(candles, period: 7) }

    private var selected: Candle? {
        guard let d = selectedDate else { return nil }
        return candles.min(by: { abs($0.bucket.timeIntervalSince(d)) < abs($1.bucket.timeIntervalSince(d)) })
    }

    private var yDomain: ClosedRange<Double> {
        let lows = candles.map(\.low), highs = candles.map(\.high)
        guard let lo = lows.min(), let hi = highs.max(), hi > lo else {
            let v = candles.last?.close ?? 1
            return (v * 0.99)...(v * 1.01)
        }
        let pad = (hi - lo) * 0.10
        return (lo - pad)...(hi + pad)
    }

    var body: some View {
        VStack(spacing: 8) {
            header
            GeometryReader { geo in
                priceChart(plotWidth: geo.size.width)
            }
            .frame(height: 240)
            activityChart
                .frame(height: 46)
            hint
        }
        .onAppear(perform: initWindow)
        .onChange(of: candles.count) { _, _ in initWindow() }
        .onChange(of: mode) { _, _ in selectedDate = nil }
    }

    // MARK: Header (legend + crosshair readout)

    private var header: some View {
        HStack(alignment: .top) {
            if let c = selected {
                crosshairReadout(c)
            } else if let last = candles.last {
                HStack(spacing: 10) {
                    legendDot(lineColor, "Fiyat")
                    legendDot(Theme.accent.opacity(0.9), "MA7")
                    Spacer(minLength: 0)
                    Text(Fmt.tl(last.close))
                        .font(Theme.number(13, weight: .bold)).foregroundStyle(Theme.text)
                }
            }
        }
        .frame(height: 30)
        .animation(.easeOut(duration: 0.12), value: selectedDate)
    }

    private func legendDot(_ color: Color, _ label: String) -> some View {
        HStack(spacing: 4) {
            Circle().fill(color).frame(width: 6, height: 6)
            Text(label).font(.system(size: 10, weight: .medium)).foregroundStyle(Theme.textDim)
        }
    }

    private func crosshairReadout(_ c: Candle) -> some View {
        let chg = c.close - c.open
        let chgPct = c.open != 0 ? chg / c.open * 100 : 0
        return HStack(spacing: 12) {
            VStack(alignment: .leading, spacing: 1) {
                Text(c.bucket.formatted(.dateTime.month(.twoDigits).day().hour().minute()))
                    .font(.system(size: 10)).foregroundStyle(Theme.textDim)
                HStack(spacing: 8) {
                    ohlc("A", c.open); ohlc("Y", c.high); ohlc("D", c.low); ohlc("K", c.close)
                }
            }
            Spacer(minLength: 0)
            Text(Fmt.signedPct(chgPct))
                .font(Theme.number(12, weight: .bold)).foregroundStyle(Theme.trend(chgPct))
        }
    }

    private func ohlc(_ k: String, _ v: Double) -> some View {
        HStack(spacing: 2) {
            Text(k).font(.system(size: 9)).foregroundStyle(Theme.textFaint)
            Text(Fmt.compactPrice(v)).font(Theme.number(10, weight: .semibold)).foregroundStyle(Theme.text)
        }
    }

    // MARK: Price chart

    @ViewBuilder
    private func priceChart(plotWidth: CGFloat) -> some View {
        let visibleCount = max(visibleLen / step, 1)
        let barW = max(2.0, min(16.0, (plotWidth / visibleCount) * 0.62))
        Chart {
            switch mode {
            case .line:
                ForEach(candles) { c in
                    AreaMark(x: .value("Zaman", c.bucket), y: .value("Fiyat", c.close))
                        .interpolationMethod(.monotone)
                        .foregroundStyle(LinearGradient(colors: [lineColor.opacity(0.22), .clear],
                                                        startPoint: .top, endPoint: .bottom))
                    LineMark(x: .value("Zaman", c.bucket), y: .value("Fiyat", c.close))
                        .interpolationMethod(.monotone)
                        .foregroundStyle(lineColor)
                        .lineStyle(StrokeStyle(lineWidth: 2))
                }
            case .candle:
                ForEach(candles) { c in
                    let up = c.close >= c.open
                    let col = up ? Theme.up : Theme.down
                    RuleMark(x: .value("Zaman", c.bucket),
                             yStart: .value("Düşük", c.low), yEnd: .value("Yüksek", c.high))
                        .foregroundStyle(col).lineStyle(StrokeStyle(lineWidth: 1))
                    RectangleMark(x: .value("Zaman", c.bucket),
                                  yStart: .value("Açılış", min(c.open, c.close)),
                                  yEnd: .value("Kapanış", max(c.open, c.close)),
                                  width: .fixed(barW))
                        .foregroundStyle(col)
                }
            }

            // MA(7) overlay
            ForEach(maSeries, id: \.date) { p in
                LineMark(x: .value("Zaman", p.date), y: .value("MA7", p.value), series: .value("s", "ma"))
                    .interpolationMethod(.monotone)
                    .foregroundStyle(Theme.accent.opacity(0.85))
                    .lineStyle(StrokeStyle(lineWidth: 1.2))
            }

            // Last price dashed line + trailing bubble
            if let last = candles.last {
                RuleMark(y: .value("Son", last.close))
                    .foregroundStyle(lineColor.opacity(0.35))
                    .lineStyle(StrokeStyle(lineWidth: 1, dash: [2, 3]))
            }

            // Crosshair
            if let c = selected {
                RuleMark(x: .value("Seçili", c.bucket))
                    .foregroundStyle(Theme.textDim.opacity(0.55))
                    .lineStyle(StrokeStyle(lineWidth: 1, dash: [3, 3]))
                PointMark(x: .value("Seçili", c.bucket), y: .value("Fiyat", c.close))
                    .symbolSize(70)
                    .foregroundStyle(Theme.text)
            }
        }
        .chartYScale(domain: yDomain)
        .chartYAxis {
            AxisMarks(position: .trailing, values: .automatic(desiredCount: 5)) { value in
                AxisGridLine().foregroundStyle(Theme.stroke)
                AxisValueLabel {
                    if let v = value.as(Double.self) {
                        Text(Fmt.compactPrice(v)).font(.system(size: 9)).foregroundStyle(Theme.textDim)
                    }
                }
            }
        }
        .chartXAxis {
            AxisMarks(values: .automatic(desiredCount: 4)) { _ in
                AxisGridLine().foregroundStyle(Theme.stroke.opacity(0.5))
                AxisValueLabel(format: .dateTime.hour().minute()).font(.system(size: 9))
                    .foregroundStyle(Theme.textDim)
            }
        }
        .chartScrollableAxes(.horizontal)
        .chartXVisibleDomain(length: visibleLen > 0 ? visibleLen : maxLen)
        .chartScrollPosition(x: $scrollX)
        .chartXSelection(value: $selectedDate)
        .chartOverlay { proxy in
            // Trailing price bubble anchored to the last close.
            GeometryReader { geo in
                if let last = candles.last, let plot = proxy.plotFrame,
                   let y = proxy.position(forY: last.close) {
                    let rect = geo[plot]
                    Text(Fmt.compactPrice(last.close))
                        .font(Theme.number(10, weight: .bold)).foregroundStyle(.white)
                        .padding(.horizontal, 5).padding(.vertical, 2)
                        .background(lineColor, in: RoundedRectangle(cornerRadius: 4))
                        .position(x: rect.maxX - 22, y: rect.minY + y)
                }
            }
        }
        .simultaneousGesture(zoomGesture)
        .onChange(of: scrollX) { _, _ in if selectedDate != nil { selectedDate = nil } }
    }

    // MARK: Activity (tick count) pane — Binance-style volume analogue

    private var activityChart: some View {
        Chart(candles) { c in
            BarMark(x: .value("Zaman", c.bucket),
                    y: .value("İşlem", c.count),
                    width: .fixed(3))
                .foregroundStyle((c.close >= c.open ? Theme.up : Theme.down).opacity(0.55))
        }
        .chartYAxis(.hidden)
        .chartXAxis(.hidden)
        .chartScrollableAxes(.horizontal)
        .chartXVisibleDomain(length: visibleLen > 0 ? visibleLen : maxLen)
        .chartScrollPosition(x: $scrollX)
        .chartPlotStyle { $0.background(Theme.bgElevated.opacity(0.4)) }
    }

    private var hint: some View {
        Text("Kaydır • İki parmakla yakınlaştır • Dokun: detay")
            .font(.system(size: 9)).foregroundStyle(Theme.textFaint)
            .frame(maxWidth: .infinity)
    }

    // MARK: Gestures / window

    private var zoomGesture: some Gesture {
        MagnifyGesture()
            .onChanged { value in
                let anchor = zoomAnchor ?? visibleLen
                if zoomAnchor == nil { zoomAnchor = anchor }
                let proposed = anchor / value.magnification
                visibleLen = min(max(proposed, minLen), maxLen)
            }
            .onEnded { _ in zoomAnchor = nil }
    }

    private func initWindow() {
        // Start showing ~40 most-recent candles (or the whole set if shorter).
        let target = min(step * 40, maxLen)
        visibleLen = max(target, minLen)
        if let last = candles.last?.bucket {
            scrollX = last.addingTimeInterval(-visibleLen + step)
        }
        selectedDate = nil
    }
}

// MARK: - Helpers

private func candleStep(_ candles: [Candle]) -> TimeInterval {
    guard candles.count >= 2 else { return 60 }
    var minGap = TimeInterval.greatestFiniteMagnitude
    for i in 1..<candles.count {
        let g = candles[i].bucket.timeIntervalSince(candles[i - 1].bucket)
        if g > 0 { minGap = min(minGap, g) }
    }
    return minGap.isFinite ? minGap : 60
}

private func movingAverage(_ candles: [Candle], period: Int) -> [(date: Date, value: Double)] {
    guard candles.count >= period else { return [] }
    var out: [(Date, Double)] = []
    out.reserveCapacity(candles.count - period + 1)
    var window = 0.0
    for i in 0..<candles.count {
        window += candles[i].close
        if i >= period { window -= candles[i - period].close }
        if i >= period - 1 { out.append((candles[i].bucket, window / Double(period))) }
    }
    return out.map { (date: $0.0, value: $0.1) }
}
