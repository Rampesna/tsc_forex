import SwiftUI

struct GamificationProgressSummaryCard: View {
    let experience: GamificationExperience

    var body: some View {
        Card {
            VStack(alignment: .leading, spacing: 10) {
                HStack {
                    Text("Gelişim Özeti")
                        .font(.system(size: 15, weight: .bold))
                        .foregroundStyle(Theme.text)
                    Spacer()
                    Text("v\(experience.ruleVersion)")
                        .font(.system(size: 10, weight: .bold))
                        .foregroundStyle(Theme.textFaint)
                }
                HStack(spacing: 8) {
                    compactStat("Ustalık", "\(experience.weeklySummary.masteredLessons)/\(experience.weeklySummary.totalLessons)")
                    compactStat("Tekrar", "\(experience.weeklySummary.dueReviews)")
                    compactStat("Karar", "\(experience.paperTrades.count)")
                }
                Text("Puanlama finansal getiriden bağımsızdır; öğrenme ve karar sürecini ölçer.")
                    .font(.system(size: 10)).foregroundStyle(Theme.textFaint)
            }
        }
    }

    private func compactStat(_ label: String, _ value: String) -> some View {
        VStack(alignment: .leading, spacing: 2) {
            Text(label).font(.system(size: 9)).foregroundStyle(Theme.textFaint)
            Text(value).font(Theme.number(16, weight: .bold)).foregroundStyle(Theme.text)
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .padding(9).background(Theme.surfaceHi, in: RoundedRectangle(cornerRadius: 10))
    }
}

struct GamificationExperienceHub: View {
    @Binding var experience: GamificationExperience?
    let token: String?
    let onProfile: (GamProfile) -> Void
    let onToast: (String) -> Void

    @EnvironmentObject private var prices: PriceStore
    @State private var busy = false
    @State private var risk = "moderate"
    @State private var watchCodes: [String] = []
    @State private var showWatchPicker = false
    @State private var showPaperPicker = false
    @State private var learningGoal = "risk_budget"
    @State private var reflection = ""
    @State private var selectedLesson = ""
    @State private var selectedAnswer: Int?
    @State private var disciplineCode = "allocation_review"
    @State private var disciplineNote = ""
    @State private var paperCode = "GRAM_GOLD"
    @State private var paperSide = "buy"
    @State private var paperQuantity = "1"
    @State private var paperPrice = ""
    @State private var paperRisk = "1"
    @State private var paperThesis = ""
    @State private var paperInvalidation = ""
    @State private var reviewTradeID = ""
    @State private var followedPlan = true
    @State private var retrospective = ""
    @State private var alias = ""
    @State private var leagueOptIn = true
    @State private var personalized = true
    @State private var socialOptIn = false
    @State private var learningNotifications = true
    @State private var groupName = ""
    @State private var inviteCode = ""

    var body: some View {
        if let exp = experience {
            VStack(spacing: 12) {
                HStack(spacing: 10) {
                    journeyLink("Günlük Plan", "3 adım · yaklaşık 1 dakika", "scope") {
                        journeyScreen("Günlük Plan") { today(exp) }
                    }
                    journeyLink("Öğrenme", "Kısa ders · tek soru", "book.closed.fill") {
                        journeyScreen("Öğrenme") { learning(exp) }
                    }
                }
                HStack(spacing: 10) {
                    journeyLink("Karar Günlüğü", "Sanal karar ve sonradan değerlendirme", "checklist") {
                        journeyScreen("Karar Günlüğü") {
                            VStack(spacing: 12) { decisionIntro; discipline(exp); paper(exp) }
                        }
                    }
                    journeyLink("Gizlilik", "Paylaşım tercihleri", "lock.shield.fill") {
                        journeyScreen("Gizlilik") { privacy(exp) }
                    }
                }
            }
            .onAppear { sync(exp) }
            .sheet(isPresented: $showWatchPicker) {
                InstrumentMultiPickerSheet(
                    instruments: prices.sorted,
                    selectedCodes: watchCodes,
                    maxSelection: 3
                ) { watchCodes = $0 }
            }
            .sheet(isPresented: $showPaperPicker) {
                InstrumentMultiPickerSheet(
                    instruments: prices.sorted,
                    selectedCodes: [paperCode],
                    maxSelection: 1
                ) { selected in
                    guard let code = selected.first else { return }
                    paperCode = code
                    if let value = prices.prices[code]?.price {
                        paperPrice = String(format: "%.2f", value)
                    }
                }
            }
        }
    }

