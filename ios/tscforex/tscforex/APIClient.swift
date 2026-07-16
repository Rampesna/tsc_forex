//
//  APIClient.swift
//  Thin async/await REST client for the TSC Forex backend.
//

import Foundation

enum APIError: Error, LocalizedError {
    case http(Int, String)
    case decoding
    case invalidResponse
    case network(URLError.Code, String)

    var errorDescription: String? {
        switch self {
        case .http(let code, let msg):
            return msg.isEmpty ? "Sunucu isteği tamamlanamadı (\(code))" : msg
        case .decoding: return "Yanıt çözümlenemedi"
        case .invalidResponse: return "Geçersiz yanıt"
        case .network(let code, let host):
            switch code {
            case .timedOut:
                return "Sunucuya ulaşılamadı (zaman aşımı): \(host). Aynı Wi-Fi'da mısınız ve backend açık mı?"
            case .cannotConnectToHost, .cannotFindHost, .networkConnectionLost, .notConnectedToInternet:
                return "Sunucuya bağlanılamadı: \(host). IP'yi Ayarlar'dan kontrol edin."
            default:
                return "Ağ hatası (\(code.rawValue)): \(host)"
            }
        }
    }
}

struct APIClient {
    var token: String?

    /// Global hook: when an authed request gets 401, this is called to refresh
    /// the access token. Returns the new token (or nil if refresh failed).
    /// Wired up in MainTabView so AuthStore owns the refresh + WS reconnect.
    static var refresher: (() async -> String?)?

    // Dedicated session with sane timeouts so requests fail fast (with a clear
    // error) instead of spinning forever when the backend is unreachable.
    private static let session: URLSession = {
        let cfg = URLSessionConfiguration.default
        cfg.timeoutIntervalForRequest = 70
        cfg.timeoutIntervalForResource = 75
        cfg.waitsForConnectivity = false
        cfg.requestCachePolicy = .reloadIgnoringLocalCacheData
        return URLSession(configuration: cfg)
    }()

    private func makeDecoder() -> JSONDecoder { JSONDecoder.api() }

    private func makeEncoder() -> JSONEncoder {
        let encoder = JSONEncoder()
        // Go's time.Time JSON decoder requires RFC3339. Swift's default Date
        // encoding is seconds since 2001 and caused portfolio requests to fail
        // binding with HTTP 400.
        encoder.dateEncodingStrategy = .iso8601
        return encoder
    }

    private func request(_ path: String, method: String = "GET", body: Encodable? = nil,
                         authed: Bool = false, overrideToken: String? = nil,
                         allowRetry: Bool = true, timeout: TimeInterval = 12) async throws -> Data {
        // Build the URL manually so query strings (e.g. "?interval=5m") are
        // preserved. appendPathComponent would percent-encode the "?".
        let full = AppConfig.apiBaseURL.absoluteString + "/" + path
        guard let url = URL(string: full) else { throw APIError.invalidResponse }
        var req = URLRequest(url: url, timeoutInterval: timeout)
        req.httpMethod = method
        req.setValue("application/json", forHTTPHeaderField: "Content-Type")
        let effToken = overrideToken ?? token
        if authed, let effToken {
            req.setValue("Bearer \(effToken)", forHTTPHeaderField: "Authorization")
        }
        if let body {
            req.httpBody = try makeEncoder().encode(AnyEncodable(body))
        }
        do {
            let (data, resp) = try await Self.session.data(for: req)
            guard let http = resp as? HTTPURLResponse else { throw APIError.invalidResponse }
            // Access token expired → refresh once and retry transparently.
            if http.statusCode == 401, authed, allowRetry, let refresher = Self.refresher {
                if let newToken = await refresher() {
                    return try await request(path, method: method, body: body, authed: authed,
                                             overrideToken: newToken, allowRetry: false, timeout: timeout)
                }
            }
            guard (200..<300).contains(http.statusCode) else {
                let msg = (try? JSONDecoder().decode(APIErrorEnvelope.self, from: data).error)
                    ?? String(data: data, encoding: .utf8)
                    ?? ""
                throw APIError.http(http.statusCode, msg)
            }
            return data
        } catch let err as URLError {
            throw APIError.network(err.code, AppConfig.host)
        }
    }

