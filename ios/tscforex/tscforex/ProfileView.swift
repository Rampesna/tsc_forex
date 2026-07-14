//
//  ProfileView.swift
//  User profile: avatar upload (shared object storage), display name edit.
//

import SwiftUI
import PhotosUI
import UIKit

struct ProfileView: View {
    @EnvironmentObject var auth: AuthStore
    @State private var name = ""
    @State private var pickerItem: PhotosPickerItem?
    @State private var uploading = false
    @State private var savedName = false

    var body: some View {
        ZStack {
            Theme.bg.ignoresSafeArea()
            VStack(spacing: 0) {
                ScreenHeader(title: "Profil", subtitle: "Hesap bilgileri")
                ScrollView {
                    VStack(spacing: 16) {
                        avatarCard
                        nameCard
                        linkedAccountsCard
                        passkeyCard
                    }
                    .padding(.horizontal, 16).padding(.top, 8).padding(.bottom, 40)
                }
            }
        }
        .onAppear {
            name = auth.displayName
            Task {
                await auth.loadProfile(); name = auth.displayName
                await auth.loadLinkedProviders()
            }
        }
        .onChange(of: pickerItem) { _, newItem in
            guard let newItem else { return }
            Task {
                uploading = true
                defer { uploading = false }
                if let data = try? await newItem.loadTransferable(type: Data.self),
                   let img = downscaledJPEG(data) {
                    await auth.uploadAvatar(img)
                }
            }
        }
    }

    private var avatarCard: some View {
        Card {
            VStack(spacing: 14) {
                ZStack(alignment: .bottomTrailing) {
                    avatarImage
                    PhotosPicker(selection: $pickerItem, matching: .images) {
                        Image(systemName: "camera.fill")
                            .font(.system(size: 13, weight: .bold))
                            .foregroundStyle(.black)
                            .padding(9)
                            .background(Theme.accent, in: Circle())
                            .overlay(Circle().stroke(Theme.bg, lineWidth: 3))
                    }
                }
                if uploading {
                    HStack(spacing: 6) {
                        ProgressView().tint(Theme.accent).scaleEffect(0.8)
                        Text("Yükleniyor…").font(.system(size: 12)).foregroundStyle(Theme.textDim)
                    }
                }
                Text(auth.displayName.isEmpty ? (auth.email ?? "Kullanıcı") : auth.displayName)
                    .font(.system(size: 18, weight: .bold)).foregroundStyle(Theme.text)
                if let email = auth.email, !email.isEmpty {
                    Text(email).font(.system(size: 13)).foregroundStyle(Theme.textDim)
                }
            }
            .frame(maxWidth: .infinity)
            .padding(.vertical, 6)
        }
    }

    @ViewBuilder
    private var avatarImage: some View {
        let size: CGFloat = 96
        if let url = URL(string: auth.avatarURL), !auth.avatarURL.isEmpty {
            AsyncImage(url: url) { phase in
                switch phase {
                case .success(let img): img.resizable().scaledToFill()
                default: placeholderAvatar
                }
            }
            .frame(width: size, height: size)
            .clipShape(Circle())
            .overlay(Circle().stroke(Theme.stroke, lineWidth: 1))
        } else {
            placeholderAvatar.frame(width: size, height: size)
        }
    }

    private var placeholderAvatar: some View {
        ZStack {
            Circle().fill(Theme.accentSoft)
            Image(systemName: "person.fill")
                .font(.system(size: 40)).foregroundStyle(Theme.accent)
        }
    }

    private var nameCard: some View {
        Card {
            VStack(alignment: .leading, spacing: 12) {
                sectionLabel("GÖRÜNEN AD")
                HStack(spacing: 10) {
                    Image(systemName: "person").foregroundStyle(Theme.textDim)
                    TextField("", text: $name,
                              prompt: Text("Adınız").foregroundColor(Theme.textFaint))
                        .foregroundStyle(Theme.text)
                }
                .padding(12)
                .background(Theme.bgElevated, in: RoundedRectangle(cornerRadius: 10))

                Button {
                    Task { await auth.updateDisplayName(name); savedName = true; Haptics.success() }
                } label: {
                    Text(savedName ? "Kaydedildi ✓" : "Kaydet")
                }
                .buttonStyle(FilledButtonStyle())
                .disabled(name.trimmingCharacters(in: .whitespaces).isEmpty)
            }
        }
        .onChange(of: name) { _, _ in savedName = false }
    }

    @State private var showPasskeys = false

