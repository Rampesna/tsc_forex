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
    @Published var displayName: String = ""
    @Published var avatarURL: String = ""
    @Published var errorMessage: String?
    @Published var isBusy = false
    @Published var linkedProviders: [String] = []

    var isLoggedIn: Bool { accessToken != nil }

    private let accessKey = "access_token"
    private let refreshKey = "refresh_token"
    private let emailKey = "user_email"
    private let nameKey = "user_display_name"
    private let avatarKey = "user_avatar_url"

    init() {
        accessToken = UserDefaults.standard.string(forKey: accessKey)
        refreshToken = UserDefaults.standard.string(forKey: refreshKey)
        email = UserDefaults.standard.string(forKey: emailKey)
        displayName = UserDefaults.standard.string(forKey: nameKey) ?? ""
        avatarURL = UserDefaults.standard.string(forKey: avatarKey) ?? ""
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
            await loadProfile()
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
            persist(tok, email: email ?? "")
            await loadProfile()
        } catch GoogleAuthError.cancelled {
            // Silent: the user chose to cancel.
        } catch {
            errorMessage = error.localizedDescription
        }
    }

    func loginWithApple() async {
        isBusy = true; errorMessage = nil
        defer { isBusy = false }
        do {
            let cred = try await AppleAuth.shared.signIn()
            let tok = try await APIClient().appleLogin(idToken: cred.idToken, fullName: cred.fullName)
            persist(tok, email: email ?? "")
            await loadProfile()
        } catch AppleAuthError.cancelled {
            // Silent: user chose to cancel.
        } catch {
            errorMessage = error.localizedDescription
        }
    }

    func loginWithPasskey() async {
        guard #available(iOS 16.0, *) else {
            errorMessage = "Geçiş anahtarı için iOS 16+ gerekli"; return
        }
        isBusy = true; errorMessage = nil
        defer { isBusy = false }
        do {
            let tok = try await PasskeyAuth.shared.login()
            persist(tok, email: email ?? "")
            await loadProfile()
        } catch PasskeyError.cancelled {
            // Silent.
        } catch {
            errorMessage = error.localizedDescription
        }
    }

    /// Registers a new passkey for the signed-in user. Returns true on success.
    func registerPasskey() async -> Bool {
        guard #available(iOS 16.0, *) else {
            errorMessage = "Geçiş anahtarı için iOS 16+ gerekli"; return false
        }
        guard let token = accessToken else { return false }
        do {
            try await PasskeyAuth.shared.register(accessToken: token)
            return true
        } catch PasskeyError.cancelled {
            return false
        } catch {
            errorMessage = error.localizedDescription
            return false
        }
    }

    /// Loads the providers currently linked to this account (Google/Apple/email).
    func loadLinkedProviders() async {
        guard accessToken != nil else { return }
        if let list = try? await APIClient(token: accessToken).linkedProviders() {
            linkedProviders = list
        }
    }

    /// Links a Google account to the signed-in user. Returns true on success.
    func linkGoogle() async -> Bool {
        do {
            let idToken = try await GoogleAuth.signIn()
            linkedProviders = try await APIClient(token: accessToken).linkGoogle(idToken: idToken)
            Haptics.success()
            return true
        } catch GoogleAuthError.cancelled {
            return false
        } catch {
            errorMessage = friendlyLinkError(error); return false
        }
    }

    /// Links an Apple account to the signed-in user. Returns true on success.
    func linkApple() async -> Bool {
        do {
            let cred = try await AppleAuth.shared.signIn()
            linkedProviders = try await APIClient(token: accessToken)
                .linkApple(idToken: cred.idToken, fullName: cred.fullName)
            Haptics.success()
            return true
        } catch AppleAuthError.cancelled {
            return false
        } catch {
            errorMessage = friendlyLinkError(error); return false
        }
    }

    private func friendlyLinkError(_ error: Error) -> String {
        if case APIError.http(409, _) = error {
            return "Bu hesap başka bir kullanıcıya bağlı."
        }
        return error.localizedDescription
    }

    /// Fetches the server-side profile (email, name, avatar) and caches it.
    func loadProfile() async {
        guard accessToken != nil else { return }
        do {
            let p = try await APIClient(token: accessToken).profile()
            applyProfile(email: p.email, name: p.displayName, avatar: p.avatarURL)
        } catch {
            // Non-fatal: keep cached values.
        }
    }

    func updateDisplayName(_ name: String) async {
        do {
            let p = try await APIClient(token: accessToken).updateProfile(displayName: name)
            applyProfile(email: p.email, name: p.displayName, avatar: p.avatarURL)
        } catch {
            errorMessage = error.localizedDescription
        }
    }

    func uploadAvatar(_ data: Data) async {
        do {
            let url = try await APIClient(token: accessToken)
                .uploadAvatar(imageData: data, filename: "avatar.jpg", contentType: "image/jpeg")
            avatarURL = url
            UserDefaults.standard.set(url, forKey: avatarKey)
        } catch {
            errorMessage = error.localizedDescription
        }
    }

    func logout() {
        accessToken = nil; refreshToken = nil; email = nil
        displayName = ""; avatarURL = ""
        [accessKey, refreshKey, emailKey, nameKey, avatarKey].forEach {
            UserDefaults.standard.removeObject(forKey: $0)
        }
    }

    private func applyProfile(email: String, name: String, avatar: String) {
        if !email.isEmpty {
            self.email = email
            UserDefaults.standard.set(email, forKey: emailKey)
        }
        displayName = name
        avatarURL = avatar
        UserDefaults.standard.set(name, forKey: nameKey)
        UserDefaults.standard.set(avatar, forKey: avatarKey)
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
