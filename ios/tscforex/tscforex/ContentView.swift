//
//  ContentView.swift
//  Root navigation with a custom, Binance-style bottom tab bar.
//

import SwiftUI

struct RootView: View {
    var body: some View {
        ZStack {
            Theme.bg.ignoresSafeArea()
            MainTabView()
        }
        .preferredColorScheme(.dark)
        .tint(Theme.accent)
    }
}

private struct RequestAuthenticationKey: EnvironmentKey {
    static let defaultValue: () -> Void = {}
}

extension EnvironmentValues {
    var requestAuthentication: () -> Void {
        get { self[RequestAuthenticationKey.self] }
        set { self[RequestAuthenticationKey.self] = newValue }
    }
}

enum AppTab: Int, CaseIterable {
    case markets, portfolio, rewards, alerts, settings
    var title: String {
        switch self {
        case .markets: return "Piyasa"
        case .portfolio: return "Portföy"
        case .rewards: return "Ödüller"
        case .alerts: return "Alarm"
        case .settings: return "Ayarlar"
        }
    }
    var icon: String {
        switch self {
        case .markets: return "chart.bar.xaxis"
        case .portfolio: return "briefcase"
        case .rewards: return "trophy"
        case .alerts: return "bell"
        case .settings: return "gearshape"
        }
    }
    var iconFilled: String {
        switch self {
        case .markets: return "chart.bar.xaxis"
        case .portfolio: return "briefcase.fill"
        case .rewards: return "trophy.fill"
        case .alerts: return "bell.fill"
        case .settings: return "gearshape.fill"
        }
    }
}

struct MainTabView: View {
    @EnvironmentObject var auth: AuthStore
    @EnvironmentObject var prices: PriceStore
    @EnvironmentObject var billing: BillingStore
    @State private var tab: AppTab = .markets
    @State private var marketCategory: MarketCategory = .all
    @State private var marketPath = NavigationPath()
    @State private var showingNotifications = false
    @State private var showingAuthentication = false

    var body: some View {
        Group {
            switch tab {
            case .markets:
                MarketsView(category: $marketCategory, path: $marketPath) {
                    if auth.isLoggedIn {
                        showingNotifications = true
                    } else {
                        showingAuthentication = true
                    }
                }
            case .portfolio:
                authenticated { PortfolioView() }
            case .rewards:
                authenticated { GamificationView() }
            case .alerts:
                authenticated { AlertsView() }
            case .settings:
                authenticated { SettingsView() }
            }
        }
        .environment(\.requestAuthentication, { showingAuthentication = true })
        .frame(maxWidth: .infinity, maxHeight: .infinity)
        .safeAreaInset(edge: .bottom) {
            CustomTabBar(selection: $tab) { tapped in
                if tapped == .markets, !marketPath.isEmpty {
                    withAnimation { marketPath = NavigationPath() }
                }
            }
        }
        // The inbox is global application state rather than a market-detail
        // route. Keeping it outside the market NavigationPath avoids nesting
        // navigation stacks and leaves the selected market category untouched.
        .fullScreenCover(isPresented: $showingNotifications) {
            NotificationsView()
        }
        .fullScreenCover(isPresented: $showingAuthentication) {
            LoginView(onContinueAsGuest: { showingAuthentication = false })
                .environmentObject(auth)
        }
        .onAppear {
            prices.connect(token: auth.accessToken)
            if auth.isLoggedIn {
                setupPush()
            } else {
                prices.clearAccountData()
            }
        }
        .task(id: auth.accessToken) {
            guard let token = auth.accessToken, !token.isEmpty else {
                billing.clearSession()
                return
            }
            await auth.loadProfile()
            await billing.refresh(token: token)
        }
        .onDisappear {
            if !auth.isLoggedIn {
                billing.clearSession()
            }
        }
        .onChange(of: auth.isLoggedIn) { _, isLoggedIn in
            showingAuthentication = false
            prices.updateToken(auth.accessToken)
            if isLoggedIn {
                setupPush()
            } else {
                showingNotifications = false
                prices.clearAccountData()
                billing.clearSession()
            }
        }
    }

    @ViewBuilder
    private func authenticated<Content: View>(@ViewBuilder content: () -> Content) -> some View {
        if auth.isLoggedIn {
            content()
        } else {
            AccountRequiredView(tab: tab) {
                showingAuthentication = true
            }
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
        if let existing = PushManager.shared.deviceToken {
            Task { try? await APIClient(token: auth.accessToken).registerDevice(token: existing) }
        }
    }
}

private struct AccountRequiredView: View {
    let tab: AppTab
    let onSignIn: () -> Void

