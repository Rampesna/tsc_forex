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
    let currency: String?   // native currency of `price`: "TRY" (default) or "USD"
    let priceTRY: Double?    // TRY equivalent (== price for TRY instruments)
    let provider: String?
    let timestamp: Date?

    enum CodingKeys: String, CodingKey {
        case code
        case displayName = "display_name"
        case kind, price, bid, ask, change
        case changePct = "change_pct"
        case currency
        case priceTRY = "price_try"
        case provider, timestamp
    }

    // True when the instrument is quoted in USD (global equities).
    var isUSD: Bool { (currency ?? "TRY") == "USD" }
    // TRY equivalent for the sub-line; falls back to price for TRY instruments.
    var tryValue: Double { priceTRY ?? price }
}

struct PricesResponse: Codable {
    let count: Int
    let prices: [InstrumentPrice]
}

// Structured, risk-oriented brief from POST /me/ai/market-insight.
// Provider credentials never reach the app; this is generated server-side.
struct MarketInsight: Codable {
    let code: String
    let title: String
    let stance: String
    let stanceLabel: String
    let summary: String
    let factors: [String]
    let riskNote: String
    let confidence: String?
    let timeHorizons: [MarketHorizon]?
    let macroDrivers: [String]?
    let geopoliticalRisks: [String]?
    let crossAssetAnalysis: [String]?
    let scenarios: [MarketScenario]?
    let dataContext: [MarketDataPoint]?
    let sources: [MarketSource]?
    let generatedAt: String
    let disclaimer: String

    enum CodingKeys: String, CodingKey {
        case code, title, stance, summary, factors, confidence, scenarios, sources, disclaimer
        case stanceLabel = "stance_label"
        case riskNote = "risk_note"
        case timeHorizons = "time_horizons"
        case macroDrivers = "macro_drivers"
        case geopoliticalRisks = "geopolitical_risks"
        case crossAssetAnalysis = "cross_asset_analysis"
        case dataContext = "data_context"
        case generatedAt = "generated_at"
    }
}

struct MarketInsightJob: Codable {
    let status: String
    let jobID: String

    enum CodingKeys: String, CodingKey {
        case status
        case jobID = "job_id"
    }
}

struct MarketHorizon: Codable, Hashable {
    let horizon: String
    let outlook: String
    let rationale: String
}

struct MarketScenario: Codable, Hashable {
    let name: String
    let likelihood: String
    let trigger: String
    let implication: String
}

struct MarketDataPoint: Codable, Hashable {
    let code: String
    let displayName: String
    let price: Double
    let currency: String
    let change1D: Double?
    let change1W: Double?
    let change1M: Double?
    let change3M: Double?
    let change1Y: Double?
    let timestamp: String

    enum CodingKeys: String, CodingKey {
        case code, price, currency, timestamp
        case displayName = "display_name"
        case change1D = "change_1d_pct"
        case change1W = "change_1w_pct"
        case change1M = "change_1m_pct"
        case change3M = "change_3m_pct"
        case change1Y = "change_1y_pct"
    }
}

struct MarketSource: Codable, Hashable {
    let title: String
    let url: String
}

struct ResearchArchiveItem: Codable, Identifiable {
    let id: String
    let code: String
    let payload: MarketInsight
    let generatedAt: Date
    let expiresAt: Date

    enum CodingKeys: String, CodingKey {
        case id, code, payload
        case generatedAt = "generated_at"
        case expiresAt = "expires_at"
    }
}

// A registered passkey (WebAuthn credential) for the signed-in user.
struct Passkey: Codable, Identifiable {
    let id: String
    let name: String
    let synced: Bool
    let createdAt: Date?

    enum CodingKeys: String, CodingKey {
        case id, name, synced
        case createdAt = "created_at"
    }
}

