//
//  WidgetBridge.swift
//  Publishes the user's top favorites into the shared App Group container so the
//  home-screen widget can render them without any authenticated networking.
//

import Foundation
import WidgetKit

enum WidgetBridge {
    static let appGroup = "group.talhacan.tscforex"
    static let storageKey = "favorites_widget_v1"
    static let widgetKind = "FavoritesWidget"

    /// One favorite row as shown in the widget. Kept in sync with the identical
    /// struct in the widget target (both encode/decode the same JSON shape).
    struct Item: Codable {
        let code: String
        let name: String
        let price: Double
        let changePct: Double
    }

    /// Persists the snapshot and asks WidgetKit to refresh the timeline.
    static func write(_ items: [Item]) {
        guard let defaults = UserDefaults(suiteName: appGroup) else { return }
        guard let data = try? JSONEncoder().encode(items) else { return }
        defaults.set(data, forKey: storageKey)
        WidgetCenter.shared.reloadTimelines(ofKind: widgetKind)
    }
}
