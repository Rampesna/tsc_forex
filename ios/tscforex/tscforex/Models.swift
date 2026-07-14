//
//  Models.swift
//  Codable models mirroring the backend API + WebSocket envelopes.
//

import Foundation

struct TokenResponse: Codable {
    let accessToken: String
    let refreshToken: String
    let tokenType: String

    enum CodingKeys: String, CodingKey {
        case accessToken = "access_token"
        case refreshToken = "refresh_token"
        case tokenType = "token_type"
    }
}

// Authenticated user's profile (GET/PATCH /me/profile).
struct ProfileResponse: Codable {
    let id: String
    let email: String
    let displayName: String
    let avatarURL: String
    let provider: String
    let role: String

    enum CodingKeys: String, CodingKey {
        case id, email, provider, role
        case displayName = "display_name"
        case avatarURL = "avatar_url"
    }
}

struct InstrumentPrice: Codable, Identifiable, Equatable {
    var id: String { code }
    let code: String
    let displayName: String
    let kind: String
    let price: Double
    let bid: Double
    let ask: Double
    let change: Double
    let changePct: Double
    let provider: String?
    let timestamp: Date?

    enum CodingKeys: String, CodingKey {
        case code
        case displayName = "display_name"
        case kind, price, bid, ask, change
        case changePct = "change_pct"
        case provider, timestamp
    }
}

struct PricesResponse: Codable {
    let count: Int
    let prices: [InstrumentPrice]
}

// OHLC candle for charts.
struct Candle: Codable, Identifiable {
    var id: Date { bucket }
    let bucket: Date
    let open: Double
    let high: Double
    let low: Double
    let close: Double
    let count: Int
}

struct CandlesResponse: Codable {
    let code: String
    let interval: String
    let candles: [Candle]?
}

// Last-N prices per instrument for the market-list mini charts.
struct SparklinesResponse: Codable {
    let points: [String: [Double]]
}

// 24h rolling stats for the detail header.
struct InstrumentStats: Codable {
    let code: String
    let open: Double
    let high: Double
    let low: Double
    let last: Double
    let change: Double
    let changePct: Double
    let count: Int

    enum CodingKeys: String, CodingKey {
        case code, open, high, low, last, change, count
        case changePct = "change_pct"
    }
}

// Rich per-kind market detail (GET /instruments/{code}/detail). Every field is
// optional so the same struct covers crypto / stock / gold / currency; the view
// renders whichever section matches `kind`.
struct InstrumentDetail: Codable {
    let code: String
    let kind: String
    var priceTry: Double?
    var priceUsd: Double?

    // Crypto — 24h (Binance) + fundamentals (CoinGecko)
    var high24h: Double?
    var low24h: Double?
    var open24h: Double?
    var prevClose: Double?
    var weightedAvg: Double?
    var volumeBase: Double?
    var quoteVolumeUsd: Double?
    var trades: Int?
    var changePct24h: Double?
    var marketCapUsd: Double?
    var fdvUsd: Double?
    var totalVolumeUsd: Double?
    var dominancePct: Double?
    var rank: Int?
    var circulatingSupply: Double?
    var totalSupply: Double?
    var maxSupply: Double?
    var athUsd: Double?
    var atlUsd: Double?
    var athDate: String?
    var atlDate: String?

    // Stock — Yahoo meta
    var dayHigh: Double?
    var dayLow: Double?
    var week52High: Double?
    var week52Low: Double?
    var volume: Double?
    var exchange: String?
    var longName: String?

    // Gold — derived
    var onsUsd: Double?
    var onsTry: Double?
    var usdTry: Double?
    var gramHasTry: Double?
    var gramWeight: Double?
    var fineness: Double?
    var purityLabel: String?

    // Currency — derived
    var inverse: Double?
    var usdCross: Double?
    var iso: String?