struct PasskeyListResponse: Codable {
    let passkeys: [Passkey]
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
    var currency: String?   // "TRY" (BIST) or "USD" (global)
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
        case code, kind, trades, rank, volume, exchange, inverse, iso, fineness, currency
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

// MARK: - Gamification (streak / XP / badges / leaderboard)

struct GamBadge: Codable, Identifiable {
    var id: String { code }
    let code: String
    let name: String
    let description: String
    let icon: String
    let earned: Bool
    let earnedAt: Date?

    enum CodingKeys: String, CodingKey {
        case code, name, description, icon, earned
        case earnedAt = "earned_at"
    }
}

struct GamQuest: Codable, Identifiable {
    var id: String { code }
    let code: String
    let title: String
    let description: String
    let xp: Int
    let completed: Bool
}

struct GamProfile: Codable {
    let xp: Int
    let level: Int
    let xpIntoLevel: Int
    let xpForNextLevel: Int
    let progress: Double
    let currentStreak: Int
    let longestStreak: Int
    let checkedInToday: Bool
    let totalCheckIns: Int
    let weeklyXp: Int
    let rank: Int
    let badges: [GamBadge]
    let quests: [GamQuest]

    enum CodingKeys: String, CodingKey {
        case xp, level, progress, rank, badges, quests
        case xpIntoLevel = "xp_into_level"
        case xpForNextLevel = "xp_for_next_level"
        case currentStreak = "current_streak"
        case longestStreak = "longest_streak"
        case checkedInToday = "checked_in_today"
        case totalCheckIns = "total_check_ins"
        case weeklyXp = "weekly_xp"
    }
}

struct GamCheckInResult: Codable {
    let profile: GamProfile
    let xpGained: Int
    let newBadges: [GamBadge]
    let alreadyCheckedIn: Bool

    enum CodingKeys: String, CodingKey {
        case profile
        case xpGained = "xp_gained"
        case newBadges = "new_badges"
        case alreadyCheckedIn = "already_checked_in"
    }
}

struct LeaderEntry: Codable, Identifiable {
    var id: String { "\(rank)-\(userId)" }
    let rank: Int
    let userId: String
    let displayName: String
    let avatarUrl: String
    let xp: Int
    let level: Int

    enum CodingKeys: String, CodingKey {
        case rank, xp, level
        case userId = "user_id"
        case displayName = "display_name"
        case avatarUrl = "avatar_url"
    }
}

struct LeaderboardResponse: Codable {
    let entries: [LeaderEntry]
    let me: LeaderEntry
}

struct DailyPlan: Codable, Identifiable {
    let id: String
    let dayKey: String
    let riskAppetite: String
    let watchCodes: [String]
    let learningGoal: String
    let reflection: String

    enum CodingKeys: String, CodingKey {
        case id, reflection
        case dayKey = "day_key"
        case riskAppetite = "risk_appetite"
        case watchCodes = "watch_codes"
        case learningGoal = "learning_goal"
    }
}

struct LessonQuestion: Codable {
    let code: String
    let text: String
    let choices: [String]
}

struct LessonStatus: Codable, Identifiable {
    var id: String { code }
    let code: String
    let title: String
    let summary: String
    let cards: [String]
    let question: LessonQuestion
    let mastery: Int
    let attempts: Int
    let completed: Bool
    let dueForReview: Bool
    let nextReviewAt: Date?

    enum CodingKeys: String, CodingKey {
        case code, title, summary, cards, question, mastery, attempts, completed
        case dueForReview = "due_for_review"
        case nextReviewAt = "next_review_at"
    }
}

struct DisciplineQuest: Codable, Identifiable {
    var id: String { code }
    let code: String
    let title: String
    let description: String
    let xp: Int
    let completed: Bool
}

struct SeasonView: Codable {
    let code: String
    let name: String
    let league: String
    let points: Int
    let rank: Int
    let endsAt: Date
    let standings: [LeaderEntry]
    let scoring: String

