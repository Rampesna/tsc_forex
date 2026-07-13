//
//  PortfolioStore.swift
//  Local portfolio holdings with live valuation. Amounts are in the
//  instrument's native unit (grams for gold, pieces for coins, units for FX).
//

import Foundation
import SwiftUI
import Combine

struct Holding: Codable, Identifiable, Equatable {
    var id = UUID()
    var code: String
    var amount: Double
    var cost: Double   // optional average cost per unit (0 = unknown)
}

@MainActor
final class PortfolioStore: ObservableObject {
    @Published var holdings: [Holding] = []
    private let key = "portfolio_holdings_v1"

    init() { load() }

    private func load() {
        if let d = UserDefaults.standard.data(forKey: key),
           let h = try? JSONDecoder().decode([Holding].self, from: d) {
            holdings = h
        }
    }

    private func save() {
        if let d = try? JSONEncoder().encode(holdings) {
            UserDefaults.standard.set(d, forKey: key)
        }
    }

    func add(code: String, amount: Double, cost: Double) {
        if let i = holdings.firstIndex(where: { $0.code == code }) {
            holdings[i].amount += amount
            if cost > 0 { holdings[i].cost = cost }
        } else {
            holdings.append(Holding(code: code, amount: amount, cost: cost))
        }
        save()
    }

    func remove(at offsets: IndexSet) {
        holdings.remove(atOffsets: offsets)
        save()
    }
}
