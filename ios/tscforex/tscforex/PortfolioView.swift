//
//  PortfolioView.swift
//  Live-valued portfolio: total value, today's change, P/L vs cost, holdings.
//

import SwiftUI

struct PortfolioView: View {
    @EnvironmentObject var portfolio: PortfolioStore
    @EnvironmentObject var prices: PriceStore
    @State private var showAdd = false

    private func price(_ code: String) -> Double { prices.prices[code]?.price ?? 0 }
    private func value(_ h: Holding) -> Double { h.amount * price(h.code) }

    private var total: Double { portfolio.holdings.reduce(0) { $0 + value($1) } }

    private var todayChange: Double {
        portfolio.holdings.reduce(0) { acc, h in
            acc + h.amount * (prices.prices[h.code]?.change ?? 0)
        }
    }

    private var totalCost: Double {
        portfolio.holdings.reduce(0) { $0 + $1.amount * $1.cost }
    }
    private var pnl: Double { totalCost > 0 ? total - totalCost : 0 }

    var body: some View {
        ZStack {
            Theme.bg.ignoresSafeArea()
            VStack(spacing: 0) {
                header
                if portfolio.holdings.isEmpty {
                    emptyState
                } else {
                    ScrollView {
                        VStack(spacing: 14) {
                            summaryCard
                            ForEach(portfolio.holdings) { h in holdingRow(h) }
                        }
                        .padding(.horizontal, 16).padding(.top, 8).padding(.bottom, 100)
                    }
                }
            }
        }
        .sheet(isPresented: $showAdd) {
            AddHoldingSheet()
                .environmentObject(portfolio).environmentObject(prices)
                .presentationDetents([.height(420)])
                .presentationBackground(Theme.bgElevated)
        }
    }

    private var header: some View {
        HStack {
            VStack(alignment: .leading, spacing: 2) {
                Text("Portföy").font(.system(size: 28, weight: .bold)).foregroundStyle(Theme.text)
                Text("Canlı değerleme").font(.system(size: 12)).foregroundStyle(Theme.textDim)
            }
            Spacer()
            Button { Haptics.tap(); showAdd = true } label: {
                Image(systemName: "plus").font(.system(size: 15, weight: .bold)).foregroundStyle(.black)
                    .frame(width: 34, height: 34).background(Theme.accent, in: Circle())
            }
        }
        .padding(.horizontal, 16).padding(.top, 6).padding(.bottom, 10)
    }

    private var summaryCard: some View {
        Card {
            VStack(alignment: .leading, spacing: 10) {
                Text("Toplam Değer").font(.system(size: 12)).foregroundStyle(Theme.textDim)
                Text("₺\(Fmt.compactPrice(total))")
                    .font(Theme.number(32, weight: .bold)).foregroundStyle(Theme.text)
                HStack(spacing: 16) {
                    VStack(alignment: .leading, spacing: 3) {
                        Text("Bugün").font(.system(size: 11)).foregroundStyle(Theme.textDim)
                        Text("\(Fmt.signed(todayChange)) ₺")
                            .font(Theme.number(14, weight: .semibold)).foregroundStyle(Theme.trend(todayChange))
                    }
                    if totalCost > 0 {
                        VStack(alignment: .leading, spacing: 3) {
                            Text("Kâr/Zarar").font(.system(size: 11)).foregroundStyle(Theme.textDim)
                            Text("\(Fmt.signed(pnl)) ₺ (\(Fmt.signedPct(totalCost > 0 ? pnl/totalCost*100 : 0)))")
                                .font(Theme.number(14, weight: .semibold)).foregroundStyle(Theme.trend(pnl))
                        }
                    }
                    Spacer()
                }
            }
        }
    }