    private func journeyLink<Destination: View>(
        _ title: String, _ subtitle: String, _ icon: String,
        @ViewBuilder destination: () -> Destination
    ) -> some View {
        NavigationLink {
            destination()
        } label: {
            Card(padding: 13) {
                VStack(alignment: .leading, spacing: 8) {
                    Image(systemName: icon).font(.system(size: 18, weight: .bold)).foregroundStyle(Theme.accent)
                    Text(title).font(.system(size: 13, weight: .bold)).foregroundStyle(Theme.text)
                    Text(subtitle).font(.system(size: 10)).foregroundStyle(Theme.textDim).lineLimit(2)
                    HStack { Spacer(); Image(systemName: "chevron.right").font(.system(size: 10, weight: .bold)).foregroundStyle(Theme.textFaint) }
                }
                .frame(maxWidth: .infinity, alignment: .leading)
            }
        }
        .buttonStyle(.plain)
    }

    private func journeyScreen<Content: View>(
        _ title: String, @ViewBuilder content: () -> Content
    ) -> some View {
        ScrollView {
            content().padding(.horizontal, 16).padding(.top, 12).padding(.bottom, 40)
        }
        .background(Theme.bg.ignoresSafeArea())
        .navigationTitle(title)
        .navigationBarTitleDisplayMode(.inline)
    }

    private func today(_ exp: GamificationExperience) -> some View {
        VStack(spacing: 12) {
            Card {
                VStack(alignment: .leading, spacing: 14) {
                    HStack(alignment: .top) {
                        VStack(alignment: .leading, spacing: 3) {
                            Label(exp.dailyPlan == nil ? "Bugünü Planla" : "Bugünün Planı", systemImage: "scope")
                                .font(.system(size: 16, weight: .bold)).foregroundStyle(Theme.text)
                            Text("Bugün neyi izleyeceğini ve risk sınırını bir kez belirle.")
                                .font(.system(size: 11)).foregroundStyle(Theme.textDim)
                        }
                        Spacer()
                        Text("~1 dk")
                            .font(.system(size: 10, weight: .bold)).foregroundStyle(Theme.accent)
                            .padding(.horizontal, 8).padding(.vertical, 5)
                            .background(Theme.accentSoft, in: Capsule())
                    }

                    stepLabel("1", "Bugünkü risk sınırın", "Tek bir seçim yap")
                    Picker("Risk", selection: $risk) {
                        Text("Düşük").tag("low")
                        Text("Orta").tag("moderate")
                        Text("Yüksek").tag("high")
                    }
                    .pickerStyle(.segmented)

                    stepLabel("2", "İzleme listen", "En fazla 3 varlık")
                    watchSelection

                    stepLabel("3", "Bugünkü öğrenme odağın", "Bir kısa ders seç")
                    Menu {
                        ForEach(exp.lessons) { lesson in
                            Button(lesson.title) { learningGoal = lesson.code }
                        }
                    } label: {
                        HStack(spacing: 10) {
                            Image(systemName: "book.closed.fill").foregroundStyle(Theme.accent)
                            Text(exp.lessons.first(where: { $0.code == learningGoal })?.title ?? "Ders seç")
                                .font(.system(size: 13, weight: .semibold)).foregroundStyle(Theme.text)
                                .lineLimit(2)
                            Spacer()
                            Image(systemName: "chevron.up.chevron.down")
                                .font(.system(size: 10, weight: .bold)).foregroundStyle(Theme.textFaint)
                        }
                        .padding(.horizontal, 12).frame(minHeight: 46)
                        .background(Theme.surfaceHi, in: RoundedRectangle(cornerRadius: 10))
                        .overlay(RoundedRectangle(cornerRadius: 10).stroke(Theme.stroke))
                    }
                    .buttonStyle(.plain)

                    VStack(alignment: .leading, spacing: 6) {
                        Text("Kısa not (isteğe bağlı)")
                            .font(.system(size: 11, weight: .semibold)).foregroundStyle(Theme.textDim)
                        input("Örn. Açılışta acele karar vermeyeceğim.", text: $reflection, axis: .vertical)
                        Text("Not yalnızca kendi karar sürecini hatırlatmak içindir.")
                            .font(.system(size: 9.5)).foregroundStyle(Theme.textFaint)
                    }

                    if exp.dailyPlan != nil {
                        Label("Bugünkü plan kaydedildi; değiştirip yeniden kaydedebilirsin.", systemImage: "checkmark.circle.fill")
                            .font(.system(size: 11, weight: .semibold)).foregroundStyle(Theme.up)
                    }
                    Button(busy ? "Kaydediliyor…" : "Bugünün Planını Kaydet") {
                        Task { await savePlan() }
                    }
                    .buttonStyle(FilledButtonStyle())
                    .disabled(busy)
                }
            }
        }
    }

