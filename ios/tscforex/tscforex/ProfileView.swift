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
    @State private var passkeyDone = false

    var body: some View {
        ZStack {
            Theme.bg.ignoresSafeArea()
            VStack(spacing: 0) {
                ScreenHeader(title: "Profil", subtitle: "Hesap bilgileri")
                ScrollView {
                    VStack(spacing: 16) {
                        avatarCard
                        nameCard
                        passkeyCard
                        infoCard
                    }
                    .padding(.horizontal, 16).padding(.top, 8).padding(.bottom, 40)
                }
            }
        }
        .onAppear {
            name = auth.displayName
            Task { await auth.loadProfile(); name = auth.displayName }
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

    private var passkeyCard: some View {
        Card {
            VStack(alignment: .leading, spacing: 10) {
                sectionLabel("GÜVENLİK")
                Text("Bu cihazda parolasız, hızlı ve güvenli giriş için bir geçiş anahtarı (passkey) oluştur.")
                    .font(.system(size: 12)).foregroundStyle(Theme.textDim)
                Button {
                    Task {
                        let ok = await auth.registerPasskey()
                        if ok { passkeyDone = true; Haptics.success() }
                    }
                } label: {
                    Label(passkeyDone ? "Geçiş anahtarı eklendi ✓" : "Geçiş anahtarı oluştur",
                          systemImage: "person.badge.key")
                }
                .buttonStyle(FilledButtonStyle(color: Theme.accentSoft, textColor: Theme.accent))
            }
        }
    }

    private var infoCard: some View {
        Card {
            VStack(alignment: .leading, spacing: 10) {
                sectionLabel("HESAP")
                row(icon: "checkmark.seal", label: "Sağlayıcı", value: providerLabel)
            }
        }
    }

    private var providerLabel: String {
        // Best-effort from cached email domain; backend owns the truth.
        if auth.email?.contains("privaterelay.appleid.com") == true { return "Apple" }
        return auth.avatarURL.contains("googleusercontent") ? "Google" : "E-posta"
    }

    private func row(icon: String, label: String, value: String) -> some View {
        HStack {
            Image(systemName: icon).foregroundStyle(Theme.textDim).frame(width: 22)
            Text(label).font(.system(size: 13)).foregroundStyle(Theme.textDim)
            Spacer()
            Text(value).font(.system(size: 13, weight: .semibold)).foregroundStyle(Theme.text)
        }
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
