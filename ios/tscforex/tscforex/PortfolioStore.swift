import Foundation
import SwiftUI
import Combine

struct Holding: Codable, Identifiable, Equatable {
    var id = UUID()
    var code: String
    var amount: Double
    var cost: Double
}

@propertyWrapper
struct LossyDouble: Codable, Equatable {
    var wrappedValue: Double

    init(wrappedValue: Double = 0) {
        self.wrappedValue = wrappedValue
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.singleValueContainer()
        if let value = try? container.decode(Double.self) {
            wrappedValue = value
        } else if let string = try? container.decode(String.self), let value = Double(string) {
            wrappedValue = value
        } else {
            wrappedValue = 0
        }
    }

    func encode(to encoder: Encoder) throws {
        var container = encoder.singleValueContainer()
        try container.encode(String(wrappedValue))
    }
}

struct CloudPortfolio: Codable, Identifiable, Equatable {
    let id: String
    let userId: String
    var name: String
    let baseCurrency: String
    @LossyDouble var netDepositsTRY: Double
    let version: Int
    let createdAt: Date
    let updatedAt: Date

    enum CodingKeys: String, CodingKey {
        case id, name, version
        case userId = "user_id"
        case baseCurrency = "base_currency"
        case netDepositsTRY = "net_deposits_try"
        case createdAt = "created_at"
        case updatedAt = "updated_at"
    }
}

struct CloudPortfolioPosition: Codable, Identifiable, Equatable {
    let id: String
    let code: String
    let displayName: String
    let kind: String
    let currency: String
    @LossyDouble var quantity: Double
    @LossyDouble var averageCostNative: Double
    @LossyDouble var averageCostTRY: Double
    @LossyDouble var currentPriceNative: Double
    @LossyDouble var currentPriceTRY: Double
    @LossyDouble var valueNative: Double
    @LossyDouble var valueTRY: Double
    @LossyDouble var costBasisTRY: Double
    @LossyDouble var unrealizedPnLTRY: Double
    @LossyDouble var unrealizedPnLPct: Double
    @LossyDouble var realizedPnLTRY: Double
    @LossyDouble var dayChangeTRY: Double
    @LossyDouble var dayChangePct: Double
    @LossyDouble var allocationPct: Double
    let updatedAt: Date

    enum CodingKeys: String, CodingKey {
        case id, code, kind, currency, quantity
        case displayName = "display_name"
        case averageCostNative = "average_cost_native"
        case averageCostTRY = "average_cost_try"
        case currentPriceNative = "current_price_native"
        case currentPriceTRY = "current_price_try"
        case valueNative = "value_native"
        case valueTRY = "value_try"
        case costBasisTRY = "cost_basis_try"
        case unrealizedPnLTRY = "unrealized_pnl_try"
        case unrealizedPnLPct = "unrealized_pnl_pct"
        case realizedPnLTRY = "realized_pnl_try"
        case dayChangeTRY = "day_change_try"
        case dayChangePct = "day_change_pct"
        case allocationPct = "allocation_pct"
        case updatedAt = "updated_at"
    }
}

struct CloudPortfolioOverview: Codable, Equatable {
    let portfolio: CloudPortfolio
    let positions: [CloudPortfolioPosition]
    @LossyDouble var totalValueTRY: Double
    @LossyDouble var totalCostTRY: Double
    @LossyDouble var unrealizedPnLTRY: Double
    @LossyDouble var realizedPnLTRY: Double
    @LossyDouble var dayChangeTRY: Double
    @LossyDouble var dayChangePct: Double
    @LossyDouble var netDepositsTRY: Double
    let activePositions: Int

    enum CodingKeys: String, CodingKey {
        case portfolio, positions
        case totalValueTRY = "total_value_try"
        case totalCostTRY = "total_cost_try"
        case unrealizedPnLTRY = "unrealized_pnl_try"
        case realizedPnLTRY = "realized_pnl_try"
        case dayChangeTRY = "day_change_try"
        case dayChangePct = "day_change_pct"
        case netDepositsTRY = "net_deposits_try"
        case activePositions = "active_positions"
    }
}

