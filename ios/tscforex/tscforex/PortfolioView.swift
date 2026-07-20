import SwiftUI
import UIKit

private enum PortfolioChartMetric: String, CaseIterable, Identifiable {
    case value
    case totalPnL
    case costBasis

    var id: String { rawValue }

    var title: String {
        switch self {
        case .value: return "Değer"
        case .totalPnL: return "Toplam K/Z"
        case .costBasis: return "Maliyet"
        }
    }

    func value(in point: PortfolioPerformancePoint) -> Double {
        switch self {
        case .value: return point.valueTRY
        case .totalPnL: return point.totalPnLTRY
        case .costBasis: return point.costBasisTRY
        }
    }
}

struct PortfolioView: View {
    @EnvironmentObject private var portfolio: PortfolioStore
    @EnvironmentObject private var prices: PriceStore
    @EnvironmentObject private var auth: AuthStore
    @EnvironmentObject private var billing: BillingStore
    @State private var showTransaction = false
    @State private var range = "1y"
    @State private var chartMetric = PortfolioChartMetric.value
    @State private var chartResetToken = 0
    @State private var selectedPerformanceDate: Date?
    @State private var performanceLoading = false
    @State private var showCreatePortfolio = false
    @State private var showRenamePortfolio = false
    @State private var confirmDeletePortfolio = false
    @State private var portfolioName = ""
    @State private var portfolioHealth: PortfolioHealth?
    @State private var premiumOperation: String?
    @State private var exportFile: PortfolioExportFile?
    @State private var showPaywall = false

    var body: some View {
        NavigationStack {
            ZStack {
                Theme.bg.ignoresSafeArea()
                if portfolio.loading, portfolio.overview == nil {
                    ProgressView("Portföy senkronize ediliyor…")
                        .tint(Theme.accent).foregroundStyle(Theme.textDim)
                } else {
                    ScrollView {
                        LazyVStack(spacing: 14) {
                            if let error = portfolio.errorMessage {
                                errorCard(error)
                            }
                            if !portfolio.legacyPreview.isEmpty {
                                legacyImportCard
                            }
                            portfolioSelectorCard
                            summaryCard
                            premiumToolsCard
                            if let portfolioHealth {
                                healthCard(portfolioHealth)
                            }
                            performanceCard
                            if portfolio.activePositions.isEmpty {
                                emptyCard
                            } else {
                                allocationCard
                                positionsCard
                            }
                        }
                        .padding(.horizontal, 16)
                        .padding(.bottom, 100)
                    }
                    .refreshable {
                        await portfolio.sync(token: auth.accessToken, prices: prices.prices, range: range)
                    }
                }
            }
            .navigationTitle("Portföy")
            .toolbar {
                ToolbarItem(placement: .topBarTrailing) {
                    Button {
                        Haptics.tap()
                        showTransaction = true
                    } label: {
                        Image(systemName: "plus")
                            .font(.system(size: 15, weight: .bold))
                            .foregroundStyle(.black)
                            .frame(width: 32, height: 32)
                            .background(Theme.accent, in: Circle())
                    }
                }
            }
        }
        .task {
            await portfolio.sync(token: auth.accessToken, prices: prices.prices, range: range)
        }
        .sheet(isPresented: $showTransaction) {
            PortfolioTransactionSheet()
                .environmentObject(portfolio)
                .environmentObject(prices)
                .presentationDetents([.large])
                .presentationBackground(Theme.bgElevated)
        }
        .sheet(isPresented: $showPaywall) {
            PaywallView()
                .environmentObject(auth)
                .environmentObject(billing)
        }
        .sheet(item: $exportFile) { file in
            PortfolioShareSheet(url: file.url)
                .presentationDetents([.medium, .large])
        }
        .alert("Yeni Portföy", isPresented: $showCreatePortfolio) {
            TextField("Portföy adı", text: $portfolioName)
            Button("Vazgeç", role: .cancel) { portfolioName = "" }
            Button("Oluştur") {
                let name = portfolioName
                portfolioName = ""
                Task {
                    do { _ = try await portfolio.createPortfolio(name: name, range: range) }
                    catch { handlePremiumError(error) }
                }
            }
        } message: {
            Text("Planınızdaki portföy hakkı kapsamında ayrı bir takip alanı oluşturun.")
        }
        .alert("Portföyü Yeniden Adlandır", isPresented: $showRenamePortfolio) {
            TextField("Portföy adı", text: $portfolioName)
            Button("Vazgeç", role: .cancel) { portfolioName = "" }
            Button("Kaydet") {
                let name = portfolioName
                portfolioName = ""
                Task { try? await portfolio.renameCurrentPortfolio(name: name) }
            }
        }
        .alert("Portföy silinsin mi?", isPresented: $confirmDeletePortfolio) {
            Button("Vazgeç", role: .cancel) {}
            Button("Kalıcı Olarak Sil", role: .destructive) {
                Task { try? await portfolio.deleteCurrentPortfolio(range: range) }
            }
        } message: {
            Text("Bu portföydeki tüm işlem ve performans geçmişi kalıcı olarak silinir.")
        }
    }

