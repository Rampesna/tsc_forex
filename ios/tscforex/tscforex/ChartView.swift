//
//  ChartView.swift
//  Swift Charts price chart: area/line and candlestick modes with a live
//  crosshair-free, Binance-style minimal aesthetic.
//

import SwiftUI
import Charts

enum ChartMode: String, CaseIterable { case line, candle }

struct PriceChart: View {
    let candles: [Candle]
    let mode: ChartMode
    let trendUp: Bool

    private var lineColor: Color { trendUp ? Theme.up : Theme.down }

    private var yDomain: ClosedRange<Double> {
        let lows = candles.map(\.low)
        let highs = candles.map(\.high)
        guard let lo = lows.min(), let hi = highs.max(), hi > lo else {
            let v = candles.last?.close ?? 1
            return (v * 0.99)...(v * 1.01)
        }
        let pad = (hi - lo) * 0.08
        return (lo - pad)...(hi + pad)
    }

    var body: some View {
        if candles.count < 2 {
            emptyState
        } else {
            chart
        }
    }

    private var emptyState: some View {
        VStack(spacing: 8) {
            Image(systemName: "chart.xyaxis.line")
                .font(.system(size: 28)).foregroundStyle(Theme.textFaint)
            Text("Grafik verisi birikiyor…")
                .font(.system(size: 13)).foregroundStyle(Theme.textDim)
            Text("Backend her fiyatı kaydediyor; birkaç dakika içinde mum çubukları oluşacak.")
                .font(.system(size: 11)).foregroundStyle(Theme.textFaint)
                .multilineTextAlignment(.center)
        }
        .frame(maxWidth: .infinity, minHeight: 220)
    }

    @ViewBuilder
    private var chart: some View {
        Chart {
            switch mode {
            case .line:
                ForEach(candles) { c in
                    LineMark(x: .value("t", c.bucket), y: .value("p", c.close))
                        .interpolationMethod(.monotone)
                        .foregroundStyle(lineColor)
                        .lineStyle(StrokeStyle(lineWidth: 2))
                    AreaMark(x: .value("t", c.bucket), y: .value("p", c.close))
                        .interpolationMethod(.monotone)
                        .foregroundStyle(
                            LinearGradient(colors: [lineColor.opacity(0.25), .clear],
                                           startPoint: .top, endPoint: .bottom))
                }
            case .candle:
                ForEach(candles) { c in
                    let up = c.close >= c.open
                    let col = up ? Theme.up : Theme.down
                    RuleMark(x: .value("t", c.bucket),
                             yStart: .value("l", c.low),
                             yEnd: .value("h", c.high))
                        .foregroundStyle(col)
                        .lineStyle(StrokeStyle(lineWidth: 1))
                    RectangleMark(x: .value("t", c.bucket),
                                  yStart: .value("o", min(c.open, c.close)),
                                  yEnd: .value("c", max(c.open, c.close)),
                                  width: 5)
                        .foregroundStyle(col)
                }
            }
        }
        .chartYScale(domain: yDomain)
        .chartYAxis {
            AxisMarks(position: .trailing, values: .automatic(desiredCount: 4)) { value in
                AxisGridLine().foregroundStyle(Theme.stroke)
                AxisValueLabel {
                    if let v = value.as(Double.self) {
                        Text(Fmt.compactPrice(v))
                            .font(.system(size: 10)).foregroundStyle(Theme.textDim)
                    }
                }
            }
        }
        .chartXAxis {
            AxisMarks(values: .automatic(desiredCount: 4)) { value in
                AxisValueLabel(format: .dateTime.hour().minute())
                    .font(.system(size: 10))
                    .foregroundStyle(Theme.textDim)
            }
        }
        .frame(minHeight: 240)
    }
}
