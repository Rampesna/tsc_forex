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

    var body: some View {
        ZStack {
            Theme.bg.ignoresSafeArea()
            VStack(spacing: 0) {
                ScreenHeader(title: "Ayarlar", subtitle: "Sunucu & hesap")
                ScrollView {
                    VStack(spacing: 14) {
                        Card {
                            VStack(alignment: .leading, spacing: 12) {
                                sectionLabel("BAĞLANTI")
                                HStack {
                                    Text("Sunucu durumu").font(.system(size: 13)).foregroundStyle(Theme.textDim)
                                    Spacer()
                                    ConnectionDot(connected: prices.connected)
                                }
                            }
                        }

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

                        if let email = auth.email {
                            Card {
                                VStack(alignment: .leading, spacing: 12) {
                                    sectionLabel("HESAP")
                                    HStack {
                                        Image(systemName: "person.circle.fill").foregroundStyle(Theme.accent)
                                        Text(email).font(.system(size: 14)).foregroundStyle(Theme.text)
                                        Spacer()
                                    }
                                    Button(role: .destructive) {
                                        prices.disconnect(); auth.logout()
                                    } label: {
                                        Label("Çıkış Yap", systemImage: "arrow.right.square")
                                    }
                                    .buttonStyle(FilledButtonStyle(color: Theme.down.opacity(0.16), textColor: Theme.down))
                                }
                            }
                        }
                    }
                    .padding(.horizontal, 16).padding(.top, 8).padding(.bottom, 100)
                }
            }
        }
        .onChange(of: host) { _, _ in saved = false }
    }

    private func sectionLabel(_ t: String) -> some View {
        Text(t).font(.system(size: 11, weight: .bold)).foregroundStyle(Theme.textDim).tracking(1)
    }
}
