import Foundation

/// AI third-party data-sharing consent belongs to an app account, not to the
/// device globally. This prevents one account's approval being reused after a
/// logout/account switch on a shared device.
enum AIResearchConsentStore {
    private static let prefix = "ai_research_disclosure_accepted_v2"

    static func isAccepted(for userID: String?) -> Bool {
        guard let userID, !userID.isEmpty else { return false }
        return UserDefaults.standard.bool(forKey: key(userID))
    }

    static func setAccepted(_ accepted: Bool, for userID: String?) {
        guard let userID, !userID.isEmpty else { return }
        if accepted {
            UserDefaults.standard.set(true, forKey: key(userID))
        } else {
            UserDefaults.standard.removeObject(forKey: key(userID))
        }
    }

    private static func key(_ userID: String) -> String { "\(prefix).\(userID)" }
}