    private func discipline(_ exp: GamificationExperience) -> some View {
        Card {
            VStack(alignment: .leading, spacing: 12) {
                Label("Bugünün Kısa Değerlendirmesi", systemImage: "text.badge.checkmark")
                    .font(.system(size: 15, weight: .bold)).foregroundStyle(Theme.text)
                Text("Bir başlık seç ve karar verirken neyi kontrol ettiğini kendi cümlenle yaz. Kâr veya zarar girmen gerekmez.")
                    .font(.system(size: 11)).foregroundStyle(Theme.textDim)
                if let quest = exp.disciplineQuests.first(where: { $0.code == disciplineCode }) {
                    Menu {
                        ForEach(exp.disciplineQuests) { item in
                            Button(item.title) {
                                disciplineCode = item.code
                                disciplineNote = ""
                            }
                        }
                    } label: {
                        HStack {
                            VStack(alignment: .leading, spacing: 2) {
                                Text(quest.title).font(.system(size: 13, weight: .bold)).foregroundStyle(Theme.text)
                                Text(quest.completed ? "Bugün tamamlandı" : "+\(quest.xp) XP")
                                    .font(.system(size: 10, weight: .semibold))
                                    .foregroundStyle(quest.completed ? Theme.up : Theme.accent)
                            }
                            Spacer()
                            Image(systemName: "chevron.up.chevron.down")
                                .font(.system(size: 10, weight: .bold)).foregroundStyle(Theme.textFaint)
                        }
                        .padding(11).background(Theme.surfaceHi, in: RoundedRectangle(cornerRadius: 10))
                    }
                    .buttonStyle(.plain)
                    Text(quest.description).font(.system(size: 11)).foregroundStyle(Theme.textDim)
                }
                input("En az 20 karakterlik kısa değerlendirmen…", text: $disciplineNote, axis: .vertical)
                characterHint(disciplineNote, minimum: 20)
                Button(busy ? "Kaydediliyor…" : "Değerlendirmeyi Kaydet") {
                    Task { await completeDiscipline() }
                }
                .buttonStyle(FilledButtonStyle())
                .disabled(busy)
            }
        }
    }

    private var decisionIntro: some View {
        Card {
            VStack(alignment: .leading, spacing: 10) {
                Label("Bu alan ne işe yarar?", systemImage: "lightbulb.fill")
                    .font(.system(size: 15, weight: .bold)).foregroundStyle(Theme.text)
                Text("Karar Günlüğü gerçek alım-satım yapmaz ve portföyünü değiştirmez. Bir fikri uygulamadan önce nedenini ve risk sınırını kaydetmene, daha sonra da sonuca kapılmadan kendi sürecini değerlendirmene yardımcı olur.")
                    .font(.system(size: 11)).foregroundStyle(Theme.textDim)
                HStack(spacing: 8) {
                    decisionStep("1", "Fikri yaz")
                    decisionStep("2", "İzle")
                    decisionStep("3", "Değerlendir")
                }
            }
        }
    }

    private var watchSelection: some View {
        VStack(alignment: .leading, spacing: 7) {
            HStack {
                Text("İzlenecek varlıklar")
                    .font(.system(size: 11, weight: .semibold))
                    .foregroundStyle(Theme.textDim)
                Spacer()
                Text("\(watchCodes.count)/3")
                    .font(Theme.number(10, weight: .bold))
                    .foregroundStyle(Theme.textFaint)
            }
            Button {
                showWatchPicker = true
            } label: {
                HStack(spacing: 10) {
                    Image(systemName: "magnifyingglass")
                        .foregroundStyle(Theme.accent)
                    Text(watchCodes.isEmpty ? "Varlık seç" : "Seçimi düzenle")
                        .font(.system(size: 13, weight: .semibold))
                        .foregroundStyle(Theme.text)
                    Spacer()
                    Image(systemName: "chevron.right")
                        .font(.system(size: 11, weight: .bold))
                        .foregroundStyle(Theme.textFaint)
                }
                .padding(.horizontal, 12)
                .frame(height: 44)
                .background(Theme.surfaceHi, in: RoundedRectangle(cornerRadius: 10))
                .overlay(RoundedRectangle(cornerRadius: 10).stroke(Theme.stroke))
            }
            .buttonStyle(.plain)
            if !selectedWatchInstruments.isEmpty {
                ScrollView(.horizontal, showsIndicators: false) {
                    HStack(spacing: 6) {
                        ForEach(selectedWatchInstruments) { instrument in
                            HStack(spacing: 5) {
                                Text(instrument.displayName)
                                    .font(.system(size: 11, weight: .semibold))
                                Button {
                                    watchCodes.removeAll { $0 == instrument.code }
                                } label: {
                                    Image(systemName: "xmark")
                                        .font(.system(size: 8, weight: .bold))
                                }
                                .buttonStyle(.plain)
                            }
                            .foregroundStyle(Theme.accent)
                            .padding(.horizontal, 9)
                            .padding(.vertical, 7)
                            .background(Theme.accentSoft, in: Capsule())
                        }
                    }
                }
            }
        }
    }

    private var selectedWatchInstruments: [InstrumentPrice] {
        watchCodes.compactMap { prices.prices[$0] }
    }

