import Foundation
import Combine
import StoreKit
import UIKit

struct PlanLimits: Codable {
    let aiResearch: Int
    let activeAlerts: Int
    let watchlist: Int
    let portfolios: Int
    let portfolioHealth: Int
    let exports: Int
    let researchArchiveDays: Int

    enum CodingKeys: String, CodingKey {
        case watchlist, portfolios, exports
        case aiResearch = "ai_research"
        case activeAlerts = "active_alerts"
        case portfolioHealth = "portfolio_health"
        case researchArchiveDays = "research_archive_days"
    }
}

struct SubscriptionPlan: Codable {
    let code: String
    let name: String
    let productID: String?
    let priceHintTRY: Double
    let limits: PlanLimits

    enum CodingKeys: String, CodingKey {
        case code, name, limits
        case productID = "product_id"
        case priceHintTRY = "price_hint_try"
    }
}

struct FeatureQuota: Codable {
    let limit: Int
    let used: Int
    let remaining: Int
    let periodEnd: Date?

    enum CodingKeys: String, CodingKey {
        case limit, used, remaining
        case periodEnd = "period_end"
    }
}

struct BillingEntitlement: Codable {
    let productID: String
    let status: String
    let environment: String
    let expiresAt: Date

    enum CodingKeys: String, CodingKey {
        case status, environment
        case productID = "product_id"
        case expiresAt = "expires_at"
    }
}

struct BillingStatus: Codable {
    let plan: SubscriptionPlan
    let entitlement: BillingEntitlement?
    let usage: [String: FeatureQuota]
    let freeResearchAvailable: Bool
    let purchaseChannel: String
    let manageURL: String

    enum CodingKeys: String, CodingKey {
        case plan, entitlement, usage
        case freeResearchAvailable = "free_research_available"
        case purchaseChannel = "purchase_channel"
        case manageURL = "manage_url"
    }
}

struct BillingFreeTrial: Codable {
    let channel: String
    let ready: Bool
}

struct BillingPlansResponse: Codable {
    let purchaseChannel: String
    let plans: [SubscriptionPlan]
    let freeTrial: BillingFreeTrial

    enum CodingKeys: String, CodingKey {
        case plans
        case purchaseChannel = "purchase_channel"
        case freeTrial = "free_trial"
    }
}

@MainActor
final class BillingStore: ObservableObject {
    /// Bootstrap identifiers let StoreKit transaction updates be delivered
    /// before the public catalog request completes. The backend catalog remains
    /// authoritative for products offered and every plan limit displayed.
    private static let bootstrapProductIDs = [
        "talhacan.tscforex.plus.monthly",
        "talhacan.tscforex.pro.monthly",
        "talhacan.tscforex.expert.monthly",
        "talhacan.tscforex.elite.monthly",
    ]

    @Published private(set) var products: [Product] = []
    @Published private(set) var purchasedProductIDs: Set<String> = []
    @Published private(set) var serverStatus: BillingStatus?
    @Published private(set) var catalogPlans: [SubscriptionPlan] = []
    @Published private(set) var freeTrialReady = false
    @Published private(set) var catalogLoaded = false
    @Published private(set) var isLoading = false
    @Published var message: String?

    private var updates: Task<Void, Never>?
    private var syncToken: String?

    init() {
        updates = observeTransactions()
        Task { await refresh() }
    }

    deinit { updates?.cancel() }

    var hasActiveSubscription: Bool {
        serverStatus?.entitlement?.status == "active"
    }

    var aiRemaining: Int? { serverStatus?.usage["ai_research"]?.remaining }

    func isCurrentPlan(_ productID: String) -> Bool {
        hasActiveSubscription && serverStatus?.entitlement?.productID == productID
    }

    /// Billing state is account-scoped. Drop the copied access token and every
    /// user-specific entitlement snapshot as soon as the app account logs out.
    func clearSession() {
        syncToken = nil
        serverStatus = nil
        purchasedProductIDs = []
        message = nil
    }

    func limits(for productID: String) -> PlanLimits? {
        catalogPlans.first(where: { $0.productID == productID })?.limits
    }

    func plan(for productID: String) -> SubscriptionPlan? {
        catalogPlans.first(where: { $0.productID == productID })
    }

