//
//  SettingsView.swift
//  Backend server address + account, themed.
//

import SwiftUI

struct SettingsView: View {
    @EnvironmentObject var auth: AuthStore
    @EnvironmentObject var prices: PriceStore
    @EnvironmentObject var billing: BillingStore
    @State private var showProfile = false
    @State private var confirmDelete = false
    @State private var deleting = false
    @State private var deleteError: String?
    @State private var showPaywall = false
    @State private var showResearchArchive = false
    @State private var acceptedAIDisclosure = false

    private let privacyURL = URL(string: "https://tsc.talhacan.com/privacy")!
    private let termsURL = URL(string: "https://tsc.talhacan.com/terms")!
    private let supportURL = URL(string: "https://tsc.talhacan.com/support")!

    var body: some View {
        ZStack {
            Theme.bg.ignoresSafeArea()
            VStack(spacing: 0) {
                ScreenHeader(title: "Ayarlar", subtitle: "Hesap & uygulama")
                ScrollView {
                    VStack(spacing: 14) {
                        Card {
                            VStack(alignment: .leading, spacing: 12) {
                                sectionLabel("HESAP")
                                Button { showProfile = true } label: {
                                    HStack(spacing: 12) {
                                        avatarThumb
                                        VStack(alignment: .leading, spacing: 2) {
                                            Text(auth.displayName.isEmpty ? (auth.email ?? "Kullanıcı") : auth.displayName)
                                                .font(.system(size: 15, weight: .semibold))
                                                .foregroundStyle(Theme.text)
                                            Text("Profili düzenle")
                                                .font(.system(size: 12)).foregroundStyle(Theme.textDim)
                                        }
                                        Spacer()
                                        Image(systemName: "chevron.right").foregroundStyle(Theme.textFaint)
                                    }
                                }
                                Button(role: .destructive) {
                                    prices.disconnect(); billing.clearSession(); auth.logout()
                                } label: {
                                    Label("Çıkış Yap", systemImage: "arrow.right.square")
                                }
                                .buttonStyle(FilledButtonStyle(color: Theme.down.opacity(0.16), textColor: Theme.down))
                            }
                        }

                        Card {
                            VStack(alignment: .leading, spacing: 12) {
                                sectionLabel("ABONELİK")
                                HStack {
                                    VStack(alignment: .leading, spacing: 2) {
                                        Text(billing.hasActiveSubscription ? "Abonelik etkin" : "Ücretsiz plan")
                                            .font(.system(size: 15, weight: .semibold)).foregroundStyle(Theme.text)
                                        Text(billing.serverStatus?.plan.name ?? "AI araştırma kullanımını planına göre yönet")
                                            .font(.system(size: 11)).foregroundStyle(Theme.textDim)
                                    }
                                    Spacer()
                                    if billing.serverStatus?.plan.code == "free" && !billing.freeTrialReady {
                                        Text("Doğrulama bekleniyor")
                                            .font(.system(size: 10, weight: .bold)).foregroundStyle(Theme.textDim)
                                    } else if let remaining = billing.aiRemaining {
                                        Text("\(remaining) hak")
                                            .font(.system(size: 11, weight: .bold)).foregroundStyle(Theme.accent)
                                    }
                                }
                                Button(billing.hasActiveSubscription ? "Planları Gör" : "Planları İncele") {
                                    billing.message = nil
                                    showPaywall = true
                                }
                                .buttonStyle(FilledButtonStyle())
                                Button("Aboneliği Apple'da Yönet") {
                                    Task { await billing.manageSubscriptions() }
                                }
                                .font(.system(size: 13, weight: .semibold)).foregroundStyle(Theme.accent)
                                Button("Satın Almaları Geri Yükle") {
                                    Task { await billing.restore(token: auth.accessToken) }
                                }
                                .font(.system(size: 13, weight: .semibold)).foregroundStyle(Theme.accent)
                                Button {
                                    showResearchArchive = true
                                } label: {
                                    HStack {
                                        Label("Araştırma Arşivi", systemImage: "archivebox")
                                        Spacer()
                                        Image(systemName: "chevron.right")
                                    }
                                }
                                .font(.system(size: 13, weight: .semibold)).foregroundStyle(Theme.accent)
                                if let error = billing.catalogError {
                                    billingStatusRow(error, retry: true)
                                }
                                if let message = billing.message {
                                    billingStatusRow(message, retry: false)
                                }
                            }
                        }

                        Card {
                            VStack(alignment: .leading, spacing: 10) {
                                sectionLabel("AI VE VERİ KULLANIMI")
                                Text("AI araştırmasında piyasa verileri, açık web kaynakları ve tek yönlü türetilen takma adlı güvenlik kimliği analiz ve kötüye kullanım önleme için OpenAI'a gönderilir; kişisel profil ve portföy tutarları gönderilmez.")
                                    .font(.system(size: 12)).foregroundStyle(Theme.textDim)
                                HStack {
                                    VStack(alignment: .leading, spacing: 2) {
                                        Text("AI araştırma onayı").font(.system(size: 13, weight: .semibold)).foregroundStyle(Theme.text)
                                        Text(acceptedAIDisclosure ? "Onay verildi" : "Onay verilmedi")
                                            .font(.system(size: 11)).foregroundStyle(Theme.textDim)
                                    }
                                    Spacer()
                                    if acceptedAIDisclosure {
                                        Button("Onayı Geri Çek") {
                                            AIResearchConsentStore.setAccepted(false, for: auth.userID)
                                            acceptedAIDisclosure = false
                                        }
                                            .font(.system(size: 11, weight: .bold)).foregroundStyle(Theme.down)
                                    }
                                }
                            }
                        }

                        Card {
                            VStack(alignment: .leading, spacing: 12) {
                                sectionLabel("YASAL")
                                Link(destination: privacyURL) { legalRow("Gizlilik Politikası", "lock.shield") }
                                Divider().overlay(Theme.stroke)
                                Link(destination: termsURL) { legalRow("Kullanım Şartları", "doc.text") }
                                Divider().overlay(Theme.stroke)
                                Link(destination: supportURL) { legalRow("Destek & Yardım", "questionmark.circle") }
                            }
                        }

                        Card {
                            VStack(alignment: .leading, spacing: 12) {
                                sectionLabel("UYGULAMA")
                                infoRow("Sürüm", AppConfig.appVersion)
                                Divider().overlay(Theme.stroke)
                                infoRow("İzlenen varlık", "\(prices.prices.count)")
                            }
                        }

                        // Account deletion — required by App Store guidelines when
                        // the app supports account creation.
                        Card {
                            VStack(alignment: .leading, spacing: 10) {
                                sectionLabel("TEHLİKELİ BÖLGE")
                                Text("Hesabını silersen izleme listen, alarmların, portföyün, bildirimlerin ve ödül verilerin kalıcı olarak kaldırılır. Apple aboneliğin otomatik iptal edilmez; iptal etmezsen faturalama sürebilir.")
                                    .font(.system(size: 12)).foregroundStyle(Theme.textDim)
                                Button("Önce Aboneliği Apple'da Yönet") {
                                    Task { await billing.manageSubscriptions() }
                                }
                                .font(.system(size: 12, weight: .bold)).foregroundStyle(Theme.accent)
                                Button(role: .destructive) {
                                    confirmDelete = true
                                } label: {
                                    HStack {
                                        if deleting { ProgressView().tint(Theme.down) }
                                        Label(deleting ? "Siliniyor…" : "Hesabı Sil", systemImage: "trash")
                                    }
                                }
                                .disabled(deleting)
                                .buttonStyle(FilledButtonStyle(color: Theme.down.opacity(0.16), textColor: Theme.down))
                                if let deleteError {
                                    Text(deleteError).font(.system(size: 11)).foregroundStyle(Theme.down)
                                }
                            }
                        }

                        Text("TSC Forex · Canlı Altın & Döviz")
                            .font(.system(size: 11)).foregroundStyle(Theme.textFaint)
                            .frame(maxWidth: .infinity)
                            .padding(.top, 4)
                    }
                    .padding(.horizontal, 16).padding(.top, 8).padding(.bottom, 120)
                }
            }
        }
        .sheet(isPresented: $showProfile) {
            NavigationStack { ProfileView() }.preferredColorScheme(.dark)
        }
        .sheet(isPresented: $showPaywall) {
            PaywallView()
                .environmentObject(auth)
                .environmentObject(billing)
        }
        .sheet(isPresented: $showResearchArchive) {
            ResearchArchiveView()
                .environmentObject(auth)
                .environmentObject(billing)
        }
        .task {
            await billing.refresh(token: auth.accessToken)
            await auth.loadLinkedProviders()
        }
        .task(id: auth.userID) {
            acceptedAIDisclosure = AIResearchConsentStore.isAccepted(for: auth.userID)
        }
        .alert("Hesabını silmek istediğinden emin misin?", isPresented: $confirmDelete) {
            Button("Vazgeç", role: .cancel) {}
            Button("Hesabı Sil", role: .destructive) { Task { await deleteAccount() } }
        } message: {
            Text("Hesabın ve kullanıcıya bağlı içeriklerin kalıcı olarak silinir. Tek ücretsiz araştırma hakkının kötüye kullanılmasını önleyen tek yönlü cihaz kaydı ile aktif aboneliği daha sonra yalnız sana geri bağlamaya yarayan tek yönlü Apple işlem/kurtarma kaydı korunabilir. Ham kurtarma anahtarı sunucuda saklanmaz. Apple aboneliğin iptal olmaz ve sen iptal edene kadar faturalama sürebilir.")
        }
    }

