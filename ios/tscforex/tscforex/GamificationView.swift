//
//  GamificationView.swift
//  "Ödüller" — daily streak, XP/levels, badges and learning journeys.
//  Designed to bring users back every day with a visible learning streak and
//  transparent progress milestones.
//

import SwiftUI

struct GamificationView: View {
    @EnvironmentObject var auth: AuthStore

    @State private var profile: GamProfile?
    @State private var experience: GamificationExperience?
    @State private var loading = true
    @State private var failed = false
    @State private var claiming = false
    @State private var toast: String?
    @State private var celebrate: [GamBadge] = []
    @State private var selectedBadge: GamBadge?

    var body: some View {
        NavigationStack {
            ScrollView {
                VStack(spacing: 16) {
                ScreenHeader(title: "Ödüller", subtitle: "Serini koru, seviye atla, rozet topla")

                if let p = profile {
                    heroCard(p)
                    statsRow(p)
                    if let experience {
                        GamificationProgressSummaryCard(experience: experience)
                        badgesCard(p)
                        GamificationExperienceHub(
                            experience: $experience,
                            token: auth.accessToken,
                            onProfile: { profile = $0 },
                            onToast: showToast
                        )
                    } else if !loading {
                        badgesCard(p)
                        experienceErrorCard
                    } else {
                        badgesCard(p)
                    }
                    questsCard(p)
                } else if loading {
                    skeleton
                } else if failed {
                    errorCard
                }
                Color.clear.frame(height: 96)
            }
                .padding(.horizontal, 16)
                .padding(.top, 8)
            }
            .background(Theme.bg.ignoresSafeArea())
            .refreshable { await load() }
        }
        .task { await load() }
        .overlay(alignment: .top) { toastBanner }
        .overlay { celebration }
        .sheet(item: $selectedBadge) { badge in badgeDetail(badge) }
    }

    // MARK: Hero (level + XP progress)

    private func heroCard(_ p: GamProfile) -> some View {
        Card {
            HStack(spacing: 16) {
                LevelRing(level: p.level, progress: p.progress)
                    .frame(width: 78, height: 78)
                VStack(alignment: .leading, spacing: 8) {
                    Text("Seviye \(p.level)")
                        .font(.system(size: 20, weight: .bold))
                        .foregroundStyle(Theme.text)
                    XPBar(progress: p.progress)
                    Text("\(p.xpIntoLevel) / \(p.xpForNextLevel) XP  •  Sonraki seviyeye \(max(0, p.xpForNextLevel - p.xpIntoLevel)) XP")
                        .font(.system(size: 12, weight: .medium))
                        .foregroundStyle(Theme.textDim)
                }
            }
        }
    }

    private func statsRow(_ p: GamProfile) -> some View {
        HStack(spacing: 10) {
            StatTile(label: "🔥 Seri", value: "\(p.currentStreak) gün", tint: p.currentStreak > 0 ? Theme.down : Theme.text)
            StatTile(label: "Haftalık XP", value: "\(p.weeklyXp)", tint: Theme.accent)
            StatTile(label: "Toplam XP", value: "\(p.xp)")
        }
    }

    // MARK: Daily reward

    @ViewBuilder private func dailyReward(_ p: GamProfile) -> some View {
        if p.checkedInToday {
            Card {
                HStack(spacing: 12) {
                    Image(systemName: "checkmark.seal.fill")
                        .font(.system(size: 24)).foregroundStyle(Theme.up)
                    VStack(alignment: .leading, spacing: 2) {
                        Text("Günlük giriş tamamlandı").font(.system(size: 15, weight: .bold)).foregroundStyle(Theme.text)
                        Text("Yarın tekrar gel, serini büyüt! En uzun serin: \(p.longestStreak) gün")
                            .font(.system(size: 12)).foregroundStyle(Theme.textDim)
                    }
                    Spacer()
                }
            }
        } else {
            Button {
                Task { await claim() }
            } label: {
                HStack(spacing: 10) {
                    if claiming {
                        ProgressView().tint(.black)
                    } else {
                        Image(systemName: "gift.fill")
                        Text("Günlük Ödülü Al")
                    }
                }
                .frame(maxWidth: .infinity)
            }
            .buttonStyle(FilledButtonStyle())
            .disabled(claiming)
        }
    }

    // MARK: Quests

