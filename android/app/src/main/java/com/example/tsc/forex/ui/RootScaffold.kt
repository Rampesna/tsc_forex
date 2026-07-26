package com.example.tsc.forex.ui

import androidx.compose.animation.EnterTransition
import androidx.compose.animation.ExitTransition
import androidx.compose.animation.animateColorAsState
import androidx.compose.animation.core.Spring
import androidx.compose.animation.core.spring
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.interaction.MutableInteractionSource
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.WindowInsets
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.navigationBarsPadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.BarChart
import androidx.compose.material.icons.filled.EmojiEvents
import androidx.compose.material.icons.filled.Notifications
import androidx.compose.material.icons.filled.Settings
import androidx.compose.material.icons.filled.Work
import androidx.compose.material.icons.outlined.BarChart
import androidx.compose.material.icons.outlined.EmojiEvents
import androidx.compose.material.icons.outlined.Notifications
import androidx.compose.material.icons.outlined.Settings
import androidx.compose.material.icons.outlined.WorkOutline
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.Icon
import androidx.compose.material3.Scaffold
import com.example.tsc.forex.ui.LocalizedText as Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.draw.shadow
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.navigation.NavGraph.Companion.findStartDestination
import androidx.navigation.compose.NavHost
import androidx.navigation.compose.composable
import androidx.navigation.compose.currentBackStackEntryAsState
import androidx.navigation.compose.rememberNavController
import com.example.tsc.forex.data.Prefs
import com.example.tsc.forex.store.AuthStore
import com.example.tsc.forex.store.BillingStore
import com.example.tsc.forex.store.PortfolioStore
import com.example.tsc.forex.store.PriceStore
import com.example.tsc.forex.ui.screens.AlertsScreen
import com.example.tsc.forex.ui.screens.AssetDetailScreen
import com.example.tsc.forex.ui.screens.LoginScreen
import com.example.tsc.forex.ui.screens.MarketsScreen
import com.example.tsc.forex.ui.screens.NotificationsScreen
import com.example.tsc.forex.ui.screens.PaywallSheet
import com.example.tsc.forex.ui.screens.PortfolioPositionScreen
import com.example.tsc.forex.ui.screens.PortfolioScreen
import com.example.tsc.forex.ui.screens.ResearchArchiveScreen
import com.example.tsc.forex.ui.screens.RewardsScreen
import com.example.tsc.forex.ui.screens.SettingsScreen
import com.example.tsc.forex.ui.theme.Th

/** Mirrors iOS AppTab: outline idle · filled active, monochrome brand tint. */
private data class Tab(
    val route: String,
    val label: String,
    val outlined: ImageVector,
    val filled: ImageVector,
)

private val TABS = listOf(
    Tab("markets", "Piyasa", Icons.Outlined.BarChart, Icons.Filled.BarChart),
    Tab("portfolio", "Portföy", Icons.Outlined.WorkOutline, Icons.Filled.Work),
    Tab("rewards", "Ödüller", Icons.Outlined.EmojiEvents, Icons.Filled.EmojiEvents),
    Tab("alerts", "Alarm", Icons.Outlined.Notifications, Icons.Filled.Notifications),
    Tab("settings", "Ayarlar", Icons.Outlined.Settings, Icons.Filled.Settings),
)

private val TabBarShape = RoundedCornerShape(26.dp)
private val TabPillShape = RoundedCornerShape(18.dp)

