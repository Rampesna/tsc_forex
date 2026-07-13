//
//  AppConfig.swift
//  Central backend address resolution.
//
//  Release builds always talk to the production API over TLS (https/wss) and
//  expose no server-configuration UI. Debug builds default to the developer's
//  Mac on the LAN and allow overriding the host from the (debug-only) Settings
//  section so the app can be tested against a local backend.
//

import Foundation

enum AppConfig {
    /// Production API host (served behind TLS via the aaPanel reverse proxy).
    static let productionHost = "api.talhacan.com"

    static let defaultsHostKey = "server_host"

    #if DEBUG
    /// Debug default: the developer Mac's Bonjour name so it survives DHCP/IP
    /// changes on the same Wi-Fi. Editable from Settings (debug only).
    static let defaultHost = "Rampsna-Macbook-7.local:8080"
    #else
    static let defaultHost = productionHost
    #endif

    /// Whether the app allows the user to change the server address at runtime.
    /// Only in debug builds — production is locked to the TLS production host.
    static var allowsServerOverride: Bool {
        #if DEBUG
        return true
        #else
        return false
        #endif
    }

    static var host: String {
        get {
            #if DEBUG
            return UserDefaults.standard.string(forKey: defaultsHostKey) ?? defaultHost
            #else
            return productionHost
            #endif
        }
        set { UserDefaults.standard.set(newValue, forKey: defaultsHostKey) }
    }

    /// TLS is used for real domains; plain http/ws only for LAN hosts (dev).
    private static var secure: Bool { host.contains("talhacan.com") }

    static var apiBaseURL: URL {
        URL(string: "\(secure ? "https" : "http")://\(host)/api/v1")!
    }

    static var wsURL: URL {
        URL(string: "\(secure ? "wss" : "ws")://\(host)/ws")!
    }
}
