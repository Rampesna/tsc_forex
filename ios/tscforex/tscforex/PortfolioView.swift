import SwiftUI

struct PortfolioView: View {
    @EnvironmentObject private var portfolio: PortfolioStore
    @EnvironmentObject private var prices: PriceStore
    @EnvironmentObject private var auth: AuthStore
    @State private var showTransaction = false
    @State private var range = "1y"

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
                            summaryCard
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
            VStack(alignment: .leading, spacing: 12) {
                HStack {
                    Text("Performans").font(.system(size: 15, weight: .semibold)).foregroundStyle(Theme.text)
                    Spacer()
                    HStack(spacing: 4) {
                        ForEach(["1m", "3m", "1y", "5y"], id: \.self) { item in
                            Button(item.uppercased()) {
                                range = item
                                Task { await portfolio.refreshPerformance(range: item) }
                            }
                            .font(.system(size: 10, weight: .semibold))
                            .foregroundStyle(range == item ? Theme.accent : Theme.textDim)
                            .padding(.horizontal, 7).padding(.vertical, 5)
                            .background(range == item ? Theme.accent.opacity(0.12) : Color.clear,
                                        in: RoundedRectangle(cornerRadius: 7))
                        }
                    }
                }
                PortfolioPerformanceChart(points: portfolio.performance)
                    .frame(height: 150)
            }
        }
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

private struct PortfolioPerformanceChart: View {
    let points: [PortfolioPerformancePoint]

    var body: some View {
        if points.count < 2 {
            Text("Henüz yeterli performans verisi yok")
                .font(.system(size: 12)).foregroundStyle(Theme.textFaint)
                .frame(maxWidth: .infinity, maxHeight: .infinity)
        } else {
            Canvas { context, size in
                let firstActive = points.firstIndex {
                    $0.valueTRY != 0 || $0.costBasisTRY != 0
                } ?? 0
                let values = Array(points.dropFirst(firstActive)).map(\.valueTRY)
                let lo = values.min() ?? 0
                let hi = values.max() ?? 1
                let span = max(hi - lo, max(hi * 0.01, 1))
                var line = Path()
                for (index, value) in values.enumerated() {
                    let x = CGFloat(index) / CGFloat(max(values.count - 1, 1)) * size.width
                    let y = size.height - CGFloat((value - lo) / span) * (size.height - 12) - 6
                    if index == 0 { line.move(to: CGPoint(x: x, y: y)) }
                    else { line.addLine(to: CGPoint(x: x, y: y)) }
                }
                var fill = line
                fill.addLine(to: CGPoint(x: size.width, y: size.height))
                fill.addLine(to: CGPoint(x: 0, y: size.height))
                fill.closeSubpath()
                context.fill(fill, with: .linearGradient(
                    Gradient(colors: [Theme.accent.opacity(0.22), Theme.accent.opacity(0)]),
                    startPoint: .zero, endPoint: CGPoint(x: 0, y: size.height)
                ))
                context.stroke(line, with: .color(Theme.accent), lineWidth: 2)
            }
        }
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
                .foregroundStyle(trend.map(Theme.trend) ?? Theme.text)
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