    private var portfolioSelectorCard: some View {
        Card(padding: 14) {
            HStack(spacing: 12) {
                VStack(alignment: .leading, spacing: 3) {
                    Text("AKTİF PORTFÖY")
                        .font(.system(size: 9, weight: .bold))
                        .foregroundStyle(Theme.textFaint)
                        .tracking(1)
                    Menu {
                        ForEach(portfolio.portfolios, id: \.portfolio.id) { item in
                            Button {
                                portfolioHealth = nil
                                Task { await portfolio.selectPortfolio(id: item.portfolio.id, range: range) }
                            } label: {
                                if item.portfolio.id == portfolio.overview?.portfolio.id {
                                    Label(item.portfolio.name, systemImage: "checkmark")
                                } else {
                                    Text(item.portfolio.name)
                                }
                            }
                        }
                    } label: {
                        HStack(spacing: 6) {
                            Text(portfolio.overview?.portfolio.name ?? "Portföy seç")
                                .font(.system(size: 16, weight: .bold))
                                .foregroundStyle(Theme.text)
                            Image(systemName: "chevron.up.chevron.down")
                                .font(.system(size: 10, weight: .bold))
                                .foregroundStyle(Theme.accent)
                        }
                    }
                }
                Spacer()
                Button {
                    portfolioName = ""
                    showCreatePortfolio = true
                } label: {
                    Image(systemName: "folder.badge.plus")
                        .font(.system(size: 17, weight: .semibold))
                        .foregroundStyle(Theme.accent)
                        .frame(width: 36, height: 36)
                        .background(Theme.accentSoft, in: RoundedRectangle(cornerRadius: 10))
                }
                Menu {
                    Button("Yeniden Adlandır", systemImage: "pencil") {
                        portfolioName = portfolio.overview?.portfolio.name ?? ""
                        showRenamePortfolio = true
                    }
                    Button("Portföyü Sil", systemImage: "trash", role: .destructive) {
                        confirmDeletePortfolio = true
                    }
                    .disabled(portfolio.portfolios.count <= 1)
                } label: {
                    Image(systemName: "ellipsis")
                        .foregroundStyle(Theme.textDim)
                        .frame(width: 32, height: 36)
                }
            }
        }
    }

    private var premiumToolsCard: some View {
        Card {
            VStack(alignment: .leading, spacing: 12) {
                HStack {
                    VStack(alignment: .leading, spacing: 2) {
                        Text("Portföy Araçları")
                            .font(.system(size: 15, weight: .bold)).foregroundStyle(Theme.text)
                        Text("Dağılımı değerlendir veya işlem defterini paylaşılabilir CSV olarak al.")
                            .font(.system(size: 10)).foregroundStyle(Theme.textDim)
                    }
                    Spacer()
                    Image(systemName: "crown.fill").foregroundStyle(Theme.accent)
                }
                HStack(spacing: 10) {
                    premiumButton(
                        title: "Sağlık Analizi",
                        subtitle: quotaText("portfolio_health"),
                        icon: "heart.text.square",
                        busy: premiumOperation == "health"
                    ) { Task { await loadPortfolioHealth() } }
                    premiumButton(
                        title: "CSV Dışa Aktar",
                        subtitle: quotaText("exports"),
                        icon: "square.and.arrow.up",
                        busy: premiumOperation == "export"
                    ) { Task { await exportPortfolio() } }
                }
            }
        }
    }

    private func premiumButton(title: String, subtitle: String, icon: String,
                               busy: Bool, action: @escaping () -> Void) -> some View {
        Button(action: action) {
            VStack(alignment: .leading, spacing: 7) {
                if busy {
                    ProgressView().tint(Theme.accent)
                } else {
                    Image(systemName: icon).foregroundStyle(Theme.accent)
                }
                Text(title).font(.system(size: 12, weight: .bold)).foregroundStyle(Theme.text)
                Text(subtitle).font(.system(size: 9)).foregroundStyle(Theme.textFaint)
            }
            .frame(maxWidth: .infinity, minHeight: 72, alignment: .leading)
            .padding(11)
            .background(Theme.surfaceHi, in: RoundedRectangle(cornerRadius: 11))
        }
        .buttonStyle(.plain)
        .disabled(premiumOperation != nil)
    }

    private func healthCard(_ report: PortfolioHealth) -> some View {
        Card {
            VStack(alignment: .leading, spacing: 12) {
                HStack(alignment: .top) {
                    VStack(alignment: .leading, spacing: 3) {
                        Text("Portföy Sağlığı")
                            .font(.system(size: 15, weight: .bold)).foregroundStyle(Theme.text)
                        Text(report.generatedAt.formatted(date: .abbreviated, time: .shortened))
                            .font(.system(size: 9)).foregroundStyle(Theme.textFaint)
                    }
                    Spacer()
                    VStack(spacing: 1) {
                        Text("\(report.score)").font(Theme.number(24, weight: .bold)).foregroundStyle(healthColor(report.score))
                        Text(report.grade).font(.system(size: 9, weight: .bold)).foregroundStyle(healthColor(report.score))
                    }
                }

                Text(report.summary).font(.system(size: 12)).foregroundStyle(Theme.textDim)
                ForEach(report.findings) { finding in
                    HStack(alignment: .top, spacing: 9) {
                        Circle().fill(severityColor(finding.severity)).frame(width: 7, height: 7).padding(.top, 5)
                        VStack(alignment: .leading, spacing: 2) {
                            Text(finding.title).font(.system(size: 12, weight: .semibold)).foregroundStyle(Theme.text)
                            Text(finding.detail).font(.system(size: 10)).foregroundStyle(Theme.textDim)
                        }
                    }
                }
                Text("Bu değerlendirme eğitim ve risk farkındalığı içindir; yatırım tavsiyesi değildir.")
                    .font(.system(size: 9)).foregroundStyle(Theme.textFaint)
            }
        }
    }