    // MARK: Auth

    func register(email: String, password: String) async throws {
        _ = try await request("auth/register", method: "POST",
                              body: ["email": email, "password": password])
    }

    func login(email: String, password: String) async throws -> TokenResponse {
        let data = try await request("auth/login", method: "POST",
                                     body: ["email": email, "password": password])
        guard let tok = try? makeDecoder().decode(TokenResponse.self, from: data) else {
            throw APIError.decoding
        }
        return tok
    }

    func refresh(_ refreshToken: String) async throws -> TokenResponse {
        let data = try await request("auth/refresh", method: "POST",
                                     body: ["refresh_token": refreshToken])
        return try makeDecoder().decode(TokenResponse.self, from: data)
    }

    /// Exchanges a Google ID token (from the OAuth flow) for our own session.
    func googleLogin(idToken: String) async throws -> TokenResponse {
        let data = try await request("auth/google", method: "POST",
                                     body: ["id_token": idToken])
        guard let tok = try? makeDecoder().decode(TokenResponse.self, from: data) else {
            throw APIError.decoding
        }
        return tok
    }

    /// Exchanges an Apple identity token (Sign in with Apple) for our session.
    func appleLogin(idToken: String, fullName: String) async throws -> TokenResponse {
        let data = try await request("auth/apple", method: "POST",
                                     body: ["id_token": idToken, "full_name": fullName])
        guard let tok = try? makeDecoder().decode(TokenResponse.self, from: data) else {
            throw APIError.decoding
        }
        return tok
    }

    // MARK: Account linking (multiple providers → one account)

    private struct ProvidersResp: Decodable { let providers: [String] }

    func linkedProviders() async throws -> [String] {
        let data = try await request("me/providers", authed: true)
        return (try? makeDecoder().decode(ProvidersResp.self, from: data).providers) ?? []
    }

    func linkGoogle(idToken: String) async throws -> [String] {
        let data = try await request("me/link/google", method: "POST",
                                     body: ["id_token": idToken], authed: true)
        return (try? makeDecoder().decode(ProvidersResp.self, from: data).providers) ?? []
    }

    func linkApple(idToken: String, fullName: String) async throws -> [String] {
        let data = try await request("me/link/apple", method: "POST",
                                     body: ["id_token": idToken, "full_name": fullName], authed: true)
        return (try? makeDecoder().decode(ProvidersResp.self, from: data).providers) ?? []
    }

    // MARK: Passkeys (management)

    func listPasskeys() async throws -> [Passkey] {
        let data = try await request("me/passkeys", authed: true)
        return (try? makeDecoder().decode(PasskeyListResponse.self, from: data).passkeys) ?? []
    }

    func deletePasskey(id: String) async throws {
        _ = try await request("me/passkeys/\(id)", method: "DELETE", authed: true)
    }

    func renamePasskey(id: String, name: String) async throws {
        _ = try await request("me/passkeys/\(id)", method: "PATCH",
                              body: ["name": name], authed: true)
    }

    // MARK: Profile

    func profile() async throws -> ProfileResponse {
        let data = try await request("me/profile", authed: true)
        return try makeDecoder().decode(ProfileResponse.self, from: data)
    }

    func updateProfile(displayName: String) async throws -> ProfileResponse {
        let data = try await request("me/profile", method: "PATCH",
                                     body: ["display_name": displayName], authed: true)
        return try makeDecoder().decode(ProfileResponse.self, from: data)
    }

