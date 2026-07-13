//
//  PriceStore.swift
//  Live price state backed by a resilient WebSocket connection. Falls back to
//  REST polling if the socket drops, and auto-reconnects. Also maintains a
//  rolling sparkline buffer per instrument and the user's local favorites.
//

import Foundation
import SwiftUI
import Combine
import Network

@MainActor
final class PriceStore: ObservableObject {
    @Published var prices: [String: InstrumentPrice] = [:]
    @Published var spark: [String: [Double]] = [:]       // rolling mini-chart buffer
    @Published var favorites: Set<String> = []
    @Published var favoriteOrder: [String] = []          // user-defined favorites order
    @Published var liveNotifications: [WSNotificationPayload] = []
    @Published var connected = false
    @Published var lastUpdate: Date?

    private var task: URLSessionWebSocketTask?
    private var reconnectAttempts = 0
    private var token: String?
    private var shouldRun = false
    private let sparkCap = 40
    private let favKey = "favorite_codes"
    private let favOrderKey = "favorite_order"

    // Network-change monitor: auto-reconnects when Wi-Fi/connectivity changes
    // so the user doesn't have to manually re-connect after switching networks.
    private let netMonitor = NWPathMonitor()
    private var netStarted = false
    private var lastReopen = Date.distantPast

    // Dedicated WS session that waits for connectivity instead of failing fast.
    private lazy var wsSession: URLSession = {
        let c = URLSessionConfiguration.default
        c.waitsForConnectivity = true
        c.timeoutIntervalForRequest = 20
        return URLSession(configuration: c)
    }()

    init() {
        favorites = Set(UserDefaults.standard.stringArray(forKey: favKey) ?? [])
        favoriteOrder = UserDefaults.standard.stringArray(forKey: favOrderKey) ?? Array(favorites)
    }

    // MARK: Derived collections

    var sorted: [InstrumentPrice] {
        prices.values.sorted { $0.code < $1.code }
    }
    func gold() -> [InstrumentPrice] { sorted.filter { $0.kind == "gold" } }
    func currencies() -> [InstrumentPrice] { sorted.filter { $0.kind == "currency" } }

    // Favorites in the user's custom order (drag-to-reorder).
    func orderedFavorites() -> [InstrumentPrice] {
        favoriteOrder.compactMap { prices[$0] }.filter { favorites.contains($0.code) }
    }

    // Top gainers / losers by daily change (for the movers strip).
    func topMovers(_ n: Int = 5) -> (gainers: [InstrumentPrice], losers: [InstrumentPrice]) {
        let all = prices.values.sorted { $0.changePct > $1.changePct }
        let gainers = all.prefix(n).filter { $0.changePct > 0 }
        let losers = all.suffix(n).sorted { $0.changePct < $1.changePct }.filter { $0.changePct < 0 }
        return (Array(gainers), Array(losers))
    }

    // The N currencies with the biggest daily gain — the "top gainers" strip.
    func topGainerCurrencies(_ n: Int = 10) -> [InstrumentPrice] {
        currencies()
            .filter { $0.changePct > 0 }
            .sorted { $0.changePct > $1.changePct }
            .prefix(n)
            .map { $0 }
    }

    // MARK: Notifications (live buffer)

    func clearLiveNotifications() { liveNotifications.removeAll() }

    func removeLiveNotification(at index: Int) {
        guard liveNotifications.indices.contains(index) else { return }
        liveNotifications.remove(at: index)
    }

    // MARK: Favorites

    func isFavorite(_ code: String) -> Bool { favorites.contains(code) }

    func toggleFavorite(_ code: String) {
        if favorites.contains(code) {
            favorites.remove(code)
            favoriteOrder.removeAll { $0 == code }
            Task { try? await APIClient(token: token).removeWatch(code) }
        } else {
            favorites.insert(code)
            if !favoriteOrder.contains(code) { favoriteOrder.append(code) }
            Task { try? await APIClient(token: token).addWatch(code) }
        }
        persistFavorites()
    }

    func moveFavorite(from source: IndexSet, to destination: Int) {
        var ordered = orderedFavorites().map(\.code)
        ordered.move(fromOffsets: source, toOffset: destination)
        // Keep any favorites without a live price at the end, preserving prior order.
        let missing = favoriteOrder.filter { !ordered.contains($0) && favorites.contains($0) }
        favoriteOrder = ordered + missing
        persistFavorites()
    }

