import CryptoKit
import DeviceCheck
import Foundation

struct DeviceAttestationChallenge: Codable {
    let challengeID: String
    let challenge: String
    let clientData: String
    let expiresAt: Date
    let needsKeyAttestation: Bool

    enum CodingKeys: String, CodingKey {
        case challenge
        case challengeID = "challenge_id"
        case clientData = "client_data"
        case expiresAt = "expires_at"
        case needsKeyAttestation = "needs_key_attestation"
    }
}

struct AppAttestProofPayload {
    let keyID: String
    let challengeID: String
    let assertion: String
    let deviceCheckToken: String
}

/// Produces a proof bound by the backend to the current user, installation,
/// request ID and instrument code. The App Attest private key never leaves the
/// Secure Enclave and the key identifier is stored only in Keychain.
@MainActor
final class AppAttestClient {
    static let shared = AppAttestClient()

    private let service = DCAppAttestService.shared
    private let keychainAccount = "app-attest-key-id-v1"

    private init() {}

    func proof(token: String, code: String, requestID: String) async throws -> AppAttestProofPayload {
        do {
            return try await makeProof(token: token, code: code, requestID: requestID)
        } catch {
            guard Self.isInvalidKey(error) else { throw error }
            // App Attest keys can be invalidated after restore or Secure Enclave
            // changes. Remove only the unusable key ID and perform one fresh
            // registration attempt; a second failure is surfaced unchanged.
            KeychainStore.delete(keychainAccount)
            return try await makeProof(token: token, code: code, requestID: requestID)
        }
    }

    private func makeProof(
        token: String, code: String, requestID: String
    ) async throws -> AppAttestProofPayload {
        guard service.isSupported else { throw AppAttestClientError.unsupportedDevice }

        var keyID = KeychainStore.read(keychainAccount)
        var challenge = try await APIClient(token: token).deviceAttestationChallenge(
            deviceID: DeviceIdentity.identifier,
            keyID: keyID,
            purpose: "free_ai",
            requestID: requestID,
            code: code
        )

        if challenge.needsKeyAttestation {
            if keyID == nil {
                keyID = try await generateKey()
                guard let keyID, KeychainStore.write(keyID, for: keychainAccount) else {
                    throw AppAttestClientError.secureStorageFailed
                }
            }
            guard let keyID else { throw AppAttestClientError.keyUnavailable }
            let clientHash = try clientDataHash(challenge.clientData)
            let attestation = try await attestKey(keyID, clientDataHash: clientHash)
            try await APIClient(token: token).registerDeviceAttestationKey(
                deviceID: DeviceIdentity.identifier,
                keyID: keyID,
                challengeID: challenge.challengeID,
                attestationObject: attestation.base64EncodedString()
            )

            // Registration consumes its challenge. Assertions are always made
            // against a fresh one-use challenge for this exact research call.
            challenge = try await APIClient(token: token).deviceAttestationChallenge(
                deviceID: DeviceIdentity.identifier,
                keyID: keyID,
                purpose: "free_ai",
                requestID: requestID,
                code: code
            )
        }

        guard let keyID, !challenge.needsKeyAttestation else {
            throw AppAttestClientError.registrationNotAccepted
        }
        // Freeze the final one-use challenge before starting concurrent Apple
        // operations; capturing the mutable `challenge` variable would be a
        // Swift 6 data-race warning.
        let assertionChallengeID = challenge.challengeID
        let assertionClientDataHash = try clientDataHash(challenge.clientData)
        async let assertionData = generateAssertion(
            keyID, clientDataHash: assertionClientDataHash
        )
        async let deviceCheckTokenData = generateDeviceCheckToken()
        let (assertion, deviceCheckToken) = try await (assertionData, deviceCheckTokenData)
        return AppAttestProofPayload(
            keyID: keyID,
            challengeID: assertionChallengeID,
            assertion: assertion.base64EncodedString(),
            deviceCheckToken: deviceCheckToken.base64EncodedString()
        )
    }

