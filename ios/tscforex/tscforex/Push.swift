//
//  Push.swift
//  Notification pipeline: permission, foreground banners (local notifications),
//  and best-effort APNs device-token capture + backend registration.
//
//  NOTE: True background/killed-app push requires an APNs Auth Key (.p8) on the
//  backend AND a Push-enabled App ID + provisioning profile. Until that is set
//  up, local notifications still surface alerts while the app is running.
//

import UIKit
import UserNotifications
import Combine

@MainActor
final class PushManager: NSObject, ObservableObject, UNUserNotificationCenterDelegate {
    static let shared = PushManager()

    @Published var authorized = false
    @Published var deviceToken: String?
    @Published var lastError: String?

    /// Called with the APNs hex token once the OS delivers it.
    var onToken: ((String) -> Void)?

    func bootstrap() {
        UNUserNotificationCenter.current().delegate = self
    }

    /// Ask for permission (works for local notifications regardless of APNs) and
    /// attempt remote registration (best-effort).
    func requestPermission() {
        UNUserNotificationCenter.current().requestAuthorization(options: [.alert, .sound, .badge]) { granted, err in
            Task { @MainActor in
                self.authorized = granted
                if let err { self.lastError = err.localizedDescription }
                if granted {
                    UIApplication.shared.registerForRemoteNotifications()
                }
            }
        }
    }

    /// Surface a notification banner while the app is running. Attaches the TSC
    /// logo so the banner is clearly branded (the app icon already shows on the
    /// left; this adds the logo thumbnail on the trailing edge).
    func showLocal(title: String, body: String) {
        let content = UNMutableNotificationContent()
        content.title = title
        content.body = body
        content.sound = .default
        if let attachment = Self.logoAttachment() {
            content.attachments = [attachment]
        }
        let req = UNNotificationRequest(identifier: UUID().uuidString, content: content, trigger: nil)
        UNUserNotificationCenter.current().add(req)
    }

    /// Writes the bundled logo to a temp file once and wraps it as a reusable
    /// notification attachment. Returns nil if the image can't be materialised.
    private static var cachedLogoURL: URL?
    private static func logoAttachment() -> UNNotificationAttachment? {
        let url: URL
        if let cached = cachedLogoURL, FileManager.default.fileExists(atPath: cached.path) {
            url = cached
        } else {
            guard let img = UIImage(named: "Logo"), let data = img.pngData() else { return nil }
            let tmp = FileManager.default.temporaryDirectory.appendingPathComponent("tsc-notif-logo.png")
            do { try data.write(to: tmp) } catch { return nil }
            cachedLogoURL = tmp
            url = tmp
        }
        return try? UNNotificationAttachment(identifier: "logo", url: url, options: nil)
    }

    // Show banners even when the app is in the foreground.
    nonisolated func userNotificationCenter(_ center: UNUserNotificationCenter,
                                            willPresent notification: UNNotification) async
        -> UNNotificationPresentationOptions {
        [.banner, .sound, .list]
    }
}

// AppDelegate bridges UIKit remote-notification callbacks into PushManager.
final class AppDelegate: NSObject, UIApplicationDelegate {
    func application(_ application: UIApplication,
                     didFinishLaunchingWithOptions launchOptions: [UIApplication.LaunchOptionsKey: Any]? = nil) -> Bool {
        Task { @MainActor in PushManager.shared.bootstrap() }
        return true
    }

    func application(_ application: UIApplication,
                     didRegisterForRemoteNotificationsWithDeviceToken deviceToken: Data) {
        let token = deviceToken.map { String(format: "%02x", $0) }.joined()
        Task { @MainActor in
            PushManager.shared.deviceToken = token
            PushManager.shared.onToken?(token)
        }
    }

    func application(_ application: UIApplication,
                     didFailToRegisterForRemoteNotificationsWithError error: Error) {
        Task { @MainActor in PushManager.shared.lastError = error.localizedDescription }
    }
}
