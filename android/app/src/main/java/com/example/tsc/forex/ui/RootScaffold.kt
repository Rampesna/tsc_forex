package com.example.tsc.forex.ui

import androidx.compose.foundation.layout.padding
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ShowChart
import androidx.compose.material.icons.filled.EmojiEvents
import androidx.compose.material.icons.filled.Notifications
import androidx.compose.material.icons.filled.NotificationsActive
import androidx.compose.material.icons.filled.PieChart
import androidx.compose.material.icons.filled.Settings
import androidx.compose.material3.Icon
import androidx.compose.material3.NavigationBar
import androidx.compose.material3.NavigationBarItem
import androidx.compose.material3.NavigationBarItemDefaults
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.unit.sp
import androidx.navigation.NavGraph.Companion.findStartDestination
import androidx.navigation.compose.NavHost
import androidx.navigation.compose.composable
import androidx.navigation.compose.currentBackStackEntryAsState
import androidx.navigation.compose.rememberNavController
import com.example.tsc.forex.data.Prefs
import com.example.tsc.forex.store.AuthStore
import com.example.tsc.forex.store.PortfolioStore
import com.example.tsc.forex.store.PriceStore
import com.example.tsc.forex.ui.screens.AlertsScreen
import com.example.tsc.forex.ui.screens.AssetDetailScreen
import com.example.tsc.forex.ui.screens.MarketsScreen
import com.example.tsc.forex.ui.screens.NotificationsScreen
import com.example.tsc.forex.ui.screens.PortfolioScreen
import com.example.tsc.forex.ui.screens.PortfolioPositionScreen
import com.example.tsc.forex.ui.screens.RewardsScreen
import com.example.tsc.forex.ui.screens.SettingsScreen
import com.example.tsc.forex.ui.theme.Th

private data class Tab(val route: String, val label: String, val icon: ImageVector)

private val TABS = listOf(
    Tab("markets", "Piyasalar", Icons.AutoMirrored.Filled.ShowChart),
    Tab("portfolio", "Portföy", Icons.Filled.PieChart),
    Tab("rewards", "Ödüller", Icons.Filled.EmojiEvents),
    Tab("alerts", "Alarmlar", Icons.Filled.NotificationsActive),
    Tab("notifications", "Bildirim", Icons.Filled.Notifications),
    Tab("settings", "Ayarlar", Icons.Filled.Settings),
)

@Composable
fun RootScaffold(auth: AuthStore, prices: PriceStore, portfolio: PortfolioStore, prefs: Prefs) {
    val nav = rememberNavController()
    val backStack by nav.currentBackStackEntryAsState()
    val current = backStack?.destination?.route

    Scaffold(
        containerColor = Th.bg,
        bottomBar = {
            // Hide the bottom bar on the detail screen for an immersive view.
            if (current == null || TABS.any { it.route == current }) {
                NavigationBar(containerColor = Th.surface) {
                    TABS.forEach { tab ->
                        NavigationBarItem(
                            selected = current == tab.route,
                            onClick = {
                                nav.navigate(tab.route) {
                                    popUpTo(nav.graph.findStartDestination().id) { saveState = true }
                                    launchSingleTop = true
                                    restoreState = true
                                }
                            },
                            icon = { Icon(tab.icon, tab.label) },
                            label = { Text(tab.label, fontSize = 10.sp) },
                            colors = NavigationBarItemDefaults.colors(
                                selectedIconColor = Th.accent,
                                selectedTextColor = Th.accent,
                                unselectedIconColor = Th.textFaint,
                                unselectedTextColor = Th.textFaint,
                                indicatorColor = Th.accent.copy(alpha = 0.14f),
                            ),
                        )
                    }
                }
            }
        },
    ) { padding ->
        NavHost(
            navController = nav,
            startDestination = "markets",
            modifier = Modifier.padding(padding),
        ) {
            composable("markets") { MarketsScreen(prices) { code -> nav.navigate("detail/$code") } }
            composable("portfolio") {
                PortfolioScreen(portfolio, prices) { code -> nav.navigate("portfolio-position/$code") }
            }
            composable("rewards") { RewardsScreen(auth, prices) }
            composable("alerts") { AlertsScreen(auth, prices) }
            composable("notifications") { NotificationsScreen(auth, prices) }
            composable("settings") { SettingsScreen(auth, prices, prefs) }
            composable("detail/{code}") { entry ->
                val code = entry.arguments?.getString("code") ?: return@composable
                AssetDetailScreen(code, auth, prices, onBack = { nav.popBackStack() })
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
}
