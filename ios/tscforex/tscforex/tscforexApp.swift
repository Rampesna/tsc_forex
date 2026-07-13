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

    var body: some Scene {
        WindowGroup {
            RootView()
                .environmentObject(auth)
                .environmentObject(prices)
                .environmentObject(portfolio)
                .onAppear {
                    prices.connect(token: auth.accessToken)
                }
        }
    }
}