    private func billingStatusRow(_ text: String, retry: Bool) -> some View {
        VStack(alignment: .leading, spacing: 8) {
            Text(text)
                .font(.system(size: 11))
                .foregroundStyle(retry ? Theme.down : Theme.textDim)
                .fixedSize(horizontal: false, vertical: true)
            HStack {
                if retry {
                    Button("Tekrar Dene") {
                        Task { await billing.refresh(token: auth.accessToken) }
                    }
                }
                Spacer()
                if !retry {
                    Button("Kapat") { billing.message = nil }
                }
            }
            .font(.system(size: 11, weight: .bold))
            .foregroundStyle(Theme.accent)
        }
        .padding(10)
        .background(Theme.bgElevated, in: RoundedRectangle(cornerRadius: 9))
    }

    private func deleteAccount() async {
        deleting = true; deleteError = nil
        do {
            if auth.userID == nil { await auth.loadProfile() }
            await auth.loadLinkedProviders()
            var authorizationCode: String?
            if auth.linkedProviders.contains("apple") {
                do {
                    authorizationCode = try await AppleAuth.shared.signIn().authorizationCode
                } catch AppleAuthError.cancelled {
                    deleteError = "Hesap silme iptal edildi. Apple ile yeniden doğrulama tamamlanmadan hesap silinemez."
                    deleting = false
                    return
                }
            }
            guard let userID = auth.userID else { throw APIError.invalidResponse }
            let recoveryToken = try await billing.recoveryTokenForAccountDeletion(userID: userID)
            try await APIClient(token: auth.accessToken)
                .deleteAccount(
                    appleAuthorizationCode: authorizationCode,
                    subscriptionRecoveryToken: recoveryToken
                )
            PendingOperationStore.clearAll(userID: userID)
            prices.disconnect()
            billing.clearSession()
            auth.logout()
            Haptics.success()
        } catch {
            deleteError = "Silme başarısız: \(error.localizedDescription)"
            Haptics.error()
        }
        deleting = false
    }