    private func quotaText(_ feature: String) -> String {
        guard let quota = billing.serverStatus?.usage[feature] else { return "Paket özelliği" }
        return "Bu ay \(quota.remaining)/\(quota.limit) kaldı"
    }

    private func healthColor(_ score: Int) -> Color {
        score >= 85 ? Theme.up : (score >= 60 ? Theme.accent : Theme.down)
    }

    private func severityColor(_ severity: String) -> Color {
        switch severity {
        case "high": return Theme.down
        case "medium": return Theme.accent
        default: return Theme.up
        }
    }

    private func loadPortfolioHealth() async {
        guard let token = auth.accessToken, let id = portfolio.overview?.portfolio.id else { return }
        if auth.userID == nil { await auth.loadProfile() }
        guard let userID = auth.userID else {
            portfolio.errorMessage = "Hesap bilgileri doğrulanamadı. Profilinizi yenileyip tekrar deneyin."
            return
        }
        let operation = "portfolio-health"
        let pending = PendingOperationStore.loadOrCreate(
            userID: userID, operation: operation, resource: id
        )
        premiumOperation = "health"
        defer { premiumOperation = nil }
        do {
            portfolioHealth = try await APIClient(token: token).portfolioHealth(
                id: id, requestID: pending.requestID
            )
            PendingOperationStore.clear(userID: userID, operation: operation, resource: id)
            await billing.refresh(token: token)
            Haptics.success()
        } catch {
            clearTerminalPendingOperation(
                error, userID: userID, operation: operation, resource: id
            )
            handlePremiumError(error)
        }
    }

    private func exportPortfolio() async {
        guard let token = auth.accessToken, let id = portfolio.overview?.portfolio.id else { return }
        if auth.userID == nil { await auth.loadProfile() }
        guard let userID = auth.userID else {
            portfolio.errorMessage = "Hesap bilgileri doğrulanamadı. Profilinizi yenileyip tekrar deneyin."
            return
        }
        let operation = "portfolio-export"
        let pending = PendingOperationStore.loadOrCreate(
            userID: userID, operation: operation, resource: id
        )
        premiumOperation = "export"
        defer { premiumOperation = nil }
        do {
            let data = try await APIClient(token: token).exportPortfolio(
                id: id, requestID: pending.requestID
            )
            let date = Date().formatted(.iso8601.year().month().day().dateSeparator(.dash))
            let url = FileManager.default.temporaryDirectory
                .appendingPathComponent("portfoy-\(id.prefix(8))-\(date).csv")
            try data.write(to: url, options: .atomic)
            PendingOperationStore.clear(userID: userID, operation: operation, resource: id)
            exportFile = PortfolioExportFile(url: url)
            await billing.refresh(token: token)
            Haptics.success()
        } catch {
            clearTerminalPendingOperation(
                error, userID: userID, operation: operation, resource: id
            )
            handlePremiumError(error)
        }
    }

    private func handlePremiumError(_ error: Error) {
        guard let apiError = error as? APIError else {
            portfolio.errorMessage = error.localizedDescription
            return
        }
        if apiError.statusCode == 402 {
            showPaywall = true
        } else if apiError.statusCode == 429 {
            portfolio.errorMessage = "Çok sık istek gönderildi. Kısa bir süre bekleyip tekrar deneyin."
        } else if apiError.statusCode == 409 && apiError.serverCode == "operation_in_progress" {
            portfolio.errorMessage = "İşlem halen tamamlanıyor. Biraz sonra tekrar deneyin; kullanım hakkınız yeniden düşülmeyecek."
        } else {
            portfolio.errorMessage = error.localizedDescription
        }
    }

    private func clearTerminalPendingOperation(
        _ error: Error, userID: String, operation: String, resource: String
    ) {
        guard let apiError = error as? APIError else { return }
        if !apiError.shouldRetainRequestID {
            PendingOperationStore.clear(
                userID: userID, operation: operation, resource: resource
            )
        }
    }

    private var summaryCard: some View {
        let value = portfolio.overview?.totalValueTRY ?? 0
        let day = portfolio.overview?.dayChangeTRY ?? 0
        let dayPct = portfolio.overview?.dayChangePct ?? 0
        let unrealized = portfolio.overview?.unrealizedPnLTRY ?? 0
        let realized = portfolio.overview?.realizedPnLTRY ?? 0
        return Card {
            VStack(alignment: .leading, spacing: 16) {
                VStack(alignment: .leading, spacing: 4) {
                    Text("Toplam Değer").font(.system(size: 12)).foregroundStyle(Theme.textDim)
                    Text("₺\(Fmt.compactPrice(value))")
                        .font(Theme.number(32, weight: .bold))
                        .foregroundStyle(Theme.text)
                }
                HStack(spacing: 10) {
                    portfolioMetric("Bugün", value: "\(Fmt.signed(day)) ₺",
                                    sub: Fmt.signedPct(dayPct), trend: day)
                    portfolioMetric("Gerçekleşmemiş", value: "\(Fmt.signed(unrealized)) ₺",
                                    trend: unrealized)
                    portfolioMetric("Gerçekleşen", value: "\(Fmt.signed(realized)) ₺",
                                    trend: realized)
                }
            }
        }
    }

