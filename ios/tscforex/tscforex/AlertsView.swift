//
//  AlertsView.swift
//  Themed list of the user's price alerts.
//

import SwiftUI

struct AlertsView: View {
    @EnvironmentObject var auth: AuthStore
    @EnvironmentObject var prices: PriceStore
    @State private var alerts: [Alert] = []
    @State private var loading = false
    @State private var editingAlert: Alert?
    @State private var deletingIDs: Set<String> = []
    @State private var errorMessage: String?

    var body: some View {
        NavigationStack {
            ZStack {
                Theme.bg.ignoresSafeArea()
                VStack(spacing: 0) {
                    ScreenHeader(title: "Alarmlar", subtitle: "Eşik geçilince bir kez bildirir")
                    if alerts.isEmpty && !loading {
                        emptyState
                    } else {
                        ScrollView {
                            LazyVStack(spacing: 10) {
                                if let errorMessage {
                                    Label(errorMessage, systemImage: "exclamationmark.triangle")
                                        .font(.system(size: 12)).foregroundStyle(Theme.down)
                                        .frame(maxWidth: .infinity, alignment: .leading)
                                        .padding(12)
                                        .background(Theme.down.opacity(0.08), in: RoundedRectangle(cornerRadius: 10))
                                }
                                oneShotInfo
                                ForEach(alerts) { a in alertCard(a) }
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
        .onChange(of: prices.liveNotifications.count) { _, _ in
            // Alert notifications arrive over the live socket. Refreshing here
            // removes the one-shot alarm immediately instead of waiting for a
            // tab reopen or manual pull-to-refresh.
            Task { await load() }
        }
        .sheet(item: $editingAlert) { alert in
            AlertEditSheet(
                alert: alert,
                displayName: prices.prices[alert.code]?.displayName ?? alert.code,
                currentPrice: prices.prices[alert.code]?.price
            ) { updated in
                if let index = alerts.firstIndex(where: { $0.id == updated.id }) {
                    alerts[index] = updated
                }
            }
            .environmentObject(auth)
            .presentationDetents([.large])
            .presentationBackground(Theme.bgElevated)
        }
    }

    private func alertCard(_ a: Alert) -> some View {
        Card(padding: 14) {
            HStack(spacing: 12) {
                AssetIcon(code: a.code, kind: prices.prices[a.code]?.kind ?? "currency", size: 34)
                VStack(alignment: .leading, spacing: 3) {
                    Text(prices.prices[a.code]?.displayName ?? a.code)
                        .font(.system(size: 15, weight: .semibold)).foregroundStyle(Theme.text)
                    HStack(spacing: 6) {
                        Image(systemName: a.direction == "above" ? "arrow.up.right" : "arrow.down.right")
                            .font(.system(size: 10, weight: .bold))
                            .foregroundStyle(a.direction == "above" ? Theme.up : Theme.down)
                        Text(a.direction == "above" ? "Üstüne çıkınca" : "Altına inince")
                            .font(.system(size: 12)).foregroundStyle(Theme.textDim)
                    }
                }
                Spacer()
                VStack(alignment: .trailing, spacing: 4) {
                    Text("₺\(Fmt.compactPrice(a.threshold))")
                        .font(Theme.number(15, weight: .semibold)).foregroundStyle(Theme.text)
                    HStack(spacing: 4) {
                        Button {
                            Haptics.tap()
                            editingAlert = a
                        } label: {
                            Image(systemName: "pencil")
                                .font(.system(size: 13, weight: .semibold))
                                .foregroundStyle(Theme.accent)
                                .frame(width: 32, height: 30)
                        }
                        .accessibilityLabel("Alarmı düzenle")

                        Button(role: .destructive) {
                            Task { await delete(a.id) }
                        } label: {
                            Group {
                                if deletingIDs.contains(a.id) {
                                    ProgressView().tint(Theme.down).scaleEffect(0.7)
                                } else {
                                    Image(systemName: "trash")
                                        .font(.system(size: 13, weight: .semibold))
                                }
                            }
                            .foregroundStyle(Theme.down)
                            .frame(width: 32, height: 30)
                        }
                        .disabled(deletingIDs.contains(a.id))
                        .accessibilityLabel("Alarmı sil")
                    }
                }
            }
        }
    }

    private var oneShotInfo: some View {
        HStack(alignment: .top, spacing: 10) {
            Image(systemName: "bell.badge")
                .font(.system(size: 15, weight: .semibold))
                .foregroundStyle(Theme.accent)
                .frame(width: 22)
            VStack(alignment: .leading, spacing: 3) {
                Text("Tek seferlik alarmlar")
                    .font(.system(size: 13, weight: .semibold)).foregroundStyle(Theme.text)
                Text("Eşik gerçekleşip bildirim oluşturulduğunda alarm otomatik olarak aktif listeden kaldırılır.")
                    .font(.system(size: 11)).foregroundStyle(Theme.textDim)
                    .fixedSize(horizontal: false, vertical: true)
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .padding(13)
        .background(Theme.accent.opacity(0.08), in: RoundedRectangle(cornerRadius: 12, style: .continuous))
        .overlay(RoundedRectangle(cornerRadius: 12, style: .continuous).stroke(Theme.accent.opacity(0.18)))
    }

    private var emptyState: some View {
        VStack(spacing: 10) {
            Image(systemName: "bell.slash").font(.system(size: 34)).foregroundStyle(Theme.textFaint)
            Text("Alarm yok").font(.system(size: 16, weight: .semibold)).foregroundStyle(Theme.text)
            Text("Piyasa sekmesinden bir varlık açıp tek seferlik fiyat alarmı kurabilirsiniz.")
                .font(.system(size: 13)).foregroundStyle(Theme.textDim)
                .multilineTextAlignment(.center).padding(.horizontal, 40)
            if let errorMessage {
                Text(errorMessage)
                    .font(.system(size: 12)).foregroundStyle(Theme.down)
                    .multilineTextAlignment(.center).padding(.horizontal, 32)
                Button("Tekrar Dene") { Task { await load() } }
                    .font(.system(size: 13, weight: .semibold)).foregroundStyle(Theme.accent)
            }
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity)
    }

    private func load() async {
        loading = true; defer { loading = false }
        do {
            alerts = try await APIClient(token: auth.accessToken).alerts().filter { $0.active }
            errorMessage = nil
        } catch {
            errorMessage = error.localizedDescription
        }
    }

    private func delete(_ id: String) async {
        guard !deletingIDs.contains(id) else { return }
        deletingIDs.insert(id)
        defer { deletingIDs.remove(id) }
        do {
            try await APIClient(token: auth.accessToken).deleteAlert(id)
            withAnimation { alerts.removeAll { $0.id == id } }
            errorMessage = nil
        } catch {
            errorMessage = error.localizedDescription
        }
    }
}

private struct AlertEditSheet: View {
    let alert: Alert
    let displayName: String
    let currentPrice: Double?
    let onSaved: (Alert) -> Void

    @EnvironmentObject private var auth: AuthStore
    @Environment(\.dismiss) private var dismiss
    @State private var direction: String
    @State private var threshold: String
    @State private var saving = false
    @State private var errorMessage: String?
    @FocusState private var thresholdFocused: Bool

    init(alert: Alert, displayName: String, currentPrice: Double?, onSaved: @escaping (Alert) -> Void) {
        self.alert = alert
        self.displayName = displayName
        self.currentPrice = currentPrice
        self.onSaved = onSaved
        _direction = State(initialValue: alert.direction)
        _threshold = State(initialValue: Self.inputValue(alert.threshold))
    }

    private var parsedThreshold: Double? {
        let normalized = threshold
            .trimmingCharacters(in: .whitespacesAndNewlines)
            .replacingOccurrences(of: ",", with: ".")
        guard let value = Double(normalized), value > 0, value.isFinite else { return nil }
        return value
    }

    private var thresholdWarning: String? {
        guard let currentPrice, let value = parsedThreshold else { return nil }
        if direction == "above", currentPrice >= value {
            return "Güncel fiyat bu eşiğin üzerinde. Alarm ilk fiyat güncellemesinde tetiklenebilir."
        }
        if direction == "below", currentPrice <= value {
            return "Güncel fiyat bu eşiğin altında. Alarm ilk fiyat güncellemesinde tetiklenebilir."
        }
        return nil
    }

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 17) {
                HStack {
                    VStack(alignment: .leading, spacing: 3) {
                        Text("Alarmı Düzenle")
                            .font(.system(size: 20, weight: .bold)).foregroundStyle(Theme.text)
                        Text("\(displayName) · Tek seferlik")
                            .font(.system(size: 12)).foregroundStyle(Theme.textDim)
                    }
                    Spacer()
                    Button { dismiss() } label: {
                        Image(systemName: "xmark")
                            .font(.system(size: 12, weight: .bold)).foregroundStyle(Theme.textDim)
                            .frame(width: 32, height: 32)
                            .background(Theme.surface, in: Circle())
                    }
                    .accessibilityLabel("Kapat")
                }

                SegBar(items: [("above", "Üstüne çıkınca"), ("below", "Altına inince")],
                       selection: $direction)

                VStack(alignment: .leading, spacing: 7) {
                    Text("Fiyat eşiği")
                        .font(.system(size: 12, weight: .semibold)).foregroundStyle(Theme.textDim)
                    HStack {
                        Text("₺").foregroundStyle(Theme.textDim)
                        TextField("0,00", text: $threshold)
                            .keyboardType(.decimalPad)
                            .focused($thresholdFocused)
                            .font(Theme.number(20))
                            .foregroundStyle(Theme.text)
                    }
                    .padding(14)
                    .background(Theme.surface, in: RoundedRectangle(cornerRadius: 10, style: .continuous))
                    .overlay(
                        RoundedRectangle(cornerRadius: 10, style: .continuous)
                            .stroke(parsedThreshold == nil ? Theme.down.opacity(0.55) : Theme.stroke)
                    )
                    if let currentPrice {
                        Text("Güncel fiyat: ₺\(Fmt.compactPrice(currentPrice))")
                            .font(.system(size: 11)).foregroundStyle(Theme.textDim)
                    }
                }

                if let warning = thresholdWarning {
                    Label(warning, systemImage: "exclamationmark.triangle")
                        .font(.system(size: 11)).foregroundStyle(Color.orange)
                        .fixedSize(horizontal: false, vertical: true)
                }
                if let errorMessage {
                    Text(errorMessage)
                        .font(.system(size: 12)).foregroundStyle(Theme.down)
                        .fixedSize(horizontal: false, vertical: true)
                }

                Button { Task { await save() } } label: {
                    if saving { ProgressView().tint(.black) } else { Text("Değişiklikleri Kaydet") }
                }
                .buttonStyle(FilledButtonStyle())
                .disabled(parsedThreshold == nil || saving)
            }
            .padding(20)
        }
        .scrollDismissesKeyboard(.interactively)
        .toolbar {
            ToolbarItemGroup(placement: .keyboard) {
                Spacer()
                Button("Bitti") {
                    thresholdFocused = false
                }
            }
        }
    }

    private func save() async {
        guard let value = parsedThreshold, !saving else { return }
        saving = true
        defer { saving = false }
        do {
            let updated = try await APIClient(token: auth.accessToken)
                .updateAlert(id: alert.id, direction: direction, threshold: value)
            Haptics.success()
            onSaved(updated)
            dismiss()
        } catch {
            errorMessage = error.localizedDescription
        }
    }

    private static func inputValue(_ value: Double) -> String {
        let formatter = NumberFormatter()
        formatter.numberStyle = .decimal
        formatter.usesGroupingSeparator = false
        formatter.minimumFractionDigits = 0
        formatter.maximumFractionDigits = value >= 1 ? 2 : 6
        formatter.decimalSeparator = ","
        return formatter.string(from: NSNumber(value: value)) ?? String(value)
    }
}

// Shared compact screen header used by secondary tabs.
struct ScreenHeader: View {
    let title: String
    var subtitle: String?
    var body: some View {
        HStack {
            VStack(alignment: .leading, spacing: 2) {
                Text(title).font(.system(size: 26, weight: .bold)).foregroundStyle(Theme.text)
                if let subtitle {
                    Text(subtitle).font(.system(size: 12)).foregroundStyle(Theme.textDim)
                }
            }
            Spacer()
        }
        .padding(.horizontal, 16).padding(.top, 8).padding(.bottom, 8)
    }
}
