import CryptoKit
import Foundation

/// Persists idempotency state so a network retry or app restart resumes the
/// same server operation instead of spending quota twice.
struct PendingOperation: Codable, Equatable {
    let requestID: String
    var jobID: String?
}

enum PendingOperationStore {
    private static let prefix = "pending-operation-v1."

    static func loadOrCreate(userID: String, operation: String, resource: String) -> PendingOperation {
        if let existing = load(userID: userID, operation: operation, resource: resource) {
            return existing
        }
        let pending = PendingOperation(requestID: UUID().uuidString.lowercased(), jobID: nil)
        save(pending, userID: userID, operation: operation, resource: resource)
        return pending
    }

    static func updateJobID(
        _ jobID: String, userID: String, operation: String, resource: String
    ) {
        var pending = loadOrCreate(userID: userID, operation: operation, resource: resource)
        pending.jobID = jobID
        save(pending, userID: userID, operation: operation, resource: resource)
    }

    static func clear(userID: String, operation: String, resource: String) {
        UserDefaults.standard.removeObject(
            forKey: key(userID: userID, operation: operation, resource: resource)
        )
    }

    static func clearAll(userID: String) {
        let accountPrefix = prefix + userID.lowercased() + "."
        for key in UserDefaults.standard.dictionaryRepresentation().keys where key.hasPrefix(accountPrefix) {
            UserDefaults.standard.removeObject(forKey: key)
        }
    }

    private static func load(
        userID: String, operation: String, resource: String
    ) -> PendingOperation? {
        guard let data = UserDefaults.standard.data(
            forKey: key(userID: userID, operation: operation, resource: resource)
        ) else { return nil }
        return try? JSONDecoder().decode(PendingOperation.self, from: data)
    }

    private static func save(
        _ value: PendingOperation, userID: String, operation: String, resource: String
    ) {
        guard let data = try? JSONEncoder().encode(value) else { return }
        UserDefaults.standard.set(
            data, forKey: key(userID: userID, operation: operation, resource: resource)
        )
    }

    private static func key(userID: String, operation: String, resource: String) -> String {
        let scope = Data((operation + "|" + resource).utf8)
        let digest = SHA256.hash(data: scope).map { String(format: "%02x", $0) }.joined()
        return prefix + userID.lowercased() + "." + digest
    }
}
