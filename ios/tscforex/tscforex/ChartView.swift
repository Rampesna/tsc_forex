//
//  ChartView.swift
//  High-performance trading chart.
//
//  Swift Charts creates one SwiftUI view per mark and does not virtualize
//  horizontally-scrolled data. A few hundred candles plus two moving averages
//  and volume bars therefore means thousands of views rebuilt on every tick.
//
//  This renderer keeps the paged candle cache separate from what is drawn:
//    • only the visible 8...160 bars (+2 overscan) are rendered
//    • Core Graphics draws the whole frame in one UIView
//    • paging preserves the time/index anchor
//    • live-price redraws are capped at 15 FPS
//    • pan has inertial deceleration; pinch zooms around the gesture focal point
//

import SwiftUI
import UIKit

enum ChartMode: String, CaseIterable { case line, candle }

struct PriceChart: View {
    let candles: [Candle]
    let mode: ChartMode
    let trendUp: Bool
    var livePrice: Double?
    var onReachStart: () -> Void = {}
    var onReachEnd: () -> Void = {}
    var canLoadOlder = false
    var canLoadNewer = false
    var resetToken = 0
    var currency: String?
    var height: CGFloat = 342
    var seriesTitle = "Fiyat"
    var showsMovingAverages = true
    var showsVolume = true
    var showsOHLCSelection = true
    var emptyTitle = "Grafik verisi birikiyor…"
    var emptySubtitle = "Kısa süre içinde mum çubukları oluşacak."
    var onSelectionChange: (Date?) -> Void = { _ in }

    var body: some View {
        if candles.count < 2 {
            emptyState
        } else {
            TradingChartRepresentable(
                candles: candles,
                mode: mode,
                trendUp: trendUp,
                livePrice: livePrice,
                onReachStart: onReachStart,
                onReachEnd: onReachEnd,
                canLoadOlder: canLoadOlder,
                canLoadNewer: canLoadNewer,
                resetToken: resetToken,
                currency: currency,
                seriesTitle: seriesTitle,
                showsMovingAverages: showsMovingAverages,
                showsVolume: showsVolume,
                showsOHLCSelection: showsOHLCSelection,
                onSelectionChange: onSelectionChange
            )
            .frame(height: height)
        }
    }

    private var emptyState: some View {
        VStack(spacing: 8) {
            Image(systemName: "chart.xyaxis.line")
                .font(.system(size: 28))
                .foregroundStyle(Theme.textFaint)
            Text(L10n.text(emptyTitle))
                .font(.system(size: 13))
                .foregroundStyle(Theme.textDim)
            Text(L10n.text(emptySubtitle))
                .font(.system(size: 11))
                .foregroundStyle(Theme.textFaint)
        }
        .frame(maxWidth: .infinity, minHeight: height)
    }
}

// MARK: - SwiftUI / UIKit bridge

private struct TradingChartRepresentable: UIViewRepresentable {
    let candles: [Candle]
    let mode: ChartMode
    let trendUp: Bool
    let livePrice: Double?
    let onReachStart: () -> Void
    let onReachEnd: () -> Void
    let canLoadOlder: Bool
    let canLoadNewer: Bool
    let resetToken: Int
    let currency: String?
    let seriesTitle: String
    let showsMovingAverages: Bool
    let showsVolume: Bool
    let showsOHLCSelection: Bool
    let onSelectionChange: (Date?) -> Void

    func makeUIView(context: Context) -> TradingChartUIView {
        TradingChartUIView()
    }

    func updateUIView(_ view: TradingChartUIView, context: Context) {
        view.apply(
            candles: candles,
            mode: mode,
            trendUp: trendUp,
            livePrice: livePrice,
            canLoadOlder: canLoadOlder,
            canLoadNewer: canLoadNewer,
            resetToken: resetToken,
            currency: currency,
            seriesTitle: seriesTitle,
            showsMovingAverages: showsMovingAverages,
            showsVolume: showsVolume,
            showsOHLCSelection: showsOHLCSelection,
            onSelectionChange: onSelectionChange,
            onReachStart: onReachStart,
            onReachEnd: onReachEnd
        )
    }

    static func dismantleUIView(_ uiView: TradingChartUIView, coordinator: ()) {
        uiView.stopAnimations()
    }
}

// MARK: - Immediate-mode chart renderer

private final class TradingChartUIView: UIView, UIGestureRecognizerDelegate {
    private var candles: [Candle] = []
    private var closePrefix: [Double] = [0]

    private var mode: ChartMode = .line
    private var trendUp = true
    private var livePrice: Double?
    private var currency: String?
    private var seriesTitle = "Fiyat"
    private var showsMovingAverages = true
    private var showsVolume = true
    private var showsOHLCSelection = true
    private var onSelectionChange: (Date?) -> Void = { _ in }
    private var canLoadOlder = false
    private var canLoadNewer = false
    private var onReachStart: () -> Void = {}
    private var onReachEnd: () -> Void = {}
    private var resetToken: Int?