    private func learning(_ exp: GamificationExperience) -> some View {
        let lesson = exp.lessons.first(where: { $0.code == selectedLesson }) ?? exp.lessons.first
        return VStack(spacing: 12) {
            Card {
                VStack(alignment: .leading, spacing: 8) {
                    Label("Kısa Öğrenme", systemImage: "book.closed.fill")
                        .font(.system(size: 16, weight: .bold)).foregroundStyle(Theme.text)
                    Text("Bir başlık seç, üç kısa kartı oku ve tek soruyu yanıtla. Yanlış yanıt ceza vermez; dersi tekrar deneyebilirsin.")
                        .font(.system(size: 11)).foregroundStyle(Theme.textDim)
                    HStack(spacing: 6) {
                        learningStep("1", "Oku")
                        learningStep("2", "Yanıtla")
                        learningStep("3", "Kontrol et")
                    }
                }
            }

            Card {
                VStack(alignment: .leading, spacing: 12) {
                    Text("Dersler").font(.system(size: 12, weight: .bold)).foregroundStyle(Theme.textDim)
                    ScrollView(.horizontal, showsIndicators: false) {
                        HStack {
                            ForEach(exp.lessons) { item in
                                Button {
                                    Haptics.tap()
                                    selectedLesson = item.code
                                    selectedAnswer = nil
                                } label: {
                                    Text("\(item.completed ? "✓ " : item.dueForReview ? "↻ " : "")\(item.title)")
                                        .font(.system(size: 11, weight: .semibold))
                                        .foregroundStyle(item.code == lesson?.code ? Theme.accent : Theme.textDim)
                                        .padding(.horizontal, 10).padding(.vertical, 7)
                                        .background(Theme.surfaceHi, in: Capsule())
                                        .overlay(Capsule().stroke(item.code == lesson?.code ? Theme.accent : Theme.stroke))
                                }
                                .buttonStyle(.plain)
                            }
                        }
                    }
                    if let lesson {
                        HStack(alignment: .top) {
                            VStack(alignment: .leading, spacing: 3) {
                                Text(lesson.title).font(.system(size: 15, weight: .bold)).foregroundStyle(Theme.text)
                                Text(lesson.summary).font(.system(size: 11)).foregroundStyle(Theme.textDim)
                            }
                            Spacer()
                            Text("%\(lesson.mastery)")
                                .font(Theme.number(13, weight: .bold)).foregroundStyle(Theme.accent)
                        }
                        ProgressView(value: Double(lesson.mastery), total: 100)
                            .tint(lesson.completed ? Theme.up : Theme.accent)
                        ForEach(Array(lesson.cards.enumerated()), id: \.offset) { index, text in
                            HStack(alignment: .top, spacing: 9) {
                                Text("\(index + 1)")
                                    .font(Theme.number(10, weight: .bold)).foregroundStyle(Theme.accent)
                                    .frame(width: 22, height: 22).background(Theme.accentSoft, in: Circle())
                                Text(text).font(.system(size: 11)).foregroundStyle(Theme.text)
                            }
                            .padding(10).frame(maxWidth: .infinity, alignment: .leading)
                            .background(Theme.surfaceHi, in: RoundedRectangle(cornerRadius: 9))
                        }
                        Divider().overlay(Theme.stroke)
                        Text("Tek soru").font(.system(size: 10, weight: .bold)).foregroundStyle(Theme.accent)
                        Text(lesson.question.text).font(.system(size: 13, weight: .bold)).foregroundStyle(Theme.text)
                        ForEach(Array(lesson.question.choices.enumerated()), id: \.offset) { index, choice in
                            Button {
                                Haptics.tap()
                                selectedAnswer = index
                            } label: {
                                HStack {
                                    Image(systemName: selectedAnswer == index ? "largecircle.fill.circle" : "circle")
                                    Text(choice).multilineTextAlignment(.leading)
                                    Spacer()
                                }
                                .font(.system(size: 12)).foregroundStyle(Theme.text)
                                .padding(10).background(selectedAnswer == index ? Theme.accentSoft : Theme.surfaceHi,
                                                        in: RoundedRectangle(cornerRadius: 9))
                            }.buttonStyle(.plain)
                        }
                        Button(busy ? "Kontrol ediliyor…" : "Yanıtımı Kontrol Et") {
                            Task { await submitQuiz(lesson) }
                        }
                        .buttonStyle(FilledButtonStyle()).disabled(busy)
                    } else {
                        Text("Dersler şu anda yüklenemedi. Ekranı aşağı çekerek yeniden deneyebilirsin.")
                            .font(.system(size: 12)).foregroundStyle(Theme.textDim)
                    }
                }
            }
        }
    }

