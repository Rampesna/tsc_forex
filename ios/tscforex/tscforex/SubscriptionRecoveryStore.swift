import Foundation
import Security

/// Account-scoped proof used only to restore an App Store transaction after
/// in-app account deletion. It is intentionally ThisDeviceOnly through
/// KeychainStore and is selected by StoreKit's original appAccountToken.
enum SubscriptionRecoveryStore {
    private static let prefix = "subscription-recovery-"

    static func existingToken(for appAccountToken: UUID) -> String? {
        KeychainStore.read(account(for: appAccountToken))
    }

    static func token(for appAccountToken: UUID) throws -> String {
        if let existing = existingToken(for: appAccountToken), !existing.isEmpty {
            return existing
        }
        var random = [UInt8](repeating: 0, count: 32)
        let status = random.withUnsafeMutableBytes { buffer -> OSStatus in
            guard let baseAddress = buffer.baseAddress else { return errSecAllocate }
            return SecRandomCopyBytes(kSecRandomDefault, buffer.count, baseAddress)
        }
        guard status == errSecSuccess else {
            throw SubscriptionRecoveryStoreError.randomGenerationFailed
        }
        let value = Data(random).base64EncodedString()
            .replacingOccurrences(of: "+", with: "-")
            .replacingOccurrences(of: "/", with: "_")
            .replacingOccurrences(of: "=", with: "")
        guard KeychainStore.write(value, for: account(for: appAccountToken)) else {
            throw SubscriptionRecoveryStoreError.secureStorageFailed
        }
        return value
    }

    private static func account(for token: UUID) -> String {
        prefix + token.uuidString.lowercased()
    }
}

enum SubscriptionRecoveryStoreError: LocalizedError {
    case randomGenerationFailed
    case secureStorageFailed

    var errorDescription: String? {
        switch self {
        case .randomGenerationFailed:
            return "Abonelik kurtarma anahtarı oluşturulamadı. Lütfen tekrar deneyin."
        case .secureStorageFailed:
            return "Abonelik kurtarma anahtarı güvenli alana kaydedilemedi. Cihazınızı yeniden başlatıp tekrar deneyin."
        }
    }
}