    private var explanation: String {
        switch tab {
        case .portfolio: return "Portföyünüzü cihazlar arasında güvenle eşitlemek için hesabınızla giriş yapın."
        case .rewards: return "Görevlerinizi, öğrenme ilerlemenizi ve rozetlerinizi hesabınıza kaydedin."
        case .alerts: return "Fiyat alarmlarınızı oluşturmak ve bildirim almak için hesabınızla giriş yapın."
        case .settings: return "Profilinizi, aboneliğinizi ve uygulama tercihlerinizi yönetmek için giriş yapın."
        case .markets: return ""
        }
    }

    var body: some View {
        ZStack {
            Theme.bg.ignoresSafeArea()
            ScrollView {
                VStack(spacing: 18) {
                    Spacer(minLength: 80)
                    Image(systemName: tab.iconFilled)
                        .font(.system(size: 34, weight: .semibold))
                        .foregroundStyle(Theme.accent)
                        .frame(width: 74, height: 74)
                        .background(Theme.accentSoft, in: RoundedRectangle(cornerRadius: 22, style: .continuous))
                    Text("\(tab.title) için giriş yapın")
                        .font(.system(size: 23, weight: .bold))
                        .foregroundStyle(Theme.text)
                    Text(explanation)
                        .font(.system(size: 14))
                        .foregroundStyle(Theme.textDim)
                        .multilineTextAlignment(.center)
                        .fixedSize(horizontal: false, vertical: true)
                    Button("Giriş Yap veya Kayıt Ol", action: onSignIn)
                        .buttonStyle(FilledButtonStyle())
                        .padding(.top, 4)

                    if tab == .settings {
                        HStack(spacing: 18) {
                            Link("Gizlilik", destination: URL(string: "https://tsc.talhacan.com/privacy")!)
                            Link("Kullanım Şartları", destination: URL(string: "https://tsc.talhacan.com/terms")!)
                        }
                        .font(.system(size: 12, weight: .semibold))
                        .foregroundStyle(Theme.accent)
                    }
                }
                .padding(.horizontal, 28)
                .frame(maxWidth: 520)
                .frame(maxWidth: .infinity)
            }
        }
    }
}

// Floating, glass-style tab bar: a rounded capsule with a sliding accent pill
// behind the active item and a filled-icon state on selection.
struct CustomTabBar: View {
    @Binding var selection: AppTab
    var onTap: (AppTab) -> Void = { _ in }
    @Namespace private var ns

    var body: some View {
        HStack(spacing: 2) {
            ForEach(AppTab.allCases, id: \.self) { t in
                item(t)
            }
        }
        .padding(.horizontal, 8)
        .padding(.vertical, 8)
        .background(
            ZStack {
                RoundedRectangle(cornerRadius: 26, style: .continuous)
                    .fill(.ultraThinMaterial)
                RoundedRectangle(cornerRadius: 26, style: .continuous)
                    .fill(Theme.bgElevated.opacity(0.72))
                RoundedRectangle(cornerRadius: 26, style: .continuous)
                    .stroke(Theme.strokeStrong, lineWidth: 1)
            }
        )
        .clipShape(RoundedRectangle(cornerRadius: 26, style: .continuous))
        .shadow(color: .black.opacity(0.45), radius: 18, x: 0, y: 8)
        .padding(.horizontal, 14)
        .padding(.bottom, 8)
    }

    @ViewBuilder
    private func item(_ t: AppTab) -> some View {
        let active = selection == t
        Button {
            Haptics.tap()
            onTap(t)
            if selection != t {
                withAnimation(.spring(response: 0.35, dampingFraction: 0.78)) { selection = t }
            }
        } label: {
            VStack(spacing: 3) {
                Image(systemName: active ? t.iconFilled : t.icon)
                    .font(.system(size: 18, weight: .semibold))
                    .frame(height: 22)
                Text(t.title)
                    .font(.system(size: 9.5, weight: active ? .bold : .medium))
                    .lineLimit(1)
            }
            .foregroundStyle(active ? Theme.accent : Theme.textDim)
            .frame(maxWidth: .infinity)
            .padding(.vertical, 8)
            .background(
                ZStack {
                    if active {
                        RoundedRectangle(cornerRadius: 18, style: .continuous)
                            .fill(Theme.accentSoft)
                            .matchedGeometryEffect(id: "tabPill", in: ns)
                    }
                }
            )
            .contentShape(Rectangle())
        }
        .buttonStyle(.plain)
    }
}

// Kept for Xcode previews / template compatibility.
struct ContentView: View {
    var body: some View { RootView() }
}

#if DEBUG
struct ContentView_Previews: PreviewProvider {
    static var previews: some View {
        RootView()
            .environmentObject(AuthStore())
            .environmentObject(PriceStore())
            .environmentObject(PortfolioStore())
            .environmentObject(BillingStore())
    }
}
#endif