    // Index-based viewport. The cache can contain hundreds of bars, but the
    // renderer never displays more than maxVisibleBars.
    private var viewportStart = 0.0
    private var visibleBars = 48.0
    private let minVisibleBars = 8.0
    private let maxVisibleBars = 160.0

    private var selectedIndex: Int?
    private var requestedOldest: Date?
    private var requestedNewest: Date?

    private var panLastX: CGFloat = 0
    private var pinchStartBars = 48.0
    private var pinchAnchorIndex = 0.0
    private var pinchAnchorFraction = 0.5

    private var displayLink: CADisplayLink?
    private var momentumBarsPerSecond = 0.0
    private var momentumLastTime: CFTimeInterval = 0

    private var pendingLiveDisplay: DispatchWorkItem?
    private var lastLiveDisplay = CACurrentMediaTime()

    private let headerHeight: CGFloat = 34
    private let volumePaneHeight: CGFloat = 46
    private let timeAxisHeight: CGFloat = 22
    private let rightAxisWidth: CGFloat = 58

    private let ma7Color = UIColor(red: 0.95, green: 0.77, blue: 0.24, alpha: 1)
    private let ma25Color = UIColor(red: 0.90, green: 0.35, blue: 0.72, alpha: 1)

    override init(frame: CGRect) {
        super.init(frame: frame)
        isOpaque = false
        backgroundColor = .clear
        contentMode = .redraw
        isAccessibilityElement = true
        accessibilityLabel = "Etkileşimli fiyat grafiği"
        installGestures()
    }

    required init?(coder: NSCoder) {
        fatalError("init(coder:) has not been implemented")
    }

    deinit {
        stopAnimations()
    }

    override func layoutSubviews() {
        super.layoutSubviews()
        clampViewport()
    }

    func apply(
        candles newCandles: [Candle],
        mode newMode: ChartMode,
        trendUp newTrendUp: Bool,
        livePrice newLivePrice: Double?,
        canLoadOlder newCanLoadOlder: Bool,
        canLoadNewer newCanLoadNewer: Bool,
        resetToken newResetToken: Int,
        currency newCurrency: String?,
        seriesTitle newSeriesTitle: String,
        showsMovingAverages newShowsMovingAverages: Bool,
        showsVolume newShowsVolume: Bool,
        showsOHLCSelection newShowsOHLCSelection: Bool,
        onSelectionChange newOnSelectionChange: @escaping (Date?) -> Void,
        onReachStart newOnReachStart: @escaping () -> Void,
        onReachEnd newOnReachEnd: @escaping () -> Void
    ) {
        onReachStart = newOnReachStart
        onReachEnd = newOnReachEnd
        canLoadOlder = newCanLoadOlder
        canLoadNewer = newCanLoadNewer
        onSelectionChange = newOnSelectionChange

        let needsReset = resetToken != newResetToken
        let dataChanged = candleDataChanged(newCandles)
        let styleChanged = mode != newMode || trendUp != newTrendUp || currency != newCurrency
            || seriesTitle != newSeriesTitle
            || showsMovingAverages != newShowsMovingAverages
            || showsVolume != newShowsVolume
            || showsOHLCSelection != newShowsOHLCSelection
        let liveChanged = livePrice != newLivePrice

        mode = newMode
        trendUp = newTrendUp
        currency = newCurrency
        seriesTitle = newSeriesTitle
        showsMovingAverages = newShowsMovingAverages
        showsVolume = newShowsVolume
        showsOHLCSelection = newShowsOHLCSelection
        resetToken = newResetToken
        accessibilityLabel = "Etkileşimli \(newSeriesTitle.lowercased()) grafiği"

        if dataChanged {
            replaceCandles(newCandles, reset: needsReset)
        } else if needsReset {
            resetToLatest()
        }

        livePrice = newLivePrice

        if dataChanged || needsReset || styleChanged {
            setNeedsDisplay()
        } else if liveChanged {
            scheduleLiveDisplay()
        }
    }

    func stopAnimations() {
        displayLink?.invalidate()
        displayLink = nil
        pendingLiveDisplay?.cancel()
        pendingLiveDisplay = nil
    }

    // MARK: Data / viewport anchoring

    private func candleDataChanged(_ next: [Candle]) -> Bool {
        guard candles.count == next.count,
              candles.first?.bucket == next.first?.bucket,
              candles.last?.bucket == next.last?.bucket else {
            return true
        }
        return candles.last?.close != next.last?.close
            || candles.last?.high != next.last?.high
            || candles.last?.low != next.last?.low
    }

    private func replaceCandles(_ next: [Candle], reset: Bool) {
        let oldFloor = Int(floor(viewportStart))
        let oldFraction = viewportStart - floor(viewportStart)
        let anchorDate = candles.indices.contains(oldFloor) ? candles[oldFloor].bucket : nil

        candles = next
        rebuildPrefix()

        if reset || candles.isEmpty {
            resetToLatest()
            requestedOldest = nil
            requestedNewest = nil
            return
        }

        // Older/newer pages may be prepended/appended while the opposite edge is
        // trimmed. Preserve the candle under the viewport's leading edge.
        if let anchorDate,
           let index = candles.firstIndex(where: { $0.bucket == anchorDate }) {
            viewportStart = Double(index) + oldFraction
        }
        clampViewport()
    }