@Composable
fun RootScaffold(
    auth: AuthStore,
    prices: PriceStore,
    portfolio: PortfolioStore,
    billing: BillingStore,
    prefs: Prefs,
) {
    val nav = rememberNavController()
    val backStack by nav.currentBackStackEntryAsState()
    val current = backStack?.destination?.route
    var showLogin by remember { mutableStateOf(false) }

    val hideBar = current == "research-archive" ||
        current == "notifications"
    val activeTab = when {
        current?.startsWith("detail/") == true -> "markets"
        current?.startsWith("portfolio-position/") == true -> "portfolio"
        else -> current ?: "markets"
    }
    val isTabContext = TABS.any { it.route == current } ||
        current?.startsWith("detail/") == true ||
        current?.startsWith("portfolio-position/") == true

    fun requestAuth() {
        showLogin = true
    }

    Scaffold(
        containerColor = Th.bg,
        contentWindowInsets = WindowInsets(0, 0, 0, 0),
        bottomBar = {
            if (!hideBar && (current == null || isTabContext)) {
                // iOS CustomTabBar: floating glass capsule above the home indicator
                Box(
                    Modifier
                        .fillMaxWidth()
                        .navigationBarsPadding()
                        .padding(horizontal = 14.dp, vertical = 8.dp),
                ) {
                    FloatingTabBar(
                        currentRoute = activeTab,
                        onSelect = { route ->
                            nav.navigate(route) {
                                popUpTo(nav.graph.findStartDestination().id) { saveState = true }
                                launchSingleTop = true
                                restoreState = true
                            }
                        },
                    )
                }
            }
        },
    ) { padding ->
        NavHost(
            navController = nav,
            startDestination = "markets",
            modifier = Modifier.padding(padding),
            enterTransition = { EnterTransition.None },
            exitTransition = { ExitTransition.None },
            popEnterTransition = { EnterTransition.None },
            popExitTransition = { ExitTransition.None },
        ) {
            composable("markets") {
                MarketsScreen(
                    store = prices,
                    auth = auth,
                    onOpen = { code -> nav.navigate("detail/$code") },
                    onNotifications = {
                        if (auth.isLoggedIn) nav.navigate("notifications")
                        else requestAuth()
                    },
                    onRequestAuth = ::requestAuth,
                )
            }
            composable("portfolio") {
                if (auth.isLoggedIn) {
                    PortfolioScreen(portfolio, prices, auth, billing, prefs) { code ->
                        nav.navigate("portfolio-position/$code")
                    }
                } else {
                    AccountRequiredPane(
                        title = "Portföy için giriş yapın",
                        explanation = "Portföyünüzü cihazlar arasında güvenle eşitlemek için hesabınızla giriş yapın.",
                        onSignIn = ::requestAuth,
                    )
                }
            }
            composable("rewards") {
                if (auth.isLoggedIn) {
                    RewardsScreen(auth, prices)
                } else {
                    AccountRequiredPane(
                        title = "Ödüller için giriş yapın",
                        explanation = "Görevlerinizi, öğrenme ilerlemenizi ve rozetlerinizi hesabınıza kaydedin.",
                        onSignIn = ::requestAuth,
                    )
                }
            }
            composable("alerts") {
                if (auth.isLoggedIn) {
                    AlertsScreen(auth, prices)
                } else {
                    AccountRequiredPane(
                        title = "Alarm için giriş yapın",
                        explanation = "Fiyat alarmlarınızı oluşturmak ve bildirim almak için hesabınızla giriş yapın.",
                        onSignIn = ::requestAuth,
                    )
                }
            }
            composable("settings") {
                if (auth.isLoggedIn) {
                    SettingsScreen(
                        auth = auth,
                        store = prices,
                        billing = billing,
                        prefs = prefs,
                        onOpenArchive = { nav.navigate("research-archive") },
                    )
                } else {
                    AccountRequiredPane(
                        title = "Ayarlar için giriş yapın",
                        explanation = "Profilinizi, aboneliğinizi ve uygulama tercihlerinizi yönetmek için giriş yapın.",
                        onSignIn = ::requestAuth,
                    )
                }
            }
            composable("notifications") {
                NotificationsScreen(auth, prices, onBack = { nav.popBackStack() })
            }
            composable("research-archive") {
                var showPaywall by remember { mutableStateOf(false) }
                ResearchArchiveScreen(
                    auth = auth,
                    billing = billing,
                    onBack = { nav.popBackStack() },
                    onOpenPaywall = { showPaywall = true },
                )
                if (showPaywall) {
                    PaywallSheet(billing = billing, onDismiss = { showPaywall = false })
                }
            }
            composable("detail/{code}") { entry ->
                val code = entry.arguments?.getString("code") ?: return@composable
                AssetDetailScreen(
                    code = code,
                    auth = auth,
                    store = prices,
                    billing = billing,
                    prefs = prefs,
                    onBack = { nav.popBackStack() },
                    onRequestAuth = ::requestAuth,
                )
            }
            composable("portfolio-position/{code}") { entry ->
                val code = entry.arguments?.getString("code") ?: return@composable
                PortfolioPositionScreen(
                    code = code,
                    portfolio = portfolio,
                    store = prices,
                    onBack = { nav.popBackStack() },
                    onAsset = { assetCode -> nav.navigate("detail/$assetCode") },
                )
            }
        }
    }

    if (showLogin) {
        LoginScreen(
            auth = auth,
            asModal = true,
            onLoggedIn = {
                showLogin = false
                billing.refresh()
            },
            onContinueAsGuest = { showLogin = false },
        )
    }
}

