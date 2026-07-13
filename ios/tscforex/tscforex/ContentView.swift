//
//  ContentView.swift
//  Root navigation with a custom, Binance-style bottom tab bar.
//

import SwiftUI

struct RootView: View {
    @EnvironmentObject var auth: AuthStore

    var body: some View {
        ZStack {
            Theme.bg.ignoresSafeArea()
            if auth.isLoggedIn {
                MainTabView()
            } else {
                LoginView()
            }
        }
        .preferredColorScheme(.dark)
        .tint(Theme.accent)
    }
}

enum AppTab: Int, CaseIterable {
    case markets, portfolio, alerts, notifications, settings
    var title: String {
        switch self {
        case .markets: return "Piyasa"
        case .portfolio: return "Portföy"
        case .alerts: return "Alarm"
        case .notifications: return "Bildirim"
        case .settings: return "Ayarlar"
        }
    }
    var icon: String {
        switch self {
        case .markets: return "chart.bar.xaxis"
        case .portfolio: return "briefcase"
        case .alerts: return "bell"
        case .notifications: return "tray.full"
        case .settings: return "gearshape"
        }
    }
}

struct MainTabView: View {
    @EnvironmentObject var auth: AuthStore
    @EnvironmentObject var prices: PriceStore
    @State private var tab: AppTab = .markets

    var body: some View {
        ZStack(alignment: .bottom) {
            Group {
                switch tab {
                case .markets:        MarketsView()
                case .portfolio:      PortfolioView()
                case .alerts:         AlertsView()
                case .notifications:  NotificationsView()
                case .settings:       SettingsView()
                }
            }
            .frame(maxWidth: .infinity, maxHeight: .infinity)

            CustomTabBar(selection: $tab, badge: prices.liveNotifications.count)
        }
        .onAppear {
            prices.connect(token: auth.accessToken)
            setupPush()
        }
    }

    // Ask for notification permission and register this device for push so the
    // backend can target it (iOS now, Android later — both stored per-user).
    private func setupPush() {
        // Auto-refresh expired access tokens, then reconnect the WS with the new one.
        APIClient.refresher = {
            let ok = await auth.refresh()
            if ok { prices.updateToken(auth.accessToken) }
            return ok ? auth.accessToken : nil
        }
        PushManager.shared.onToken = { token in
            Task { try? await APIClient(token: auth.accessToken).registerDevice(token: token) }
        }
        PushManager.shared.requestPermission()
        if let existing = PushManager.shared.deviceToken {
            Task { try? await APIClient(token: auth.accessToken).registerDevice(token: existing) }
        }
    }
}

struct CustomTabBar: View {
    @Binding var selection: AppTab
    var badge: Int = 0

    var body: some View {
        HStack {
            ForEach(AppTab.allCases, id: \.self) { t in
                Button {
                    Haptics.tap()
                    withAnimation(.easeInOut(duration: 0.15)) { selection = t }
                } label: {
                    VStack(spacing: 4) {
                        ZStack(alignment: .topTrailing) {
                            Image(systemName: t.icon)
                                .font(.system(size: 19, weight: .semibold))
                            if t == .notifications && badge > 0 {
                                Circle().fill(Theme.down).frame(width: 8, height: 8).offset(x: 6, y: -2)
                            }
                        }
                        Text(t.title).font(.system(size: 10, weight: .medium))
                    }
                    .foregroundStyle(selection == t ? Theme.accent : Theme.textDim)
                    .frame(maxWidth: .infinity)
                }
                .buttonStyle(.plain)
            }
        }
        .padding(.top, 10)
        .padding(.bottom, 26)
        .background(
            Theme.bgElevated
                .overlay(Rectangle().fill(Theme.stroke).frame(height: 1), alignment: .top)
                .ignoresSafeArea(edges: .bottom)
        )
    }
}

// Kept for Xcode previews / template compatibility.
struct ContentView: View {
    var body: some View { RootView() }
}

#Preview {
    RootView()
        .environmentObject(AuthStore())
        .environmentObject(PriceStore())
        .environmentObject(PortfolioStore())
}