    private func paper(_ exp: GamificationExperience) -> some View {
        VStack(spacing: 12) {
            Card {
                VStack(alignment: .leading, spacing: 12) {
                    Label("Sanal Karar Oluştur", systemImage: "square.and.pencil")
                        .font(.system(size: 15, weight: .bold)).foregroundStyle(Theme.text)
                    Text("Buradaki kayıt emir değildir. Fikrini, kabul ettiğin riski ve vazgeçme koşulunu karar anında sabitler.")
                        .font(.system(size: 11)).foregroundStyle(Theme.textDim)

                    fieldLabel("Varlık")
                    Button {
                        Haptics.tap()
                        showPaperPicker = true
                    } label: {
                        HStack(spacing: 10) {
                            Image(systemName: "magnifyingglass").foregroundStyle(Theme.accent)
                            VStack(alignment: .leading, spacing: 2) {
                                Text(prices.prices[paperCode]?.displayName ?? paperCode)
                                    .font(.system(size: 13, weight: .semibold)).foregroundStyle(Theme.text)
                                Text(paperCode).font(Theme.number(9, weight: .semibold)).foregroundStyle(Theme.textFaint)
                            }
                            Spacer()
                            Text("Değiştir").font(.system(size: 11, weight: .bold)).foregroundStyle(Theme.accent)
                        }
                        .padding(11).background(Theme.surfaceHi, in: RoundedRectangle(cornerRadius: 10))
                    }
                    .buttonStyle(.plain)

                    fieldLabel("Beklentin")
                    Picker("Yön", selection: $paperSide) {
                        Text("Yükseliş").tag("buy")
                        Text("Düşüş").tag("sell")
                    }
                    .pickerStyle(.segmented)

                    HStack(alignment: .top, spacing: 8) {
                        labeledInput("Miktar", placeholder: "1", text: $paperQuantity)
                            .keyboardType(.decimalPad)
                        labeledInput("Referans fiyat", placeholder: "0,00", text: $paperPrice)
                            .keyboardType(.decimalPad)
                    }
                    VStack(alignment: .leading, spacing: 5) {
                        labeledInput("Risk bütçesi (%)", placeholder: "1", text: $paperRisk)
                            .keyboardType(.decimalPad)
                        Text("0 ile 5 arasında olmalı; bu oran gerçek para ayırmaz.")
                            .font(.system(size: 9.5)).foregroundStyle(Theme.textFaint)
                    }

                    fieldLabel("Bu fikrin dayanağı ne?")
                    input("En az 20 karakterle tezini yaz…", text: $paperThesis, axis: .vertical)
                    characterHint(paperThesis, minimum: 20)
                    fieldLabel("Hangi koşulda bu fikirden vazgeçersin?")
                    input("En az 10 karakterle geçersizleşme koşulunu yaz…", text: $paperInvalidation, axis: .vertical)
                    characterHint(paperInvalidation, minimum: 10)
                    Button(busy ? "Kaydediliyor…" : "Sanal Kararı Kaydet") {
                        Task { await createPaperTrade() }
                    }
                    .buttonStyle(FilledButtonStyle())
                    .disabled(busy)
                }
            }
            if exp.paperTrades.isEmpty {
                Card {
                    VStack(spacing: 8) {
                        Image(systemName: "doc.text.magnifyingglass")
                            .font(.system(size: 25)).foregroundStyle(Theme.textFaint)
                        Text("Henüz kayıtlı sanal karar yok")
                            .font(.system(size: 13, weight: .bold)).foregroundStyle(Theme.text)
                        Text("İlk kaydını oluşturduğunda burada tezini ve sonraki değerlendirmeni birlikte göreceksin.")
                            .font(.system(size: 10.5)).foregroundStyle(Theme.textDim)
                            .multilineTextAlignment(.center)
                    }
                    .frame(maxWidth: .infinity).padding(.vertical, 8)
                }
            }
            ForEach(exp.paperTrades.prefix(8)) { trade in
                Card {
                    VStack(alignment: .leading, spacing: 9) {
                        HStack {
                            Text("\(prices.prices[trade.code]?.displayName ?? trade.code) · \(trade.side == "buy" ? "Yükseliş" : "Düşüş")")
                                .font(.system(size: 13, weight: .bold)).foregroundStyle(Theme.text)
                            Spacer()
                            Text(trade.reviewedAt == nil ? "Değerlendirme bekliyor" : "Süreç \(trade.processScore)/100")
                                .font(.system(size: 10, weight: .bold))
                                .foregroundStyle(trade.reviewedAt == nil ? Theme.textDim : Theme.up)
                        }
                        Text("\(trade.quantity.formatted()) adet · Referans \(Fmt.compactPrice(trade.entryPrice)) · Risk %\(trade.riskBudgetPct.formatted())")
                            .font(Theme.number(10, weight: .medium)).foregroundStyle(Theme.textFaint)
                        VStack(alignment: .leading, spacing: 3) {
                            Text("TEZ").font(.system(size: 9, weight: .bold)).foregroundStyle(Theme.accent)
                            Text(trade.thesis).font(.system(size: 11)).foregroundStyle(Theme.textDim)
                            Text("VAZGEÇME KOŞULU").font(.system(size: 9, weight: .bold)).foregroundStyle(Theme.down)
                            Text(trade.invalidation).font(.system(size: 11)).foregroundStyle(Theme.textDim)
                        }
                        if trade.reviewedAt == nil {
                            Button {
                                Haptics.tap()
                                reviewTradeID = reviewTradeID == trade.id ? "" : trade.id
                                retrospective = ""
                            } label: {
                                Label(
                                    reviewTradeID == trade.id ? "Değerlendirmeyi Kapat" : "Bu Kararı Değerlendir",
                                    systemImage: reviewTradeID == trade.id ? "chevron.up" : "checklist"
                                )
                                .font(.system(size: 11, weight: .bold)).foregroundStyle(Theme.accent)
                                .frame(maxWidth: .infinity).padding(.vertical, 9)
                                .background(Theme.accentSoft, in: RoundedRectangle(cornerRadius: 9))
                            }
                            .buttonStyle(.plain)
                        }
                        if reviewTradeID == trade.id {
                            Toggle("Planıma uydum", isOn: $followedPlan).font(.system(size: 12)).tint(Theme.accent)
                            Text("Sonucun iyi veya kötü olmasından bağımsız, plana ne kadar uyduğunu yaz.")
                                .font(.system(size: 10.5)).foregroundStyle(Theme.textDim)
                            input("En az 20 karakterlik süreç değerlendirmen…", text: $retrospective, axis: .vertical)
                            characterHint(retrospective, minimum: 20)
                            Button(busy ? "Kaydediliyor…" : "Değerlendirmeyi Tamamla") {
                                Task { await reviewPaperTrade(trade.id) }
                            }.buttonStyle(FilledButtonStyle()).disabled(busy)
                        } else if trade.reviewedAt != nil, !trade.retrospective.isEmpty {
                            Label(trade.retrospective, systemImage: "checkmark.seal.fill")
                                .font(.system(size: 11)).foregroundStyle(Theme.up)
                        }
                    }
                }
            }
        }
    }