    private func persistFavorites() {
        UserDefaults.standard.set(Array(favorites), forKey: favKey)
        UserDefaults.standard.set(favoriteOrder, forKey: favOrderKey)
    }

    // MARK: Lifecycle

    func connect(token: String?) {
        self.token = token
        shouldRun = true
        reconnectAttempts = 0
        startNetMonitor()
        Task { await bootstrapREST() }
        Task { await syncFavorites() }
        openSocket()
    }

    private func startNetMonitor() {
        guard !netStarted else { return }
        netStarted = true
        netMonitor.pathUpdateHandler = { [weak self] path in
            Task { @MainActor in
                guard let self, self.shouldRun, path.status == .satisfied else { return }
                // Debounce: avoid reopening on every minor path update.
                if Date().timeIntervalSince(self.lastReopen) < 2 { return }
                self.lastReopen = Date()
                self.reconnectAttempts = 0
                self.task?.cancel(with: .goingAway, reason: nil)
                self.openSocket()
                Task { await self.bootstrapREST() }
            }
        }
        netMonitor.start(queue: DispatchQueue.global(qos: .utility))
    }

    func disconnect() {
        shouldRun = false
        task?.cancel(with: .goingAway, reason: nil)
        task = nil
        connected = false
    }

    // Reconnect the socket with a refreshed token (after token rotation).
    func updateToken(_ newToken: String?) {
        token = newToken
        guard shouldRun else { return }
        task?.cancel(with: .goingAway, reason: nil)
        openSocket()
    }

    private func bootstrapREST() async {
        let client = APIClient(token: token)
        // Seed sparklines with the last 10 DB records per instrument first, so the
        // mini charts show real recent history immediately.
        if let seeds = try? await client.sparklines(points: 10) {
            for (code, series) in seeds where !series.isEmpty {
                spark[code] = Array(series.suffix(sparkCap))
            }
        }
        if let snapshot = try? await client.prices() {
            for p in snapshot { ingest(p) }
        }
    }

    // Manual refresh (pull-to-refresh): re-pull snapshot + reseed sparklines.
    func refresh() async {
        await bootstrapREST()
    }

    private func syncFavorites() async {
        guard token != nil else { return }
        if let remote = try? await APIClient(token: token).watchlist(), !remote.isEmpty {
            favorites.formUnion(remote)
            UserDefaults.standard.set(Array(favorites), forKey: favKey)
        }
    }

    // MARK: Socket

    private func openSocket() {
        guard shouldRun else { return }
        var url = AppConfig.wsURL
        if let token {
            url = URL(string: "\(AppConfig.wsURL.absoluteString)?token=\(token)") ?? url
        }
        let t = wsSession.webSocketTask(with: url)
        task = t
        t.resume()
        connected = true
        reconnectAttempts = 0
        receive()
    }

    private func receive() {
        task?.receive { [weak self] result in
            guard let self else { return }
            switch result {
            case .success(let message):
                if case .string(let text) = message {
                    Task { @MainActor in self.handle(text) }
                }
                self.receive()
            case .failure:
                Task { @MainActor in self.scheduleReconnect() }
            }
        }
    }

    private func handle(_ text: String) {
        guard let data = text.data(using: .utf8) else { return }
        let dec = JSONDecoder.api()
        guard let env = try? dec.decode(WSEnvelope.self, from: data) else { return }
        switch env.type {
        case "instrument_update":
            if let e = try? dec.decode(WSInstrumentEnvelope.self, from: data) {
                ingest(e.payload)
                lastUpdate = Date()
            }
        case "notification":
            if let e = try? dec.decode(WSNotificationEnvelope.self, from: data) {
                liveNotifications.insert(e.payload, at: 0)
                PushManager.shared.showLocal(title: e.payload.title, body: e.payload.body)
                Haptics.warning()
            }
        default:
            break
        }
    }

    private func ingest(_ p: InstrumentPrice) {
        prices[p.code] = p
        var buf = spark[p.code] ?? []
        buf.append(p.price)
        if buf.count > sparkCap { buf.removeFirst(buf.count - sparkCap) }
        spark[p.code] = buf
    }

    private func scheduleReconnect() {
        connected = false
        guard shouldRun else { return }
        reconnectAttempts += 1
        let delay = min(Double(reconnectAttempts) * 1.5, 15)
        DispatchQueue.main.asyncAfter(deadline: .now() + delay) { [weak self] in
            self?.openSocket()
        }
    }
}
