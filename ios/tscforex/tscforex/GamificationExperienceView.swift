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
                    journeyLink("Günlük Plan", "İzleme ve risk planı", "scope") {
                        journeyScreen("Günlük Plan") { today(exp) }
                    }
                    journeyLink("Öğrenme", "Mikro ders ve tekrar", "book.closed.fill") {
                        journeyScreen("Öğrenme") { learning(exp) }
                    }
                }
                HStack(spacing: 10) {
                    journeyLink("Karar Günlüğü", "Süreç ve paper kararlar", "checklist") {
                        journeyScreen("Karar Günlüğü") {
                            VStack(spacing: 12) { discipline(exp); paper(exp) }
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
                VStack(alignment: .leading, spacing: 10) {
                    Label(exp.dailyPlan == nil ? "Bugünü Planla" : "Bugünün Planı", systemImage: "scope")
                        .font(.system(size: 15, weight: .bold)).foregroundStyle(Theme.text)
                    Picker("Risk", selection: $risk) {
                        Text("Düşük").tag("low")
                        Text("Orta").tag("moderate")
                        Text("Yüksek").tag("high")
                    }.pickerStyle(.segmented)
                    watchSelection
                    Picker("Öğrenme hedefi", selection: $learningGoal) {
                        ForEach(exp.lessons) { lesson in Text(lesson.title).tag(lesson.code) }
                    }
                    .tint(Theme.accent)
                    input("Dünkü plandan kısa not", text: $reflection, axis: .vertical)
                    Button(busy ? "Kaydediliyor…" : "Planı Kaydet ve Seriyi Sürdür") {
                        Task { await savePlan() }
                    }
                    .buttonStyle(FilledButtonStyle())
                    .disabled(busy || learningGoal.isEmpty || watchCodes.isEmpty)
                }
            }

        }
    }

    private func discipline(_ exp: GamificationExperience) -> some View {
        Card {
            VStack(alignment: .leading, spacing: 10) {
                Text("Portföy Disiplini").font(.system(size: 15, weight: .bold)).foregroundStyle(Theme.text)
                Text("Sonuca değil karar sürecine puan verilir.")
                    .font(.system(size: 11)).foregroundStyle(Theme.textDim)
                Picker("Görev", selection: $disciplineCode) {
                    ForEach(exp.disciplineQuests) { quest in Text(quest.title).tag(quest.code) }
                }.tint(Theme.accent)
                if let quest = exp.disciplineQuests.first(where: { $0.code == disciplineCode }) {
                    Text(quest.description).font(.system(size: 11)).foregroundStyle(Theme.textFaint)
                }
                input("Risk, tez veya plan sapmasını değerlendir…", text: $disciplineNote, axis: .vertical)
                Button(busy ? "Kaydediliyor…" : "Süreç Değerlendirmesini Tamamla") {
                    Task { await completeDiscipline() }
                }
                .buttonStyle(FilledButtonStyle())
                .disabled(busy || disciplineNote.trimmingCharacters(in: .whitespacesAndNewlines).count < 20)
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
        return Card {
            VStack(alignment: .leading, spacing: 11) {
                Text("Mikro Ders ve Ustalık").font(.system(size: 15, weight: .bold)).foregroundStyle(Theme.text)
                ScrollView(.horizontal, showsIndicators: false) {
                    HStack {
                        ForEach(exp.lessons) { item in
                            Button {
                                selectedLesson = item.code
                                selectedAnswer = nil
                            } label: {
                                Text("\(item.completed ? "✓ " : item.dueForReview ? "↻ " : "")\(item.title)")
                                    .font(.system(size: 11, weight: .semibold))
                                    .foregroundStyle(item.code == lesson?.code ? Theme.accent : Theme.textDim)
                                    .padding(.horizontal, 10).padding(.vertical, 7)
                                    .background(Theme.surfaceHi, in: Capsule())
                                    .overlay(Capsule().stroke(item.code == lesson?.code ? Theme.accent : Theme.stroke))
                            }.buttonStyle(.plain)
                        }
                    }
                }
                if let lesson {
                    HStack {
                        VStack(alignment: .leading, spacing: 2) {
                            Text(lesson.title).font(.system(size: 15, weight: .bold)).foregroundStyle(Theme.text)
                            Text(lesson.summary).font(.system(size: 11)).foregroundStyle(Theme.textDim)
                        }
                        Spacer()
                        Text("%\(lesson.mastery)").font(Theme.number(13, weight: .bold)).foregroundStyle(Theme.accent)
                    }
                    ForEach(Array(lesson.cards.enumerated()), id: \.offset) { index, text in
                        Text("\(index + 1). \(text)")
                            .font(.system(size: 11)).foregroundStyle(Theme.text)
                            .padding(10).frame(maxWidth: .infinity, alignment: .leading)
                            .background(Theme.surfaceHi, in: RoundedRectangle(cornerRadius: 9))
                    }
                    Text(lesson.question.text).font(.system(size: 13, weight: .bold)).foregroundStyle(Theme.text)
                    ForEach(Array(lesson.question.choices.enumerated()), id: \.offset) { index, choice in
                        Button {
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
                    Button(busy ? "Kontrol ediliyor…" : "Bilgimi Kontrol Et") {
                        Task { await submitQuiz(lesson) }
                    }
                    .buttonStyle(FilledButtonStyle()).disabled(busy || selectedAnswer == nil)
                }
            }
        }
    }

    private func paper(_ exp: GamificationExperience) -> some View {
        VStack(spacing: 12) {
            Card {
                VStack(alignment: .leading, spacing: 9) {
                    Text("Paper Karar Laboratuvarı").font(.system(size: 15, weight: .bold)).foregroundStyle(Theme.text)
                    Text("Getiri değil; tez, risk, geçersizleşme ve retrospektif puanlanır.")
                        .font(.system(size: 11)).foregroundStyle(Theme.textDim)
                    HStack {
                        input("Kod", text: $paperCode)
                        Picker("Yön", selection: $paperSide) {
                            Text("Alış").tag("buy"); Text("Satış").tag("sell")
                        }.frame(maxWidth: 120)
                    }
                    HStack {
                        input("Miktar", text: $paperQuantity).keyboardType(.decimalPad)
                        input("Giriş fiyatı", text: $paperPrice).keyboardType(.decimalPad)
                        input("Risk %", text: $paperRisk).keyboardType(.decimalPad)
                    }
                    input("Tez (en az 20 karakter)", text: $paperThesis, axis: .vertical)
                    input("Tezi geçersiz kılacak koşul", text: $paperInvalidation, axis: .vertical)
                    Button(busy ? "Kaydediliyor…" : "Kararı Kilitle") {
                        Task { await createPaperTrade() }
                    }
                    .buttonStyle(FilledButtonStyle())
                    .disabled(busy || paperThesis.count < 20 || paperInvalidation.count < 10 ||
                              Double(paperPrice.replacingOccurrences(of: ",", with: ".")) == nil)
                }
            }
            ForEach(exp.paperTrades.prefix(8)) { trade in
                Card {
                    VStack(alignment: .leading, spacing: 7) {
                        HStack {
                            Text("\(trade.code) · \(trade.side == "buy" ? "Alış" : "Satış")")
                                .font(.system(size: 13, weight: .bold)).foregroundStyle(Theme.text)
                            Spacer()
                            Text(trade.reviewedAt == nil ? "Retrospektif bekliyor" : "Süreç \(trade.processScore)/100")
                                .font(.system(size: 10, weight: .bold))
                                .foregroundStyle(trade.reviewedAt == nil ? Theme.textDim : Theme.up)
                        }
                        Text(trade.thesis).font(.system(size: 11)).foregroundStyle(Theme.textDim)
                        if trade.reviewedAt == nil {
                            Button("Kararı değerlendir") {
                                reviewTradeID = reviewTradeID == trade.id ? "" : trade.id
                            }.font(.system(size: 11, weight: .bold)).foregroundStyle(Theme.accent)
                        }
                        if reviewTradeID == trade.id {
                            Toggle("Planıma uydum", isOn: $followedPlan).font(.system(size: 12)).tint(Theme.accent)
                            input("Sonuçtan bağımsız süreç değerlendirmesi", text: $retrospective, axis: .vertical)
                            Button(busy ? "Kaydediliyor…" : "Retrospektifi Tamamla") {
                                Task { await reviewPaperTrade(trade.id) }
                            }.buttonStyle(FilledButtonStyle()).disabled(busy || retrospective.count < 20)
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
            .textInputAutocapitalization(.characters)
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
        if selectedLesson.isEmpty { selectedLesson = exp.lessons.first?.code ?? "" }
        disciplineCode = exp.disciplineQuests.first?.code ?? disciplineCode
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
        busy = true
        defer { busy = false }
        do {
            let result = try await APIClient(token: token).saveDailyPlan(
                risk: risk, codes: Array(watchCodes.prefix(3)), learningGoal: learningGoal, reflection: reflection
            )
            onProfile(result.profile)
            onToast("Plan kaydedildi · +\(result.xpGained) XP · 🔥 \(result.profile.currentStreak) gün")
            Haptics.success()
            await reload()
        } catch { onToast(error.localizedDescription) }
    }

    private func submitQuiz(_ lesson: LessonStatus) async {
        guard let answer = selectedAnswer else { return }
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
        } catch { onToast(error.localizedDescription) }
    }

    private func completeDiscipline() async {
        busy = true
        defer { busy = false }
        do {
            let result = try await APIClient(token: token).completeDiscipline(code: disciplineCode, note: disciplineNote)
            onToast(result.awarded ? "Süreç tamam · +\(result.xpGained) XP" : "Bugün zaten tamamlandı")
            disciplineNote = ""
            await reload()
        } catch { onToast(error.localizedDescription) }
    }

    private func createPaperTrade() async {
        busy = true
        defer { busy = false }
        do {
            _ = try await APIClient(token: token).createPaperTrade(
                code: paperCode.uppercased(), side: paperSide,
                quantity: Double(paperQuantity.replacingOccurrences(of: ",", with: ".")) ?? 0,
                entryPrice: Double(paperPrice.replacingOccurrences(of: ",", with: ".")) ?? 0,
                thesis: paperThesis,
                riskBudgetPct: Double(paperRisk.replacingOccurrences(of: ",", with: ".")) ?? 0,
                invalidation: paperInvalidation
            )
            onToast("Paper karar günlüğüne eklendi")
            paperThesis = ""; paperInvalidation = ""
            await reload()
        } catch { onToast(error.localizedDescription) }
    }

    private func reviewPaperTrade(_ id: String) async {
        busy = true
        defer { busy = false }
        do {
            let result = try await APIClient(token: token).reviewPaperTrade(
                id: id, followedPlan: followedPlan, retrospective: retrospective
            )
            onToast("Retrospektif tamam · +\(result.xpGained) XP")
            reviewTradeID = ""; retrospective = ""
            await reload()
        } catch { onToast(error.localizedDescription) }
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