    private func privacy(_ exp: GamificationExperience) -> some View {
        VStack(spacing: 12) {
            Card {
                VStack(alignment: .leading, spacing: 9) {
                    Text("Gizlilik Kontrolleri").font(.system(size: 15, weight: .bold)).foregroundStyle(Theme.text)
                    Text("Para, pozisyon ve getiri sosyal yüzeylerde paylaşılmaz.")
                        .font(.system(size: 11)).foregroundStyle(Theme.textDim)
                    input("Takma ad", text: $alias)
                    Toggle("Kişiselleştirilmiş öğrenme", isOn: $personalized)
                    Toggle("Öğrenme bildirimleri", isOn: $learningNotifications)
                    Toggle("Gizlilik güvenli çalışma grupları", isOn: $socialOptIn)
                    Button(busy ? "Kaydediliyor…" : "Tercihleri Kaydet") {
                        Task { await savePrivacy(exp.privacy.userId) }
                    }.buttonStyle(FilledButtonStyle()).disabled(busy)
                }
                .font(.system(size: 12)).tint(Theme.accent)
            }
            if socialOptIn {
                Card {
                    VStack(alignment: .leading, spacing: 9) {
                        Text("Çalışma Grupları").font(.system(size: 15, weight: .bold)).foregroundStyle(Theme.text)
                        HStack {
                            input("Yeni grup adı", text: $groupName)
                            Button("Oluştur") { Task { await createGroup() } }
                                .disabled(groupName.count < 3 || busy)
                        }
                        HStack {
                            input("Davet kodu", text: $inviteCode)
                            Button("Katıl") { Task { await joinGroup() } }
                                .disabled(inviteCode.count < 6 || busy)
                        }
                        ForEach(exp.studyGroups) { group in
                            HStack {
                                Text("\(group.name) · \(group.memberCount)/8")
                                    .font(.system(size: 11, weight: .semibold)).foregroundStyle(Theme.text)
                                Spacer()
                                Text(group.inviteCode).font(Theme.number(11, weight: .bold)).foregroundStyle(Theme.accent)
                            }
                            .padding(9).background(Theme.surfaceHi, in: RoundedRectangle(cornerRadius: 9))
                        }
                    }
                }
            }
        }
    }

    private func input(_ placeholder: String, text: Binding<String>, axis: Axis = .horizontal) -> some View {
        TextField(placeholder, text: text, axis: axis)
            .font(.system(size: 12)).foregroundStyle(Theme.text)
            .padding(10).background(Theme.surfaceHi, in: RoundedRectangle(cornerRadius: 9))
            .overlay(RoundedRectangle(cornerRadius: 9).stroke(Theme.stroke))
            .textInputAutocapitalization(.sentences)
    }

    private func labeledInput(
        _ title: String, placeholder: String, text: Binding<String>
    ) -> some View {
        VStack(alignment: .leading, spacing: 5) {
            fieldLabel(title)
            input(placeholder, text: text)
        }
        .frame(maxWidth: .infinity)
    }

    private func fieldLabel(_ text: String) -> some View {
        Text(text)
            .font(.system(size: 11, weight: .semibold))
            .foregroundStyle(Theme.textDim)
    }

