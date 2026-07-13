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