    private var passkeyCard: some View {
        Card {
            VStack(alignment: .leading, spacing: 10) {
                sectionLabel("GÜVENLİK")
                Text("Parolasız, hızlı ve güvenli giriş için geçiş anahtarlarını (passkey) yönetin; ekleyin, yeniden adlandırın veya silin.")
                    .font(.system(size: 12)).foregroundStyle(Theme.textDim)
                if let msg = auth.errorMessage, !msg.isEmpty {
                    Text(msg)
                        .font(.system(size: 12))
                        .foregroundStyle(Theme.down)
                        .frame(maxWidth: .infinity, alignment: .leading)
                        .padding(10)
                        .background(Theme.down.opacity(0.12), in: RoundedRectangle(cornerRadius: 10))
                }
                Button {
                    auth.errorMessage = nil
                    showPasskeys = true
                } label: {
                    Label("Geçiş anahtarlarını yönet", systemImage: "person.badge.key")
                }
                .buttonStyle(FilledButtonStyle(color: Theme.accentSoft, textColor: Theme.accent))
            }
        }
        .sheet(isPresented: $showPasskeys) {
            PasskeyManagerView().environmentObject(auth)
        }
    }

    @State private var linkingGoogle = false
    @State private var linkingApple = false

    private var linkedAccountsCard: some View {
        Card {
            VStack(alignment: .leading, spacing: 12) {
                sectionLabel("BAĞLI HESAPLAR")
                Text("Aynı hesaba birden fazla giriş yöntemi bağlayabilirsin.")
                    .font(.system(size: 12)).foregroundStyle(Theme.textDim)

                providerRow(
                    name: "Google", system: "g.circle.fill", tint: Color(hex: 0x4285F4),
                    linked: auth.linkedProviders.contains("google"),
                    busy: linkingGoogle
                ) {
                    linkingGoogle = true
                    Task { _ = await auth.linkGoogle(); linkingGoogle = false }
                }

                providerRow(
                    name: "Apple", system: "apple.logo", tint: Theme.text,
                    linked: auth.linkedProviders.contains("apple"),
                    busy: linkingApple
                ) {
                    linkingApple = true
                    Task { _ = await auth.linkApple(); linkingApple = false }
                }

                if auth.linkedProviders.contains("local") {
                    HStack(spacing: 10) {
                        Image(systemName: "envelope.fill").foregroundStyle(Theme.textDim).frame(width: 24)
                        Text("E-posta / Parola").font(.system(size: 14, weight: .semibold)).foregroundStyle(Theme.text)
                        Spacer()
                        linkedBadge
                    }
                }
            }
        }
    }

    @ViewBuilder
    private func providerRow(name: String, system: String, tint: Color,
                             linked: Bool, busy: Bool, action: @escaping () -> Void) -> some View {
        HStack(spacing: 10) {
            Image(systemName: system).foregroundStyle(tint).frame(width: 24)
            Text(name).font(.system(size: 14, weight: .semibold)).foregroundStyle(Theme.text)
            Spacer()
            if busy {
                ProgressView().tint(Theme.accent).scaleEffect(0.8)
            } else if linked {
                linkedBadge
            } else {
                Button(action: action) {
                    Text("Bağla").font(.system(size: 13, weight: .bold)).foregroundStyle(Theme.accent)
                        .padding(.horizontal, 14).padding(.vertical, 7)
                        .background(Theme.accentSoft, in: Capsule())
                }
            }
        }
    }

    private var linkedBadge: some View {
        HStack(spacing: 4) {
            Image(systemName: "checkmark.circle.fill")
            Text("Bağlı")
        }
        .font(.system(size: 12, weight: .bold)).foregroundStyle(Theme.up)
    }

    private func sectionLabel(_ t: String) -> some View {
        Text(t).font(.system(size: 11, weight: .bold)).foregroundStyle(Theme.textDim).tracking(1)
    }

    /// Re-encodes to JPEG and caps the largest side at ~512px to keep uploads small.
    private func downscaledJPEG(_ data: Data, maxSide: CGFloat = 512) -> Data? {
        guard let ui = UIImage(data: data) else { return nil }
        let scale = min(1, maxSide / max(ui.size.width, ui.size.height))
        let target = CGSize(width: ui.size.width * scale, height: ui.size.height * scale)
        let renderer = UIGraphicsImageRenderer(size: target)
        let resized = renderer.image { _ in ui.draw(in: CGRect(origin: .zero, size: target)) }
        return resized.jpegData(compressionQuality: 0.85)
    }
}
