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
                                sectionLabel("UYGULAMA")
                                infoRow("Sürüm", AppConfig.appVersion)
                                Divider().overlay(Theme.stroke)
                                infoRow("İzlenen varlık", "\(prices.prices.count)")
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