    private func portfolioMetric(_ title: String, value: String, sub: String? = nil,
                                 trend: Double) -> some View {
        VStack(alignment: .leading, spacing: 4) {
            Text(title).font(.system(size: 10)).foregroundStyle(Theme.textDim).lineLimit(1)
            Text(value).font(Theme.number(12, weight: .semibold))
                .foregroundStyle(Theme.trend(trend)).lineLimit(1).minimumScaleFactor(0.7)
            if let sub {
                Text(sub).font(Theme.number(9)).foregroundStyle(Theme.trend(trend))
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
    }

    private var performanceCard: some View {
        Card {
            VStack(alignment: .leading, spacing: 14) {
                HStack {
                    VStack(alignment: .leading, spacing: 2) {
                        Text("Portföy Performansı")
                            .font(.system(size: 15, weight: .semibold)).foregroundStyle(Theme.text)
                        Text("Varlıkların gerçek günlük fiyat geçmişine göre")
                            .font(.system(size: 9)).foregroundStyle(Theme.textFaint)
                    }
                    Spacer()
                    if performanceLoading {
                        ProgressView().tint(Theme.accent).scaleEffect(0.8)
                    }
                }

                if let message = portfolio.performanceErrorMessage {
                    Label(message, systemImage: "exclamationmark.triangle.fill")
                        .font(.system(size: 10, weight: .medium))
                        .foregroundStyle(Theme.accent)
                }

                HStack(spacing: 5) {
                    ForEach(PortfolioChartMetric.allCases) { metric in
                        Button {
                            chartMetric = metric
                            selectedPerformanceDate = nil
                            chartResetToken += 1
                            Haptics.tap()
                        } label: {
                            Text(metric.title)
                                .font(.system(size: 10, weight: .semibold))
                                .foregroundStyle(chartMetric == metric ? .black : Theme.textDim)
                                .frame(maxWidth: .infinity)
                                .padding(.vertical, 7)
                                .background(
                                    chartMetric == metric ? Theme.accent : Theme.surfaceHi,
                                    in: RoundedRectangle(cornerRadius: 8)
                                )
                        }
                        .buttonStyle(.plain)
                    }
                }

                performanceSnapshot

                PortfolioPerformanceChart(
                    points: portfolio.performance,
                    metric: chartMetric,
                    resetToken: chartResetToken,
                    selectedDate: $selectedPerformanceDate
                )

                HStack(spacing: 4) {
                    ForEach([
                        ("1m", "1 AY"), ("3m", "3 AY"),
                        ("1y", "1 YIL"), ("5y", "5 YIL")
                    ], id: \.0) { item in
                        Button(item.1) {
                            guard range != item.0 else { return }
                            performanceLoading = true
                            Task {
                                let loaded = await portfolio.refreshPerformance(range: item.0)
                                if loaded {
                                    range = item.0
                                    selectedPerformanceDate = nil
                                    chartResetToken += 1
                                }
                                performanceLoading = false
                            }
                        }
                        .font(.system(size: 10, weight: .semibold))
                        .foregroundStyle(range == item.0 ? Theme.accent : Theme.textDim)
                        .frame(maxWidth: .infinity)
                        .padding(.vertical, 7)
                        .background(
                            range == item.0 ? Theme.accent.opacity(0.12) : Theme.surfaceHi,
                            in: RoundedRectangle(cornerRadius: 8)
                        )
                    }
                }

                Label("Yatay sürükle, sıkıştırarak yakınlaştır, dokunarak günü seç", systemImage: "hand.draw")
                    .font(.system(size: 9)).foregroundStyle(Theme.textFaint)
            }
        }
    }

    @ViewBuilder private var performanceSnapshot: some View {
        if let point = selectedPerformancePoint ?? portfolio.performance.last {
            VStack(alignment: .leading, spacing: 8) {
                Text((selectedPerformancePoint == nil ? "SON DURUM · " : "SEÇİLİ GÜN · ") + point.displayDate)
                    .font(.system(size: 9, weight: .bold))
                    .foregroundStyle(Theme.textFaint)
                    .tracking(0.6)
                LazyVGrid(columns: [GridItem(.flexible()), GridItem(.flexible())], spacing: 8) {
                    performanceValue("Değer", point.valueTRY)
                    performanceValue("Maliyet", point.costBasisTRY)
                    performanceValue("Toplam K/Z", point.totalPnLTRY, trend: point.totalPnLTRY)
                    performanceValue("Net Para Akışı", point.netDepositsTRY)
                }
            }
            .padding(10)
            .background(Theme.surfaceHi.opacity(0.72), in: RoundedRectangle(cornerRadius: 10))
        }
    }

    private var selectedPerformancePoint: PortfolioPerformancePoint? {
        guard let selectedPerformanceDate else { return nil }
        return portfolio.performance.first { point in
            guard let pointDate = point.chartDate else { return false }
            return Calendar(identifier: .gregorian).isDate(pointDate, inSameDayAs: selectedPerformanceDate)
        }
    }

    private func performanceValue(_ title: String, _ value: Double, trend: Double? = nil) -> some View {
        let color = trend.map { Theme.trend($0) } ?? Theme.text
        return VStack(alignment: .leading, spacing: 2) {
            Text(title).font(.system(size: 9)).foregroundStyle(Theme.textFaint)
            Text("₺\(Fmt.compactPrice(value))")
                .font(Theme.number(12, weight: .semibold))
                .foregroundStyle(color)
        }
        .frame(maxWidth: .infinity, alignment: .leading)
    }

    private var allocationCard: some View {
        Card {
            VStack(alignment: .leading, spacing: 12) {
                Text("Dağılım").font(.system(size: 15, weight: .semibold)).foregroundStyle(Theme.text)
                ForEach(portfolio.activePositions.prefix(8)) { position in
                    VStack(spacing: 5) {
                        HStack {
                            Text(position.code).font(.system(size: 11, weight: .medium)).foregroundStyle(Theme.text)
                            Spacer()
                            Text("%\(String(format: "%.1f", position.allocationPct))")
                                .font(Theme.number(11)).foregroundStyle(Theme.textDim)
                        }
                        GeometryReader { geo in
                            ZStack(alignment: .leading) {
                                Capsule().fill(Theme.surface)
                                Capsule().fill(Theme.accent)
                                    .frame(width: geo.size.width * min(max(position.allocationPct / 100, 0), 1))
                            }
                        }
                        .frame(height: 5)
                    }
                }
            }
        }
    }

    private var positionsCard: some View {
        Card(padding: 0) {
            VStack(spacing: 0) {
                HStack {
                    Text("Pozisyonlar").font(.system(size: 15, weight: .semibold)).foregroundStyle(Theme.text)
                    Spacer()
                    Text("\(portfolio.activePositions.count)")
                        .font(Theme.number(12)).foregroundStyle(Theme.textDim)
                }
                .padding(16)
                ForEach(Array(portfolio.activePositions.enumerated()), id: \.element.id) { index, position in
                    if index > 0 { Divider().overlay(Theme.stroke) }
                    NavigationLink {
                        PortfolioPositionView(position: position)
                            .environmentObject(portfolio)
                            .environmentObject(prices)
                            .environmentObject(auth)
                    } label: {
                        positionRow(position)
                    }
                    .buttonStyle(.plain)
                }
            }
        }
    }

    private func positionRow(_ position: CloudPortfolioPosition) -> some View {
        HStack(spacing: 12) {
            AssetIcon(code: position.code, kind: position.kind, size: 38)
            VStack(alignment: .leading, spacing: 3) {
                Text(position.displayName).font(.system(size: 14, weight: .semibold)).foregroundStyle(Theme.text)
                Text("\(Fmt.compactPrice(position.quantity)) adet • Ort. \(position.currency == "USD" ? "$" : "₺")\(Fmt.compactPrice(position.averageCostNative))")
                    .font(.system(size: 10)).foregroundStyle(Theme.textDim)
            }
            Spacer()
            VStack(alignment: .trailing, spacing: 3) {
                Text("₺\(Fmt.compactPrice(position.valueTRY))")
                    .font(Theme.number(14, weight: .bold)).foregroundStyle(Theme.text)
                Text("\(Fmt.signed(position.unrealizedPnLTRY)) ₺ (\(Fmt.signedPct(position.unrealizedPnLPct)))")
                    .font(Theme.number(10, weight: .semibold))
                    .foregroundStyle(Theme.trend(position.unrealizedPnLTRY))
            }
            Image(systemName: "chevron.right").font(.system(size: 10)).foregroundStyle(Theme.textFaint)
        }
        .padding(.horizontal, 16).padding(.vertical, 13)
        .contentShape(Rectangle())
    }

    private var legacyImportCard: some View {
        Card {
            VStack(alignment: .leading, spacing: 9) {
                Text("Eski portföy bulundu").font(.system(size: 15, weight: .semibold)).foregroundStyle(Theme.text)
                Text("Bu cihazdaki \(portfolio.legacyPreview.count) pozisyonu mevcut bulut portföyüne eklemeden önce onaylayın.")
                    .font(.system(size: 12)).foregroundStyle(Theme.textDim)
                ForEach(portfolio.legacyPreview.prefix(8)) { holding in
                    HStack {
                        Text(holding.code).font(.system(size: 12, weight: .semibold))
                        Spacer()
                        Text("\(Fmt.compactPrice(holding.amount)) adet")
                            .font(Theme.number(11)).foregroundStyle(Theme.textDim)
                    }
                }
                HStack {
                    Button("İçe Aktar") {
                        Task { await portfolio.confirmLegacyImport(prices: prices.prices) }
                    }.buttonStyle(FilledButtonStyle())
                    Button("Atla") { portfolio.skipLegacyImport() }
                        .foregroundStyle(Theme.textDim)
                }
            }
        }
    }

    private var emptyCard: some View {
        Card {
            VStack(spacing: 11) {
                Image(systemName: "briefcase").font(.system(size: 30)).foregroundStyle(Theme.textFaint)
                Text("Portföyün boş").font(.system(size: 17, weight: .semibold)).foregroundStyle(Theme.text)
                Text("Alış, satış, temettü ve ücretleri işlem defterinde takip etmeye başlayın.")
                    .font(.system(size: 12)).foregroundStyle(Theme.textDim).multilineTextAlignment(.center)
                Button("İlk İşlemi Ekle") { showTransaction = true }
                    .buttonStyle(FilledButtonStyle()).frame(maxWidth: 220)
            }
            .frame(maxWidth: .infinity).padding(.vertical, 14)
        }
    }

    private func errorCard(_ message: String) -> some View {
        Card {
            HStack {
                Image(systemName: "exclamationmark.triangle").foregroundStyle(Theme.down)
                Text(message).font(.system(size: 12)).foregroundStyle(Theme.textDim)
            }
        }
    }
}

private struct PortfolioExportFile: Identifiable {
    let id = UUID()
    let url: URL
}

private struct PortfolioShareSheet: UIViewControllerRepresentable {
    let url: URL

    func makeUIViewController(context: Context) -> UIActivityViewController {
        UIActivityViewController(activityItems: [url], applicationActivities: nil)
    }

    func updateUIViewController(_ uiViewController: UIActivityViewController, context: Context) {}
}

private struct PortfolioPerformanceChart: View {
    let points: [PortfolioPerformancePoint]
    let metric: PortfolioChartMetric
    let resetToken: Int
    @Binding var selectedDate: Date?

    private var activePoints: [PortfolioPerformancePoint] {
        guard let first = points.firstIndex(where: {
            $0.valueTRY != 0 || $0.costBasisTRY != 0 || $0.netDepositsTRY != 0
        }) else { return points }
        return Array(points[first...])
    }

    private var candles: [Candle] {
        let dated = activePoints.compactMap { point -> (PortfolioPerformancePoint, Date)? in
            guard let date = point.chartDate else { return nil }
            return (point, date)
        }
        return dated.enumerated().map { index, entry in
            let close = metric.value(in: entry.0)
            let open = index > 0 ? metric.value(in: dated[index - 1].0) : close
            return Candle(
                bucket: entry.1,
                open: open,
                high: max(open, close),
                low: min(open, close),
                close: close,
                count: 0
            )
        }
    }

    private var trendUp: Bool {
        guard let first = candles.first?.close, let last = candles.last?.close else { return true }
        return last >= first
    }

    var body: some View {
        PriceChart(
            candles: candles,
            mode: .line,
            trendUp: trendUp,
            resetToken: resetToken,
            currency: "TRY",
            height: 238,
            seriesTitle: metric.title,
            showsMovingAverages: false,
            showsVolume: false,
            showsOHLCSelection: false,
            emptyTitle: "Henüz yeterli performans verisi yok",
            emptySubtitle: "İşlem ekledikçe geçmiş portföy değeri burada oluşur.",
            onSelectionChange: { selectedDate = $0 }
        )
    }
}

private extension PortfolioPerformancePoint {
    var chartDate: Date? {
        let parts = date.split(separator: "-").compactMap { Int($0) }
        guard parts.count == 3 else { return nil }
        var components = DateComponents()
        components.calendar = Calendar(identifier: .gregorian)
        components.timeZone = TimeZone(secondsFromGMT: 0)
        components.year = parts[0]
        components.month = parts[1]
        components.day = parts[2]
        return components.date
    }

    var displayDate: String {
        chartDate?.formatted(date: .abbreviated, time: .omitted) ?? date
    }
}

private struct PortfolioPositionView: View {
    let position: CloudPortfolioPosition
    @EnvironmentObject private var portfolio: PortfolioStore
    @EnvironmentObject private var prices: PriceStore
    @State private var detail: PortfolioPositionDetail?
    @State private var editor: CloudPortfolioTransaction?
    @State private var showAdd = false

    var body: some View {
        ZStack {
            Theme.bg.ignoresSafeArea()
            ScrollView {
                VStack(spacing: 14) {
                    Card {
                        VStack(alignment: .leading, spacing: 12) {
                            HStack {
                                AssetIcon(code: position.code, kind: position.kind, size: 44)
                                VStack(alignment: .leading) {
                                    Text(position.displayName).font(.system(size: 19, weight: .bold)).foregroundStyle(Theme.text)
                                    Text("\(position.code) • %\(String(format: "%.1f", position.allocationPct)) pay")
                                        .font(.system(size: 11)).foregroundStyle(Theme.textDim)
                                }
                            }
                            LazyVGrid(columns: [GridItem(.flexible()), GridItem(.flexible())], spacing: 12) {
                                detailMetric("Miktar", Fmt.compactPrice(position.quantity))
                                detailMetric(
                                    "Ortalama Maliyet",
                                    "\(position.currency == "USD" ? "$" : "₺")\(Fmt.compactPrice(position.averageCostNative))"
                                )
                                detailMetric(
                                    "Native Değer",
                                    "\(position.currency == "USD" ? "$" : "₺")\(Fmt.compactPrice(position.valueNative))"
                                )
                                detailMetric("TRY Değeri", "₺\(Fmt.compactPrice(position.valueTRY))")
                                detailMetric(
                                    "Günlük K/Z",
                                    "\(Fmt.signed(position.dayChangeTRY)) ₺ · \(Fmt.signed(position.dayChangePct))%",
                                    trend: position.dayChangeTRY
                                )
                                detailMetric(
                                    "Toplam K/Z",
                                    "\(Fmt.signed(position.unrealizedPnLTRY)) ₺ · \(Fmt.signed(position.unrealizedPnLPct))%",
                                    trend: position.unrealizedPnLTRY
                                )
                            }
                        }
                    }
                    NavigationLink {
                        AssetDetailView(code: position.code)
                    } label: {
                        Label("Varlık detayına ve grafiğe git", systemImage: "chart.xyaxis.line")
                            .font(.system(size: 13, weight: .semibold))
                            .frame(maxWidth: .infinity).padding(13)
                            .background(Theme.surface, in: RoundedRectangle(cornerRadius: 12))
                    }
                    .buttonStyle(.plain).foregroundStyle(Theme.accent)
                    Card(padding: 0) {
                        VStack(spacing: 0) {
                            HStack {
                                Text("İşlem Geçmişi").font(.system(size: 15, weight: .semibold)).foregroundStyle(Theme.text)
                                Spacer()
                                Button("Ekle") { showAdd = true }.font(.system(size: 12, weight: .semibold))
                            }.padding(16)
                            ForEach(detail?.transactions ?? []) { transaction in
                                Divider().overlay(Theme.stroke)
                                transactionRow(transaction)
                            }
                        }
                    }
                }.padding(16).padding(.bottom, 80)
            }
        }
        .navigationTitle(position.code).navigationBarTitleDisplayMode(.inline)
        .task { detail = try? await portfolio.positionDetail(code: position.code) }
        .sheet(isPresented: $showAdd) {
            PortfolioTransactionSheet(initialCode: position.code)
                .environmentObject(portfolio).environmentObject(prices)
                .presentationDetents([.large]).presentationBackground(Theme.bgElevated)
        }
        .sheet(item: $editor) { transaction in
            PortfolioTransactionSheet(initialCode: position.code, transaction: transaction)
                .environmentObject(portfolio).environmentObject(prices)
                .presentationDetents([.large]).presentationBackground(Theme.bgElevated)
        }
    }

    private func detailMetric(_ title: String, _ value: String, trend: Double? = nil) -> some View {
        VStack(alignment: .leading, spacing: 3) {
            Text(title).font(.system(size: 10)).foregroundStyle(Theme.textDim)
            Text(value).font(Theme.number(12, weight: .semibold))
                .foregroundStyle(trend == nil ? Theme.text : (trend! >= 0 ? Theme.up : Theme.down))
        }.frame(maxWidth: .infinity, alignment: .leading)
    }

    private func transactionRow(_ transaction: CloudPortfolioTransaction) -> some View {
        HStack {
            VStack(alignment: .leading, spacing: 3) {
                Text(transaction.type.title).font(.system(size: 13, weight: .semibold)).foregroundStyle(Theme.text)
                Text(transaction.occurredAt.formatted(date: .abbreviated, time: .omitted))
                    .font(.system(size: 10)).foregroundStyle(Theme.textDim)
            }
            Spacer()
            Text(transaction.type.isTrade
                 ? "\(Fmt.compactPrice(transaction.quantity)) adet"
                 : "\(transaction.currency == "USD" ? "$" : "₺")\(Fmt.compactPrice(transaction.amountNative))")
                .font(Theme.number(12)).foregroundStyle(Theme.text)
            Menu {
                Button("Düzenle") { editor = transaction }
                Button("Sil", role: .destructive) {
                    Task {
                        await portfolio.delete(transactionId: transaction.id, prices: prices.prices)
                        detail = try? await portfolio.positionDetail(code: position.code)
                    }
                }
            } label: {
                Image(systemName: "ellipsis").frame(width: 32, height: 32).foregroundStyle(Theme.textDim)
            }
        }.padding(.horizontal, 16).padding(.vertical, 11)
    }
}

struct PortfolioTransactionSheet: View {
    var initialCode: String?
    var transaction: CloudPortfolioTransaction?
    @EnvironmentObject private var portfolio: PortfolioStore
    @EnvironmentObject private var prices: PriceStore
    @Environment(\.dismiss) private var dismiss
    @State private var type: PortfolioTransactionType
    @State private var assetKind: String
    @State private var code: String
    @State private var quantity: String
    @State private var unitPrice: String
    @State private var amount: String
    @State private var fee: String
    @State private var note: String
    @State private var occurredAt: Date
    @State private var saving = false
    @State private var showCategoryPicker = false
    @State private var showInstrumentPicker = false

    init(initialCode: String? = nil, transaction: CloudPortfolioTransaction? = nil) {
        self.initialCode = initialCode
        self.transaction = transaction
        let selectedCode = transaction?.code ?? initialCode ?? ""
        _type = State(initialValue: transaction?.type ?? .buy)
        _assetKind = State(initialValue: "")
        _code = State(initialValue: selectedCode)
        _quantity = State(initialValue: transaction.map { String($0.quantity) } ?? "")
        _unitPrice = State(initialValue: transaction.map { String($0.unitPriceNative) } ?? "")
        _amount = State(initialValue: transaction.map { String($0.amountNative) } ?? "")
        _fee = State(initialValue: transaction.map { String($0.feeNative) } ?? "0")
        _note = State(initialValue: transaction?.note ?? "")
        _occurredAt = State(initialValue: transaction?.occurredAt ?? Date())
    }

    var body: some View {
        NavigationStack {
            Form {
                Picker("İşlem", selection: $type) {
                    ForEach(PortfolioTransactionType.allCases) { item in Text(item.title).tag(item) }
                }
                if type.usesAsset {
                    LabeledContent("Varlık türü") {
                        Button {
                            showCategoryPicker = true
                        } label: {
                            HStack(spacing: 5) {
                                Text(selectedCategory?.title ?? "Tür seç")
                                Image(systemName: "chevron.up.chevron.down")
                                    .font(.system(size: 10, weight: .bold))
                            }
                        }
                    }
                    LabeledContent("Varlık") {
                        Button {
                            showInstrumentPicker = selectedCategory != nil
                        } label: {
                            HStack(spacing: 5) {
                                Text(selectedInstrument?.displayName ?? "Varlık seç")
                                    .lineLimit(1)
                                Image(systemName: "magnifyingglass")
                                    .font(.system(size: 11, weight: .bold))
                            }
                        }
                        .disabled(selectedCategory == nil)
                    }
                }
                if type.isTrade {
                    LabeledContent("Miktar") {
                        TextField("0", text: $quantity)
                            .keyboardType(.decimalPad).multilineTextAlignment(.trailing)
                    }
                    LabeledContent("Birim fiyatı") {
                        TextField("0", text: $unitPrice)
                            .keyboardType(.decimalPad).multilineTextAlignment(.trailing)
                    }
                    LabeledContent("Komisyon") {
                        TextField("0", text: $fee)
                            .keyboardType(.decimalPad).multilineTextAlignment(.trailing)
                    }
                } else {
                    LabeledContent("Tutar") {
                        TextField("0", text: $amount)
                            .keyboardType(.decimalPad).multilineTextAlignment(.trailing)
                    }
                }
                DatePicker("İşlem tarihi", selection: $occurredAt, in: ...Date(), displayedComponents: .date)
                TextField("Not (opsiyonel)", text: $note)
            }
            .scrollContentBackground(.hidden).background(Theme.bg)
            .navigationTitle(transaction == nil ? "Yeni İşlem" : "İşlemi Düzenle")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) { Button("Vazgeç") { dismiss() } }
                ToolbarItem(placement: .confirmationAction) {
                    Button(saving ? "Kaydediliyor…" : "Kaydet") {
                        Task { await save() }
                    }.disabled(saving || !isValid)
                }
            }
            .onAppear {
                if let instrument = prices.prices[code] {
                    assetKind = instrument.kind
                    if unitPrice.isEmpty { unitPrice = String(instrument.price) }
                }
            }
            .sheet(isPresented: $showCategoryPicker) {
                InstrumentCategoryPickerSheet(
                    instruments: prices.sorted,
                    selected: selectedCategory
                ) { category in
                    assetKind = category.id
                    code = ""
                    unitPrice = ""
                }
            }
            .sheet(isPresented: $showInstrumentPicker) {
                if let category = selectedCategory {
                    InstrumentPickerSheet(
                        category: category,
                        instruments: prices.sorted,
                        selectedCode: code
                    ) { instrument in
                        code = instrument.code
                        if transaction == nil { unitPrice = String(instrument.price) }
                    }
                }
            }
        }
    }

