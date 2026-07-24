package com.example.tsc.forex.store

import android.app.Activity
import android.app.Application
import android.util.Log
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import com.android.billingclient.api.AcknowledgePurchaseParams
import com.android.billingclient.api.BillingClient
import com.android.billingclient.api.BillingClientStateListener
import com.android.billingclient.api.BillingFlowParams
import com.android.billingclient.api.BillingResult
import com.android.billingclient.api.PendingPurchasesParams
import com.android.billingclient.api.ProductDetails
import com.android.billingclient.api.Purchase
import com.android.billingclient.api.PurchasesUpdatedListener
import com.android.billingclient.api.QueryProductDetailsParams
import com.android.billingclient.api.QueryPurchasesParams
import com.android.billingclient.api.acknowledgePurchase
import com.android.billingclient.api.queryProductDetails
import com.android.billingclient.api.queryPurchasesAsync
import com.example.tsc.forex.data.ApiClient
import com.example.tsc.forex.data.BillingStatus
import com.example.tsc.forex.data.DeviceIdentity
import com.example.tsc.forex.data.Prefs
import com.example.tsc.forex.data.SubscriptionPlan
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.launch
import kotlinx.coroutines.sync.Mutex
import kotlinx.coroutines.sync.withLock
import kotlinx.coroutines.withContext

/**
 * Google Play Billing + server entitlement sync.
 *
 * Purchase tokens are verified via `POST /me/billing/google/sync`. Apple
 * StoreKit on iOS is a separate path and is never touched from here.
 */
