//
//  GoogleAuth.swift
//  Native "Sign in with Google" using the system browser (ASWebAuthenticationSession)
//  and the OAuth 2.0 Authorization Code + PKCE flow — no third-party SDK needed.
//
//  Flow: open Google consent in a secure system browser → receive an auth code on
//  our reversed-client-id redirect → exchange the code (with PKCE verifier, no
//  client secret for iOS clients) for an ID token → hand the ID token to our
//  backend (/auth/google), which verifies it and returns our own session.
//

import Foundation
import AuthenticationServices
import CryptoKit

enum GoogleAuthError: Error, LocalizedError {
    case cancelled
    case noCode
    case tokenExchange(String)

    var errorDescription: String? {
        switch self {
        case .cancelled: return "Google girişi iptal edildi"
        case .noCode: return "Google yetkilendirme kodu alınamadı"
        case .tokenExchange(let m): return "Google token değişimi başarısız: \(m)"
        }
    }
}

enum GoogleAuth {
    // Public iOS OAuth client (not a secret).
    static let clientID = "83380535128-16nfuuc3p0jok4l4htitlubgpsjah622.apps.googleusercontent.com"
    static let reversedClientID = "com.googleusercontent.apps.83380535128-16nfuuc3p0jok4l4htitlubgpsjah622"
    private static let redirectURI = reversedClientID + ":/oauth2redirect"

    /// Runs the full flow and returns a verified Google ID token.
    @MainActor
    static func signIn() async throws -> String {
        let verifier = randomURLSafe(64)
        let challenge = codeChallenge(for: verifier)

        var comps = URLComponents(string: "https://accounts.google.com/o/oauth2/v2/auth")!
        comps.queryItems = [
            .init(name: "client_id", value: clientID),
            .init(name: "redirect_uri", value: redirectURI),
            .init(name: "response_type", value: "code"),
            .init(name: "scope", value: "openid email profile"),
            .init(name: "code_challenge", value: challenge),
            .init(name: "code_challenge_method", value: "S256"),
            .init(name: "prompt", value: "select_account"),
        ]

        let callbackURL = try await presentWebAuth(authURL: comps.url!)
        guard let code = URLComponents(url: callbackURL, resolvingAgainstBaseURL: false)?
            .queryItems?.first(where: { $0.name == "code" })?.value else {
            throw GoogleAuthError.noCode
        }
        return try await exchange(code: code, verifier: verifier)
    }

    // MARK: - Web auth session

    @MainActor
    private static func presentWebAuth(authURL: URL) async throws -> URL {
        try await withCheckedThrowingContinuation { cont in
            let session = ASWebAuthenticationSession(
                url: authURL,
                callbackURLScheme: reversedClientID
            ) { url, error in
                if let url { cont.resume(returning: url) }
                else if let error {
                    let ns = error as NSError
                    if ns.domain == ASWebAuthenticationSessionError.errorDomain,
                       ns.code == ASWebAuthenticationSessionError.canceledLogin.rawValue {
                        cont.resume(throwing: GoogleAuthError.cancelled)
                    } else {
                        cont.resume(throwing: error)
                    }
                } else {
                    cont.resume(throwing: GoogleAuthError.cancelled)
                }
            }
            session.presentationContextProvider = presenter
            session.prefersEphemeralWebBrowserSession = false
            if !session.start() { cont.resume(throwing: GoogleAuthError.cancelled) }
        }
    }

    private static let presenter = WebAuthPresenter()

    // MARK: - Token exchange (PKCE, no client secret for iOS clients)

    private static func exchange(code: String, verifier: String) async throws -> String {
        var req = URLRequest(url: URL(string: "https://oauth2.googleapis.com/token")!)
        req.httpMethod = "POST"
        req.setValue("application/x-www-form-urlencoded", forHTTPHeaderField: "Content-Type")
        let form: [String: String] = [
            "client_id": clientID,
            "code": code,
            "code_verifier": verifier,
            "grant_type": "authorization_code",
            "redirect_uri": redirectURI,
        ]
        req.httpBody = form.map { "\($0.key)=\(formEncode($0.value))" }
            .joined(separator: "&").data(using: .utf8)

        let (data, resp) = try await URLSession.shared.data(for: req)
        guard let http = resp as? HTTPURLResponse, (200..<300).contains(http.statusCode) else {
            throw GoogleAuthError.tokenExchange(String(data: data, encoding: .utf8) ?? "unknown")
        }
        struct TokenResp: Decodable { let id_token: String? }
        guard let idToken = (try? JSONDecoder().decode(TokenResp.self, from: data))?.id_token else {
            throw GoogleAuthError.tokenExchange("id_token missing")
        }
        return idToken
    }

    // MARK: - PKCE helpers

    private static func randomURLSafe(_ n: Int) -> String {
        var bytes = [UInt8](repeating: 0, count: n)
        _ = SecRandomCopyBytes(kSecRandomDefault, n, &bytes)
        return Data(bytes).base64URLEncoded()
    }

    private static func codeChallenge(for verifier: String) -> String {
        let hash = SHA256.hash(data: Data(verifier.utf8))
        return Data(hash).base64URLEncoded()
    }

    private static func formEncode(_ s: String) -> String {
        var allowed = CharacterSet.alphanumerics
        allowed.insert(charactersIn: "-._~")
        return s.addingPercentEncoding(withAllowedCharacters: allowed) ?? s
    }
}

private extension Data {
    func base64URLEncoded() -> String {
        base64EncodedString()
            .replacingOccurrences(of: "+", with: "-")
            .replacingOccurrences(of: "/", with: "_")
            .replacingOccurrences(of: "=", with: "")
    }
}

/// Supplies the key window for the auth sheet.
final class WebAuthPresenter: NSObject, ASWebAuthenticationPresentationContextProviding {
    func presentationAnchor(for session: ASWebAuthenticationSession) -> ASPresentationAnchor {
        let scenes = UIApplication.shared.connectedScenes.compactMap { $0 as? UIWindowScene }
        let window = scenes.flatMap { $0.windows }.first { $0.isKeyWindow }
        return window ?? ASPresentationAnchor()
    }
}