    private func holdingRow(_ h: Holding) -> some View {
        let v = value(h)
        let share = total > 0 ? v / total * 100 : 0
        return Card(padding: 14) {
            HStack(spacing: 12) {
                AssetIcon(code: h.code, kind: prices.prices[h.code]?.kind ?? "gold", size: 36)
                VStack(alignment: .leading, spacing: 3) {
                    Text(prices.prices[h.code]?.displayName ?? h.code)
                        .font(.system(size: 15, weight: .semibold)).foregroundStyle(Theme.text)
                    Text("\(Fmt.compactPrice(h.amount)) birim • %\(String(format: "%.1f", share))")
                        .font(.system(size: 11)).foregroundStyle(Theme.textDim)
                }
                Spacer()
                VStack(alignment: .trailing, spacing: 3) {
                    Text("₺\(Fmt.compactPrice(v))").font(Theme.number(15, weight: .bold)).foregroundStyle(Theme.text)
                    Text(Fmt.signedPct(prices.prices[h.code]?.changePct ?? 0))
                        .font(Theme.number(11, weight: .semibold))
                        .foregroundStyle(Theme.trend(prices.prices[h.code]?.changePct ?? 0))
                }
            }
        }
        .contextMenu {
            Button(role: .destructive) {
                if let idx = portfolio.holdings.firstIndex(of: h) {
                    portfolio.remove(at: IndexSet(integer: idx))
                }
            } label: { Label("Sil", systemImage: "trash") }
        }
    }

    private var emptyState: some View {
        VStack(spacing: 12) {
            Image(systemName: "briefcase").font(.system(size: 36)).foregroundStyle(Theme.textFaint)
            Text("Portföyün boş").font(.system(size: 17, weight: .semibold)).foregroundStyle(Theme.text)
            Text("Sahip olduğun altın/dövizi ekle, değerini canlı takip et.")
                .font(.system(size: 13)).foregroundStyle(Theme.textDim)
                .multilineTextAlignment(.center).padding(.horizontal, 40)
            Button { showAdd = true } label: { Text("Varlık Ekle") }
                .buttonStyle(FilledButtonStyle()).frame(width: 200).padding(.top, 6)
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity)
    }
}

struct AddHoldingSheet: View {
    @EnvironmentObject var portfolio: PortfolioStore
    @EnvironmentObject var prices: PriceStore
    @Environment(\.dismiss) private var dismiss

    @State private var code = "GRAM_GOLD"
    @State private var amountText = ""
    @State private var costText = ""

    private var options: [InstrumentPrice] {
        prices.sorted
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 16) {
            Text("Varlık Ekle").font(.system(size: 20, weight: .bold)).foregroundStyle(Theme.text)

            Menu {
                ForEach(options) { p in
                    Button(p.displayName) { code = p.code }
                }
            } label: {
                HStack {
                    Text(prices.prices[code]?.displayName ?? code).foregroundStyle(Theme.text)
                    Spacer()
                    Image(systemName: "chevron.up.chevron.down").font(.system(size: 12)).foregroundStyle(Theme.textDim)
                }
                .padding(14)
                .background(Theme.surface, in: RoundedRectangle(cornerRadius: 10))
            }

            labeledField("Miktar (birim)", text: $amountText, placeholder: "örn. 10")
            labeledField("Ortalama maliyet / birim (opsiyonel)", text: $costText, placeholder: "örn. 6000")

            Button {
                let amt = Double(amountText.replacingOccurrences(of: ",", with: ".")) ?? 0
                let cost = Double(costText.replacingOccurrences(of: ",", with: ".")) ?? 0
                if amt > 0 {
                    portfolio.add(code: code, amount: amt, cost: cost)
                    Haptics.success()
                    dismiss()
                }
            } label: { Text("Ekle") }
            .buttonStyle(FilledButtonStyle())
            .disabled((Double(amountText.replacingOccurrences(of: ",", with: ".")) ?? 0) <= 0)

            Spacer(minLength: 0)
        }
        .padding(20)
    }

    private func labeledField(_ label: String, text: Binding<String>, placeholder: String) -> some View {
        VStack(alignment: .leading, spacing: 6) {
            Text(label).font(.system(size: 12)).foregroundStyle(Theme.textDim)
            TextField("", text: text, prompt: Text(placeholder).foregroundColor(Theme.textFaint))
                .keyboardType(.decimalPad)
                .font(Theme.number(18)).foregroundStyle(Theme.text)
                .padding(12)
                .background(Theme.surface, in: RoundedRectangle(cornerRadius: 10))
        }
    }
}
