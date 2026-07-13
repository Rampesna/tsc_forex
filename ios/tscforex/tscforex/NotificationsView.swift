//
//  NotificationsView.swift
//  Live (WS) notifications merged with persisted history (REST), themed.
//

import SwiftUI

struct NotificationsView: View {
    @EnvironmentObject var auth: AuthStore
    @EnvironmentObject var prices: PriceStore
    @State private var history: [AppNotification] = []
    @State private var showClearConfirm = false

    private var isEmpty: Bool { history.isEmpty && prices.liveNotifications.isEmpty }

    var body: some View {
        NavigationStack {
            ZStack {
                Theme.bg.ignoresSafeArea()
                VStack(spacing: 0) {
                    header
                    if isEmpty {
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
                            .padding(.horizontal, 16).padding(.top, 8).padding(.bottom, 100)
                        }
                        .refreshable { await load() }
                    }
                }
            }
            .toolbar(.hidden, for: .navigationBar)
        }
        .task { await load() }
        .confirmationDialog("Tüm bildirimler silinsin mi?", isPresented: $showClearConfirm, titleVisibility: .visible) {
            Button("Tümünü Sil", role: .destructive) { Task { await clearAll() } }
            Button("Vazgeç", role: .cancel) {}
        }
    }

    private var header: some View {
        HStack(alignment: .center) {
            VStack(alignment: .leading, spacing: 2) {
                Text("Bildirimler").font(.system(size: 28, weight: .bold)).foregroundStyle(Theme.text)
                Text("Tetiklenen alarmlar").font(.system(size: 12)).foregroundStyle(Theme.textDim)
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

    private func load() async {
        if let n = try? await APIClient(token: auth.accessToken).notifications() { history = n }
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
