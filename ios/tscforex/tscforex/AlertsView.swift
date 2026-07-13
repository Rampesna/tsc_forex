//
//  AlertsView.swift
//  Themed list of the user's price alerts.
//

import SwiftUI

struct AlertsView: View {
    @EnvironmentObject var auth: AuthStore
    @EnvironmentObject var prices: PriceStore
    @State private var alerts: [Alert] = []
    @State private var loading = false

    var body: some View {
        NavigationStack {
            ZStack {
                Theme.bg.ignoresSafeArea()
                VStack(spacing: 0) {
                    ScreenHeader(title: "Alarmlar", subtitle: "Eşik geçişinde bildirim")
                    if alerts.isEmpty && !loading {
                        emptyState
                    } else {
                        ScrollView {
                            LazyVStack(spacing: 10) {
                                ForEach(alerts) { a in alertCard(a) }
                            }
                            .padding(.horizontal, 16).padding(.top, 8).padding(.bottom, 100)
                        }
                        .refreshable { await load() }
                    }
                }
            }
            .toolbar(.hidden, for: .navigationBar)
        }
        .task { await load() }
    }

    private func alertCard(_ a: Alert) -> some View {
        Card(padding: 14) {
            HStack(spacing: 12) {
                AssetIcon(code: a.code, kind: prices.prices[a.code]?.kind ?? "currency", size: 34)
                VStack(alignment: .leading, spacing: 3) {
                    Text(prices.prices[a.code]?.displayName ?? a.code)
                        .font(.system(size: 15, weight: .semibold)).foregroundStyle(Theme.text)
                    HStack(spacing: 6) {
                        Image(systemName: a.direction == "above" ? "arrow.up.right" : "arrow.down.right")
                            .font(.system(size: 10, weight: .bold))
                            .foregroundStyle(a.direction == "above" ? Theme.up : Theme.down)
                        Text(a.direction == "above" ? "Üstüne çıkınca" : "Altına inince")
                            .font(.system(size: 12)).foregroundStyle(Theme.textDim)
                    }
                }
                Spacer()
                VStack(alignment: .trailing, spacing: 4) {
                    Text("₺\(Fmt.compactPrice(a.threshold))")
                        .font(Theme.number(15, weight: .semibold)).foregroundStyle(Theme.text)
                    Button(role: .destructive) {
                        Task { await delete(a.id) }
                    } label: {
                        Image(systemName: "trash").font(.system(size: 13)).foregroundStyle(Theme.down)
                    }
                }
            }
        }
    }

    private var emptyState: some View {
        VStack(spacing: 10) {
            Image(systemName: "bell.slash").font(.system(size: 34)).foregroundStyle(Theme.textFaint)
            Text("Alarm yok").font(.system(size: 16, weight: .semibold)).foregroundStyle(Theme.text)
            Text("Piyasa sekmesinden bir ürün açıp alarm kurabilirsiniz.")
                .font(.system(size: 13)).foregroundStyle(Theme.textDim)
                .multilineTextAlignment(.center).padding(.horizontal, 40)
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity)
    }

    private func load() async {
        loading = true; defer { loading = false }
        if let a = try? await APIClient(token: auth.accessToken).alerts() { alerts = a }
    }

    private func delete(_ id: String) async {
        try? await APIClient(token: auth.accessToken).deleteAlert(id)
        await load()
    }
}

// Shared compact screen header used by secondary tabs.
struct ScreenHeader: View {
    let title: String
    var subtitle: String?
    var body: some View {
        HStack {
            VStack(alignment: .leading, spacing: 2) {
                Text(title).font(.system(size: 26, weight: .bold)).foregroundStyle(Theme.text)
                if let subtitle {
                    Text(subtitle).font(.system(size: 12)).foregroundStyle(Theme.textDim)
                }
            }
            Spacer()
        }
        .padding(.horizontal, 16).padding(.top, 8).padding(.bottom, 8)
    }
}
