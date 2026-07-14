//
//  AppConfig.swift
//  Central backend address resolution.
//
//  The app always talks to the production API over TLS (https/wss). There is no
//  runtime server-configuration UI — this is a shipped product, not a dev tool.
//

import Foundation

enum AppConfig {
    /// Production API host (served behind TLS via the aaPanel reverse proxy).
    static let productionHost = "tsc-api.talhacan.com"

    static let defaultHost = productionHost

    /// The server address is fixed to production and cannot be changed at runtime.
    static var allowsServerOverride: Bool { false }

    static var host: String {
        get { productionHost }
        set { _ = newValue } // no-op: production host is fixed.
    }

    static var apiBaseURL: URL {
        URL(string: "https://\(productionHost)/api/v1")!
    }

    static var wsURL: URL {
        URL(string: "wss://\(productionHost)/ws")!
    }

    /// App marketing version + build for the Settings "About" row.
    static var appVersion: String {
        let v = Bundle.main.infoDictionary?["CFBundleShortVersionString"] as? String ?? "1.0"
        let b = Bundle.main.infoDictionary?["CFBundleVersion"] as? String ?? "1"
        return "\(v) (\(b))"
    }
}
