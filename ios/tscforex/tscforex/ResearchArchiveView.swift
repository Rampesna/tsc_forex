import SwiftUI

struct ResearchArchiveView: View {
    @EnvironmentObject private var auth: AuthStore
    @EnvironmentObject private var billing: BillingStore
    @State private var items: [ResearchArchiveItem] = []
    @State private var loading = true
    @State private var errorMessage: String?
    @State private var showPaywall = false

    var body: some View {
        NavigationStack {
            ZStack {
                Theme.bg.ignoresSafeArea()
                if loading, items.isEmpty {
                    ProgressView("Araştırmalar yükleniyor…")
                        .tint(Theme.accent)
                        .foregroundStyle(Theme.textDim)
                } else {
                    ScrollView {
                        LazyVStack(spacing: 12) {
                            if let errorMessage {
                                Card {
                                    Label(errorMessage, systemImage: "exclamationmark.triangle")
                                        .font(.system(size: 12))
                                        .foregroundStyle(Theme.down)
                                }
                            }
                            if items.isEmpty {
                                emptyCard
                            } else {
                                ForEach(items) { item in
                                    NavigationLink {
                                        ResearchArchiveDetailView(item: item)
                                    } label: {
                                        archiveRow(item)
                                    }
                                    .buttonStyle(.plain)
                                }
                            }
                        }
                        .padding(16)
                        .padding(.bottom, 32)
                    }
                    .refreshable { await load() }
                }
            }
            .navigationTitle("Araştırma Arşivi")
            .navigationBarTitleDisplayMode(.inline)
        }
        .task { await load() }
        .sheet(isPresented: $showPaywall) {
            PaywallView()
                .environmentObject(auth)
                .environmentObject(billing)
        }
        .preferredColorScheme(.dark)
    }

    private func archiveRow(_ item: ResearchArchiveItem) -> some View {
        Card {
            HStack(spacing: 12) {
                Image(systemName: "doc.text.magnifyingglass")
                    .font(.system(size: 20, weight: .semibold))
                    .foregroundStyle(Theme.accent)
                    .frame(width: 42, height: 42)
                    .background(Theme.accentSoft, in: RoundedRectangle(cornerRadius: 11))
                VStack(alignment: .leading, spacing: 4) {
                    HStack {
                        Text(item.code)
                            .font(.system(size: 10, weight: .bold))
                            .foregroundStyle(Theme.accent)
                        Text(item.generatedAt.formatted(date: .abbreviated, time: .shortened))
                            .font(.system(size: 9)).foregroundStyle(Theme.textFaint)
                    }
                    Text(item.payload.title)
                        .font(.system(size: 14, weight: .bold))
                        .foregroundStyle(Theme.text)
                        .lineLimit(2)
                    Text(item.payload.summary)
                        .font(.system(size: 10))
                        .foregroundStyle(Theme.textDim)
                        .lineLimit(2)
                    Text("\(item.expiresAt.formatted(date: .abbreviated, time: .omitted)) tarihine kadar arşivde")
                        .font(.system(size: 9)).foregroundStyle(Theme.textFaint)
                }
                Spacer(minLength: 4)
                Image(systemName: "chevron.right")
                    .font(.system(size: 10, weight: .bold))
                    .foregroundStyle(Theme.textFaint)
            }
        }
    }

    private var emptyCard: some View {
        Card {
            VStack(spacing: 12) {
                Image(systemName: "archivebox")
                    .font(.system(size: 34)).foregroundStyle(Theme.textFaint)
                Text("Arşiviniz henüz boş")
                    .font(.system(size: 17, weight: .bold)).foregroundStyle(Theme.text)
                Text("Paketiniz etkinken oluşturduğunuz kaynaklı piyasa araştırmaları, planınızın saklama süresi boyunca burada görünür.")
                    .font(.system(size: 12)).foregroundStyle(Theme.textDim)
                    .multilineTextAlignment(.center)
                if !billing.hasActiveSubscription {
                    Button("Planları İncele") { showPaywall = true }
                        .buttonStyle(FilledButtonStyle())
                        .frame(maxWidth: 230)
                }
            }
            .frame(maxWidth: .infinity)
            .padding(.vertical, 18)
        }
    }

    private func load() async {
        guard let token = auth.accessToken else {
            loading = false
            return
        }
        loading = true
        errorMessage = nil
        do { items = try await APIClient(token: token).researchHistory() }
        catch { errorMessage = error.localizedDescription }
        loading = false
    }
}

private struct ResearchArchiveDetailView: View {
    let item: ResearchArchiveItem