    private func stepLabel(_ number: String, _ title: String, _ detail: String) -> some View {
        HStack(spacing: 8) {
            Text(number)
                .font(Theme.number(10, weight: .bold)).foregroundStyle(Theme.accent)
                .frame(width: 22, height: 22).background(Theme.accentSoft, in: Circle())
            Text(title).font(.system(size: 12, weight: .bold)).foregroundStyle(Theme.text)
            Spacer()
            Text(detail).font(.system(size: 9.5)).foregroundStyle(Theme.textFaint)
        }
    }

    private func decisionStep(_ number: String, _ title: String) -> some View {
        VStack(spacing: 4) {
            Text(number)
                .font(Theme.number(10, weight: .bold)).foregroundStyle(Theme.accent)
                .frame(width: 22, height: 22).background(Theme.accentSoft, in: Circle())
            Text(title).font(.system(size: 9.5, weight: .semibold)).foregroundStyle(Theme.textDim)
        }
        .frame(maxWidth: .infinity).padding(.vertical, 7)
        .background(Theme.surfaceHi, in: RoundedRectangle(cornerRadius: 9))
    }

    private func learningStep(_ number: String, _ title: String) -> some View {
        HStack(spacing: 5) {
            Text(number).font(Theme.number(9, weight: .bold)).foregroundStyle(Theme.accent)
            Text(title).font(.system(size: 9.5, weight: .semibold)).foregroundStyle(Theme.textDim)
        }
        .frame(maxWidth: .infinity).padding(.vertical, 7)
        .background(Theme.surfaceHi, in: RoundedRectangle(cornerRadius: 8))
    }

    private func characterHint(_ text: String, minimum: Int) -> some View {
        let count = text.trimmingCharacters(in: .whitespacesAndNewlines).count
        return HStack {
            Text(count >= minimum ? "Hazır" : "\(minimum - count) karakter daha gerekli")
            Spacer()
            Text("\(count)/\(minimum)")
        }
        .font(.system(size: 9.5, weight: .medium))
        .foregroundStyle(count >= minimum ? Theme.up : Theme.textFaint)
    }

    private func sync(_ exp: GamificationExperience) {
        if let plan = exp.dailyPlan {
            risk = plan.riskAppetite
            watchCodes = Array(plan.watchCodes.prefix(3))
            learningGoal = plan.learningGoal
            reflection = plan.reflection
        } else if let first = exp.lessons.first {
            learningGoal = first.code
        }
        if selectedLesson.isEmpty || !exp.lessons.contains(where: { $0.code == selectedLesson }) {
            selectedLesson = exp.lessons.first?.code ?? ""
        }
        if disciplineCode.isEmpty ||
            exp.disciplineQuests.first(where: { $0.code == disciplineCode })?.completed == true {
            disciplineCode = exp.disciplineQuests.first(where: { !$0.completed })?.code
                ?? exp.disciplineQuests.first?.code
                ?? disciplineCode
        }
        if paperPrice.isEmpty, let currentPrice = prices.prices[paperCode]?.price {
            paperPrice = String(format: "%.2f", currentPrice)
        }
        alias = exp.privacy.alias
        leagueOptIn = exp.privacy.leagueOptIn
        personalized = exp.privacy.personalizedLearning
        socialOptIn = exp.privacy.socialOptIn
        learningNotifications = exp.privacy.learningNotifications
    }

    private func reload() async {
        if let value = try? await APIClient(token: token).gamificationExperience() {
            experience = value
            sync(value)
        }
    }

    private func savePlan() async {
        guard !watchCodes.isEmpty else {
            Haptics.warning()
            onToast("Plan için en az bir varlık seçmelisin.")
            return
        }
        guard !learningGoal.isEmpty else {
            Haptics.warning()
            onToast("Bugünkü öğrenme odağını seçmelisin.")
            return
        }
        guard reflection.count <= 500 else {
            Haptics.warning()
            onToast("Kısa not en fazla 500 karakter olabilir.")
            return
        }
        busy = true
        defer { busy = false }
        do {
            let result = try await APIClient(token: token).saveDailyPlan(
                risk: risk, codes: Array(watchCodes.prefix(3)), learningGoal: learningGoal, reflection: reflection
            )
            onProfile(result.profile)
            let reward = result.xpGained > 0 ? " · +\(result.xpGained) XP" : ""
            onToast("Bugünün planı kaydedildi\(reward) · 🔥 \(result.profile.currentStreak) gün")
            Haptics.success()
            await reload()
        } catch { onToast(actionError(error)) }
    }

    private func submitQuiz(_ lesson: LessonStatus) async {
        guard let answer = selectedAnswer else {
            Haptics.warning()
            onToast("Kontrol etmeden önce bir yanıt seçmelisin.")
            return
        }
        busy = true
        defer { busy = false }
        do {
            let result = try await APIClient(token: token).submitQuiz(
                lesson: lesson.code, question: lesson.question.code, answer: answer
            )
            onToast(result.correct ? "Doğru · +\(result.xpGained) XP" : "Henüz değil; kartları tekrar gözden geçir.")
            if result.correct { Haptics.success() }
            selectedAnswer = nil
            await reload()
        } catch { onToast(actionError(error)) }
    }

