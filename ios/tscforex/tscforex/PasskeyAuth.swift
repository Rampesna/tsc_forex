//
//  PasskeyAuth.swift
//  Passwordless login/registration with platform passkeys (WebAuthn) via
//  AuthenticationServices. The backend (/auth/passkey/*) drives the ceremony;
//  this file runs the native authenticator and relays the results.
//
//  Requires: "Associated Domains" capability with
//  webcredentials:tsc-api.talhacan.com and the matching AASA file served by the
//  backend at /.well-known/apple-app-site-association.
//

import Foundation
import AuthenticationServices

enum PasskeyError: Error, LocalizedError {
    case cancelled
    case unsupported
    case badServerResponse
    case failed(String)

    var errorDescription: String? {
        switch self {
        case .cancelled: return "Geçiş anahtarı iptal edildi"
        case .unsupported: return "Bu cihaz geçiş anahtarını desteklemiyor"
        case .badServerResponse: return "Sunucudan geçersiz yanıt"
        case .failed(let m): return "Geçiş anahtarı hatası: \(m)"
        }
    }
}

@available(iOS 16.0, *)
@MainActor
final class PasskeyAuth: NSObject {
    static let shared = PasskeyAuth()

    private let rpID = "tsc-api.talhacan.com"
    private var continuation: CheckedContinuation<ASAuthorization, Error>?

    // MARK: - Public API

    /// Registers a new passkey for the signed-in user. Requires an access token.
    func register(accessToken: String) async throws {
        let begin = try await post(path: "me/passkey/register/begin", body: [:], token: accessToken)
        guard let options = (begin["options"] as? [String: Any])?["publicKey"] as? [String: Any],
              let session = begin["session"] as? String,
              let challenge = (options["challenge"] as? String).flatMap(Data.fromBase64URL),
              let user = options["user"] as? [String: Any],
              let userID = (user["id"] as? String).flatMap(Data.fromBase64URL) else {
            throw PasskeyError.badServerResponse
        }
        let provider = ASAuthorizationPlatformPublicKeyCredentialProvider(relyingPartyIdentifier: rpID)
        let request = provider.createCredentialRegistrationRequest(
            challenge: challenge,
            name: (user["name"] as? String) ?? "user",
            userID: userID
        )
        let auth = try await perform(request)
        guard let reg = auth.credential as? ASAuthorizationPlatformPublicKeyCredentialRegistration,
              let attestation = reg.rawAttestationObject else {
            throw PasskeyError.failed("kayıt yanıtı eksik")
        }
        let credential: [String: Any] = [
            "id": reg.credentialID.base64URL,
            "rawId": reg.credentialID.base64URL,
            "type": "public-key",
            "response": [
                "attestationObject": attestation.base64URL,
                "clientDataJSON": reg.rawClientDataJSON.base64URL,
            ],
        ]
        _ = try await post(path: "me/passkey/register/finish",
                           body: ["session": session, "credential": credential],
                           token: accessToken)
    }

    /// Usernameless passkey login. Returns our session tokens on success.
    func login() async throws -> TokenResponse {
        let begin = try await post(path: "auth/passkey/login/begin", body: [:], token: nil)
        guard let options = (begin["options"] as? [String: Any])?["publicKey"] as? [String: Any],
              let session = begin["session"] as? String,
              let challenge = (options["challenge"] as? String).flatMap(Data.fromBase64URL) else {
            throw PasskeyError.badServerResponse
        }
        let provider = ASAuthorizationPlatformPublicKeyCredentialProvider(relyingPartyIdentifier: rpID)
        let request = provider.createCredentialAssertionRequest(challenge: challenge)
        let auth = try await perform(request)
        guard let assertion = auth.credential as? ASAuthorizationPlatformPublicKeyCredentialAssertion else {
            throw PasskeyError.failed("giriş yanıtı eksik")
        }
        let credential: [String: Any] = [
            "id": assertion.credentialID.base64URL,
            "rawId": assertion.credentialID.base64URL,
            "type": "public-key",
            "response": [
                "authenticatorData": assertion.rawAuthenticatorData.base64URL,
                "clientDataJSON": assertion.rawClientDataJSON.base64URL,
                "signature": assertion.signature.base64URL,
                "userHandle": assertion.userID.base64URL,
            ],
        ]
        let finish = try await post(path: "auth/passkey/login/finish",
                                    body: ["session": session, "credential": credential],
                                    token: nil)
        guard let access = finish["access_token"] as? String,
              let refresh = finish["refresh_token"] as? String else {
            throw PasskeyError.badServerResponse
        }
        return TokenResponse(accessToken: access, refreshToken: refresh, tokenType: "Bearer")
    }

    // MARK: - AuthenticationServices bridge

    private func perform(_ request: ASAuthorizationRequest) async throws -> ASAuthorization {
        try await withCheckedThrowingContinuation { cont in
            self.continuation = cont
            let controller = ASAuthorizationController(authorizationRequests: [request])
            controller.delegate = self
            controller.presentationContextProvider = self
            controller.performRequests()
        }
    }

    // MARK: - Networking

    private func post(path: String, body: [String: Any], token: String?) async throws -> [String: Any] {
        let url = URL(string: AppConfig.apiBaseURL.absoluteString + "/" + path)!
        var req = URLRequest(url: url, timeoutInterval: 20)
        req.httpMethod = "POST"
        req.setValue("application/json", forHTTPHeaderField: "Content-Type")
        if let token { req.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization") }
        req.httpBody = try JSONSerialization.data(withJSONObject: body)
        let (data, resp) = try await URLSession.shared.data(for: req)
        guard let http = resp as? HTTPURLResponse, (200..<300).contains(http.statusCode) else {
            throw PasskeyError.failed(String(data: data, encoding: .utf8) ?? "bilinmeyen")
        }
        return (try? JSONSerialization.jsonObject(with: data) as? [String: Any]) ?? [:]
    }
}

@available(iOS 16.0, *)
extension PasskeyAuth: ASAuthorizationControllerDelegate {
    func authorizationController(controller: ASAuthorizationController,
                                didCompleteWithAuthorization authorization: ASAuthorization) {
        continuation?.resume(returning: authorization); continuation = nil
    }

    func authorizationController(controller: ASAuthorizationController,
                                didCompleteWithError error: Error) {
        let ns = error as NSError
        if ns.domain == ASAuthorizationError.errorDomain,
           ns.code == ASAuthorizationError.canceled.rawValue {
            continuation?.resume(throwing: PasskeyError.cancelled)
        } else {
            continuation?.resume(throwing: PasskeyError.failed(error.localizedDescription))
        }
        continuation = nil
    }
}

@available(iOS 16.0, *)
extension PasskeyAuth: ASAuthorizationControllerPresentationContextProviding {
    func presentationAnchor(for controller: ASAuthorizationController) -> ASPresentationAnchor {
        let scenes = UIApplication.shared.connectedScenes.compactMap { $0 as? UIWindowScene }
        return scenes.flatMap { $0.windows }.first { $0.isKeyWindow } ?? ASPresentationAnchor()
    }
}

// MARK: - base64url helpers

extension Data {
    var base64URL: String {
        base64EncodedString()
            .replacingOccurrences(of: "+", with: "-")
            .replacingOccurrences(of: "/", with: "_")
            .replacingOccurrences(of: "=", with: "")
    }

    static func fromBase64URL(_ s: String) -> Data? {
        var str = s.replacingOccurrences(of: "-", with: "+").replacingOccurrences(of: "_", with: "/")
        while str.count % 4 != 0 { str.append("=") }
        return Data(base64Encoded: str)
    }
}
