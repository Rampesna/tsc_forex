//
//  SettingsView.swift
//  Backend server address + account, themed.
//

import SwiftUI

struct SettingsView: View {
    @EnvironmentObject var auth: AuthStore
    @EnvironmentObject var prices: PriceStore
    @State private var host = AppConfig.host
    @State private var saved = false
    @State private var showProfile = false
    @State private var confirmDelete = false
    @State private var deleting = false
    @State private var deleteError: String?

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
                        if AppConfig.allowsServerOverride {
                            Card {
                                VStack(alignment: .leading, spacing: 12) {
                                    sectionLabel("GELİŞTİRİCİ · SUNUCU")
                                    HStack(spacing: 10) {
                                        Image(systemName: "network").foregroundStyle(Theme.textDim)
                                        TextField("", text: $host,
                                                  prompt: Text("host:port").foregroundColor(Theme.textFaint))
                                            .textInputAutocapitalization(.never)
                                            .autocorrectionDisabled()
                                            .keyboardType(.URL)
                                            .foregroundStyle(Theme.text)
                                    }
                                    .padding(12)
                                    .background(Theme.bgElevated, in: RoundedRectangle(cornerRadius: 10))

                                    Button {
                                        AppConfig.host = host
                                        prices.disconnect()
                                        prices.connect(token: auth.accessToken)
                                        saved = true
                                        Haptics.success()
                                    } label: {
                                        Text(saved ? "Kaydedildi ✓" : "Kaydet ve Yeniden Bağlan")
                                    }
                                    .buttonStyle(FilledButtonStyle(color: Theme.surfaceHi, textColor: Theme.text))

                                    Button {
                                        host = AppConfig.defaultHost
                                        AppConfig.host = host
                                        prices.disconnect()
                                        prices.connect(token: auth.accessToken)
                                        saved = true
                                        Haptics.tap()
                                    } label: {
                                        Label("Otomatik adres (.local)", systemImage: "wand.and.stars")
                                            .font(.system(size: 13, weight: .semibold))
                                    }
                                    .buttonStyle(FilledButtonStyle(color: Theme.accentSoft, textColor: Theme.accent))

                                    Text("Yalnızca geliştirici derlemesinde görünür.")
                                        .font(.system(size: 11)).foregroundStyle(Theme.textFaint)
                                }
                            }
                        }

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
                                    prices.disconnect(); auth.logout()
                                } label: {
                                    Label("Çıkış Yap", systemImage: "arrow.right.square")
                                }
                                .buttonStyle(FilledButtonStyle(color: Theme.down.opacity(0.16), textColor: Theme.down))
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
                                Text("Hesabını silersen izleme listen, alarmların, portföyün, bildirimlerin ve ödül verilerin kalıcı olarak kaldırılır. Bu işlem geri alınamaz.")
                                    .font(.system(size: 12)).foregroundStyle(Theme.textDim)
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
        .onChange(of: host) { _, _ in saved = false }
        .sheet(isPresented: $showProfile) {
            NavigationStack { ProfileView() }.preferredColorScheme(.dark)
        }
        .alert("Hesabını silmek istediğinden emin misin?", isPresented: $confirmDelete) {
            Button("Vazgeç", role: .cancel) {}
            Button("Hesabı Sil", role: .destructive) { Task { await deleteAccount() } }
        } message: {
            Text("Tüm verilerin kalıcı olarak silinecek. Bu işlem geri alınamaz.")
        }
    }

    private func deleteAccount() async {
        deleting = true; deleteError = nil
        do {
            try await APIClient(token: auth.accessToken).deleteAccount()
            prices.disconnect()
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