    private func rebuildPrefix() {
        closePrefix = Array(repeating: 0, count: candles.count + 1)
        for index in candles.indices {
            closePrefix[index + 1] = closePrefix[index] + candles[index].close
        }
    }

    private func resetToLatest() {
        selectedIndex = nil
        visibleBars = min(48, max(minVisibleBars, Double(candles.count)))
        viewportStart = max(0, Double(candles.count) - visibleBars)
        clampViewport()
        setNeedsDisplay()
    }

    private func clampViewport() {
        guard !candles.isEmpty else {
            viewportStart = 0
            return
        }
        visibleBars = min(max(visibleBars, minVisibleBars),
                          min(maxVisibleBars, Double(candles.count)))
        viewportStart = min(max(viewportStart, 0),
                            max(0, Double(candles.count) - visibleBars))
    }

    private var visibleIndexRange: ClosedRange<Int>? {
        guard !candles.isEmpty else { return nil }
        let lower = max(0, Int(floor(viewportStart)) - 2)
        let upper = min(candles.count - 1, Int(ceil(viewportStart + visibleBars)) + 2)
        return lower...max(lower, upper)
    }

    // MARK: Gestures

    private func installGestures() {
        let pan = UIPanGestureRecognizer(target: self, action: #selector(handlePan(_:)))
        pan.maximumNumberOfTouches = 1
        pan.delegate = self
        addGestureRecognizer(pan)

        let pinch = UIPinchGestureRecognizer(target: self, action: #selector(handlePinch(_:)))
        pinch.delegate = self
        addGestureRecognizer(pinch)

        let longPress = UILongPressGestureRecognizer(target: self, action: #selector(handleLongPress(_:)))
        longPress.minimumPressDuration = 0.22
        longPress.allowableMovement = 16
        longPress.delegate = self
        addGestureRecognizer(longPress)

        let tap = UITapGestureRecognizer(target: self, action: #selector(handleTap(_:)))
        tap.delegate = self
        addGestureRecognizer(tap)

        let doubleTap = UITapGestureRecognizer(target: self, action: #selector(handleDoubleTap(_:)))
        doubleTap.numberOfTapsRequired = 2
        tap.require(toFail: doubleTap)
        addGestureRecognizer(doubleTap)
    }

    func gestureRecognizer(
        _ gestureRecognizer: UIGestureRecognizer,
        shouldRecognizeSimultaneouslyWith otherGestureRecognizer: UIGestureRecognizer
    ) -> Bool {
        gestureRecognizer is UIPinchGestureRecognizer || otherGestureRecognizer is UIPinchGestureRecognizer
    }

    override func gestureRecognizerShouldBegin(_ gestureRecognizer: UIGestureRecognizer) -> Bool {
        // Let the parent vertical ScrollView keep vertical drags that begin over
        // the chart; claim only clearly horizontal chart pans.
        guard let pan = gestureRecognizer as? UIPanGestureRecognizer else { return true }
        let velocity = pan.velocity(in: self)
        return abs(velocity.x) > abs(velocity.y)
    }

    @objc private func handlePan(_ recognizer: UIPanGestureRecognizer) {
        guard !candles.isEmpty else { return }
        switch recognizer.state {
        case .began:
            stopMomentum()
            clearSelection()
            panLastX = recognizer.translation(in: self).x
        case .changed:
            let currentX = recognizer.translation(in: self).x
            let deltaX = currentX - panLastX
            panLastX = currentX
            let width = max(plotRect.width, 1)
            viewportStart -= Double(deltaX / width) * visibleBars
            clampViewport()
            requestPageIfNeeded()
            setNeedsDisplay()
        case .ended:
            let velocityX = recognizer.velocity(in: self).x
            let barsPerSecond = -Double(velocityX / max(plotRect.width, 1)) * visibleBars
            startMomentum(barsPerSecond: barsPerSecond)
        default:
            break
        }
    }

    @objc private func handlePinch(_ recognizer: UIPinchGestureRecognizer) {
        guard !candles.isEmpty else { return }
        let location = recognizer.location(in: self)
        let fraction = min(max((location.x - plotRect.minX) / max(plotRect.width, 1), 0), 1)

        switch recognizer.state {
        case .began:
            stopMomentum()
            clearSelection()
            pinchStartBars = visibleBars
            pinchAnchorFraction = Double(fraction)
            pinchAnchorIndex = viewportStart + pinchAnchorFraction * visibleBars
        case .changed:
            visibleBars = min(max(pinchStartBars / Double(recognizer.scale), minVisibleBars),
                              min(maxVisibleBars, Double(candles.count)))
            viewportStart = pinchAnchorIndex - pinchAnchorFraction * visibleBars
            clampViewport()
            requestPageIfNeeded()
            setNeedsDisplay()
        default:
            break
        }
    }

    @objc private func handleLongPress(_ recognizer: UILongPressGestureRecognizer) {
        guard recognizer.state == .began || recognizer.state == .changed else { return }
        stopMomentum()
        updateSelection(index(atX: recognizer.location(in: self).x))
        setNeedsDisplay()
    }

    @objc private func handleTap(_ recognizer: UITapGestureRecognizer) {
        stopMomentum()
        if selectedIndex == nil {
            updateSelection(index(atX: recognizer.location(in: self).x))
        } else {
            clearSelection()
        }
        setNeedsDisplay()
    }

    @objc private func handleDoubleTap(_ recognizer: UITapGestureRecognizer) {
        stopMomentum()
        clearSelection()
        resetToLatest()
    }

    private func updateSelection(_ index: Int?) {
        guard selectedIndex != index else { return }
        selectedIndex = index
        onSelectionChange(index.flatMap { candles.indices.contains($0) ? candles[$0].bucket : nil })
    }

    private func clearSelection() {
        updateSelection(nil)
    }

    private func index(atX x: CGFloat) -> Int? {
        guard !candles.isEmpty else { return nil }
        let fraction = min(max((x - plotRect.minX) / max(plotRect.width, 1), 0), 1)
        let raw = viewportStart + Double(fraction) * visibleBars
        return min(max(Int(raw.rounded()), 0), candles.count - 1)
    }

    // MARK: Inertial pan

    private func startMomentum(barsPerSecond: Double) {
        guard abs(barsPerSecond) > 0.4 else {
            requestPageIfNeeded()
            return
        }
        momentumBarsPerSecond = barsPerSecond
        momentumLastTime = 0
        displayLink?.invalidate()
        let link = CADisplayLink(target: self, selector: #selector(momentumTick(_:)))
        link.add(to: .main, forMode: .common)
        displayLink = link
    }

    private func stopMomentum() {
        displayLink?.invalidate()
        displayLink = nil
        momentumBarsPerSecond = 0
        momentumLastTime = 0
    }

    @objc private func momentumTick(_ link: CADisplayLink) {
        if momentumLastTime == 0 {
            momentumLastTime = link.timestamp
            return
        }
        let delta = min(link.timestamp - momentumLastTime, 1.0 / 30.0)
        momentumLastTime = link.timestamp

        let before = viewportStart
        viewportStart += momentumBarsPerSecond * delta
        clampViewport()
        requestPageIfNeeded()
        setNeedsDisplay()

        // Frame-rate-independent deceleration.
        momentumBarsPerSecond *= pow(0.035, delta)
        if abs(momentumBarsPerSecond) < 0.25 || abs(viewportStart - before) < 0.0001 {
            stopMomentum()
        }
    }

    // MARK: Edge prefetch

    private func requestPageIfNeeded() {
        guard !candles.isEmpty else { return }
        let threshold = max(8, visibleBars * 0.35)

        if canLoadOlder,
           viewportStart <= threshold,
           requestedOldest != candles.first?.bucket {
            requestedOldest = candles.first?.bucket
            onReachStart()
        }

        if canLoadNewer,
           viewportStart + visibleBars >= Double(candles.count) - threshold,
           requestedNewest != candles.last?.bucket {
            requestedNewest = candles.last?.bucket
            onReachEnd()
        }
    }

    // MARK: Drawing

    override func draw(_ rect: CGRect) {
        guard let context = UIGraphicsGetCurrentContext(),
              let visibleRange = visibleIndexRange,
              !candles.isEmpty,
              plotRect.width > 0,
              plotRect.height > 0 else {
            return
        }

        context.saveGState()
        defer { context.restoreGState() }

        drawHeader(in: context)

        let domain = yDomain(for: visibleRange)
        drawGridAndAxes(in: context, domain: domain)

        context.saveGState()
        context.clip(to: plotRect)
        switch mode {
        case .line:
            drawLineSeries(in: context, range: visibleRange, domain: domain)
        case .candle:
            drawCandles(in: context, range: visibleRange, domain: domain)
        }
        if showsMovingAverages {
            drawMovingAverage(in: context, range: visibleRange, period: 7, color: ma7Color, domain: domain)
            drawMovingAverage(in: context, range: visibleRange, period: 25, color: ma25Color, domain: domain)
        }
        drawLastPrice(in: context, domain: domain)
        drawCrosshair(in: context, domain: domain)
        context.restoreGState()

        if showsVolume {
            drawVolume(in: context, range: visibleRange)
        }
        drawTimeAxis(in: context)
        drawPriceTags(in: context, domain: domain)
    }

    private var plotRect: CGRect {
        let volumeHeight = showsVolume ? volumePaneHeight : 0
        let paneGap: CGFloat = showsVolume ? 5 : 0
        let height = max(1, bounds.height - headerHeight - volumeHeight - timeAxisHeight - paneGap)
        return CGRect(
            x: 0,
            y: headerHeight,
            width: max(1, bounds.width - rightAxisWidth),
            height: height
        )
    }

    private var volumeRect: CGRect {
        let paneGap: CGFloat = showsVolume ? 5 : 0
        return CGRect(
            x: 0,
            y: plotRect.maxY + paneGap,
            width: plotRect.width,
            height: showsVolume ? volumePaneHeight : 0
        )
    }

    private var lineColor: UIColor {
        UIColor(trendUp ? Theme.up : Theme.down)
    }

    private func xPosition(for index: Double) -> CGFloat {
        plotRect.minX + CGFloat((index - viewportStart) / visibleBars) * plotRect.width
    }

    private func yPosition(for value: Double, domain: (low: Double, high: Double)) -> CGFloat {
        let span = max(domain.high - domain.low, 0.000_000_1)
        let ratio = (value - domain.low) / span
        return plotRect.maxY - CGFloat(ratio) * plotRect.height
    }

    private func effectiveClose(at index: Int) -> Double {
        if index == candles.count - 1, let livePrice, livePrice > 0 {
            return livePrice
        }
        return candles[index].close
    }

    private func effectiveHigh(at index: Int) -> Double {
        max(candles[index].high, index == candles.count - 1 ? (livePrice ?? 0) : 0)
    }

    private func effectiveLow(at index: Int) -> Double {
        guard index == candles.count - 1, let livePrice, livePrice > 0 else {
            return candles[index].low
        }
        return min(candles[index].low, livePrice)
    }

    private func yDomain(for range: ClosedRange<Int>) -> (low: Double, high: Double) {
        var low = Double.greatestFiniteMagnitude
        var high = -Double.greatestFiniteMagnitude
        for index in range {
            low = min(low, effectiveLow(at: index))
            high = max(high, effectiveHigh(at: index))
        }
        if !low.isFinite || !high.isFinite || high <= low {
            let value = effectiveClose(at: range.upperBound)
            let pad = max(abs(value) * 0.01, 1)
            return (value - pad, value + pad)
        }
        let pad = max((high - low) * 0.09, abs(high) * 0.0005)
        return (low - pad, high + pad)
    }

    private func drawHeader(in context: CGContext) {
        if let selectedIndex, candles.indices.contains(selectedIndex) {
            let candle = candles[selectedIndex]
            let close = effectiveClose(at: selectedIndex)
            let change = close - candle.open

            drawText(dateTimeLabel(candle.bucket), at: CGPoint(x: 0, y: 1),
                     color: UIColor(Theme.textDim), size: 10)
            let values: String
            if showsOHLCSelection {
                values = "A \(Fmt.compactPrice(candle.open))   Y \(Fmt.compactPrice(effectiveHigh(at: selectedIndex)))   D \(Fmt.compactPrice(effectiveLow(at: selectedIndex)))   K \(Fmt.compactPrice(close))"
            } else {
                values = "\(seriesTitle)  \(Fmt.money(close, currency: currency))"
            }
            drawText(values, at: CGPoint(x: 0, y: 16), color: UIColor(Theme.text), size: 10, monospaced: true)
            if showsOHLCSelection {
                let percent = candle.open == 0 ? 0 : change / candle.open * 100
                drawText(Fmt.signedPct(percent), rightX: bounds.width, y: 15,
                         color: UIColor(Theme.trend(percent)), size: 11, bold: true, monospaced: true)
            } else {
                drawText(Fmt.money(change, currency: currency), rightX: bounds.width, y: 15,
                         color: UIColor(Theme.trend(change)), size: 11, bold: true, monospaced: true)
            }
            return
        }

        var x: CGFloat = 0
        x = drawLegend(seriesTitle, color: lineColor, x: x)
        if showsMovingAverages {
            x = drawLegend("MA7", color: ma7Color, x: x + 10)
            _ = drawLegend("MA25", color: ma25Color, x: x + 10)
        }

        if let last = candles.indices.last {
            drawText(Fmt.money(effectiveClose(at: last), currency: currency),
                     rightX: bounds.width, y: 8,
                     color: UIColor(Theme.text), size: 12, bold: true, monospaced: true)
        }
    }

    @discardableResult
    private func drawLegend(_ title: String, color: UIColor, x: CGFloat) -> CGFloat {
        let y: CGFloat = 15
        color.setFill()
        UIBezierPath(roundedRect: CGRect(x: x, y: y, width: 10, height: 3), cornerRadius: 1).fill()
        let size = drawText(title, at: CGPoint(x: x + 14, y: 8),
                            color: UIColor(Theme.textDim), size: 10)
        return x + 14 + size.width
    }

    private func drawGridAndAxes(in context: CGContext, domain: (low: Double, high: Double)) {
        context.saveGState()
        defer { context.restoreGState() }

        context.setLineWidth(0.5)
        context.setStrokeColor(UIColor(Theme.stroke).withAlphaComponent(0.7).cgColor)

        let horizontalLines = 4
        for index in 0...horizontalLines {
            let ratio = CGFloat(index) / CGFloat(horizontalLines)
            let y = plotRect.minY + ratio * plotRect.height
            context.move(to: CGPoint(x: plotRect.minX, y: y))
            context.addLine(to: CGPoint(x: plotRect.maxX, y: y))
            context.strokePath()

            let value = domain.high - Double(ratio) * (domain.high - domain.low)
            drawText(Fmt.compactPrice(value),
                     at: CGPoint(x: plotRect.maxX + 5, y: y - 6),
                     color: UIColor(Theme.textDim), size: 9, monospaced: true)
        }

        context.setStrokeColor(UIColor(Theme.stroke).withAlphaComponent(0.35).cgColor)
        for index in 0...4 {
            let x = plotRect.minX + CGFloat(index) / 4 * plotRect.width
            context.move(to: CGPoint(x: x, y: plotRect.minY))
            context.addLine(to: CGPoint(x: x, y: showsVolume ? volumeRect.maxY : plotRect.maxY))
            context.strokePath()
        }
    }

    private func drawLineSeries(
        in context: CGContext,
        range: ClosedRange<Int>,
        domain: (low: Double, high: Double)
    ) {
        let path = CGMutablePath()
        for index in range {
            let point = CGPoint(
                x: xPosition(for: Double(index) + 0.5),
                y: yPosition(for: effectiveClose(at: index), domain: domain)
            )
            index == range.lowerBound ? path.move(to: point) : path.addLine(to: point)
        }

        let area = path.mutableCopy() ?? CGMutablePath()
        area.addLine(to: CGPoint(x: xPosition(for: Double(range.upperBound) + 0.5), y: plotRect.maxY))
        area.addLine(to: CGPoint(x: xPosition(for: Double(range.lowerBound) + 0.5), y: plotRect.maxY))
        area.closeSubpath()

        context.saveGState()
        context.addPath(area)
        context.clip()
        let colors = [
            lineColor.withAlphaComponent(0.24).cgColor,
            lineColor.withAlphaComponent(0).cgColor
        ] as CFArray
        if let gradient = CGGradient(colorsSpace: CGColorSpaceCreateDeviceRGB(), colors: colors, locations: [0, 1]) {
            context.drawLinearGradient(
                gradient,
                start: CGPoint(x: 0, y: plotRect.minY),
                end: CGPoint(x: 0, y: plotRect.maxY),
                options: []
            )
        }
        context.restoreGState()

        context.addPath(path)
        context.setStrokeColor(lineColor.cgColor)
        context.setLineWidth(1.8)
        context.setLineJoin(.round)
        context.setLineCap(.round)
        context.strokePath()
    }

    private func drawCandles(
        in context: CGContext,
        range: ClosedRange<Int>,
        domain: (low: Double, high: Double)
    ) {
        let spacing = plotRect.width / CGFloat(visibleBars)
        let bodyWidth = min(max(spacing * 0.64, 1.5), 14)

        for index in range {
            let candle = candles[index]
            let close = effectiveClose(at: index)
            let color = close >= candle.open ? UIColor(Theme.up) : UIColor(Theme.down)
            let x = xPosition(for: Double(index) + 0.5)
            let highY = yPosition(for: effectiveHigh(at: index), domain: domain)
            let lowY = yPosition(for: effectiveLow(at: index), domain: domain)
            let openY = yPosition(for: candle.open, domain: domain)
            let closeY = yPosition(for: close, domain: domain)

            context.setStrokeColor(color.cgColor)
            context.setLineWidth(1)
            context.move(to: CGPoint(x: x, y: highY))
            context.addLine(to: CGPoint(x: x, y: lowY))
            context.strokePath()

            color.setFill()
            let top = min(openY, closeY)
            let height = max(abs(closeY - openY), 1)
            context.fill(CGRect(x: x - bodyWidth / 2, y: top, width: bodyWidth, height: height))
        }
    }

    private func drawMovingAverage(
        in context: CGContext,
        range: ClosedRange<Int>,
        period: Int,
        color: UIColor,
        domain: (low: Double, high: Double)
    ) {
        guard candles.count >= period else { return }
        let start = max(range.lowerBound, period - 1)
        guard start <= range.upperBound else { return }

        let path = CGMutablePath()
        var isFirst = true
        for index in start...range.upperBound {
            let value = movingAverage(at: index, period: period)
            let point = CGPoint(
                x: xPosition(for: Double(index) + 0.5),
                y: yPosition(for: value, domain: domain)
            )
            if isFirst {
                path.move(to: point)
                isFirst = false
            } else {
                path.addLine(to: point)
            }
        }
        context.addPath(path)
        context.setStrokeColor(color.withAlphaComponent(0.9).cgColor)
        context.setLineWidth(1.05)
        context.setLineJoin(.round)
        context.strokePath()
    }

    private func movingAverage(at index: Int, period: Int) -> Double {
        let lower = index - period + 1
        guard lower >= 0, closePrefix.indices.contains(index + 1) else {
            return candles[index].close
        }
        var sum = closePrefix[index + 1] - closePrefix[lower]
        if index == candles.count - 1, let livePrice {
            sum += livePrice - candles[index].close
        }
        return sum / Double(period)
    }

    private func drawVolume(in context: CGContext, range: ClosedRange<Int>) {
        UIColor(Theme.bgElevated).withAlphaComponent(0.38).setFill()
        context.fill(volumeRect)

        let maxCount = max(1, range.map { candles[$0].count }.max() ?? 1)
        let spacing = volumeRect.width / CGFloat(visibleBars)
        let width = min(max(spacing * 0.64, 1), 12)

        context.saveGState()
        context.clip(to: volumeRect)
        for index in range {
            let candle = candles[index]
            guard candle.count > 0 else { continue }
            let close = effectiveClose(at: index)
            let color = close >= candle.open ? UIColor(Theme.up) : UIColor(Theme.down)
            let height = CGFloat(candle.count) / CGFloat(maxCount) * volumeRect.height
            let x = xPosition(for: Double(index) + 0.5)
            color.withAlphaComponent(0.5).setFill()
            context.fill(CGRect(
                x: x - width / 2,
                y: volumeRect.maxY - height,
                width: width,
                height: height
            ))
        }
        context.restoreGState()
    }

    private func drawLastPrice(in context: CGContext, domain: (low: Double, high: Double)) {
        guard let lastIndex = candles.indices.last,
              livePrice != nil,
              Double(lastIndex) >= viewportStart - 1,
              Double(lastIndex) <= viewportStart + visibleBars + 1 else {
            return
        }
        let y = yPosition(for: effectiveClose(at: lastIndex), domain: domain)
        context.saveGState()
        context.setStrokeColor(lineColor.withAlphaComponent(0.45).cgColor)
        context.setLineWidth(0.8)
        context.setLineDash(phase: 0, lengths: [3, 3])
        context.move(to: CGPoint(x: plotRect.minX, y: y))
        context.addLine(to: CGPoint(x: plotRect.maxX, y: y))
        context.strokePath()
        context.restoreGState()
    }

    private func drawCrosshair(in context: CGContext, domain: (low: Double, high: Double)) {
        guard let selectedIndex, candles.indices.contains(selectedIndex) else { return }
        let x = xPosition(for: Double(selectedIndex) + 0.5)
        let y = yPosition(for: effectiveClose(at: selectedIndex), domain: domain)

        context.saveGState()
        context.setStrokeColor(UIColor(Theme.textDim).withAlphaComponent(0.75).cgColor)
        context.setLineWidth(0.8)
        context.setLineDash(phase: 0, lengths: [3, 3])
        context.move(to: CGPoint(x: x, y: plotRect.minY))
        context.addLine(to: CGPoint(x: x, y: showsVolume ? volumeRect.maxY : plotRect.maxY))
        context.move(to: CGPoint(x: plotRect.minX, y: y))
        context.addLine(to: CGPoint(x: plotRect.maxX, y: y))
        context.strokePath()
        context.restoreGState()

        UIColor(Theme.text).setFill()
        context.fillEllipse(in: CGRect(x: x - 2.5, y: y - 2.5, width: 5, height: 5))
    }

    private func drawPriceTags(in context: CGContext, domain: (low: Double, high: Double)) {
        if let selectedIndex, candles.indices.contains(selectedIndex) {
            let value = effectiveClose(at: selectedIndex)
            drawTag(
                Fmt.compactPrice(value),
                atY: yPosition(for: value, domain: domain),
                background: UIColor(Theme.surfaceHi),
                foreground: UIColor(Theme.text)
            )
            return
        }

        guard let lastIndex = candles.indices.last,
              livePrice != nil,
              Double(lastIndex) >= viewportStart - 1,
              Double(lastIndex) <= viewportStart + visibleBars + 1 else {
            return
        }
        let value = effectiveClose(at: lastIndex)
        drawTag(Fmt.compactPrice(value), atY: yPosition(for: value, domain: domain),
                background: lineColor, foreground: .white)
    }

    private func drawTag(_ text: String, atY y: CGFloat, background: UIColor, foreground: UIColor) {
        let attributes = textAttributes(color: foreground, size: 9, bold: true, monospaced: true)
        let size = (text as NSString).size(withAttributes: attributes)
        let tagRect = CGRect(
            x: plotRect.maxX + 2,
            y: y - size.height / 2 - 2,
            width: min(rightAxisWidth - 3, size.width + 7),
            height: size.height + 4
        )
        background.setFill()
        UIBezierPath(roundedRect: tagRect, cornerRadius: 3).fill()
        (text as NSString).draw(
            at: CGPoint(x: tagRect.minX + 3.5, y: tagRect.minY + 2),
            withAttributes: attributes
        )
    }

    private func drawTimeAxis(in context: CGContext) {
        let y = volumeRect.maxY + 4
        for tick in 0...3 {
            let fraction = Double(tick) / 3
            let rawIndex = viewportStart + fraction * visibleBars
            let index = min(max(Int(rawIndex.rounded()), 0), candles.count - 1)
            let text = axisDateLabel(candles[index].bucket)
            let attributes = textAttributes(color: UIColor(Theme.textDim), size: 9)
            let size = (text as NSString).size(withAttributes: attributes)
            let x = xPosition(for: rawIndex) - size.width / 2
            (text as NSString).draw(
                at: CGPoint(x: min(max(x, plotRect.minX), plotRect.maxX - size.width), y: y),
                withAttributes: attributes
            )
        }

        if let selectedIndex, candles.indices.contains(selectedIndex) {
            let text = dateTimeLabel(candles[selectedIndex].bucket)
            let attributes = textAttributes(color: UIColor(Theme.text), size: 9, bold: true, monospaced: true)
            let size = (text as NSString).size(withAttributes: attributes)
            let centerX = xPosition(for: Double(selectedIndex) + 0.5)
            let rect = CGRect(
                x: min(max(centerX - size.width / 2 - 4, plotRect.minX), plotRect.maxX - size.width - 8),
                y: y - 2,
                width: size.width + 8,
                height: size.height + 4
            )
            UIColor(Theme.surfaceHi).setFill()
            UIBezierPath(roundedRect: rect, cornerRadius: 3).fill()
            (text as NSString).draw(
                at: CGPoint(x: rect.minX + 4, y: rect.minY + 2),
                withAttributes: attributes
            )
        }
    }

    // MARK: Text / date helpers

    @discardableResult
    private func drawText(
        _ text: String,
        at point: CGPoint,
        color: UIColor,
        size: CGFloat,
        bold: Bool = false,
        monospaced: Bool = false
    ) -> CGSize {
        let attributes = textAttributes(color: color, size: size, bold: bold, monospaced: monospaced)
        (text as NSString).draw(at: point, withAttributes: attributes)
        return (text as NSString).size(withAttributes: attributes)
    }

    private func drawText(
        _ text: String,
        rightX: CGFloat,
        y: CGFloat,
        color: UIColor,
        size: CGFloat,
        bold: Bool = false,
        monospaced: Bool = false
    ) {
        let attributes = textAttributes(color: color, size: size, bold: bold, monospaced: monospaced)
        let measured = (text as NSString).size(withAttributes: attributes)
        (text as NSString).draw(
            at: CGPoint(x: rightX - measured.width, y: y),
            withAttributes: attributes
        )
    }

    private func textAttributes(
        color: UIColor,
        size: CGFloat,
        bold: Bool = false,
        monospaced: Bool = false
    ) -> [NSAttributedString.Key: Any] {
        let weight: UIFont.Weight = bold ? .semibold : .regular
        let font = monospaced
            ? UIFont.monospacedDigitSystemFont(ofSize: size, weight: weight)
            : UIFont.systemFont(ofSize: size, weight: weight)
        return [.font: font, .foregroundColor: color]
    }

    private func axisDateLabel(_ date: Date) -> String {
        let span = visibleDateSpan
        if span <= 3 * 86_400 {
            return Self.timeFormatter.string(from: date)
        } else if span <= 365 * 86_400 {
            return Self.dayFormatter.string(from: date)
        } else {
            return Self.monthFormatter.string(from: date)
        }
    }

    private func dateTimeLabel(_ date: Date) -> String {
        Self.detailFormatter.string(from: date)
    }

    private var visibleDateSpan: TimeInterval {
        guard !candles.isEmpty else { return 0 }
        let first = min(max(Int(floor(viewportStart)), 0), candles.count - 1)
        let last = min(max(Int(ceil(viewportStart + visibleBars)), 0), candles.count - 1)
        return max(0, candles[last].bucket.timeIntervalSince(candles[first].bucket))
    }

    private static func formatter(_ template: String) -> DateFormatter {
        let formatter = DateFormatter()
        formatter.locale = Locale(identifier: "tr_TR")
        formatter.setLocalizedDateFormatFromTemplate(template)
        return formatter
    }

    private static let timeFormatter = formatter("HH:mm")
    private static let dayFormatter = formatter("dd MMM")
    private static let monthFormatter = formatter("MMM yy")
    private static let detailFormatter = formatter("dd MMM HH:mm")

    // MARK: Live redraw throttling

    private func scheduleLiveDisplay() {
        pendingLiveDisplay?.cancel()
        let now = CACurrentMediaTime()
        let frameInterval = 1.0 / 15.0
        let delay = max(0, frameInterval - (now - lastLiveDisplay))

        if delay == 0 {
            lastLiveDisplay = now
            setNeedsDisplay()
            return
        }

        let work = DispatchWorkItem { [weak self] in
            guard let self else { return }
            self.lastLiveDisplay = CACurrentMediaTime()
            self.setNeedsDisplay()
        }
        pendingLiveDisplay = work
        DispatchQueue.main.asyncAfter(deadline: .now() + delay, execute: work)
    }
}
