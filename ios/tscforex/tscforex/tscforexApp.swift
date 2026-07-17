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

    var body: some Scene {
        WindowGroup {
            RootView()
                .environmentObject(auth)
                .environmentObject(prices)
                .environmentObject(portfolio)
                .environmentObject(billing)
                .onAppear {
                    prices.connect(token: auth.accessToken)
                }
        }
    }
}
