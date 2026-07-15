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
        case .http(let code, let msg): return "Sunucu hatası (\(code)): \(msg)"
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
        cfg.timeoutIntervalForRequest = 12
        cfg.timeoutIntervalForResource = 20
        cfg.waitsForConnectivity = false
        cfg.requestCachePolicy = .reloadIgnoringLocalCacheData
        return URLSession(configuration: cfg)
    }()

    private func makeDecoder() -> JSONDecoder { JSONDecoder.api() }

    private func request(_ path: String, method: String = "GET", body: Encodable? = nil,
                         authed: Bool = false, overrideToken: String? = nil,
                         allowRetry: Bool = true) async throws -> Data {
        // Build the URL manually so query strings (e.g. "?interval=5m") are
        // preserved. appendPathComponent would percent-encode the "?".
        let full = AppConfig.apiBaseURL.absoluteString + "/" + path
        guard let url = URL(string: full) else { throw APIError.invalidResponse }
        var req = URLRequest(url: url, timeoutInterval: 12)
        req.httpMethod = method
        req.setValue("application/json", forHTTPHeaderField: "Content-Type")
        let effToken = overrideToken ?? token
        if authed, let effToken {
            req.setValue("Bearer \(effToken)", forHTTPHeaderField: "Authorization")
        }
        if let body {
            req.httpBody = try JSONEncoder().encode(AnyEncodable(body))
        }
        do {
            let (data, resp) = try await Self.session.data(for: req)
            guard let http = resp as? HTTPURLResponse else { throw APIError.invalidResponse }
            // Access token expired → refresh once and retry transparently.
            if http.statusCode == 401, authed, allowRetry, let refresher = Self.refresher {
                if let newToken = await refresher() {
                    return try await request(path, method: method, body: body, authed: authed,
                                             overrideToken: newToken, allowRetry: false)
                }
            }
            guard (200..<300).contains(http.statusCode) else {
                let msg = String(data: data, encoding: .utf8) ?? ""
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

    func candles(code: String, interval: String, from: Date? = nil) async throws -> [Candle] {
        var path = "instruments/\(code)/history?interval=\(interval)"
        if let from {
            let iso = ISO8601DateFormatter().string(from: from)
            if let enc = iso.addingPercentEncoding(withAllowedCharacters: .urlQueryAllowed) {
                path += "&from=\(enc)"
            }
        }
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
