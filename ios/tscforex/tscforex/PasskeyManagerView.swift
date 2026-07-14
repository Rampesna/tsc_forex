//
//  PasskeyManagerView.swift
//  Lists the signed-in user's registered passkeys and lets them add, rename or
//  delete a credential. Backed by /me/passkeys (list), PATCH (rename) and
//  DELETE endpoints; new keys are created via the native WebAuthn ceremony.
//

import SwiftUI

struct PasskeyManagerView: View {
    @EnvironmentObject var auth: AuthStore
    @Environment(\.dismiss) private var dismiss

    @State private var passkeys: [Passkey] = []
    @State private var loading = true
    @State private var working = false
    @State private var error: String?

    // Rename flow
    @State private var renameTarget: Passkey?
    @State private var renameText = ""
    @State private var showRename = false

    // Delete flow
    @State private var deleteTarget: Passkey?
    @State private var showDelete = false

    var body: some View {
        NavigationStack {
            ZStack {
                Theme.bg.ignoresSafeArea()
                content
            }
            .navigationTitle("Geçiş Anahtarları")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .topBarLeading) {
                    Button("Kapat") { dismiss() }.foregroundStyle(Theme.textDim)
                }
                ToolbarItem(placement: .topBarTrailing) {
                    Button { Task { await addPasskey() } } label: {
                        Image(systemName: "plus")
                    }
                    .disabled(working)
                    .foregroundStyle(Theme.accent)
                }
            }
            .task { await load() }
            .alert("Yeniden adlandır", isPresented: $showRename) {
                TextField("İsim", text: $renameText)
                Button("İptal", role: .cancel) {}
                Button("Kaydet") { Task { await commitRename() } }
            } message: {
                Text("Bu geçiş anahtarı için bir isim girin.")
            }
            .alert("Geçiş anahtarı silinsin mi?", isPresented: $showDelete) {
                Button("İptal", role: .cancel) {}
                Button("Sil", role: .destructive) { Task { await commitDelete() } }
            } message: {
                Text("Bu geçiş anahtarıyla artık giriş yapamazsınız. Bu işlem geri alınamaz.")
            }
        }
    }

    @ViewBuilder
    private var content: some View {
        ScrollView {
            VStack(spacing: 14) {
                infoCard
                if let error {
                    Text(error)
                        .font(.system(size: 12))
                        .foregroundStyle(Theme.down)
                        .frame(maxWidth: .infinity, alignment: .leading)
                        .padding(12)
                        .background(Theme.down.opacity(0.12), in: RoundedRectangle(cornerRadius: 10))
                }
                if loading {
                    ProgressView().tint(Theme.accent).padding(.top, 30)
                } else if passkeys.isEmpty {
                    emptyState
                } else {
                    ForEach(passkeys) { pk in row(pk) }
                }
                addButton
            }
            .padding(.horizontal, 16).padding(.top, 12).padding(.bottom, 40)
        }
    }

    private var infoCard: some View {
        Card {
            VStack(alignment: .leading, spacing: 8) {
                HStack(spacing: 8) {
                    Image(systemName: "person.badge.key.fill").foregroundStyle(Theme.accent)
                    Text("Parolasız Giriş").font(.system(size: 15, weight: .bold)).foregroundStyle(Theme.text)
                }
                Text("Geçiş anahtarları Face ID / Touch ID ile parolasız ve güvenli giriş sağlar. Birden fazla cihaz için birden fazla anahtar ekleyebilirsiniz.")
                    .font(.system(size: 12)).foregroundStyle(Theme.textDim)
            }
        }
    }

    private func row(_ pk: Passkey) -> some View {
        Card {
            HStack(spacing: 12) {
                ZStack {
                    Circle().fill(Theme.accentSoft).frame(width: 40, height: 40)
                    Image(systemName: pk.synced ? "key.icloud.fill" : "key.fill")
                        .foregroundStyle(Theme.accent)
                }
                VStack(alignment: .leading, spacing: 3) {
                    Text(pk.name).font(.system(size: 15, weight: .semibold)).foregroundStyle(Theme.text)
                    HStack(spacing: 6) {
                        if pk.synced {
                            Label("Eşitlenmiş", systemImage: "checkmark.icloud")
                                .font(.system(size: 11)).foregroundStyle(Theme.up)
                        }
                        if let d = pk.createdAt {
                            Text(d.formatted(date: .abbreviated, time: .shortened))
                                .font(.system(size: 11)).foregroundStyle(Theme.textFaint)
                        }
                    }
                }
                Spacer()
                Menu {
                    Button { startRename(pk) } label: { Label("Yeniden adlandır", systemImage: "pencil") }
                    Button(role: .destructive) { startDelete(pk) } label: { Label("Sil", systemImage: "trash") }
                } label: {
                    Image(systemName: "ellipsis")
                        .foregroundStyle(Theme.textDim)
                        .frame(width: 32, height: 32)
                        .contentShape(Rectangle())
                }
            }
        }
    }

    private var emptyState: some View {
        VStack(spacing: 8) {
            Image(systemName: "key.slash").font(.system(size: 32)).foregroundStyle(Theme.textFaint)
            Text("Henüz geçiş anahtarı yok").font(.system(size: 14, weight: .semibold)).foregroundStyle(Theme.text)
            Text("Aşağıdaki butonla bu cihaza bir geçiş anahtarı ekleyin.")
                .font(.system(size: 12)).foregroundStyle(Theme.textDim).multilineTextAlignment(.center)
        }
        .frame(maxWidth: .infinity).padding(.vertical, 24)
    }

    private var addButton: some View {
        Button { Task { await addPasskey() } } label: {
            HStack {
                if working { ProgressView().tint(Theme.accent).scaleEffect(0.9) }
                Label("Yeni geçiş anahtarı ekle", systemImage: "plus.circle.fill")
            }
        }
        .buttonStyle(FilledButtonStyle(color: Theme.accentSoft, textColor: Theme.accent))
        .disabled(working)
        .padding(.top, 4)
    }

    // MARK: Actions

    private func load() async {
        error = nil
        do {
            passkeys = try await APIClient(token: auth.accessToken).listPasskeys()
        } catch {
            self.error = "Geçiş anahtarları yüklenemedi."
        }
        loading = false
    }

    private func addPasskey() async {
        guard !working else { return }
        working = true; error = nil
        defer { working = false }
        let ok = await auth.registerPasskey()
        if ok {
            Haptics.success()
            await load()
        } else if let msg = auth.errorMessage {
            error = msg
        }
    }

    private func startRename(_ pk: Passkey) {
        renameTarget = pk
        renameText = pk.name
        showRename = true
    }

    private func commitRename() async {
        guard let pk = renameTarget else { return }
        let name = renameText.trimmingCharacters(in: .whitespaces)
        guard !name.isEmpty else { return }
        do {
            try await APIClient(token: auth.accessToken).renamePasskey(id: pk.id, name: name)
            Haptics.tap()
            await load()
        } catch {
            self.error = "Yeniden adlandırılamadı."
        }
    }

    private func startDelete(_ pk: Passkey) {
        deleteTarget = pk
        showDelete = true
    }

    private func commitDelete() async {
        guard let pk = deleteTarget else { return }
        do {
            try await APIClient(token: auth.accessToken).deletePasskey(id: pk.id)
            Haptics.warning()
            await load()
        } catch {
            self.error = "Silinemedi."
        }
    }
}