enum PortfolioTransactionType: String, Codable, CaseIterable, Identifiable {
    case buy, sell, dividend, fee, deposit, withdraw
    var id: String { rawValue }
    var title: String {
        switch self {
        case .buy: return "Alış"
        case .sell: return "Satış"
        case .dividend: return "Temettü"
        case .fee: return "Ücret"
        case .deposit: return "Para yatırma"
        case .withdraw: return "Para çekme"
        }
    }
    var usesAsset: Bool { self != .deposit && self != .withdraw }
    var isTrade: Bool { self == .buy || self == .sell }
}

struct CloudPortfolioTransaction: Codable, Identifiable, Equatable {
    let id: String
    let portfolioId: String
    let userId: String
    let clientId: String
    let code: String
    let type: PortfolioTransactionType
    @LossyDouble var quantity: Double
    @LossyDouble var unitPriceNative: Double
    @LossyDouble var amountNative: Double
    @LossyDouble var feeNative: Double
    let currency: String
    @LossyDouble var usdtry: Double
    let occurredAt: Date
    let note: String
    let createdAt: Date
    let updatedAt: Date

    enum CodingKeys: String, CodingKey {
        case id, code, type, quantity, currency, note, usdtry
        case portfolioId = "portfolio_id"
        case userId = "user_id"
        case clientId = "client_id"
        case unitPriceNative = "unit_price_native"
        case amountNative = "amount_native"
        case feeNative = "fee_native"
        case occurredAt = "occurred_at"
        case createdAt = "created_at"
        case updatedAt = "updated_at"
    }
}

struct PortfolioPerformancePoint: Codable, Identifiable, Equatable {
    var id: String { date }
    let date: String
    @LossyDouble var valueTRY: Double
    @LossyDouble var costBasisTRY: Double
    @LossyDouble var realizedPnLTRY: Double
    @LossyDouble var totalPnLTRY: Double
    @LossyDouble var netDepositsTRY: Double

    enum CodingKeys: String, CodingKey {
        case date
        case valueTRY = "value_try"
        case costBasisTRY = "cost_basis_try"
        case realizedPnLTRY = "realized_pnl_try"
        case totalPnLTRY = "total_pnl_try"
        case netDepositsTRY = "net_deposits_try"
    }
}

struct PortfolioTransactionBody: Codable {
    let clientId: String
    let code: String
    let type: PortfolioTransactionType
    let quantity: String
    let unitPriceNative: String
    let amountNative: String
    let feeNative: String
    let currency: String
    let usdtry: String
    let occurredAt: Date
    let note: String

    enum CodingKeys: String, CodingKey {
        case code, type, quantity, currency, note, usdtry
        case clientId = "client_id"
        case unitPriceNative = "unit_price_native"
        case amountNative = "amount_native"
        case feeNative = "fee_native"
        case occurredAt = "occurred_at"
    }
}

struct PortfolioPositionDetail: Codable {
    let position: CloudPortfolioPosition
    let transactions: [CloudPortfolioTransaction]
}

private struct PortfolioOutboxItem: Codable, Identifiable {
    let id: UUID
    let portfolioId: String
    let body: PortfolioTransactionBody
}

@MainActor
final class PortfolioStore: ObservableObject {
    @Published private(set) var overview: CloudPortfolioOverview?
    @Published private(set) var performance: [PortfolioPerformancePoint] = []
    @Published private(set) var loading = false
    @Published private(set) var syncing = false
    @Published var errorMessage: String?
    @Published var legacyPreview: [Holding] = []

    private let legacyKey = "portfolio_holdings_v1"
    private let cacheKey = "portfolio_cloud_cache_v2"
    private let outboxKey = "portfolio_outbox_v1"
    private let migrationKey = "portfolio_legacy_migrated_v2"
    private var userScope: String?
    private var token: String?
    private var outbox: [PortfolioOutboxItem] = []
    private var legacy: [Holding] = []

    init() {
        if let data = UserDefaults.standard.data(forKey: legacyKey) {
            legacy = (try? JSONDecoder().decode([Holding].self, from: data)) ?? []
        }
    }

