import StoreKit
import SwiftUI

struct PaywallView: View {
    @Environment(\.dismiss) private var dismiss
    @EnvironmentObject private var auth: AuthStore
    @EnvironmentObject private var billing: BillingStore

    var body: some View {
        NavigationStack {
            ScrollView {
                VStack(spacing: 16) {
                    Image(systemName: "sparkles.rectangle.stack.fill")
                        .font(.system(size: 44)).foregroundStyle(Theme.accent)
                    Text("AI Araştırma Planları")
                        .font(.system(size: 24, weight: .bold)).foregroundStyle(Theme.text)
                    Text("Daha fazla kaynaklı piyasa araştırması ve planınıza göre artan aylık kullanım hakkı.")
                        .font(.system(size: 13)).foregroundStyle(Theme.textDim)
                        .multilineTextAlignment(.center)

                    subscriptionDisclosure

                    if let message = billing.message {
                        inlineStatus(message, isError: false)
                    }

                    if billing.products.isEmpty {
                        Card {
                            VStack(spacing: 10) {
                                if billing.isLoading {
                                    ProgressView().tint(Theme.accent)
                                    Text("App Store seçenekleri yükleniyor…")
                                        .font(.system(size: 12)).foregroundStyle(Theme.textDim)
                                } else if let error = billing.catalogError {
                                    Text(L10n.text(error))
                                        .font(.system(size: 12)).foregroundStyle(Theme.down)
                                        .multilineTextAlignment(.center)
                                    Button("Tekrar Dene") {
                                        Task { await billing.refresh(token: auth.accessToken) }
                                    }
                                    .font(.system(size: 12, weight: .bold)).foregroundStyle(Theme.accent)
                                } else {
                                    Text("Abonelik seçenekleri bulunamadı.")
                                        .font(.system(size: 12)).foregroundStyle(Theme.textDim)
                                }
                            }.frame(maxWidth: .infinity)
                        }
                    } else {
                        ForEach(billing.products, id: \.id) { product in
                            productCard(product)
                        }
                    }

                    Button("Aboneliği Apple'da Yönet") {
                        Task { await billing.manageSubscriptions() }
                    }
                    .buttonStyle(FilledButtonStyle(color: Theme.surfaceHi, textColor: Theme.accent))

                    Button("Satın Almaları Geri Yükle") {
                        Task { await billing.restore(token: auth.accessToken) }
                    }
                    .font(.system(size: 13, weight: .bold)).foregroundStyle(Theme.accent)

                }
                .padding(20)
            }
            .background(Theme.bg.ignoresSafeArea())
            .navigationTitle("Abonelik")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .topBarTrailing) {
                    Button("Kapat") { dismiss() }.foregroundStyle(Theme.accent)
                }
            }
            .task {
                if auth.userID == nil { await auth.loadProfile() }
                await billing.refresh(token: auth.accessToken)
            }
        }
        .preferredColorScheme(.dark)
    }

    private func inlineStatus(_ text: String, isError: Bool) -> some View {
        Card {
            HStack(alignment: .top, spacing: 10) {
                Image(systemName: isError ? "exclamationmark.triangle" : "info.circle")
                    .foregroundStyle(isError ? Theme.down : Theme.accent)
                Text(L10n.text(text))
                    .font(.system(size: 11)).foregroundStyle(Theme.textDim)
                    .frame(maxWidth: .infinity, alignment: .leading)
                Button("Kapat") { billing.message = nil }
                    .font(.system(size: 11, weight: .bold)).foregroundStyle(Theme.accent)
            }
        }
    }

    private var subscriptionDisclosure: some View {
        VStack(alignment: .leading, spacing: 9) {
            Text("Ödeme Apple hesabınızdan tahsil edilir. Aylık abonelik, mevcut dönemin bitiminden en az 24 saat önce iptal edilmezse otomatik yenilenir; yenileme sırasında geçerli ücret Apple hesabınızdan tahsil edilir. Apple Hesabı > Abonelikler bölümünden yönetilebilir veya iptal edilebilir.")
                .font(.system(size: 10)).foregroundStyle(Theme.textDim)
                .fixedSize(horizontal: false, vertical: true)
            HStack(spacing: 18) {
                Link("Kullanım Şartları", destination: URL(string: "https://tsc.talhacan.com/terms")!)
                Link("Gizlilik Politikası", destination: URL(string: "https://tsc.talhacan.com/privacy")!)
            }
            .font(.system(size: 11, weight: .semibold)).foregroundStyle(Theme.accent)
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .padding(12)
        .background(Theme.bgElevated, in: RoundedRectangle(cornerRadius: 10))
    }

    private func productCard(_ product: Product) -> some View {
        // The backend entitlement is the account-scoped source of truth. A
        // StoreKit transaction may belong to another app account on this
        // device and must not make this account look subscribed.
        let selected = billing.isCurrentPlan(product.id)
        let catalogPlan = billing.plan(for: product.id)
        return Card {
            VStack(alignment: .leading, spacing: 9) {
                HStack {
                    VStack(alignment: .leading, spacing: 2) {
                        Text(catalogPlan?.name ?? product.displayName).font(.system(size: 17, weight: .bold)).foregroundStyle(Theme.text)
                        Text(product.description).font(.system(size: 11)).foregroundStyle(Theme.textDim)
                    }
                    Spacer()
                    Text("\(product.displayPrice) / ay")
                        .font(.system(size: 13, weight: .bold)).foregroundStyle(Theme.accent)
                }
                if let limits = billing.limits(for: product.id) {
                    VStack(alignment: .leading, spacing: 5) {
                        quota("AI araştırması", limits.aiResearch, suffix: "/ ay")
                        quota("Aktif alarm", limits.activeAlerts)
                        quota("İzleme listesi", limits.watchlist)
                        quota("Portföy", limits.portfolios)
                        quota("Portföy sağlık analizi", limits.portfolioHealth, suffix: "/ ay")
                        quota("Dışa aktarma", limits.exports, suffix: "/ ay")
                        quota("Araştırma arşivi", limits.researchArchiveDays, suffix: " gün")
                    }
                    .padding(10).background(Theme.bgElevated, in: RoundedRectangle(cornerRadius: 9))
                }
                Text("1 aylık abonelik · otomatik yenilenir · \(product.displayPrice)/ay")
                    .font(.system(size: 10, weight: .medium))
                    .foregroundStyle(Theme.textDim)
                    .fixedSize(horizontal: false, vertical: true)
                Button(selected ? "Etkin Plan" : "Aylık Abone Ol") {
                    Task {
                        await billing.purchase(
                            product, token: auth.accessToken, userID: auth.userID
                        )
                    }
                }
                .buttonStyle(FilledButtonStyle())
                .disabled(selected || billing.isLoading)
            }
        }
    }

    private func quota(_ label: String, _ value: Int, suffix: String = "") -> some View {
        HStack {
            Text(label).foregroundStyle(Theme.textDim)
            Spacer()
            Text("\(value)\(suffix)").fontWeight(.bold).foregroundStyle(Theme.text)
        }
        .font(.system(size: 10))
    }
}
