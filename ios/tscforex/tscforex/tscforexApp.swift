//
//  tscforexApp.swift
//  tscforex
//
//  Live Gram Gold & FX tracker — TSC Forex.
//

import SwiftUI

@main
struct tscforexApp: App {
    @UIApplicationDelegateAdaptor(AppDelegate.self) private var appDelegate
    @StateObject private var auth = AuthStore()
    @StateObject private var prices = PriceStore()
    @StateObject private var portfolio = PortfolioStore()
    @StateObject private var billing = BillingStore()
    @StateObject private var language = LanguageStore()

    var body: some Scene {
        WindowGroup {
            RootView()
                .environmentObject(auth)
                .environmentObject(prices)
                .environmentObject(portfolio)
                .environmentObject(billing)
                .environmentObject(language)
                .environment(\.locale, language.current.locale)
                .environment(\.layoutDirection, language.current.layoutDirection)
                .onAppear {
                    prices.connect(token: auth.accessToken)
                }
                .onChange(of: language.current) { _, _ in
                    prices.reloadLocale()
                }
        }
    }
}