    var activePositions: [CloudPortfolioPosition] {
        (overview?.positions ?? []).filter { $0.quantity > 0 }
    }

    func sync(token: String?, prices: [String: InstrumentPrice], range: String = "1y") async {
        guard let token else { return }
        activateScope(userId: Self.jwtUserId(token))
        self.token = token
        loading = overview == nil
        syncing = true
        errorMessage = nil
        defer { loading = false; syncing = false }
        do {
            let client = APIClient(token: token)
            var portfolios = try await client.portfolios()
            if portfolios.isEmpty {
                let created = try await client.createPortfolio(name: "Ana Portföy")
                portfolios = [try await client.portfolio(id: created.id)]
            }
            var next = portfolios[0]
            if userScope == nil {
                activateScope(userId: next.portfolio.userId)
            }
            if !legacy.isEmpty && !UserDefaults.standard.bool(forKey: storageKey(migrationKey)) {
                if next.positions.contains(where: { $0.quantity > 0 }) {
                    legacyPreview = legacy
                } else {
                    try await importLegacy(portfolioId: next.portfolio.id, prices: prices)
                    next = try await client.portfolio(id: next.portfolio.id)
                }
            }
            overview = next
            saveCache()
            if await flushOutbox() {
                next = try await client.portfolio(id: next.portfolio.id)
                overview = next
                saveCache()
            }
            performance = (try? await client.portfolioPerformance(id: next.portfolio.id, range: range)) ?? []
        } catch {
            errorMessage = error.localizedDescription
        }
    }

    func refreshPerformance(range: String) async {
        guard let token, let id = overview?.portfolio.id else { return }
        performance = (try? await APIClient(token: token).portfolioPerformance(id: id, range: range)) ?? []
    }

    func positionDetail(code: String) async throws -> PortfolioPositionDetail {
        guard let token, let id = overview?.portfolio.id else { throw APIError.invalidResponse }
        return try await APIClient(token: token).portfolioPosition(id: id, code: code)
    }

    func save(body: PortfolioTransactionBody, transactionId: String? = nil,
              prices: [String: InstrumentPrice]) async {
        guard let token, let id = overview?.portfolio.id else { return }
        errorMessage = nil
        do {
            let client = APIClient(token: token)
            if let transactionId {
                _ = try await client.updatePortfolioTransaction(
                    portfolioId: id, transactionId: transactionId, body: body
                )
            } else {
                _ = try await client.createPortfolioTransaction(portfolioId: id, body: body)
            }
            await sync(token: token, prices: prices)
        } catch let error as APIError {
            if transactionId == nil, case .network = error {
                outbox.append(PortfolioOutboxItem(id: UUID(), portfolioId: id, body: body))
                persistOutbox()
                errorMessage = "İşlem çevrimdışı kuyruğa alındı."
            } else {
                errorMessage = error.localizedDescription
            }
        } catch {
            errorMessage = error.localizedDescription
        }
    }

    func delete(transactionId: String, prices: [String: InstrumentPrice]) async {
        guard let token, let id = overview?.portfolio.id else { return }
        do {
            try await APIClient(token: token).deletePortfolioTransaction(
                portfolioId: id, transactionId: transactionId
            )
            await sync(token: token, prices: prices)
        } catch {
            errorMessage = error.localizedDescription
        }
    }

    func confirmLegacyImport(prices: [String: InstrumentPrice]) async {
        guard let id = overview?.portfolio.id else { return }
        do {
            try await importLegacy(portfolioId: id, prices: prices)
            legacyPreview = []
            await sync(token: token, prices: prices)
        } catch {
            errorMessage = error.localizedDescription
        }
    }

    func skipLegacyImport() {
        UserDefaults.standard.set(true, forKey: storageKey(migrationKey))
        legacyPreview = []
    }