    private func questsCard(_ p: GamProfile) -> some View {
        Card {
            VStack(alignment: .leading, spacing: 12) {
                HStack {
                    Text("Günlük Görevler").font(.system(size: 15, weight: .bold)).foregroundStyle(Theme.text)
                    Spacer()
                    let done = p.quests.filter { $0.completed }.count
                    Text("\(done)/\(p.quests.count)").font(.system(size: 12, weight: .bold)).foregroundStyle(Theme.textDim)
                }
                ForEach(p.quests) { q in
                    HStack(spacing: 12) {
                        Image(systemName: q.completed ? "checkmark.circle.fill" : "circle")
                            .font(.system(size: 20))
                            .foregroundStyle(q.completed ? Theme.up : Theme.textFaint)
                        VStack(alignment: .leading, spacing: 1) {
                            Text(q.title).font(.system(size: 14, weight: .semibold))
                                .foregroundStyle(q.completed ? Theme.textDim : Theme.text)
                                .strikethrough(q.completed, color: Theme.textDim)
                            Text(q.description).font(.system(size: 11)).foregroundStyle(Theme.textFaint)
                        }
                        Spacer()
                        Text("+\(q.xp)").font(Theme.number(13, weight: .bold)).foregroundStyle(Theme.accent)
                    }
                }
            }
        }
    }

    // MARK: Badges

    private func badgesCard(_ p: GamProfile) -> some View {
        Card {
            VStack(alignment: .leading, spacing: 14) {
                HStack {
                    Text("Rozetler").font(.system(size: 15, weight: .bold)).foregroundStyle(Theme.text)
                    Spacer()
                    let earned = p.badges.filter { $0.earned }.count
                    Text("\(earned)/\(p.badges.count)").font(.system(size: 12, weight: .bold)).foregroundStyle(Theme.textDim)
                }
                LazyVGrid(columns: Array(repeating: GridItem(.flexible(), spacing: 10), count: 4), spacing: 14) {
                    ForEach(p.badges) { b in
                        Button { selectedBadge = b; Haptics.tap() } label: { VStack(spacing: 5) {
                            ZStack {
                                Circle()
                                    .fill(b.earned ? Theme.accentSoft : Theme.surfaceHi)
                                    .frame(width: 52, height: 52)
                                Text(b.icon).font(.system(size: 24))
                                    .opacity(b.earned ? 1 : 0.35)
                                if !b.earned {
                                    Image(systemName: "lock.fill")
                                        .font(.system(size: 11, weight: .bold))
                                        .foregroundStyle(Theme.textFaint)
                                        .offset(x: 16, y: 16)
                                }
                            }
                            Text(b.name)
                                .font(.system(size: 9.5, weight: .medium))
                                .foregroundStyle(b.earned ? Theme.text : Theme.textFaint)
                                .multilineTextAlignment(.center)
                                .lineLimit(2)
                                .frame(height: 24)
                        } }
                        .buttonStyle(.plain)
                    }
                }
            }
        }
    }

    private func badgeDetail(_ badge: GamBadge) -> some View {
        VStack(spacing: 16) {
            Text(badge.icon).font(.system(size: 54))
            Text(badge.name).font(.system(size: 21, weight: .bold)).foregroundStyle(Theme.text)
            VStack(alignment: .leading, spacing: 6) {
                Text("Kazanma koşulu").font(.system(size: 11, weight: .bold)).foregroundStyle(Theme.accent)
                Text(badge.description).font(.system(size: 13)).foregroundStyle(Theme.textDim)
            }.frame(maxWidth: .infinity, alignment: .leading)
            if badge.earned, let date = badge.earnedAt {
                Label("Kazanıldı: \(date.formatted(date: .long, time: .omitted))", systemImage: "checkmark.seal.fill")
                    .font(.system(size: 12, weight: .semibold)).foregroundStyle(Theme.up)
            } else {
                Label("Henüz kazanılmadı", systemImage: "lock.fill")
                    .font(.system(size: 12, weight: .semibold)).foregroundStyle(Theme.textDim)
            }
            Spacer()
        }
        .padding(24).background(Theme.bg.ignoresSafeArea()).presentationDetents([.medium])
    }

    // MARK: Loading / error

    private var skeleton: some View {
        VStack(spacing: 16) {
            SkeletonBlock().frame(height: 110)
            SkeletonBlock().frame(height: 64)
            SkeletonBlock().frame(height: 150)
            SkeletonBlock().frame(height: 180)
        }
        .shimmer()
    }

    private var errorCard: some View {
        Card {
            VStack(spacing: 10) {
                Image(systemName: "trophy").font(.system(size: 30)).foregroundStyle(Theme.textFaint)
                Text("Ödüller yüklenemedi").font(.system(size: 15, weight: .semibold)).foregroundStyle(Theme.text)
                Button("Tekrar dene") { Task { await load() } }
                    .buttonStyle(FilledButtonStyle())
            }
            .frame(maxWidth: .infinity)
            .padding(.vertical, 12)
        }
    }

    private var experienceErrorCard: some View {
        Card {
            HStack(spacing: 12) {
                Image(systemName: "arrow.triangle.2.circlepath")
                    .font(.system(size: 22)).foregroundStyle(Theme.accent)
                VStack(alignment: .leading, spacing: 2) {
                    Text("Gelişim alanları yüklenemedi")
                        .font(.system(size: 14, weight: .bold)).foregroundStyle(Theme.text)
                    Text("Günlük plan, dersler ve sezon için tekrar deneyebilirsin.")
                        .font(.system(size: 11)).foregroundStyle(Theme.textDim)
                }
                Spacer()
                Button("Yenile") { Task { await load() } }
                    .font(.system(size: 12, weight: .bold))
            }
        }
    }