    private var selectedCategory: InstrumentCategory? {
        InstrumentCategory.all.first(where: { $0.id == assetKind })
    }

    private var selectedInstrument: InstrumentPrice? {
        prices.prices[code]
    }

    private var isValid: Bool {
        if type.isTrade {
            return selectedCategory != nil && selectedInstrument != nil
                && decimal(quantity) > 0 && decimal(unitPrice) > 0 && decimal(fee) >= 0
        }
        return decimal(amount) > 0 && (!type.usesAsset || selectedInstrument != nil)
    }

    private func decimal(_ value: String) -> Double {
        Double(value.replacingOccurrences(of: ",", with: ".")) ?? 0
    }

    private func save() async {
        saving = true
        let instrument = prices.prices[code]
        let body = PortfolioTransactionBody(
            clientId: transaction?.clientId ?? UUID().uuidString,
            code: type.usesAsset ? code : "",
            type: type,
            quantity: type.isTrade ? String(decimal(quantity)) : "0",
            unitPriceNative: type.isTrade ? String(decimal(unitPrice)) : "0",
            amountNative: type.isTrade ? "0" : String(decimal(amount)),
            feeNative: type.isTrade ? String(decimal(fee)) : "0",
            currency: type.usesAsset ? (instrument?.currency ?? "TRY") : "TRY",
            usdtry: "0",
            occurredAt: occurredAt,
            note: note
        )
        let saved = await portfolio.save(
            body: body, transactionId: transaction?.id, prices: prices.prices
        )
        saving = false
        if saved {
            Haptics.success()
            dismiss()
        }
    }
}