    private func legalRow(_ title: String, _ icon: String) -> some View {
        HStack(spacing: 12) {
            Image(systemName: icon).foregroundStyle(Theme.accent).frame(width: 22)
            Text(title).font(.system(size: 14)).foregroundStyle(Theme.text)
            Spacer()
            Image(systemName: "arrow.up.right").font(.system(size: 11)).foregroundStyle(Theme.textFaint)
        }
    }

    @ViewBuilder
    private var avatarThumb: some View {
        if let url = URL(string: auth.avatarURL), !auth.avatarURL.isEmpty {
            AsyncImage(url: url) { phase in
                if case .success(let img) = phase { img.resizable().scaledToFill() }
                else { Image(systemName: "person.circle.fill").resizable().foregroundStyle(Theme.accent) }
            }
            .frame(width: 44, height: 44).clipShape(Circle())
        } else {
            Image(systemName: "person.circle.fill")
                .resizable().frame(width: 44, height: 44).foregroundStyle(Theme.accent)
        }
    }

    private func sectionLabel(_ t: String) -> some View {
        Text(t).font(.system(size: 11, weight: .bold)).foregroundStyle(Theme.textDim).tracking(1)
    }

    private func infoRow(_ label: String, _ value: String) -> some View {
        HStack {
            Text(label).font(.system(size: 13)).foregroundStyle(Theme.textDim)
            Spacer()
            Text(value).font(.system(size: 13, weight: .semibold)).foregroundStyle(Theme.text)
        }
    }
}