    func refresh(token: String? = nil) async {
        if let token { syncToken = token }
        isLoading = true
        defer { isLoading = false }
        do {
            let catalog = try await APIClient().billingPlans()
            catalogPlans = catalog.plans.filter { plan in
                guard let productID = plan.productID else { return false }
                return !productID.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
            }
            freeTrialReady = catalog.freeTrial.ready
            catalogLoaded = true
        } catch {
            catalogLoaded = false
            freeTrialReady = false
            message = "Abonelik kataloğu şu anda doğrulanamadı. Lütfen daha sonra tekrar deneyin."
        }

        let productIDs = catalogPlans.compactMap(\.productID)
        do {
            products = productIDs.isEmpty ? [] : try await Product.products(for: productIDs)
                .sorted { productRank($0.id) < productRank($1.id) }
        } catch {
            message = "Abonelik seçenekleri şu anda yüklenemedi. Lütfen daha sonra tekrar deneyin."
        }

        if let token {
            let syncError = await syncCurrentEntitlements(token: token)
            do { serverStatus = try await APIClient(token: token).billingStatus() }
            catch {
                if message == nil { message = "Abonelik durumu sunucudan doğrulanamadı: \(error.localizedDescription)" }
            }
            reconcilePurchasedProductIDs()
            if let syncError {
                message = "App Store aboneliği sunucuyla doğrulanamadı; işlem yeniden denenecek: \(syncError.localizedDescription)"
            }
        } else {
            await refreshLocalEntitlements()
        }
    }

    func purchase(_ product: Product, token: String?, userID: String?) async {
        isLoading = true
        defer { isLoading = false }
        do {
            guard let token, !token.isEmpty else { throw StoreError.missingSession }
            guard let userID, let accountToken = UUID(uuidString: userID) else {
                throw StoreError.invalidAccountID
            }
            _ = try SubscriptionRecoveryStore.token(for: accountToken)
            syncToken = token
            let result = try await product.purchase(options: [.appAccountToken(accountToken)])
            switch result {
            case .success(let verification):
                let transaction = try verified(verification)
                let status = try await sync(
                    transaction: transaction,
                    signedTransaction: verification.jwsRepresentation,
                    token: token
                )
                serverStatus = status
                await transaction.finish()
                reconcilePurchasedProductIDs()
                message = "Aboneliğiniz etkinleştirildi."
            case .pending:
                message = "Satın alma onay bekliyor. Onaylandıktan sonra otomatik etkinleşir."
            case .userCancelled:
                break
            @unknown default:
                message = "Satın alma tamamlanamadı."
            }
        } catch {
            message = "Satın alma tamamlanamadı: \(error.localizedDescription)"
        }
    }

    func restore(token: String?) async {
        isLoading = true
        defer { isLoading = false }
        do {
            guard let token, !token.isEmpty else { throw StoreError.missingSession }
            syncToken = token
            try await AppStore.sync()
            if let syncError = await syncCurrentEntitlements(token: token) {
                throw syncError
            }
            serverStatus = try await APIClient(token: token).billingStatus()
            reconcilePurchasedProductIDs()
            message = hasActiveSubscription
                ? "Satın almalarınız doğrulandı ve aboneliğiniz etkin."
                : "Bu hesap için etkin abonelik bulunamadı."
        } catch {
            message = "Satın almalar geri yüklenemedi: \(error.localizedDescription)"
        }
    }

    /// Uses the appAccountToken of the entitlement that was synchronized with
    /// this account. This matters after a legitimate tombstone restore, where
    /// the transaction's original token can differ from the new user UUID.
    func recoveryTokenForAccountDeletion(userID: String) async throws -> String {
        if let productID = serverStatus?.entitlement?.productID {
            for await result in Transaction.currentEntitlements {
                guard case .verified(let transaction) = result,
                      transaction.revocationDate == nil,
                      transaction.productID == productID,
                      let accountToken = transaction.appAccountToken else { continue }
                return try SubscriptionRecoveryStore.token(for: accountToken)
            }
        }
        guard let accountToken = UUID(uuidString: userID) else {
            throw StoreError.invalidAccountID
        }
        return try SubscriptionRecoveryStore.token(for: accountToken)
    }

