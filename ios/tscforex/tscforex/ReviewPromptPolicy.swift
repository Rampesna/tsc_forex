import Foundation

/// Records genuinely useful actions and only lets StoreKit request a review at a
/// respectful moment. There is no custom pre-prompt, sentiment gate or reward.
enum ReviewPromptPolicy {
    private static let firstActionKey = "review.firstMeaningfulActionAt"
    private static let actionCountKey = "review.meaningfulActionCount"
    private static let lastAttemptKey = "review.lastAttemptAt"
    private static let lastVersionKey = "review.lastAttemptVersion"

    private static let minimumActions = 3
    private static let minimumExperience: TimeInterval = 3 * 24 * 60 * 60
    private static let retryInterval: TimeInterval = 120 * 24 * 60 * 60

    /// Returns `true` when the system review sheet may be requested. StoreKit
    /// still decides whether a sheet is displayed.
    static func recordMeaningfulAction(
        now: Date = Date(),
        defaults: UserDefaults = .standard
    ) -> Bool {
        if defaults.object(forKey: firstActionKey) == nil {
            defaults.set(now.timeIntervalSince1970, forKey: firstActionKey)
        }

        let actions = defaults.integer(forKey: actionCountKey) + 1
        defaults.set(actions, forKey: actionCountKey)

        let firstAction = Date(
            timeIntervalSince1970: defaults.double(forKey: firstActionKey)
        )
        guard actions >= minimumActions,
              now.timeIntervalSince(firstAction) >= minimumExperience else {
            return false
        }

        let version = Bundle.main.object(
            forInfoDictionaryKey: "CFBundleShortVersionString"
        ) as? String ?? "unknown"
        guard defaults.string(forKey: lastVersionKey) != version else {
            return false
        }

        let lastAttemptTimestamp = defaults.double(forKey: lastAttemptKey)
        if lastAttemptTimestamp > 0 {
            let lastAttempt = Date(timeIntervalSince1970: lastAttemptTimestamp)
            guard now.timeIntervalSince(lastAttempt) >= retryInterval else {
                return false
            }
        }

        defaults.set(now.timeIntervalSince1970, forKey: lastAttemptKey)
        defaults.set(version, forKey: lastVersionKey)
        defaults.set(0, forKey: actionCountKey)
        return true
    }
}