    // MARK: Toast + celebration overlays

    @ViewBuilder private var toastBanner: some View {
        if let toast {
            Text(toast)
                .font(.system(size: 14, weight: .bold))
                .foregroundStyle(Theme.text)
                .padding(.horizontal, 18).padding(.vertical, 12)
                .background(Theme.bgElevated, in: Capsule())
                .overlay(Capsule().stroke(Theme.strokeStrong, lineWidth: 1))
                .shadow(color: .black.opacity(0.4), radius: 12, y: 6)
                .padding(.top, 12)
                .transition(.move(edge: .top).combined(with: .opacity))
        }
    }

    @ViewBuilder private var celebration: some View {
        if !celebrate.isEmpty {
            ZStack {
                Color.black.opacity(0.55).ignoresSafeArea()
                    .onTapGesture { withAnimation { celebrate = [] } }
                Card {
                    VStack(spacing: 14) {
                        Text("🎉").font(.system(size: 44))
                        Text(celebrate.count > 1 ? "Yeni Rozetler!" : "Yeni Rozet!")
                            .font(.system(size: 18, weight: .bold)).foregroundStyle(Theme.text)
                        ForEach(celebrate) { b in
                            HStack(spacing: 12) {
                                Text(b.icon).font(.system(size: 30))
                                VStack(alignment: .leading, spacing: 2) {
                                    Text(b.name).font(.system(size: 15, weight: .bold)).foregroundStyle(Theme.text)
                                    Text(b.description).font(.system(size: 12)).foregroundStyle(Theme.textDim)
                                }
                                Spacer()
                            }
                        }
                        Button("Harika!") { withAnimation { celebrate = [] } }
                            .buttonStyle(FilledButtonStyle())
                    }
                    .frame(maxWidth: .infinity)
                }
                .padding(.horizontal, 40)
            }
            .transition(.opacity)
        }
    }

    // MARK: Actions

    private func load() async {
        let c = APIClient(token: auth.accessToken)
        async let profFetch = c.gamification()
        async let experienceFetch = c.gamificationExperience()
        let prof = try? await profFetch
        let experience = try? await experienceFetch
        await MainActor.run {
            if let prof { self.profile = prof; self.failed = false }
            else if self.profile == nil { self.failed = true }
            if let experience { self.experience = experience }
            self.loading = false
        }
    }

    private func claim() async {
        guard !claiming else { return }
        claiming = true
        Haptics.tap()
        let c = APIClient(token: auth.accessToken)
        guard let res = try? await c.gamificationCheckIn() else {
            await MainActor.run { claiming = false; showToast("Bağlantı hatası, tekrar dene") }
            return
        }
        await MainActor.run {
            self.profile = res.profile
            self.claiming = false
            if res.alreadyCheckedIn {
                showToast("Bugün zaten giriş yaptın ✓")
            } else {
                Haptics.success()
                showToast("+\(res.xpGained) XP  •  🔥 \(res.profile.currentStreak) gün")
                if !res.newBadges.isEmpty {
                    withAnimation(.spring) { self.celebrate = res.newBadges }
                }
            }
        }
    }

    private func showToast(_ msg: String) {
        withAnimation(.spring) { toast = msg }
        Task {
            try? await Task.sleep(nanoseconds: 2_600_000_000)
            await MainActor.run { withAnimation { if toast == msg { toast = nil } } }
        }
    }

}

// MARK: - Small reusable pieces

private struct LevelRing: View {
    let level: Int
    let progress: Double
    var body: some View {
        ZStack {
            Circle().stroke(Theme.surfaceHi, lineWidth: 7)
            Circle()
                .trim(from: 0, to: CGFloat(min(1, max(0.02, progress))))
                .stroke(
                    AngularGradient(colors: [Theme.accent, Theme.up, Theme.accent], center: .center),
                    style: StrokeStyle(lineWidth: 7, lineCap: .round)
                )
                .rotationEffect(.degrees(-90))
            VStack(spacing: -1) {
                Text("SEVİYE").font(.system(size: 8, weight: .bold)).foregroundStyle(Theme.textDim)
                Text("\(level)").font(Theme.number(26, weight: .bold)).foregroundStyle(Theme.text)
            }
        }
    }
}

private struct XPBar: View {
    let progress: Double
    var body: some View {
        GeometryReader { geo in
            ZStack(alignment: .leading) {
                Capsule().fill(Theme.surfaceHi)
                Capsule()
                    .fill(LinearGradient(colors: [Theme.accent, Theme.up], startPoint: .leading, endPoint: .trailing))
                    .frame(width: max(8, geo.size.width * CGFloat(min(1, max(0, progress)))))
            }
        }
        .frame(height: 10)
    }
}
