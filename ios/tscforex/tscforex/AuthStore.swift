//
//  AuthStore.swift
//  Holds auth state and keeps authentication secrets in the system Keychain.
//  Non-sensitive profile presentation fields remain cached in UserDefaults.
//

import Foundation
import SwiftUI
import Combine

@MainActor
final class AuthStore: ObservableObject {
    @Published var accessToken: String?
    @Published var refreshToken: String?
    @Published var userID: String?
    @Published var email: String?
    @Published var displayName: String = ""
    @Published var avatarURL: String = ""
    @Published var errorMessage: String?
    @Published var isBusy = false
    @Published var linkedProviders: [String] = []

    var isLoggedIn: Bool { accessToken != nil }

    private let accessKey = "access_token"
    private let refreshKey = "refresh_token"
    private let userIDKey = "user_id"
    private let emailKey = "user_email"
    private let nameKey = "user_display_name"
    private let avatarKey = "user_avatar_url"

    init() {
        accessToken = Self.secureValue(key: accessKey)
        refreshToken = Self.secureValue(key: refreshKey)
        userID = Self.secureValue(key: userIDKey)
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
                return persist(tok, email: email ?? "")
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
            guard persist(tok, email: email) else { return }
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
            guard persist(tok, email: email ?? "") else { return }
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
            guard persist(tok, email: email ?? "") else { return }
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
            guard persist(tok, email: email ?? "") else { return }
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
        if case APIError.http(409, _, _) = error {
            return "Bu hesap başka bir kullanıcıya bağlı."
        }
        return error.localizedDescription
    }

    /// Fetches the server-side profile (email, name, avatar) and caches it.
    func loadProfile() async {
        guard accessToken != nil else { return }
        do {
            let p = try await APIClient(token: accessToken).profile()
            applyProfile(p)
        } catch {
            // Non-fatal: keep cached values.
        }
    }

    func updateDisplayName(_ name: String) async {
        do {
            let p = try await APIClient(token: accessToken).updateProfile(displayName: name)
            applyProfile(p)
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
        userID = nil
        displayName = ""; avatarURL = ""
        [accessKey, refreshKey, userIDKey].forEach { KeychainStore.delete($0) }
        [accessKey, refreshKey, userIDKey, emailKey, nameKey, avatarKey].forEach {
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

    private func applyProfile(_ profile: ProfileResponse) {
        userID = profile.id
        if KeychainStore.write(profile.id, for: userIDKey) {
            UserDefaults.standard.removeObject(forKey: userIDKey)
        } else {
            UserDefaults.standard.set(profile.id, forKey: userIDKey)
        }
        applyProfile(email: profile.email, name: profile.displayName, avatar: profile.avatarURL)
    }

    @discardableResult
    private func persist(_ tok: TokenResponse, email: String) -> Bool {
        let accessStored = KeychainStore.write(tok.accessToken, for: accessKey)
        let refreshStored = KeychainStore.write(tok.refreshToken, for: refreshKey)
        guard accessStored && refreshStored else {
            [accessKey, refreshKey].forEach {
                KeychainStore.delete($0)
                UserDefaults.standard.removeObject(forKey: $0)
            }
            accessToken = nil
            refreshToken = nil
            self.email = nil
            errorMessage = "Oturum güvenli alana kaydedilemedi. Cihazınızı yeniden başlatıp tekrar deneyin."
            return false
        }
        UserDefaults.standard.removeObject(forKey: accessKey)
        UserDefaults.standard.removeObject(forKey: refreshKey)
        accessToken = tok.accessToken
        refreshToken = tok.refreshToken
        self.email = email
        UserDefaults.standard.set(email, forKey: emailKey)
        return true
    }

    /// Reads secure storage first and migrates legacy token/user-ID defaults
    /// atomically enough to preserve existing signed-in sessions on upgrade.
    private static func secureValue(key: String) -> String? {
        if let value = KeychainStore.read(key), !value.isEmpty { return value }
        guard let legacy = UserDefaults.standard.string(forKey: key), !legacy.isEmpty else { return nil }
        UserDefaults.standard.removeObject(forKey: key)
        guard KeychainStore.write(legacy, for: key) else { return nil }
        return legacy
    }
}