    private static func isInvalidKey(_ error: Error) -> Bool {
        let nsError = error as NSError
        return nsError.domain == DCErrorDomain && nsError.code == DCError.invalidKey.rawValue
    }

    private func clientDataHash(_ encoded: String) throws -> Data {
        guard let clientData = Data.fromBase64URL(encoded), !clientData.isEmpty else {
            throw AppAttestClientError.invalidChallenge
        }
        return Data(SHA256.hash(data: clientData))
    }

    private func generateKey() async throws -> String {
        try await withCheckedThrowingContinuation { continuation in
            service.generateKey { keyID, error in
                if let keyID { continuation.resume(returning: keyID) }
                else { continuation.resume(throwing: error ?? AppAttestClientError.keyUnavailable) }
            }
        }
    }

    private func attestKey(_ keyID: String, clientDataHash: Data) async throws -> Data {
        try await withCheckedThrowingContinuation { continuation in
            service.attestKey(keyID, clientDataHash: clientDataHash) { object, error in
                if let object { continuation.resume(returning: object) }
                else { continuation.resume(throwing: error ?? AppAttestClientError.attestationFailed) }
            }
        }
    }

    private func generateAssertion(_ keyID: String, clientDataHash: Data) async throws -> Data {
        try await withCheckedThrowingContinuation { continuation in
            service.generateAssertion(keyID, clientDataHash: clientDataHash) { assertion, error in
                if let assertion { continuation.resume(returning: assertion) }
                else { continuation.resume(throwing: error ?? AppAttestClientError.assertionFailed) }
            }
        }
    }

    private func generateDeviceCheckToken() async throws -> Data {
        let device = DCDevice.current
        guard device.isSupported else { throw AppAttestClientError.deviceCheckUnsupported }
        return try await withCheckedThrowingContinuation { continuation in
            device.generateToken { token, error in
                if let token { continuation.resume(returning: token) }
                else { continuation.resume(throwing: error ?? AppAttestClientError.deviceCheckFailed) }
            }
        }
    }
}

enum AppAttestClientError: LocalizedError {
    case unsupportedDevice
    case secureStorageFailed
    case keyUnavailable
    case invalidChallenge
    case attestationFailed
    case assertionFailed
    case registrationNotAccepted
    case deviceCheckUnsupported
    case deviceCheckFailed

    var errorDescription: String? {
        switch self {
        case .unsupportedDevice:
            return "Bu cihaz ücretsiz AI araştırması için gereken Apple güvenlik doğrulamasını desteklemiyor. Desteklenen gerçek bir iPhone veya iPad kullanın ya da ücretli bir planla devam edin."
        case .secureStorageFailed:
            return "Cihaz doğrulama anahtarı güvenli biçimde saklanamadı. Cihazınızı yeniden başlatıp tekrar deneyin."
        case .keyUnavailable:
            return "Apple cihaz doğrulama anahtarı oluşturulamadı. Bir süre sonra tekrar deneyin."
        case .invalidChallenge:
            return "Cihaz doğrulama isteği geçersizdi. Araştırmayı yeniden başlatın."
        case .attestationFailed:
            return "Apple cihaz kaydı doğrulanamadı. Ağ bağlantınızı kontrol edip tekrar deneyin."
        case .assertionFailed:
            return "Bu araştırma için cihaz doğrulaması üretilemedi. Araştırmayı yeniden başlatın."
        case .registrationNotAccepted:
            return "Cihaz güvenlik kaydı sunucu tarafından doğrulanmadı. Destek ile iletişime geçin."
        case .deviceCheckUnsupported:
            return "Bu cihaz tek kullanımlık ücretsiz araştırma hakkı için gereken Apple DeviceCheck doğrulamasını desteklemiyor. Ücretli bir planla devam edebilirsiniz."
        case .deviceCheckFailed:
            return "Fiziksel cihaz doğrulaması üretilemedi. Ağ bağlantınızı kontrol edip tekrar deneyin."
        }
    }
}