class BillingStore(
    app: Application,
    private val prefs: Prefs,
    private val auth: AuthStore,
) : PurchasesUpdatedListener {
    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.Main)
    private val syncMutex = Mutex()
    private val packageName: String = app.packageName

    private val billingClient: BillingClient = BillingClient.newBuilder(app)
        .setListener(this)
        .enablePendingPurchases(
            PendingPurchasesParams.newBuilder().enableOneTimeProducts().build(),
        )
        .build()

    var catalogPlans by mutableStateOf<List<SubscriptionPlan>>(emptyList())
        private set
    var freeTrialReady by mutableStateOf(false)
        private set
    var catalogLoaded by mutableStateOf(false)
        private set
    var catalogError by mutableStateOf<String?>(null)
    var serverStatus by mutableStateOf<BillingStatus?>(null)
        private set
    var isLoading by mutableStateOf(false)
        private set
    var purchasing by mutableStateOf(false)
        private set
    var message by mutableStateOf<String?>(null)
    var playProductDetails by mutableStateOf<Map<String, ProductDetails>>(emptyMap())
        private set
    var billingReady by mutableStateOf(false)
        private set

    val hasActiveSubscription: Boolean
        get() = serverStatus?.hasActiveSubscription == true

    val aiRemaining: Int?
        get() = serverStatus?.remaining("ai_research")

    init {
        connectBilling()
    }

    fun clearSession() {
        serverStatus = null
        catalogError = null
        message = null
    }

    fun refresh() {
        scope.launch { refreshNow() }
    }

    suspend fun refreshNow() {
        isLoading = true
        catalogError = null
        try {
            val catalog = ApiClient().billingPlans()
            catalogPlans = catalog.plans.filter { it.productId.isNotBlank() || it.code == "free" }
            freeTrialReady = catalog.freeTrialReady
            catalogLoaded = true
            ensureConnected()
            refreshPlayProducts()
        } catch (_: Exception) {
            catalogLoaded = false
            freeTrialReady = false
            catalogError = "Abonelik kataloğu şu anda doğrulanamadı. Lütfen tekrar deneyin."
        }
        if (auth.isLoggedIn) {
            try {
                serverStatus = auth.newApiClient().billingStatus(DeviceIdentity.id(prefs))
            } catch (_: Exception) {
                if (catalogError == null) {
                    catalogError = "Abonelik durumu sunucudan doğrulanamadı. Lütfen tekrar deneyin."
                }
            }
        }
        isLoading = false
    }

    fun limits(forProductId: String) =
        catalogPlans.firstOrNull { it.productId == forProductId }?.limits

    fun plan(forProductId: String) =
        catalogPlans.firstOrNull { it.productId == forProductId }

    fun isCurrentPlan(productId: String): Boolean =
        hasActiveSubscription && serverStatus?.entitlement?.productId == productId

    fun formattedPlayPrice(productId: String): String? {
        val details = playProductDetails[productId] ?: return null
        val offer = details.subscriptionOfferDetails?.firstOrNull() ?: return null
        return offer.pricingPhases.pricingPhaseList.firstOrNull()?.formattedPrice
    }

    fun purchase(activity: Activity, productId: String) {
        scope.launch {
            purchasing = true
            message = null
            try {
                if (!auth.isLoggedIn) {
                    message = "Satın alma için önce giriş yapın."
                    return@launch
                }
                val userId = auth.ensureUserId()
                if (userId.isNullOrBlank()) {
                    message = "Hesap kimliği doğrulanamadı. Lütfen yeniden giriş yapın."
                    return@launch
                }
                ensureConnected()
                if (!billingReady) {
                    message = "Google Play Billing şu anda kullanılamıyor."
                    return@launch
                }
                var details = playProductDetails[productId]
                if (details == null) {
                    refreshPlayProducts()
                    details = playProductDetails[productId]
                }
                if (details == null) {
                    message = "Ürün Play’de bulunamadı. Abonelik ürünlerinin yayında olduğundan emin olun."
                    return@launch
                }
                val offerToken = details.subscriptionOfferDetails?.firstOrNull()?.offerToken
                if (offerToken.isNullOrBlank()) {
                    message = "Bu plan için Play teklifi bulunamadı."
                    return@launch
                }
                val params = BillingFlowParams.newBuilder()
                    .setProductDetailsParamsList(
                        listOf(
                            BillingFlowParams.ProductDetailsParams.newBuilder()
                                .setProductDetails(details)
                                .setOfferToken(offerToken)
                                .build(),
                        ),
                    )
                    .setObfuscatedAccountId(userId.take(64))
                    .build()
                val result = billingClient.launchBillingFlow(activity, params)
                if (result.responseCode != BillingClient.BillingResponseCode.OK) {
                    message = playBillingMessage(result)
                }
            } catch (e: Exception) {
                message = "Satın alma başlatılamadı: ${e.localizedMessage ?: "bilinmeyen hata"}"
            } finally {
                // Keep purchasing=true until PurchasesUpdatedListener finishes
                // unless launch itself failed.
                if (message != null) purchasing = false
            }
        }
    }

    fun restore() {
        scope.launch {
            isLoading = true
            message = null
            try {
                if (!auth.isLoggedIn) {
                    message = "Geri yükleme için önce giriş yapın."
                    return@launch
                }
                ensureConnected()
                val purchases = queryOwnedSubscriptions()
                if (purchases.isEmpty()) {
                    serverStatus = auth.newApiClient().billingStatus(DeviceIdentity.id(prefs))
                    message = if (hasActiveSubscription) {
                        "Aboneliğiniz sunucuda etkin."
                    } else {
                        "Bu Google hesabında geri yüklenecek abonelik bulunamadı."
                    }
                    return@launch
                }
                for (purchase in purchases) {
                    syncAndAcknowledge(purchase)
                }
                message = if (hasActiveSubscription) {
                    "Satın almalarınız doğrulandı ve aboneliğiniz etkin."
                } else {
                    "Satın almalar doğrulanamadı. Lütfen tekrar deneyin."
                }
            } catch (e: Exception) {
                message = "Satın almalar geri yüklenemedi: ${e.localizedMessage ?: "bilinmeyen hata"}"
            } finally {
                isLoading = false
            }
        }
    }

    override fun onPurchasesUpdated(result: BillingResult, purchases: MutableList<Purchase>?) {
        scope.launch {
            try {
                when (result.responseCode) {
                    BillingClient.BillingResponseCode.OK -> {
                        if (purchases.isNullOrEmpty()) {
                            message = "Satın alma tamamlanamadı."
                            return@launch
                        }
                        for (purchase in purchases) {
                            syncAndAcknowledge(purchase)
                        }
                        message = "Aboneliğiniz etkinleştirildi."
                    }
                    BillingClient.BillingResponseCode.USER_CANCELED -> Unit
                    BillingClient.BillingResponseCode.ITEM_ALREADY_OWNED -> {
                        restore()
                    }
                    else -> message = playBillingMessage(result)
                }
            } finally {
                purchasing = false
            }
        }
    }

    private fun connectBilling() {
        if (billingClient.isReady) {
            billingReady = true
            return
        }
        billingClient.startConnection(object : BillingClientStateListener {
            override fun onBillingSetupFinished(result: BillingResult) {
                billingReady = result.responseCode == BillingClient.BillingResponseCode.OK
                if (billingReady) {
                    scope.launch {
                        refreshPlayProducts()
                        // Catch unfinished purchases after process death.
                        runCatching { queryOwnedSubscriptions().forEach { syncAndAcknowledge(it) } }
                    }
                } else {
                    Log.w(TAG, "Billing setup failed: ${result.debugMessage}")
                }
            }

            override fun onBillingServiceDisconnected() {
                billingReady = false
            }
        })
    }

    private suspend fun ensureConnected() {
        if (billingClient.isReady) {
            billingReady = true
            return
        }
        withContext(Dispatchers.Main) { connectBilling() }
        // Brief wait loop for connection; Play callbacks are async.
        repeat(20) {
            if (billingClient.isReady) {
                billingReady = true
                return
            }
            kotlinx.coroutines.delay(100)
        }
    }

    private suspend fun refreshPlayProducts() {
        val ids = catalogPlans.map { it.productId }.filter { it.isNotBlank() }.distinct()
        if (ids.isEmpty() || !billingClient.isReady) return
        val params = QueryProductDetailsParams.newBuilder()
            .setProductList(
                ids.map { id ->
                    QueryProductDetailsParams.Product.newBuilder()
                        .setProductId(id)
                        .setProductType(BillingClient.ProductType.SUBS)
                        .build()
                },
            )
            .build()
        val result = billingClient.queryProductDetails(params)
        if (result.billingResult.responseCode != BillingClient.BillingResponseCode.OK) {
            Log.w(TAG, "queryProductDetails: ${result.billingResult.debugMessage}")
            return
        }
        playProductDetails = result.productDetailsList
            .orEmpty()
            .associateBy { it.productId }
    }

    private suspend fun queryOwnedSubscriptions(): List<Purchase> {
        if (!billingClient.isReady) return emptyList()
        val result = billingClient.queryPurchasesAsync(
            QueryPurchasesParams.newBuilder()
                .setProductType(BillingClient.ProductType.SUBS)
                .build(),
        )
        if (result.billingResult.responseCode != BillingClient.BillingResponseCode.OK) {
            return emptyList()
        }
        return result.purchasesList.filter {
            it.purchaseState == Purchase.PurchaseState.PURCHASED
        }
    }

    private suspend fun syncAndAcknowledge(purchase: Purchase) = syncMutex.withLock {
        if (purchase.purchaseState != Purchase.PurchaseState.PURCHASED) return
        val productId = purchase.products.firstOrNull().orEmpty()
        if (productId.isBlank() || purchase.purchaseToken.isBlank()) return
        val status = auth.newApiClient().syncGooglePurchase(
            packageName = packageName,
            productId = productId,
            purchaseToken = purchase.purchaseToken,
            deviceId = DeviceIdentity.id(prefs),
        )
        serverStatus = status
        if (!purchase.isAcknowledged) {
            val ack = billingClient.acknowledgePurchase(
                AcknowledgePurchaseParams.newBuilder()
                    .setPurchaseToken(purchase.purchaseToken)
                    .build(),
            )
            if (ack.responseCode != BillingClient.BillingResponseCode.OK) {
                Log.w(TAG, "acknowledge failed: ${ack.debugMessage}")
            }
        }
    }

    private fun playBillingMessage(result: BillingResult): String {
        val detail = result.debugMessage.trim().ifBlank { "kod ${result.responseCode}" }
        return "Google Play: $detail"
    }

    companion object {
        private const val TAG = "BillingStore"
    }
}