    private func completeDiscipline() async {
        let note = disciplineNote.trimmingCharacters(in: .whitespacesAndNewlines)
        guard note.count >= 20 else {
            Haptics.warning()
            onToast("Değerlendirmeni en az 20 karakterle açıklamalısın.")
            return
        }
        busy = true
        defer { busy = false }
        do {
            let result = try await APIClient(token: token).completeDiscipline(code: disciplineCode, note: note)
            onToast(result.awarded ? "Süreç tamam · +\(result.xpGained) XP" : "Bugün zaten tamamlandı")
            disciplineNote = ""
            await reload()
        } catch { onToast(actionError(error)) }
    }

    private func createPaperTrade() async {
        let quantity = numericValue(paperQuantity)
        let entryPrice = numericValue(paperPrice)
        let riskBudget = numericValue(paperRisk)
        let thesis = paperThesis.trimmingCharacters(in: .whitespacesAndNewlines)
        let invalidation = paperInvalidation.trimmingCharacters(in: .whitespacesAndNewlines)
        guard prices.prices[paperCode] != nil else {
            Haptics.warning(); onToast("Önce listeden bir varlık seçmelisin."); return
        }
        guard quantity > 0 else {
            Haptics.warning(); onToast("Miktar sıfırdan büyük olmalı."); return
        }
        guard entryPrice > 0 else {
            Haptics.warning(); onToast("Geçerli bir referans fiyat girmelisin."); return
        }
        guard riskBudget > 0, riskBudget <= 5 else {
            Haptics.warning(); onToast("Risk bütçesi %0 ile %5 arasında olmalı."); return
        }
        guard thesis.count >= 20 else {
            Haptics.warning(); onToast("Fikrinin dayanağını en az 20 karakterle yazmalısın."); return
        }
        guard invalidation.count >= 10 else {
            Haptics.warning(); onToast("Vazgeçme koşulunu en az 10 karakterle yazmalısın."); return
        }
        busy = true
        defer { busy = false }
        do {
            _ = try await APIClient(token: token).createPaperTrade(
                code: paperCode.uppercased(), side: paperSide,
                quantity: quantity, entryPrice: entryPrice, thesis: thesis,
                riskBudgetPct: riskBudget, invalidation: invalidation
            )
            onToast("Sanal karar günlüğüne eklendi")
            Haptics.success()
            paperThesis = ""; paperInvalidation = ""
            await reload()
        } catch { onToast(actionError(error)) }
    }

    private func reviewPaperTrade(_ id: String) async {
        let note = retrospective.trimmingCharacters(in: .whitespacesAndNewlines)
        guard note.count >= 20 else {
            Haptics.warning()
            onToast("Süreç değerlendirmeni en az 20 karakterle yazmalısın.")
            return
        }
        busy = true
        defer { busy = false }
        do {
            let result = try await APIClient(token: token).reviewPaperTrade(
                id: id, followedPlan: followedPlan, retrospective: note
            )
            onToast("Karar değerlendirmesi tamam · +\(result.xpGained) XP")
            Haptics.success()
            reviewTradeID = ""; retrospective = ""
            await reload()
        } catch { onToast(actionError(error)) }
    }

    private func numericValue(_ text: String) -> Double {
        Double(text.replacingOccurrences(of: ",", with: ".")) ?? 0
    }

    private func actionError(_ error: Error) -> String {
        if case APIError.decoding = error {
            return "Sunucu yanıtı tamamlanamadı. Lütfen yeniden dene."
        }
        return error.localizedDescription
    }

    private func savePrivacy(_ userID: String) async {
        busy = true
        defer { busy = false }
        do {
            _ = try await APIClient(token: token).saveGamificationPrivacy(GamificationPrivacy(
                userId: userID, alias: alias, leagueOptIn: leagueOptIn,
                personalizedLearning: personalized, socialOptIn: socialOptIn,
                learningNotifications: learningNotifications
            ))
            onToast("Gizlilik tercihleri kaydedildi")
            await reload()
        } catch { onToast(error.localizedDescription) }
    }

    private func createGroup() async {
        busy = true
        defer { busy = false }
        do {
            let group = try await APIClient(token: token).createStudyGroup(name: groupName)
            onToast("Grup hazır · kod \(group.inviteCode)")
            groupName = ""
            await reload()
        } catch { onToast(error.localizedDescription) }
    }

    private func joinGroup() async {
        busy = true
        defer { busy = false }
        do {
            try await APIClient(token: token).joinStudyGroup(inviteCode: inviteCode.uppercased())
            onToast("Çalışma grubuna katıldın")
            inviteCode = ""
            await reload()
        } catch { onToast(error.localizedDescription) }
    }
}
