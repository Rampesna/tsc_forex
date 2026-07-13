//
//  AuthStore.swift
//  Holds auth state + tokens, persisted in the Keychain-like UserDefaults store.
//  (For production, migrate token storage to Keychain.)
//

import Foundation
import SwiftUI
import Combine

@MainActor
final class AuthStore: ObservableObject {
    @Published var accessToken: String?
    @Published var refreshToken: String?
    @Published var email: String?
    @Published var errorMessage: String?
    @Published var isBusy = false

    var isLoggedIn: Bool { accessToken != nil }

    private let accessKey = "access_token"
    private let refreshKey = "refresh_token"
    private let emailKey = "user_email"

    init() {
        accessToken = UserDefaults.standard.string(forKey: accessKey)
        refreshToken = UserDefaults.standard.string(forKey: refreshKey)
        email = UserDefaults.standard.string(forKey: emailKey)
    }

    private var client: APIClient { APIClient(token: accessToken) }
    private var refreshTask: Task<Bool, Never>?

    // Single-flight refresh: rotates the refresh token exactly once even if many
    // requests hit 401 simultaneously. Logs out if the refresh token is invalid.
    func refresh() async -> Bool {
        if let t = refreshTask { return await t.value }
        let t = Task { () -> Bool in
            defer { refreshTask = nil }
            guard let rt = refreshToken else { return false }
            do {
                let tok = try await APIClient().refresh(rt)
                persist(tok, email: email ?? "")
                return true
            } catch {
                logout()
                return false
            }
        }
        refreshTask = t
        return await t.value
    }

    func register(email: String, password: String) async {
        isBusy = true; errorMessage = nil
        defer { isBusy = false }
        do {
            try await client.register(email: email, password: password)
            await login(email: email, password: password)
        } catch {
            errorMessage = error.localizedDescription
        }
    }

    func login(email: String, password: String) async {
        isBusy = true; errorMessage = nil
        defer { isBusy = false }
        do {
            let tok = try await client.login(email: email, password: password)
            persist(tok, email: email)
        } catch {
            errorMessage = error.localizedDescription
        }
    }

    func loginWithGoogle() async {
        isBusy = true; errorMessage = nil
        defer { isBusy = false }
        do {
            let idToken = try await GoogleAuth.signIn()
            let tok = try await APIClient().googleLogin(idToken: idToken)
            // We don't know the email locally; backend owns it. Persist tokens and
            // let the app read profile from the session as features land.
            persist(tok, email: email ?? "")
        } catch GoogleAuthError.cancelled {
            // Silent: the user chose to cancel.
        } catch {
            errorMessage = error.localizedDescription
        }
    }

    func logout() {
        accessToken = nil; refreshToken = nil; email = nil
        UserDefaults.standard.removeObject(forKey: accessKey)
        UserDefaults.standard.removeObject(forKey: refreshKey)
        UserDefaults.standard.removeObject(forKey: emailKey)
    }

    private func persist(_ tok: TokenResponse, email: String) {
        accessToken = tok.accessToken
        refreshToken = tok.refreshToken
        self.email = email
        UserDefaults.standard.set(tok.accessToken, forKey: accessKey)
        UserDefaults.standard.set(tok.refreshToken, forKey: refreshKey)
        UserDefaults.standard.set(email, forKey: emailKey)
    }
}