    var body: some View {
        ScrollView {
            VStack(spacing: 12) {
                Card {
                    VStack(alignment: .leading, spacing: 10) {
                        HStack {
                            Text(item.code).font(.system(size: 11, weight: .bold)).foregroundStyle(Theme.accent)
                            Spacer()
                            Text(item.payload.stanceLabel)
                                .font(.system(size: 10, weight: .bold)).foregroundStyle(Theme.accent)
                                .padding(.horizontal, 8).padding(.vertical, 4)
                                .background(Theme.accentSoft, in: Capsule())
                        }
                        Text(item.payload.title).font(.system(size: 20, weight: .bold)).foregroundStyle(Theme.text)
                        Text(item.payload.summary).font(.system(size: 13)).foregroundStyle(Theme.textDim)
                        Text("Oluşturulma: \(item.generatedAt.formatted(date: .long, time: .shortened))")
                            .font(.system(size: 9)).foregroundStyle(Theme.textFaint)
                    }
                }
                bulletCard("Öne Çıkan Etkenler", icon: "list.bullet", items: item.payload.factors)
                if let values = item.payload.macroDrivers, !values.isEmpty {
                    bulletCard("Makroekonomik Etkenler", icon: "building.columns", items: values)
                }
                if let values = item.payload.geopoliticalRisks, !values.isEmpty {
                    bulletCard("Jeopolitik Riskler", icon: "globe.europe.africa", items: values)
                }
                if let values = item.payload.crossAssetAnalysis, !values.isEmpty {
                    bulletCard("Çapraz Varlık İlişkileri", icon: "arrow.triangle.branch", items: values)
                }
                if let horizons = item.payload.timeHorizons, !horizons.isEmpty {
                    Card {
                        VStack(alignment: .leading, spacing: 10) {
                            sectionTitle("Zaman Dilimleri", icon: "clock.arrow.circlepath")
                            ForEach(horizons, id: \.self) { horizon in
                                VStack(alignment: .leading, spacing: 3) {
                                    Text("\(horizon.horizon) · \(horizon.outlook)")
                                        .font(.system(size: 12, weight: .bold)).foregroundStyle(Theme.text)
                                    Text(horizon.rationale).font(.system(size: 10)).foregroundStyle(Theme.textDim)
                                }
                                .padding(10).background(Theme.surfaceHi, in: RoundedRectangle(cornerRadius: 9))
                            }
                        }
                    }
                }
                if let scenarios = item.payload.scenarios, !scenarios.isEmpty {
                    Card {
                        VStack(alignment: .leading, spacing: 10) {
                            sectionTitle("Senaryolar", icon: "point.3.connected.trianglepath.dotted")
                            ForEach(scenarios, id: \.self) { scenario in
                                VStack(alignment: .leading, spacing: 3) {
                                    Text("\(scenario.name) · \(scenario.likelihood)")
                                        .font(.system(size: 12, weight: .bold)).foregroundStyle(Theme.text)
                                    Text("Tetikleyici: \(scenario.trigger)")
                                        .font(.system(size: 10)).foregroundStyle(Theme.textDim)
                                    Text(scenario.implication).font(.system(size: 10)).foregroundStyle(Theme.textDim)
                                }
                                .padding(10).background(Theme.surfaceHi, in: RoundedRectangle(cornerRadius: 9))
                            }
                        }
                    }
                }
                if let playbook = item.payload.whatIWouldDo {
                    Card {
                        VStack(alignment: .leading, spacing: 10) {
                            sectionTitle("Ben Olsam Ne Yapardım", icon: "person.crop.circle.badge.questionmark")
                            Text(playbook.intro).font(.system(size: 12, weight: .semibold)).foregroundStyle(Theme.text)
                            Text("Kısa vade: \(playbook.shortTerm)").font(.system(size: 11)).foregroundStyle(Theme.textDim)
                            Text("Orta vade: \(playbook.mediumTerm)").font(.system(size: 11)).foregroundStyle(Theme.textDim)
                            Text("Uzun vade: \(playbook.longTerm)").font(.system(size: 11)).foregroundStyle(Theme.textDim)
                        }
                    }
                }
                if let sources = item.payload.sources, !sources.isEmpty {
                    Card {
                        VStack(alignment: .leading, spacing: 10) {
                            sectionTitle("Kaynaklar", icon: "link")
                            ForEach(sources, id: \.self) { source in
                                if let url = URL(string: source.url) {
                                    Link(destination: url) {
                                        HStack {
                                            Text(source.title).font(.system(size: 11, weight: .semibold)).lineLimit(2)
                                            Spacer()
                                            Image(systemName: "arrow.up.right")
                                        }
                                        .foregroundStyle(Theme.accent)
                                    }
                                }
                            }
                        }
                    }
                }
                Card {
                    VStack(alignment: .leading, spacing: 6) {
                        sectionTitle("Risk Notu", icon: "exclamationmark.triangle")
                        Text(item.payload.riskNote).font(.system(size: 11)).foregroundStyle(Theme.textDim)
                        Text(item.payload.disclaimer).font(.system(size: 9)).foregroundStyle(Theme.textFaint)
                    }
                }
            }
            .padding(16)
            .padding(.bottom, 32)
        }
        .background(Theme.bg.ignoresSafeArea())
        .navigationTitle("Araştırma Detayı")
        .navigationBarTitleDisplayMode(.inline)
    }

    private func bulletCard(_ title: String, icon: String, items: [String]) -> some View {
        Card {
            VStack(alignment: .leading, spacing: 9) {
                sectionTitle(title, icon: icon)
                ForEach(items, id: \.self) { value in
                    Text("• \(value)").font(.system(size: 11)).foregroundStyle(Theme.textDim)
                }
            }
        }
    }

    private func sectionTitle(_ title: String, icon: String) -> some View {
        Label(title, systemImage: icon)
            .font(.system(size: 12, weight: .bold))
            .foregroundStyle(Theme.text)
    }
}
