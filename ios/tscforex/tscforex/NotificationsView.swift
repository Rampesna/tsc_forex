//
//  NotificationsView.swift
//  Live (WS) notifications merged with persisted history (REST), themed.
//

import SwiftUI

struct NotificationsView: View {
    @EnvironmentObject var auth: AuthStore
    @EnvironmentObject var prices: PriceStore
    @Environment(\.dismiss) private var dismiss
    @State private var history: [AppNotification] = []
    @State private var showClearConfirm = false
    @State private var loading = true
    @State private var loadError: String?

    private var isEmpty: Bool { history.isEmpty && prices.liveNotifications.isEmpty }

    var body: some View {
        ZStack {
            Theme.bg.ignoresSafeArea()
            VStack(spacing: 0) {
                header
                if loading && isEmpty {
                    ProgressView("Bildirimler yükleniyor…")
                        .tint(Theme.accent)
                        .foregroundStyle(Theme.textDim)
                        .frame(maxWidth: .infinity, maxHeight: .infinity)
                } else if let loadError, isEmpty {
                    errorState(loadError)
                } else if isEmpty {
                    emptyState
                } else {
                    ScrollView {
                        LazyVStack(spacing: 10) {
                            ForEach(Array(prices.liveNotifications.enumerated()), id: \.offset) { idx, n in
                                notifCard(title: n.title, body: n.body, code: n.code, live: true, date: nil)
                                    .overlay(alignment: .topTrailing) {
                                        deleteButton { prices.removeLiveNotification(at: idx) }
                                    }
                            }
                            ForEach(history) { n in
                                notifCard(title: n.title, body: n.body, code: n.code, live: false, date: n.createdAt)
                                    .overlay(alignment: .topTrailing) {
                                        deleteButton { await deleteHistory(n) }
                                    }
                            }
                        }
                        .padding(.horizontal, 16).padding(.top, 8).padding(.bottom, 24)
                    }
                    .refreshable { await load() }
                }
            }
        }
        .task(id: auth.accessToken) { await load() }
        .confirmationDialog("Tüm bildirimler silinsin mi?", isPresented: $showClearConfirm, titleVisibility: .visible) {
            Button("Tümünü Sil", role: .destructive) { Task { await clearAll() } }
            Button("Vazgeç", role: .cancel) {}
        }
    }

    private var header: some View {
        HStack(alignment: .center) {
            Button {
                Haptics.tap()
                dismiss()
            } label: {
                Image(systemName: "chevron.left")
                    .font(.system(size: 17, weight: .bold))
                    .foregroundStyle(Theme.text)
                    .frame(width: 38, height: 38)
                    .background(Theme.surface, in: Circle())
            }
            .buttonStyle(.plain)
            VStack(alignment: .leading, spacing: 2) {
                Text("Bildirimler").font(.system(size: 28, weight: .bold)).foregroundStyle(Theme.text)
                Text("Alarmlar ve hesap hareketleri").font(.system(size: 12)).foregroundStyle(Theme.textDim)
            }
            Spacer()
            if !isEmpty {
                Button {
                    Haptics.tap(); showClearConfirm = true
                } label: {
                    Text("Temizle").font(.system(size: 14, weight: .semibold)).foregroundStyle(Theme.down)
                }
            }
        }
        .padding(.horizontal, 16).padding(.top, 6).padding(.bottom, 10)
    }

    private func deleteButton(_ action: @escaping () async -> Void) -> some View {
        Button {
            Haptics.tap()
            Task { await action() }
        } label: {
            Image(systemName: "xmark.circle.fill")
                .font(.system(size: 16))
                .foregroundStyle(Theme.textFaint)
                .padding(10)
        }
    }

    private func notifCard(title: String, body: String, code: String, live: Bool, date: Date?) -> some View {
        Card(padding: 14) {
            HStack(spacing: 12) {
                ZStack {
                    Circle().fill(live ? Theme.up.opacity(0.16) : Theme.accentSoft).frame(width: 34, height: 34)
                    Image(systemName: "bell.fill").font(.system(size: 14))
                        .foregroundStyle(live ? Theme.up : Theme.accent)
                }
                VStack(alignment: .leading, spacing: 3) {
                    HStack(spacing: 6) {
                        Text(title).font(.system(size: 14, weight: .semibold)).foregroundStyle(Theme.text)
                        if live {
                            Text("CANLI").font(.system(size: 8, weight: .bold)).foregroundStyle(Theme.up)
                                .padding(.horizontal, 5).padding(.vertical, 2)
                                .background(Theme.up.opacity(0.15), in: Capsule())
                        }
                    }
                    Text(body).font(.system(size: 12)).foregroundStyle(Theme.textDim).lineLimit(2)
                    if let date {
                        Text(date.formatted(.relative(presentation: .named)))
                            .font(.system(size: 10, weight: .medium))
                            .foregroundStyle(Theme.textFaint)
                    } else if !code.isEmpty {
                        Text(code).font(Theme.number(10, weight: .semibold)).foregroundStyle(Theme.textFaint)
                    }
                }
                Spacer()
            }
        }
    }

    private var emptyState: some View {
        VStack(spacing: 10) {
            Image(systemName: "tray").font(.system(size: 34)).foregroundStyle(Theme.textFaint)
            Text("Bildirim yok").font(.system(size: 16, weight: .semibold)).foregroundStyle(Theme.text)
            Text("Alarm eşikleri tetiklendiğinde burada görünür.")
                .font(.system(size: 13)).foregroundStyle(Theme.textDim)
                .multilineTextAlignment(.center).padding(.horizontal, 40)
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity)
    }

    private func errorState(_ message: String) -> some View {
        VStack(spacing: 12) {
            Image(systemName: "wifi.exclamationmark")
                .font(.system(size: 34)).foregroundStyle(Theme.textFaint)
            Text("Bildirimler alınamadı")
                .font(.system(size: 16, weight: .semibold)).foregroundStyle(Theme.text)
            Text(message)
                .font(.system(size: 12)).foregroundStyle(Theme.textDim)
                .multilineTextAlignment(.center).padding(.horizontal, 40)
            Button("Tekrar Dene") { Task { await load() } }
                .buttonStyle(FilledButtonStyle())
                .padding(.horizontal, 56)
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity)
    }

    private func load() async {
        loading = true
        defer { loading = false }
        do {
            history = try await APIClient(token: auth.accessToken).notifications()
            loadError = nil
        } catch {
            loadError = error.localizedDescription
        }
    }

    private func deleteHistory(_ n: AppNotification) async {
        try? await APIClient(token: auth.accessToken).deleteNotification(n.id)
        history.removeAll { $0.id == n.id }
    }

    private func clearAll() async {
        prices.clearLiveNotifications()
        try? await APIClient(token: auth.accessToken).clearNotifications()
        history.removeAll()
        Haptics.success()
    }
}
