//
//  APIClient.swift
//  Thin async/await REST client for the TSC Forex backend.
//

import Foundation

struct APIErrorDetails: Decodable, Equatable {
    let code: String?
    let feature: String?
    let limit: Int?
    let used: Int?
}

enum APIError: Error, LocalizedError {
    case http(Int, String, APIErrorDetails?)
    case decoding
    case invalidResponse
    case network(URLError.Code, String)

    var statusCode: Int? {
        guard case .http(let status, _, _) = self else { return nil }
        return status
    }

    var serverCode: String? {
        guard case .http(_, _, let details) = self else { return nil }
        return details?.code
    }

    var feature: String? {
        guard case .http(_, _, let details) = self else { return nil }
        return details?.feature
    }

    var limit: Int? {
        guard case .http(_, _, let details) = self else { return nil }
        return details?.limit
    }

    var used: Int? {
        guard case .http(_, _, let details) = self else { return nil }
        return details?.used
    }

    /// A retry with the same request identifier is safe and required for these
    /// failures because the server may already have reserved quota or output.
    var shouldRetainRequestID: Bool {
        switch self {
        case .network, .invalidResponse, .decoding:
            return true
        case .http(let status, _, let details):
            if status == 409 { return details?.code == "operation_in_progress" }
            return status == 408 || status == 429 || status == 500 ||
                status == 503 || status == 504
        }
    }

    var errorDescription: String? {
        switch self {
        case .http(let code, let msg, _):
            return msg.isEmpty ? "Sunucu isteği tamamlanamadı (\(code))" : msg
        case .decoding: return "Yanıt çözümlenemedi"
        case .invalidResponse: return "Geçersiz yanıt"
        case .network(let code, let host):
            switch code {
            case .timedOut:
                return "Bağlantı zaman aşımına uğradı. İnternet bağlantınızı kontrol edip tekrar deneyin. (\(host))"
            case .cannotConnectToHost, .cannotFindHost, .networkConnectionLost, .notConnectedToInternet:
                return "Hizmete ulaşılamadı. İnternet bağlantınızı kontrol edip tekrar deneyin. (\(host))"
            case .cancelled:
                // Pull-to-refresh / view teardown cancels in-flight requests
                // (URLError -999). Never surface this as a user-facing failure.
                return nil
            default:
                return "Ağ hatası (\(code.rawValue)): \(host)"
            }
        }
    }

    /// True when the failure is an intentional cancellation (SwiftUI refresh,
    /// navigation pop, Task cancel) rather than a real network outage.
    var isCancellation: Bool {
        if case .network(let code, _) = self, code == .cancelled { return true }
        return false
    }
}