    enum CodingKeys: String, CodingKey {
        case code, name, league, points, rank, standings, scoring
        case endsAt = "ends_at"
    }
}

struct PaperTrade: Codable, Identifiable {
    let id: String
    let clientId: String
    let code: String
    let side: String
    let quantity: Double
    let entryPrice: Double
    let thesis: String
    let riskBudgetPct: Double
    let invalidation: String
    let occurredAt: Date
    let followedPlan: Bool?
    let retrospective: String
    let processScore: Int
    let reviewedAt: Date?

    enum CodingKeys: String, CodingKey {
        case id, code, side, quantity, thesis, invalidation, retrospective
        case clientId = "client_id"
        case entryPrice = "entry_price"
        case riskBudgetPct = "risk_budget_pct"
        case occurredAt = "occurred_at"
        case followedPlan = "followed_plan"
        case processScore = "process_score"
        case reviewedAt = "reviewed_at"
    }
}

struct GamificationPrivacy: Codable {
    let userId: String
    var alias: String
    var leagueOptIn: Bool
    var personalizedLearning: Bool
    var socialOptIn: Bool
    var learningNotifications: Bool

    enum CodingKeys: String, CodingKey {
        case alias
        case userId = "user_id"
        case leagueOptIn = "league_opt_in"
        case personalizedLearning = "personalized_learning"
        case socialOptIn = "social_opt_in"
        case learningNotifications = "learning_notifications"
    }
}

struct StudyGroupOverview: Codable, Identifiable {
    let id: String
    let ownerId: String
    let name: String
    let inviteCode: String
    let memberCount: Int

    enum CodingKeys: String, CodingKey {
        case id, name
        case ownerId = "owner_id"
        case inviteCode = "invite_code"
        case memberCount = "member_count"
    }
}

struct WeeklyLearningSummary: Codable {
    let masteredLessons: Int
    let totalLessons: Int
    let seasonPoints: Int
    let dueReviews: Int

    enum CodingKeys: String, CodingKey {
        case masteredLessons = "mastered_lessons"
        case totalLessons = "total_lessons"
        case seasonPoints = "season_points"
        case dueReviews = "due_reviews"
    }
}

struct GamificationExperience: Codable {
    let ruleVersion: String
    let dailyPlan: DailyPlan?
    let lessons: [LessonStatus]
    let disciplineQuests: [DisciplineQuest]
    let season: SeasonView
    let paperTrades: [PaperTrade]
    var privacy: GamificationPrivacy
    let studyGroups: [StudyGroupOverview]
    let weeklySummary: WeeklyLearningSummary

    enum CodingKeys: String, CodingKey {
        case lessons, season, privacy
        case ruleVersion = "rule_version"
        case dailyPlan = "daily_plan"
        case disciplineQuests = "discipline_quests"
        case paperTrades = "paper_trades"
        case studyGroups = "study_groups"
        case weeklySummary = "weekly_summary"
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        ruleVersion = try container.decode(String.self, forKey: .ruleVersion)
        dailyPlan = try container.decodeIfPresent(DailyPlan.self, forKey: .dailyPlan)
        lessons = try container.decodeIfPresent([LessonStatus].self, forKey: .lessons) ?? []
        disciplineQuests = try container.decodeIfPresent(
            [DisciplineQuest].self, forKey: .disciplineQuests
        ) ?? []
        season = try container.decode(SeasonView.self, forKey: .season)
        paperTrades = try container.decodeIfPresent([PaperTrade].self, forKey: .paperTrades) ?? []
        privacy = try container.decode(GamificationPrivacy.self, forKey: .privacy)
        studyGroups = try container.decodeIfPresent(
            [StudyGroupOverview].self, forKey: .studyGroups
        ) ?? []
        weeklySummary = try container.decode(WeeklyLearningSummary.self, forKey: .weeklySummary)
    }
}

struct QuizAttemptResult: Codable {
    let correct: Bool
    let xpGained: Int
    let created: Bool
    let lesson: LessonStatus

    enum CodingKeys: String, CodingKey {
        case correct, created, lesson
        case xpGained = "xp_gained"
    }
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