    enum CodingKeys: String, CodingKey {
        case code, kind, trades, rank, volume, exchange, inverse, iso, fineness
        case priceTry = "price_try"
        case priceUsd = "price_usd"
        case high24h, low24h, open24h
        case prevClose = "prev_close"
        case weightedAvg = "weighted_avg"
        case volumeBase = "volume_base"
        case quoteVolumeUsd = "quote_volume_usd"
        case changePct24h = "change_pct24h"
        case marketCapUsd = "market_cap_usd"
        case fdvUsd = "fdv_usd"
        case totalVolumeUsd = "total_volume_usd"
        case dominancePct = "dominance_pct"
        case circulatingSupply = "circulating_supply"
        case totalSupply = "total_supply"
        case maxSupply = "max_supply"
        case athUsd = "ath_usd"
        case atlUsd = "atl_usd"
        case athDate = "ath_date"
        case atlDate = "atl_date"
        case dayHigh = "day_high"
        case dayLow = "day_low"
        case week52High = "week52_high"
        case week52Low = "week52_low"
        case longName = "long_name"
        case onsUsd = "ons_usd"
        case onsTry = "ons_try"
        case usdTry = "usd_try"
        case gramHasTry = "gram_has_try"
        case gramWeight = "gram_weight"
        case purityLabel = "purity_label"
        case usdCross = "usd_cross"
    }
}

struct CatalogItem: Codable, Identifiable {
    var id: String { code }
    let code: String
    let displayName: String
    let kind: String
    let unit: String

    enum CodingKeys: String, CodingKey {
        case code
        case displayName = "display_name"
        case kind, unit
    }
}

struct CatalogResponse: Codable {
    let count: Int
    let instruments: [CatalogItem]
}

struct Alert: Codable, Identifiable {
    let id: String
    let code: String
    let direction: String
    let threshold: Double
    let active: Bool
    let lastFiredAt: Date?
    let createdAt: Date?

    enum CodingKeys: String, CodingKey {
        case id, code, direction, threshold, active
        case lastFiredAt = "last_fired_at"
        case createdAt = "created_at"
    }
}

struct AlertsResponse: Codable {
    let count: Int
    let alerts: [Alert]
}

struct AppNotification: Codable, Identifiable {
    let id: String
    let code: String
    let title: String
    let body: String
    let price: Double
    let readAt: Date?
    let createdAt: Date?

    enum CodingKeys: String, CodingKey {
        case id, code, title, body, price
        case readAt = "read_at"
        case createdAt = "created_at"
    }
}

struct NotificationsResponse: Codable {
    let count: Int
    let notifications: [AppNotification]
}

struct WatchlistResponse: Codable {
    let count: Int
    let codes: [String]
}

// WebSocket envelope: { "type": ..., "payload": {...}, "timestamp": ... }
struct WSEnvelope: Decodable {
    let type: String
}

struct WSInstrumentEnvelope: Decodable {
    let type: String
    let payload: InstrumentPrice
}

struct WSNotificationPayload: Decodable {
    let code: String
    let title: String
    let body: String
    let price: Double
}

struct WSNotificationEnvelope: Decodable {
    let type: String
    let payload: WSNotificationPayload
}

// MARK: - Shared JSON decoder (handles Go's RFC3339 with/without fractional seconds)

extension JSONDecoder {
    static func api() -> JSONDecoder {
        let d = JSONDecoder()
        d.dateDecodingStrategy = .custom { decoder in
            let container = try decoder.singleValueContainer()
            let str = try container.decode(String.self)
            if let date = AppDateParser.frac.date(from: str) ?? AppDateParser.plain.date(from: str) {
                return date
            }
            throw DecodingError.dataCorruptedError(in: container,
                debugDescription: "Bad date: \(str)")
        }
        return d
    }
}

enum AppDateParser {
    static let frac: ISO8601DateFormatter = {
        let f = ISO8601DateFormatter()
        f.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
        return f
    }()
    static let plain: ISO8601DateFormatter = {
        let f = ISO8601DateFormatter()
        f.formatOptions = [.withInternetDateTime]
        return f
    }()
}