extension Error {
    /// Pull-to-refresh cancels the previous `.task` load; treating that as a
    /// visible "Ağ hatası (-999)" confuses users. Callers should ignore these.
    var isRequestCancellation: Bool {
        if self is CancellationError { return true }
        if let api = self as? APIError { return api.isCancellation }
        if let url = self as? URLError { return url.code == .cancelled }
        return false
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
        cfg.timeoutIntervalForRequest = 90
        cfg.timeoutIntervalForResource = 100
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

    private func applyClientHeaders(to request: inout URLRequest) {
        request.setValue("ios", forHTTPHeaderField: "X-TSC-Client-Platform")
        request.setValue(
            L10n.language.rawValue,
            forHTTPHeaderField: "X-TSC-Client-Locale"
        )
        if let version = Bundle.main.object(
            forInfoDictionaryKey: "CFBundleShortVersionString"
        ) as? String {
            request.setValue(version, forHTTPHeaderField: "X-TSC-App-Version")
        }
        request.setValue("Apple mobile device", forHTTPHeaderField: "X-TSC-Device-Model")
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
        applyClientHeaders(to: &req)
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
                let envelope = try? JSONDecoder().decode(APIErrorEnvelope.self, from: data)
                let msg = envelope?.error ?? String(data: data, encoding: .utf8) ?? ""
                throw APIError.http(http.statusCode, msg, envelope?.details)
            }
            return data
        } catch let err as URLError {
            throw APIError.network(err.code, AppConfig.host)
        }
    }

    /// Retries only the server's explicit in-progress response and always uses
    /// the caller-supplied request ID unchanged. Other failures are surfaced so
    /// the UI can preserve that ID across user retries and process restarts.
    private func requestIdempotentOperation(
        _ path: String, body: Encodable, timeout: TimeInterval
    ) async throws -> Data {
        for attempt in 0..<5 {
            do {
                return try await request(
                    path, method: "POST", body: body, authed: true, timeout: timeout
                )
            } catch let error as APIError where
                error.statusCode == 409 && error.serverCode == "operation_in_progress" {
                guard attempt < 4 else { throw error }
                try await Task.sleep(nanoseconds: 1_000_000_000)
            }
        }
        throw APIError.invalidResponse
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
        applyClientHeaders(to: &req)
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
                let envelope = try? JSONDecoder().decode(APIErrorEnvelope.self, from: data)
                let message = envelope?.error ?? String(data: data, encoding: .utf8) ?? ""
                throw APIError.http(http.statusCode, message, envelope?.details)
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

    func candles(
        code: String,
        interval: String,
        from: Date? = nil,
        to: Date? = nil,
        limit: Int? = nil,
        latestSession: Bool = false
    ) async throws -> [Candle] {
        var path = "instruments/\(code)/history?interval=\(interval)"
        let iso = ISO8601DateFormatter()
        func enc(_ d: Date) -> String? {
            iso.string(from: d).addingPercentEncoding(withAllowedCharacters: .urlQueryAllowed)
        }
        if let from, let e = enc(from) { path += "&from=\(e)" }
        if let to, let e = enc(to) { path += "&to=\(e)" }
        if let limit { path += "&limit=\(limit)" }
        if latestSession { path += "&session=latest" }
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

    func marketInsight(code: String, userID: String) async throws -> MarketInsight {
        let operation = PendingOperationStore.loadOrCreate(
            userID: userID, operation: "ai-research", resource: code.uppercased()
        )
        var jobID = operation.jobID
        let requestID = operation.requestID
        var proof: AppAttestProofPayload?
        var requestedAttestation = false
        do {
            for _ in 0..<120 {
                var body = [
                    "code": code,
                    "device_id": DeviceIdentity.identifier,
                    "request_id": requestID,
                ]
                if let jobID { body["job_id"] = jobID }
                if jobID == nil, let proof {
                    body["attestation_key_id"] = proof.keyID
                    body["attestation_challenge_id"] = proof.challengeID
                    body["attestation_assertion"] = proof.assertion
                    body["device_check_token"] = proof.deviceCheckToken
                }
                let data: Data
                do {
                    data = try await request(
                        "me/ai/market-insight", method: "POST",
                        body: body, authed: true, timeout: 75
                    )
                } catch let error as APIError where
                    error.statusCode == 403 &&
                    error.serverCode == "device_attestation_required" &&
                    jobID == nil && !requestedAttestation {
                    guard let token, !token.isEmpty else { throw APIError.invalidResponse }
                    requestedAttestation = true
                    proof = try await AppAttestClient.shared.proof(
                        token: token, code: code, requestID: requestID
                    )
                    continue
                } catch let error as APIError where
                    error.statusCode == 409 && error.serverCode == "operation_in_progress" {
                    try await Task.sleep(nanoseconds: 1_500_000_000)
                    continue
                }
                if let job = try? makeDecoder().decode(MarketInsightJob.self, from: data), !job.jobID.isEmpty {
                    jobID = job.jobID
                    PendingOperationStore.updateJobID(
                        job.jobID, userID: userID, operation: "ai-research", resource: code.uppercased()
                    )
                    try await Task.sleep(nanoseconds: 3_000_000_000)
                    continue
                }
                let result = try makeDecoder().decode(MarketInsight.self, from: data)
                PendingOperationStore.clear(
                    userID: userID, operation: "ai-research", resource: code.uppercased()
                )
                return result
            }
            throw APIError.http(
                504, "Derin araştırma beklenenden uzun sürdü; biraz sonra yeniden deneyin.", nil
            )
        } catch let error as APIError {
            if !error.shouldRetainRequestID {
                PendingOperationStore.clear(
                    userID: userID, operation: "ai-research", resource: code.uppercased()
                )
            }
            throw error
        } catch {
            // Unknown transport/decoding failures can happen after the backend
            // accepted the operation, so retain the identifier for a safe retry.
            throw error
        }
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

    func updateAlert(id: String, direction: String, threshold: Double) async throws -> Alert {
        struct Body: Encodable { let direction: String; let threshold: Double }
        let data = try await request("me/alerts/\(id)", method: "PATCH",
                                     body: Body(direction: direction, threshold: threshold),
                                     authed: true)
        return try makeDecoder().decode(Alert.self, from: data)
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

    func updatePortfolio(id: String, name: String) async throws -> CloudPortfolio {
        let data = try await request(
            "me/portfolios/\(id)", method: "PATCH", body: ["name": name], authed: true
        )
        return try makeDecoder().decode(CloudPortfolio.self, from: data)
    }

    func deletePortfolio(id: String) async throws {
        _ = try await request("me/portfolios/\(id)", method: "DELETE", authed: true)
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

    func portfolioHealth(id: String, requestID: String) async throws -> PortfolioHealth {
        let data = try await requestIdempotentOperation(
            "me/portfolios/\(id)/health-check",
            body: ["request_id": requestID], timeout: 30
        )
        return try makeDecoder().decode(PortfolioHealth.self, from: data)
    }

    func exportPortfolio(id: String, requestID: String) async throws -> Data {
        try await requestIdempotentOperation(
            "me/portfolios/\(id)/export",
            body: ["request_id": requestID], timeout: 30
        )
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
                              body: [
                                "platform": "ios",
                                "token": deviceToken,
                                "device_id": DeviceIdentity.identifier,
                              ], authed: true)
    }

    // MARK: App Attest

    func deviceAttestationChallenge(deviceID: String, keyID: String?, purpose: String,
                                    requestID: String, code: String) async throws -> DeviceAttestationChallenge {
        let body = DeviceAttestationChallengeBody(
            deviceID: deviceID, keyID: keyID, purpose: purpose,
            requestID: requestID, code: code
        )
        let data = try await request(
            "me/device-attestation/challenge", method: "POST", body: body,
            authed: true, timeout: 20
        )
        return try makeDecoder().decode(DeviceAttestationChallenge.self, from: data)
    }

    func registerDeviceAttestationKey(deviceID: String, keyID: String, challengeID: String,
                                      attestationObject: String) async throws {
        let body = DeviceAttestationRegistrationBody(
            deviceID: deviceID, keyID: keyID, challengeID: challengeID,
            attestationObject: attestationObject
        )
        _ = try await request(
            "me/device-attestation/register", method: "POST", body: body,
            authed: true, timeout: 30
        )
    }

    // MARK: Billing

    func billingPlans() async throws -> BillingPlansResponse {
        let data = try await request("billing/plans")
        return try makeDecoder().decode(BillingPlansResponse.self, from: data)
    }

    func billingStatus() async throws -> BillingStatus {
        let id = DeviceIdentity.identifier.addingPercentEncoding(withAllowedCharacters: .urlQueryAllowed)
            ?? DeviceIdentity.identifier
        let data = try await request("me/billing/status?device_id=\(id)", authed: true)
        return try makeDecoder().decode(BillingStatus.self, from: data)
    }

    func researchHistory(limit: Int = 50) async throws -> [ResearchArchiveItem] {
        struct Response: Decodable { let items: [ResearchArchiveItem] }
        let bounded = min(max(limit, 1), 100)
        let data = try await request("me/ai/research-history?limit=\(bounded)", authed: true)
        return try makeDecoder().decode(Response.self, from: data).items
    }

    func syncApplePurchase(productID: String, transactionID: String,
                           originalTransactionID: String, signedTransaction: String,
                           deviceID: String, subscriptionRecoveryToken: String) async throws -> BillingStatus {
        let body = ApplePurchaseSyncBody(
            productID: productID,
            transactionID: transactionID,
            originalTransactionID: originalTransactionID,
            signedTransaction: signedTransaction,
            deviceID: deviceID,
            subscriptionRecoveryToken: subscriptionRecoveryToken
        )
        let data = try await request(
            "me/billing/apple/sync", method: "POST", body: body, authed: true, timeout: 30
        )
        return try makeDecoder().decode(BillingStatus.self, from: data)
    }

    // MARK: Account

    /// Permanently deletes the signed-in user and all their data. Required for
    /// App Store in-app account-deletion compliance.
    func deleteAccount(appleAuthorizationCode: String? = nil,
                       subscriptionRecoveryToken: String? = nil) async throws {
        let body = AccountDeletionBody(
            appleAuthorizationCode: appleAuthorizationCode,
            subscriptionRecoveryToken: subscriptionRecoveryToken
        )
        _ = try await request("me/account", method: "DELETE", body: body, authed: true)
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
    let code: String?
    let feature: String?
    let limit: Int?
    let used: Int?

    var details: APIErrorDetails? {
        guard code != nil || feature != nil || limit != nil || used != nil else { return nil }
        return APIErrorDetails(code: code, feature: feature, limit: limit, used: used)
    }
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

private struct ApplePurchaseSyncBody: Encodable {
    let productID: String
    let transactionID: String
    let originalTransactionID: String
    let signedTransaction: String
    let deviceID: String
    let subscriptionRecoveryToken: String

    enum CodingKeys: String, CodingKey {
        case productID = "product_id"
        case transactionID = "transaction_id"
        case originalTransactionID = "original_transaction_id"
        case signedTransaction = "signed_transaction"
        case deviceID = "device_id"
        case subscriptionRecoveryToken = "subscription_recovery_token"
    }
}

private struct AccountDeletionBody: Encodable {
    let appleAuthorizationCode: String?
    let subscriptionRecoveryToken: String?

    enum CodingKeys: String, CodingKey {
        case appleAuthorizationCode = "apple_authorization_code"
        case subscriptionRecoveryToken = "subscription_recovery_token"
    }
}

private struct DeviceAttestationChallengeBody: Encodable {
    let deviceID: String
    let keyID: String?
    let purpose: String
    let requestID: String
    let code: String

    enum CodingKeys: String, CodingKey {
        case purpose, code
        case deviceID = "device_id"
        case keyID = "key_id"
        case requestID = "request_id"
    }
}

private struct DeviceAttestationRegistrationBody: Encodable {
    let deviceID: String
    let keyID: String
    let challengeID: String
    let attestationObject: String

    enum CodingKeys: String, CodingKey {
        case deviceID = "device_id"
        case keyID = "key_id"
        case challengeID = "challenge_id"
        case attestationObject = "attestation_object"
    }
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