    /// Uploads avatar image bytes as multipart/form-data; returns the public URL.
    func uploadAvatar(imageData: Data, filename: String, contentType: String) async throws -> String {
        let full = AppConfig.apiBaseURL.absoluteString + "/me/avatar"
        guard let url = URL(string: full) else { throw APIError.invalidResponse }
        let boundary = "Boundary-\(UUID().uuidString)"
        var req = URLRequest(url: url, timeoutInterval: 30)
        req.httpMethod = "POST"
        req.setValue("multipart/form-data; boundary=\(boundary)", forHTTPHeaderField: "Content-Type")
        if let token { req.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization") }

        var body = Data()
        body.append("--\(boundary)\r\n".data(using: .utf8)!)
        body.append("Content-Disposition: form-data; name=\"avatar\"; filename=\"\(filename)\"\r\n".data(using: .utf8)!)
        body.append("Content-Type: \(contentType)\r\n\r\n".data(using: .utf8)!)
        body.append(imageData)
        body.append("\r\n--\(boundary)--\r\n".data(using: .utf8)!)
        req.httpBody = body

        do {
            let (data, resp) = try await Self.session.data(for: req)
            guard let http = resp as? HTTPURLResponse else { throw APIError.invalidResponse }
            // On expiry, refresh once and retry.
            if http.statusCode == 401, let refresher = Self.refresher, let newToken = await refresher() {
                return try await APIClient(token: newToken).uploadAvatar(imageData: imageData, filename: filename, contentType: contentType)
            }
            guard (200..<300).contains(http.statusCode) else {
                throw APIError.http(http.statusCode, String(data: data, encoding: .utf8) ?? "")
            }
            struct Resp: Decodable { let avatar_url: String }
            guard let url = try? JSONDecoder().decode(Resp.self, from: data).avatar_url else {
                throw APIError.decoding
            }
            return url
        } catch let err as URLError {
            throw APIError.network(err.code, AppConfig.host)
        }
    }

    // MARK: Instruments

    func catalog() async throws -> [CatalogItem] {
        let data = try await request("instruments")
        return try makeDecoder().decode(CatalogResponse.self, from: data).instruments
    }

    func prices() async throws -> [InstrumentPrice] {
        let data = try await request("instruments/prices")
        return try makeDecoder().decode(PricesResponse.self, from: data).prices
    }

    func sparklines(points: Int = 10) async throws -> [String: [Double]] {
        let data = try await request("instruments/sparklines?points=\(points)")
        return try makeDecoder().decode(SparklinesResponse.self, from: data).points
    }

    func candles(code: String, interval: String, from: Date? = nil, to: Date? = nil, limit: Int? = nil) async throws -> [Candle] {
        var path = "instruments/\(code)/history?interval=\(interval)"
        let iso = ISO8601DateFormatter()
        func enc(_ d: Date) -> String? {
            iso.string(from: d).addingPercentEncoding(withAllowedCharacters: .urlQueryAllowed)
        }
        if let from, let e = enc(from) { path += "&from=\(e)" }
        if let to, let e = enc(to) { path += "&to=\(e)" }
        if let limit { path += "&limit=\(limit)" }
        let data = try await request(path)
        return try makeDecoder().decode(CandlesResponse.self, from: data).candles ?? []
    }

    func stats(code: String) async throws -> InstrumentStats {
        let data = try await request("instruments/\(code)/stats")
        return try makeDecoder().decode(InstrumentStats.self, from: data)
    }

    func detail(code: String) async throws -> InstrumentDetail {
        let data = try await request("instruments/\(code)/detail")
        return try makeDecoder().decode(InstrumentDetail.self, from: data)
    }

    func marketInsight(code: String) async throws -> MarketInsight {
        var jobID: String?
        for _ in 0..<120 {
            var body = ["code": code]
            if let jobID { body["job_id"] = jobID }
            let data = try await request(
                "me/ai/market-insight", method: "POST",
                body: body, authed: true, timeout: 30
            )
            if let job = try? makeDecoder().decode(MarketInsightJob.self, from: data), !job.jobID.isEmpty {
                jobID = job.jobID
                try await Task.sleep(nanoseconds: 3_000_000_000)
                continue
            }
            return try makeDecoder().decode(MarketInsight.self, from: data)
        }
        throw APIError.http(504, "Derin araştırma beklenenden uzun sürdü; biraz sonra yeniden deneyin.")
    }

    // MARK: Watchlist

    func watchlist() async throws -> [String] {
        let data = try await request("me/watchlist", authed: true)
        return try makeDecoder().decode(WatchlistResponse.self, from: data).codes
    }

    func addWatch(_ code: String) async throws {
        _ = try await request("me/watchlist", method: "POST", body: ["code": code], authed: true)
    }

    func removeWatch(_ code: String) async throws {
        _ = try await request("me/watchlist/\(code)", method: "DELETE", authed: true)
    }

    // MARK: Alerts

    func alerts() async throws -> [Alert] {
        let data = try await request("me/alerts", authed: true)
        return try makeDecoder().decode(AlertsResponse.self, from: data).alerts
    }

    func createAlert(code: String, direction: String, threshold: Double) async throws {
        struct Body: Encodable { let code: String; let direction: String; let threshold: Double }
        _ = try await request("me/alerts", method: "POST",
                              body: Body(code: code, direction: direction, threshold: threshold),
                              authed: true)
    }

    func deleteAlert(_ id: String) async throws {
        _ = try await request("me/alerts/\(id)", method: "DELETE", authed: true)
    }

    // MARK: Portfolio

    func portfolios() async throws -> [CloudPortfolioOverview] {
        struct Response: Decodable { let portfolios: [CloudPortfolioOverview] }
        let data = try await request("me/portfolios", authed: true)
        return try makeDecoder().decode(Response.self, from: data).portfolios
    }

    func createPortfolio(name: String) async throws -> CloudPortfolio {
        let data = try await request(
            "me/portfolios", method: "POST", body: ["name": name], authed: true
        )
        return try makeDecoder().decode(CloudPortfolio.self, from: data)
    }

    func portfolio(id: String) async throws -> CloudPortfolioOverview {
        let data = try await request("me/portfolios/\(id)", authed: true)
        return try makeDecoder().decode(CloudPortfolioOverview.self, from: data)
    }

    func portfolioPosition(id: String, code: String) async throws -> PortfolioPositionDetail {
        let safeCode = code.addingPercentEncoding(withAllowedCharacters: .urlPathAllowed) ?? code
        let data = try await request("me/portfolios/\(id)/positions/\(safeCode)", authed: true)
        return try makeDecoder().decode(PortfolioPositionDetail.self, from: data)
    }

    func createPortfolioTransaction(
        portfolioId: String, body: PortfolioTransactionBody
    ) async throws -> CloudPortfolioTransaction {
        struct Response: Decodable {
            let created: Bool
            let transaction: CloudPortfolioTransaction
        }
        let data = try await request(
            "me/portfolios/\(portfolioId)/transactions",
            method: "POST", body: body, authed: true
        )
        return try makeDecoder().decode(Response.self, from: data).transaction
    }

    func updatePortfolioTransaction(
        portfolioId: String, transactionId: String, body: PortfolioTransactionBody
    ) async throws -> CloudPortfolioTransaction {
        let data = try await request(
            "me/portfolios/\(portfolioId)/transactions/\(transactionId)",
            method: "PATCH", body: body, authed: true
        )
        return try makeDecoder().decode(CloudPortfolioTransaction.self, from: data)
    }

    func deletePortfolioTransaction(portfolioId: String, transactionId: String) async throws {
        _ = try await request(
            "me/portfolios/\(portfolioId)/transactions/\(transactionId)",
            method: "DELETE", authed: true
        )
    }

    func portfolioPerformance(id: String, range: String) async throws -> [PortfolioPerformancePoint] {
        struct Response: Decodable { let points: [PortfolioPerformancePoint] }
        let safeRange = range.addingPercentEncoding(withAllowedCharacters: .urlQueryAllowed) ?? "1y"
        let data = try await request(
            "me/portfolios/\(id)/performance?range=\(safeRange)&points=365",
            authed: true
        )
        return try makeDecoder().decode(Response.self, from: data).points
    }

    // MARK: Notifications

    func notifications() async throws -> [AppNotification] {
        let data = try await request("me/notifications", authed: true)
        return try makeDecoder().decode(NotificationsResponse.self, from: data).notifications
    }

    func deleteNotification(_ id: String) async throws {
        _ = try await request("me/notifications/\(id)", method: "DELETE", authed: true)
    }

    func clearNotifications() async throws {
        _ = try await request("me/notifications", method: "DELETE", authed: true)
    }

    // MARK: Devices

    func registerDevice(token deviceToken: String) async throws {
        _ = try await request("me/devices", method: "POST",
                              body: ["platform": "ios", "token": deviceToken], authed: true)
    }

    // MARK: Account

    /// Permanently deletes the signed-in user and all their data. Required for
    /// App Store in-app account-deletion compliance.
    func deleteAccount() async throws {
        _ = try await request("me/account", method: "DELETE", authed: true)
    }

    // MARK: Gamification

    func gamification() async throws -> GamProfile {
        let data = try await request("me/gamification", authed: true)
        return try makeDecoder().decode(GamProfile.self, from: data)
    }

    func gamificationCheckIn() async throws -> GamCheckInResult {
        let data = try await request("me/gamification/checkin", method: "POST", authed: true)
        return try makeDecoder().decode(GamCheckInResult.self, from: data)
    }

    func leaderboard(limit: Int = 50) async throws -> LeaderboardResponse {
        let data = try await request("me/gamification/leaderboard?limit=\(limit)", authed: true)
        return try makeDecoder().decode(LeaderboardResponse.self, from: data)
    }

    func gamificationExperience() async throws -> GamificationExperience {
        let data = try await request("me/gamification/experience", authed: true)
        return try makeDecoder().decode(GamificationExperience.self, from: data)
    }

    func saveDailyPlan(risk: String, codes: [String], learningGoal: String,
                       reflection: String) async throws -> GamCheckInResult {
        let body = DailyPlanBody(riskAppetite: risk, watchCodes: codes,
                                 learningGoal: learningGoal, reflection: reflection)
        let data = try await request("me/gamification/daily-plan", method: "POST",
                                     body: body, authed: true)
        return try makeDecoder().decode(GamCheckInResult.self, from: data)
    }

    func submitQuiz(lesson: String, question: String, answer: Int,
                    idempotencyKey: String = UUID().uuidString) async throws -> QuizAttemptResult {
        let body = QuizAttemptBody(lessonCode: lesson, questionCode: question,
                                   answer: answer, idempotencyKey: idempotencyKey)
        let data = try await request("me/gamification/quiz-attempts", method: "POST",
                                     body: body, authed: true)
        return try makeDecoder().decode(QuizAttemptResult.self, from: data)
    }

    func completeDiscipline(code: String, note: String) async throws -> XPAwardResponse {
        let data = try await request("me/gamification/discipline/\(code)", method: "POST",
                                     body: ["note": note], authed: true)
        return try makeDecoder().decode(XPAwardResponse.self, from: data)
    }

    func createPaperTrade(code: String, side: String, quantity: Double, entryPrice: Double,
                          thesis: String, riskBudgetPct: Double,
                          invalidation: String) async throws -> PaperTrade {
        let body = PaperTradeBody(
            clientId: UUID().uuidString, code: code, side: side, quantity: quantity,
            entryPrice: entryPrice, thesis: thesis, riskBudgetPct: riskBudgetPct,
            invalidation: invalidation, occurredAt: AppDateParser.plain.string(from: Date())
        )
        let data = try await request("me/gamification/paper-trades", method: "POST",
                                     body: body, authed: true)
        return try makeDecoder().decode(PaperTradeEnvelope.self, from: data).trade
    }

    func reviewPaperTrade(id: String, followedPlan: Bool,
                          retrospective: String) async throws -> PaperTradeReviewResponse {
        let body = PaperTradeReviewBody(followedPlan: followedPlan, retrospective: retrospective)
        let data = try await request("me/gamification/paper-trades/\(id)/review",
                                     method: "PATCH", body: body, authed: true)
        return try makeDecoder().decode(PaperTradeReviewResponse.self, from: data)
    }

    func saveGamificationPrivacy(_ privacy: GamificationPrivacy) async throws -> GamificationPrivacy {
        let data = try await request("me/gamification/privacy", method: "PATCH",
                                     body: privacy, authed: true)
        return try makeDecoder().decode(GamificationPrivacy.self, from: data)
    }

    func createStudyGroup(name: String) async throws -> StudyGroupOverview {
        let data = try await request("me/gamification/study-groups", method: "POST",
                                     body: ["name": name], authed: true)
        return try makeDecoder().decode(StudyGroupOverview.self, from: data)
    }

    func joinStudyGroup(inviteCode: String) async throws {
        _ = try await request("me/gamification/study-groups/join", method: "POST",
                              body: ["invite_code": inviteCode], authed: true)
    }
}

private struct APIErrorEnvelope: Decodable {
    let error: String
}

private struct DailyPlanBody: Encodable {
    let riskAppetite: String
    let watchCodes: [String]
    let learningGoal: String
    let reflection: String
    enum CodingKeys: String, CodingKey {
        case reflection
        case riskAppetite = "risk_appetite"
        case watchCodes = "watch_codes"
        case learningGoal = "learning_goal"
    }
}

private struct QuizAttemptBody: Encodable {
    let lessonCode: String
    let questionCode: String
    let answer: Int
    let idempotencyKey: String
    enum CodingKeys: String, CodingKey {
        case answer
        case lessonCode = "lesson_code"
        case questionCode = "question_code"
        case idempotencyKey = "idempotency_key"
    }
}

private struct PaperTradeBody: Encodable {
    let clientId: String
    let code: String
    let side: String
    let quantity: Double
    let entryPrice: Double
    let thesis: String
    let riskBudgetPct: Double
    let invalidation: String
    let occurredAt: String
    enum CodingKeys: String, CodingKey {
        case code, side, quantity, thesis, invalidation
        case clientId = "client_id"
        case entryPrice = "entry_price"
        case riskBudgetPct = "risk_budget_pct"
        case occurredAt = "occurred_at"
    }
}

private struct PaperTradeReviewBody: Encodable {
    let followedPlan: Bool
    let retrospective: String
    enum CodingKeys: String, CodingKey {
        case followedPlan = "followed_plan"
        case retrospective
    }
}

private struct PaperTradeEnvelope: Decodable {
    let trade: PaperTrade
}

struct PaperTradeReviewResponse: Decodable {
    let trade: PaperTrade
    let xpGained: Int
    enum CodingKeys: String, CodingKey {
        case trade
        case xpGained = "xp_gained"
    }
}

struct XPAwardResponse: Decodable {
    let awarded: Bool
    let xpGained: Int
    enum CodingKeys: String, CodingKey {
        case awarded
        case xpGained = "xp_gained"
    }
}

// AnyEncodable lets us pass dictionaries and structs uniformly as request bodies.
struct AnyEncodable: Encodable {
    private let encodeFunc: (Encoder) throws -> Void
    init(_ wrapped: Encodable) {
        encodeFunc = wrapped.encode
    }
    func encode(to encoder: Encoder) throws {
        try encodeFunc(encoder)
    }
}
