import SwiftUI
import Combine

enum AppLanguage: String, CaseIterable, Identifiable {
    case turkish = "tr"
    case english = "en"
    case arabic = "ar"
    case german = "de"
    case italian = "it"
    case spanish = "es"

    var id: String { rawValue }

    var nativeName: String {
        switch self {
        case .turkish: return "Türkçe"
        case .english: return "English"
        case .arabic: return "العربية"
        case .german: return "Deutsch"
        case .italian: return "Italiano"
        case .spanish: return "Español"
        }
    }

    var locale: Locale { Locale(identifier: rawValue) }
    var layoutDirection: LayoutDirection { self == .arabic ? .rightToLeft : .leftToRight }

    static func preferred() -> AppLanguage {
        let preferred = Locale.preferredLanguages.first?
            .split(separator: "-")
            .first
            .map(String.init)
        return preferred.flatMap(AppLanguage.init(rawValue:)) ?? .turkish
    }
}

@MainActor
final class LanguageStore: ObservableObject {
    static let storageKey = "app.language.v1"

    @Published private(set) var current: AppLanguage

    init() {
        if let saved = UserDefaults.standard.string(forKey: Self.storageKey),
           let language = AppLanguage(rawValue: saved) {
            current = language
        } else {
            current = .preferred()
        }
    }

    func select(_ language: AppLanguage) {
        guard current != language else { return }
        current = language
        UserDefaults.standard.set(language.rawValue, forKey: Self.storageKey)
    }
}

enum L10n {
    static var language: AppLanguage {
        if let saved = UserDefaults.standard.string(forKey: LanguageStore.storageKey),
           let language = AppLanguage(rawValue: saved) {
            return language
        }
        return .preferred()
    }

    static func text(_ key: String) -> String {
        guard language != .turkish,
              let path = Bundle.main.path(forResource: language.rawValue, ofType: "lproj"),
              let bundle = Bundle(path: path)
        else {
            return key
        }
        return bundle.localizedString(forKey: key, value: key, table: nil)
    }

    static func format(_ key: String, _ arguments: CVarArg...) -> String {
        String(format: text(key), locale: language.locale, arguments: arguments)
    }
}

extension String {
    var localized: String { L10n.text(self) }
}