    private func importLegacy(portfolioId: String, prices: [String: InstrumentPrice]) async throws {
        guard let token else { return }
        let client = APIClient(token: token)
        let usdtry = max(prices["USDTRY"]?.price ?? 1, 1)
        for holding in legacy {
            guard holding.amount > 0 else { continue }
            let live = prices[holding.code]
            let isUSD = live?.isUSD == true
            let nativeCost: Double
            if holding.cost > 0 {
                nativeCost = isUSD ? holding.cost / usdtry : holding.cost
            } else {
                nativeCost = live?.price ?? 0
            }
            guard nativeCost > 0 else { continue }
            let body = PortfolioTransactionBody(
                clientId: "ios-legacy-v2-\(holding.id.uuidString)",
                code: holding.code,
                type: .buy,
                quantity: String(holding.amount),
                unitPriceNative: String(nativeCost),
                amountNative: "0",
                feeNative: "0",
                currency: isUSD ? "USD" : "TRY",
                usdtry: isUSD ? String(usdtry) : "1",
                occurredAt: Date(),
                note: "Eski cihaz portföyünden aktarıldı"
            )
            _ = try await client.createPortfolioTransaction(portfolioId: portfolioId, body: body)
        }
        UserDefaults.standard.set(true, forKey: storageKey(migrationKey))
        UserDefaults.standard.removeObject(forKey: legacyKey)
        legacy = []
    }

    private func flushOutbox() async -> Bool {
        guard let token, !outbox.isEmpty else { return false }
        let client = APIClient(token: token)
        var remaining: [PortfolioOutboxItem] = []
        var sent = false
        for item in outbox {
            do {
                _ = try await client.createPortfolioTransaction(
                    portfolioId: item.portfolioId, body: item.body
                )
                sent = true
            } catch {
                remaining.append(item)
            }
        }
        outbox = remaining
        persistOutbox()
        return sent
    }

    private func saveCache() {
        if let overview, let data = try? JSONEncoder().encode(overview) {
            UserDefaults.standard.set(data, forKey: storageKey(cacheKey))
        }
    }

    private func persistOutbox() {
        if let data = try? JSONEncoder().encode(outbox) {
            UserDefaults.standard.set(data, forKey: storageKey(outboxKey))
        }
    }

    private func storageKey(_ base: String) -> String {
        guard let userScope else { return "\(base).unresolved" }
        return "\(base).\(userScope)"
    }

    private func activateScope(userId: String?) {
        guard userScope != userId else { return }
        userScope = userId
        overview = nil
        performance = []
        legacyPreview = []
        outbox = []
        guard let userId else { return }

        let defaults = UserDefaults.standard
        if let data = defaults.data(forKey: storageKey(cacheKey)) {
            overview = try? JSONDecoder().decode(CloudPortfolioOverview.self, from: data)
        } else if let data = defaults.data(forKey: cacheKey),
                  let old = try? JSONDecoder().decode(CloudPortfolioOverview.self, from: data),
                  old.portfolio.userId == userId {
            overview = old
            defaults.set(data, forKey: storageKey(cacheKey))
            if let oldOutbox = defaults.data(forKey: outboxKey) {
                defaults.set(oldOutbox, forKey: storageKey(outboxKey))
            }
            if defaults.bool(forKey: migrationKey) {
                defaults.set(true, forKey: storageKey(migrationKey))
            }
            defaults.removeObject(forKey: cacheKey)
            defaults.removeObject(forKey: outboxKey)
            defaults.removeObject(forKey: migrationKey)
        }
        if let data = defaults.data(forKey: storageKey(outboxKey)) {
            outbox = (try? JSONDecoder().decode([PortfolioOutboxItem].self, from: data)) ?? []
        }
    }

    private static func jwtUserId(_ token: String) -> String? {
        let parts = token.split(separator: ".")
        guard parts.count > 1 else { return nil }
        var value = String(parts[1]).replacingOccurrences(of: "-", with: "+")
            .replacingOccurrences(of: "_", with: "/")
        value += String(repeating: "=", count: (4 - value.count % 4) % 4)
        guard let data = Data(base64Encoded: value),
              let json = try? JSONSerialization.jsonObject(with: data) as? [String: Any] else {
            return nil
        }
        return json["uid"] as? String
    }
}