/**
 * iOS [CustomTabBar] parity: floating rounded capsule, glass fill, stroke,
 * soft shadow, and teal pill behind the active tab.
 */
@Composable
private fun FloatingTabBar(
    currentRoute: String,
    onSelect: (String) -> Unit,
) {
    Row(
        Modifier
            .fillMaxWidth()
            .shadow(
                elevation = 18.dp,
                shape = TabBarShape,
                ambientColor = Color.Black.copy(alpha = 0.45f),
                spotColor = Color.Black.copy(alpha = 0.45f),
            )
            .clip(TabBarShape)
            .background(Color(0xF212161C)) // Theme.bgElevated @ ~0.95 + glass feel
            .border(1.dp, Color.White.copy(alpha = 0.12f), TabBarShape)
            .padding(horizontal = 6.dp, vertical = 6.dp),
        horizontalArrangement = Arrangement.spacedBy(2.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        TABS.forEach { tab ->
            val selected = currentRoute == tab.route
            val pill by animateColorAsState(
                targetValue = if (selected) Th.accent.copy(alpha = 0.16f) else Color.Transparent,
                animationSpec = spring(stiffness = Spring.StiffnessMediumLow),
                label = "tabPill",
            )
            val tint by animateColorAsState(
                targetValue = if (selected) Th.accent else Th.textDim,
                animationSpec = spring(stiffness = Spring.StiffnessMediumLow),
                label = "tabTint",
            )
            Column(
                Modifier
                    .weight(1f)
                    .clip(TabPillShape)
                    .background(pill)
                    .clickable(
                        interactionSource = remember { MutableInteractionSource() },
                        indication = null,
                        onClick = { onSelect(tab.route) },
                    )
                    .heightIn(min = 54.dp)
                    .padding(vertical = 6.dp),
                horizontalAlignment = Alignment.CenterHorizontally,
                verticalArrangement = Arrangement.spacedBy(3.dp),
            ) {
                Icon(
                    imageVector = if (selected) tab.filled else tab.outlined,
                    contentDescription = tab.label,
                    tint = tint,
                    modifier = Modifier.size(21.dp),
                )
                Text(
                    tab.label,
                    color = tint,
                    fontSize = 10.sp,
                    fontWeight = if (selected) FontWeight.Bold else FontWeight.Medium,
                    maxLines = 1,
                )
            }
        }
    }
}

@Composable
private fun AccountRequiredPane(title: String, explanation: String, onSignIn: () -> Unit) {
    Column(
        Modifier
            .fillMaxSize()
            .background(Th.bg)
            .padding(28.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.Center,
    ) {
        Text(title, color = Th.text, fontSize = 22.sp, fontWeight = FontWeight.Bold, textAlign = TextAlign.Center)
        Spacer(Modifier.height(10.dp))
        Text(explanation, color = Th.textDim, fontSize = 13.sp, textAlign = TextAlign.Center)
        Spacer(Modifier.height(22.dp))
        Button(
            onClick = onSignIn,
            colors = ButtonDefaults.buttonColors(containerColor = Th.accent, contentColor = Th.bg),
            shape = RoundedCornerShape(12.dp),
            modifier = Modifier.fillMaxWidth(),
        ) { Text("Giriş Yap / Kayıt Ol", fontWeight = FontWeight.Bold) }
    }
}