    func manageSubscriptions() async {
        guard let scene = UIApplication.shared.connectedScenes
            .compactMap({ $0 as? UIWindowScene }).first(where: { $0.activationState == .foregroundActive })
        else {
            message = "Abonelik yönetimi şu anda açılamıyor."
            return
        }
        do { try await AppStore.showManageSubscriptions(in: scene) }
        catch { message = "Abonelik yönetimi açılamadı: \(error.localizedDescription)" }
    }

    private func refreshLocalEntitlements() async {
        var active: Set<String> = []
        for await result in Transaction.currentEntitlements {
            guard case .verified(let transaction) = result,
                  transaction.revocationDate == nil,
                  accepts(transaction.productID) else { continue }
            active.insert(transaction.productID)
        }
        purchasedProductIDs = active
    }

    /// Reconciles every current verified App Store entitlement with the backend.
    /// A transaction remains unfinished when server verification fails so
    /// StoreKit can redeliver it after connectivity/account recovery.
    private func syncCurrentEntitlements(token: String) async -> Error? {
        var firstError: Error?
        for await result in Transaction.currentEntitlements {
            guard case .verified(let transaction) = result,
                  transaction.revocationDate == nil,
                  accepts(transaction.productID) else { continue }
            do {
                serverStatus = try await sync(
                    transaction: transaction,
                    signedTransaction: result.jwsRepresentation,
                    token: token
                )
                await transaction.finish()
            } catch {
                if firstError == nil { firstError = error }
            }
        }
        reconcilePurchasedProductIDs()
        return firstError
    }

    private func observeTransactions() -> Task<Void, Never> {
        Task { [weak self] in
            for await result in Transaction.updates {
                guard case .verified(let transaction) = result,
                      self?.accepts(transaction.productID) == true else { continue }
                do {
                    guard let self else { return }
                    self.serverStatus = try await self.sync(
                        transaction: transaction,
                        signedTransaction: result.jwsRepresentation,
                        token: self.syncToken
                    )
                    await transaction.finish()
                    self.reconcilePurchasedProductIDs()
                } catch {
                    self?.message = "App Store işlemi sunucuyla doğrulanamadı; otomatik olarak yeniden denenecek: \(error.localizedDescription)"
                }
            }
        }
    }

    private func sync(transaction: Transaction, signedTransaction: String,
                      token: String?) async throws -> BillingStatus {
        guard let token, !token.isEmpty else { throw StoreError.missingSession }
        guard let appAccountToken = transaction.appAccountToken else {
            throw StoreError.missingAccountToken
        }
        let recoveryToken = try SubscriptionRecoveryStore.token(for: appAccountToken)
        return try await APIClient(token: token).syncApplePurchase(
            productID: transaction.productID,
            transactionID: String(transaction.id),
            originalTransactionID: String(transaction.originalID),
            signedTransaction: signedTransaction,
            deviceID: DeviceIdentity.identifier,
            subscriptionRecoveryToken: recoveryToken
        )
    }

    private func reconcilePurchasedProductIDs() {
        guard hasActiveSubscription, let productID = serverStatus?.entitlement?.productID else {
            purchasedProductIDs = []
            return
        }
        purchasedProductIDs = [productID]
    }

    private func verified<T>(_ result: VerificationResult<T>) throws -> T {
        switch result {
        case .verified(let value): return value
        case .unverified: throw StoreError.failedVerification
        }
    }

    private func productRank(_ id: String) -> Int {
        catalogPlans.firstIndex(where: { $0.productID == id }) ?? .max
    }

    private func accepts(_ productID: String) -> Bool {
        let catalogIDs = Set(catalogPlans.compactMap(\.productID))
        return catalogIDs.contains(productID) ||
            (!catalogLoaded && Self.bootstrapProductIDs.contains(productID))
    }

    private enum StoreError: LocalizedError {
        case failedVerification
        case missingSession
        case invalidAccountID
        case missingAccountToken
        var errorDescription: String? {
            switch self {
            case .failedVerification: return "Satın alma App Store tarafından doğrulanamadı."
            case .missingSession: return "Satın alma için hesabınızla yeniden giriş yapın."
            case .invalidAccountID: return "Hesap kimliği doğrulanamadı; profilinizi yenileyip tekrar deneyin."
            case .missingAccountToken:
                return "Bu eski App Store işlemi güvenli bir uygulama hesabına bağlı değil. Destek ile iletişime geçin."
            }
        }
    }
}
