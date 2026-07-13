//
//  AppleAuth.swift
//  Native "Sign in with Apple" using AuthenticationServices. Returns the Apple
//  identity token (a signed JWT) plus the full name, which Apple only provides
//  on the very first authorization. The backend (/auth/apple) verifies the
//  token against Apple's public keys and issues our own session.
//

import Foundation
import AuthenticationServices

enum AppleAuthError: Error, LocalizedError {
    case cancelled
    case noToken
    case failed(String)

    var errorDescription: String? {
        switch self {
        case .cancelled: return "Apple girişi iptal edildi"
        case .noToken: return "Apple kimlik doğrulama belirteci alınamadı"
        case .failed(let m): return "Apple girişi başarısız: \(m)"
        }
    }
}

struct AppleCredential {
    let idToken: String
    let fullName: String
}

@MainActor
final class AppleAuth: NSObject {
    static let shared = AppleAuth()

    private var continuation: CheckedContinuation<AppleCredential, Error>?

    /// Presents the system Sign in with Apple sheet and returns the credential.
    func signIn() async throws -> AppleCredential {
        try await withCheckedThrowingContinuation { cont in
            self.continuation = cont
            let request = ASAuthorizationAppleIDProvider().createRequest()
            request.requestedScopes = [.fullName, .email]
            let controller = ASAuthorizationController(authorizationRequests: [request])
            controller.delegate = self
            controller.presentationContextProvider = self
            controller.performRequests()
        }
    }
}

extension AppleAuth: ASAuthorizationControllerDelegate {
    func authorizationController(controller: ASAuthorizationController,
                                didCompleteWithAuthorization authorization: ASAuthorization) {
        defer { continuation = nil }
        guard let cred = authorization.credential as? ASAuthorizationAppleIDCredential,
              let tokenData = cred.identityToken,
              let idToken = String(data: tokenData, encoding: .utf8) else {
            continuation?.resume(throwing: AppleAuthError.noToken)
            return
        }
        var name = ""
        if let full = cred.fullName {
            name = [full.givenName, full.familyName]
                .compactMap { $0 }
                .joined(separator: " ")
                .trimmingCharacters(in: .whitespaces)
        }
        continuation?.resume(returning: AppleCredential(idToken: idToken, fullName: name))
    }

    func authorizationController(controller: ASAuthorizationController,
                                didCompleteWithError error: Error) {
        defer { continuation = nil }
        let ns = error as NSError
        if ns.domain == ASAuthorizationError.errorDomain,
           ns.code == ASAuthorizationError.canceled.rawValue {
            continuation?.resume(throwing: AppleAuthError.cancelled)
        } else {
            continuation?.resume(throwing: AppleAuthError.failed(error.localizedDescription))
        }
    }
}

extension AppleAuth: ASAuthorizationControllerPresentationContextProviding {
    func presentationAnchor(for controller: ASAuthorizationController) -> ASPresentationAnchor {
        let scenes = UIApplication.shared.connectedScenes.compactMap { $0 as? UIWindowScene }
        let window = scenes.flatMap { $0.windows }.first { $0.isKeyWindow }
        return window ?? ASPresentationAnchor()
    }
}
