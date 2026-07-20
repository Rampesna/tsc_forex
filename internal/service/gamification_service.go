package service

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/tsc-forex/tsc_forex/internal/models"
	"github.com/tsc-forex/tsc_forex/internal/repository"
)

// Gamification event types (also used as XP-ledger row types).
const (
	EventCheckIn       = "checkin"
	EventFavorite      = "favorite_add"
	EventAlert         = "alert_create"
	EventDailyPlan     = "daily_plan"
	EventLesson        = "lesson_mastery"
	EventDiscipline    = "discipline_review"
	EventPaperReview   = "paper_review"
	EventDailyComplete = "daily_complete"

	gamificationRuleVersion = "2026.1"
	checkInBaseXP           = 20
	favoriteXP              = 10
	alertXP                 = 15
	dailyBonusXP            = 25
)

// BadgeDef is a static achievement definition.
type BadgeDef struct {
	Code        string
	Name        string
	Description string
	Icon        string
	XP          int
}

// badgeCatalog is the full set of earnable achievements. Level badges grant 0
// XP to avoid a feedback loop (earning XP that immediately unlocks the next
// level badge in the same pass).
var badgeCatalog = []BadgeDef{
	{"first_steps", "İlk Adım", "İlk günlük girişini yaptın.", "🌱", 20},
	{"streak_3", "Isınıyor", "3 gün üst üste giriş yaptın.", "🔥", 30},
	{"streak_7", "Haftalık Seri", "7 gün üst üste giriş yaptın.", "🔥", 70},
	{"streak_14", "İki Haftalık Ritim", "14 gün üst üste giriş yaptın.", "🗓️", 120},
	{"streak_30", "Aylık Disiplin", "30 gün üst üste giriş yaptın.", "⚡️", 300},
	{"streak_60", "Kararlı Takip", "60 gün üst üste giriş yaptın.", "💪", 500},
	{"streak_100", "Efsane Seri", "100 gün üst üste giriş yaptın.", "🏆", 1000},
	{"streak_180", "Yarım Yıl", "180 gün üst üste giriş yaptın.", "🌟", 1500},
	{"streak_365", "Bir Yıllık Seri", "365 gün üst üste giriş yaptın.", "👑", 2500},
	{"collector_1", "İlk Favori", "1 varlığı favorilere ekledin.", "☆", 15},
	{"collector", "Koleksiyoncu", "5 varlığı favorilere ekledin.", "⭐️", 50},
	{"collector_10", "Takip Listesi", "10 varlığı favorilere ekledin.", "📌", 90},
	{"collector_25", "Piyasa Radarı", "25 varlığı favorilere ekledin.", "📡", 180},
	{"collector_50", "Geniş Radar", "50 varlığı favorilere ekledin.", "🛰️", 350},
	{"watchful_1", "İlk Alarm", "1 fiyat alarmı kurdun.", "🔕", 20},
	{"watchful", "Tetikte", "5 fiyat alarmı kurdun.", "🔔", 60},
	{"watchful_10", "Nöbetçi", "10 fiyat alarmı kurdun.", "⏰", 100},
	{"watchful_25", "Risk Gözcüsü", "25 fiyat alarmı kurdun.", "👁️", 200},
	{"watchful_50", "Erken Uyarı", "50 fiyat alarmı kurdun.", "🚨", 400},
	{"level_3", "Çırak", "3. seviyeye ulaştın.", "🟤", 0},
	{"level_5", "Yükselen", "5. seviyeye ulaştın.", "🥉", 0},
	{"level_10", "Deneyimli", "10. seviyeye ulaştın.", "🥈", 0},
	{"level_15", "Analist", "15. seviyeye ulaştın.", "🔷", 0},
	{"level_25", "Usta Yatırımcı", "25. seviyeye ulaştın.", "🥇", 0},
	{"level_50", "Piyasa Ustası", "50. seviyeye ulaştın.", "💎", 0},
	{"committed_7", "İlk Hafta", "Toplam 7 gün giriş yaptın.", "📆", 60},
	{"committed", "Sadık Takipçi", "Toplam 30 gün giriş yaptın.", "📅", 200},
	{"committed_90", "Düzenli Takipçi", "Toplam 90 gün giriş yaptın.", "🧭", 450},
	{"committed_180", "Uzun Yol", "Toplam 180 gün giriş yaptın.", "🛤️", 800},
	{"committed_365", "Yıllık Yolculuk", "Toplam 365 gün giriş yaptın.", "🎖️", 1600},
}

// QuestDef is a daily task that resets each Istanbul day.
type QuestDef struct {
	Code        string
	Title       string
	Description string
	XP          int
}

var dailyQuests = []QuestDef{
	{EventDailyPlan, "Günü Planla", "Risk sınırını, takip listesini ve öğrenme hedefini belirle.", 25},
	{EventLesson, "Bilgini Sına", "Bir mikro dersin bilgi kontrolünü doğru tamamla.", 20},
	{EventDiscipline, "Süreci Değerlendir", "Portföy kararlarından birini kısa bir notla gözden geçir.", 25},
}

// BadgeStatus is a badge plus whether the current user has earned it.
type BadgeStatus struct {
	Code        string     `json:"code"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Icon        string     `json:"icon"`
	Earned      bool       `json:"earned"`
	EarnedAt    *time.Time `json:"earned_at,omitempty"`
	Current     int64      `json:"current"`
	Target      int64      `json:"target"`
	Progress    float64    `json:"progress"`
}

// QuestStatus is a daily quest plus today's completion state.
type QuestStatus struct {
	Code        string `json:"code"`
	Title       string `json:"title"`
	Description string `json:"description"`
	XP          int    `json:"xp"`
	Completed   bool   `json:"completed"`
}

// GamificationProfile is the full engagement snapshot returned to clients.
type GamificationProfile struct {
	XP             int64         `json:"xp"`
	Level          int           `json:"level"`
	XPIntoLevel    int64         `json:"xp_into_level"`
	XPForNextLevel int64         `json:"xp_for_next_level"`
	Progress       float64       `json:"progress"`
	CurrentStreak  int           `json:"current_streak"`
	LongestStreak  int           `json:"longest_streak"`
	CheckedInToday bool          `json:"checked_in_today"`
	TotalCheckIns  int           `json:"total_check_ins"`
	WeeklyXP       int64         `json:"weekly_xp"`
	Rank           int           `json:"rank"`
	Badges         []BadgeStatus `json:"badges"`
	Quests         []QuestStatus `json:"quests"`
}

// CheckInResult is returned by the daily check-in endpoint.
type CheckInResult struct {
	Profile          GamificationProfile `json:"profile"`
	XPGained         int                 `json:"xp_gained"`
	NewBadges        []BadgeStatus       `json:"new_badges"`
	AlreadyCheckedIn bool                `json:"already_checked_in"`
}

// LeaderEntry is one row of the weekly leaderboard.
type LeaderEntry struct {
	Rank        int    `json:"rank"`
	UserID      string `json:"user_id"`
	DisplayName string `json:"display_name"`
	AvatarURL   string `json:"avatar_url"`
	XP          int64  `json:"xp"`
	Level       int    `json:"level"`
}

// GamificationService implements streaks, XP/levels, badges, daily quests and
// the weekly leaderboard. It is server-authoritative: XP is only granted for
// server-verified actions (daily check-in, adding a favorite, creating an
// alert) and milestone badges — never trusted from the client.
type GamificationService struct {
	repo     repository.GamificationRepository
	notifier *Notifier // optional: badge/quest push + WS notifications
	log      *zap.Logger
	loc      *time.Location
}

// NewGamificationService builds the service. notifier may be nil.
func NewGamificationService(repo repository.GamificationRepository, notifier *Notifier, log *zap.Logger) *GamificationService {
	if log == nil {
		log = zap.NewNop()
	}
	loc, err := time.LoadLocation("Europe/Istanbul")
	if err != nil {
		loc = time.UTC
	}
	return &GamificationService{repo: repo, notifier: notifier, log: log, loc: loc}
}

func (g *GamificationService) dayKey(t time.Time) string { return t.In(g.loc).Format("2006-01-02") }
func (g *GamificationService) today() string             { return g.dayKey(time.Now()) }

// weekStart returns Monday 00:00 Istanbul (in UTC) for the current week.
func (g *GamificationService) weekStart() time.Time {
	now := time.Now().In(g.loc)
	off := (int(now.Weekday()) + 6) % 7 // Monday=0
	d := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, g.loc).AddDate(0, 0, -off)
	return d.UTC()
}

// levelForXP maps cumulative XP to a level (level N needs 50·(N-1)² XP).
func levelForXP(xp int64) int {
	if xp <= 0 {
		return 1
	}
	l := int(math.Floor(math.Sqrt(float64(xp)/50.0))) + 1
	if l < 1 {
		l = 1
	}
	return l
}

func xpForLevel(level int) int64 {
	if level <= 1 {
		return 0
	}
	n := int64(level - 1)
	return 50 * n * n
}

// Profile returns the full engagement snapshot, creating state on first use.
func (g *GamificationService) Profile(ctx context.Context, uid uuid.UUID) (GamificationProfile, error) {
	st, err := g.repo.State(ctx, uid)
	if err != nil {
		return GamificationProfile{}, err
	}
	return g.buildProfile(ctx, uid, st), nil
}

func (g *GamificationService) buildProfile(ctx context.Context, uid uuid.UUID, st models.UserGamification) GamificationProfile {
	today := g.today()
	todayTypes, _ := g.repo.TodayTypes(ctx, uid, today)
	badges, _ := g.repo.Badges(ctx, uid)
	favCount, _ := g.repo.CountByType(ctx, uid, EventFavorite)
	alertCount, _ := g.repo.CountByType(ctx, uid, EventAlert)
	earned := make(map[string]time.Time, len(badges))
	for _, b := range badges {
		earned[b.Code] = b.EarnedAt
	}
	since := g.weekStart()
	weekly, _ := g.repo.WeeklyXP(ctx, uid, since)
	rank, _ := g.repo.RankOf(ctx, uid, since, weekly)

	base := xpForLevel(st.Level)
	next := xpForLevel(st.Level + 1)
	into := st.XP - base
	span := next - base
	prog := 0.0
	if span > 0 {
		prog = float64(into) / float64(span)
		prog = math.Max(0, math.Min(1, prog))
	}

	bs := make([]BadgeStatus, 0, len(badgeCatalog))
	for _, def := range badgeCatalog {
		at, ok := earned[def.Code]
		var atp *time.Time
		if ok {
			t := at
			atp = &t
		}
		bs = append(bs, badgeStatus(def, st, favCount, alertCount, ok, atp))
	}
	qs := make([]QuestStatus, 0, len(dailyQuests))
	for _, q := range dailyQuests {
		qs = append(qs, QuestStatus{q.Code, q.Title, q.Description, q.XP, todayTypes[q.Code]})
	}

	return GamificationProfile{
		XP:             st.XP,
		Level:          st.Level,
		XPIntoLevel:    into,
		XPForNextLevel: next - base,
		Progress:       prog,
		CurrentStreak:  st.CurrentStreak,
		LongestStreak:  st.LongestStreak,
		CheckedInToday: st.LastCheckInDay == today,
		TotalCheckIns:  st.TotalCheckIns,
		WeeklyXP:       weekly,
		Rank:           rank,
		Badges:         bs,
		Quests:         qs,
	}
}

// CheckIn performs the once-per-day check-in: advances/resets the streak,
// grants XP, then evaluates badges and the daily-quest bonus. Idempotent within
// the same Istanbul day.
func (g *GamificationService) CheckIn(ctx context.Context, uid uuid.UUID) (CheckInResult, error) {
	today := g.today()
	gained := 0
	out, err := g.repo.ApplyAward(ctx, repository.AwardSpec{
		UserID: uid, IdempotencyKey: "checkin:" + today, EventType: EventCheckIn,
		SourceType: "calendar_day", SourceID: today, Reason: "Günlük plan serisi",
		RuleVersion: gamificationRuleVersion, DayKey: today,
	}, func(st *models.UserGamification) int {
		yesterday := g.dayKey(time.Now().AddDate(0, 0, -1))
		if st.LastCheckInDay == yesterday {
			st.CurrentStreak++
		} else {
			st.CurrentStreak = 1
		}
		if st.CurrentStreak > st.LongestStreak {
			st.LongestStreak = st.CurrentStreak
		}
		st.LastCheckInDay = today
		st.TotalCheckIns++
		gained = checkInBaseXP + min(st.CurrentStreak-1, 10)*3
		st.XP += int64(gained)
		st.Level = levelForXP(st.XP)
		return gained
	})
	if err != nil {
		return CheckInResult{}, err
	}
	if !out.Created {
		return CheckInResult{
			Profile:          g.buildProfile(ctx, uid, out.State),
			NewBadges:        make([]BadgeStatus, 0),
			AlreadyCheckedIn: true,
		}, nil
	}

	newBadges := g.evaluate(ctx, uid, out.State, today)
	if newBadges == nil {
		newBadges = make([]BadgeStatus, 0)
	}
	st, stateErr := g.repo.State(ctx, uid)
	if stateErr != nil {
		return CheckInResult{}, stateErr
	}

	return CheckInResult{
		Profile:   g.buildProfile(ctx, uid, st),
		XPGained:  gained,
		NewBadges: newBadges,
	}, nil
}

// Award grants XP for a server-verified engagement action. sourceID ties the
// reward to the durable domain object, so delete/recreate loops cannot farm XP.
func (g *GamificationService) Award(ctx context.Context, uid uuid.UUID, action string, sourceIDs ...string) {
	if action != EventFavorite && action != EventAlert {
		return
	}
	today := g.today()
	sourceID := today
	if len(sourceIDs) > 0 && sourceIDs[0] != "" {
		sourceID = sourceIDs[0]
	}
	xp := favoriteXP
	if action == EventAlert {
		xp = alertXP
	}
	out, err := g.awardXP(ctx, uid, action+":"+sourceID, action, "engagement", sourceID,
		"Doğrulanmış uygulama eylemi", xp, 0)
	if err != nil || !out.Created {
		return
	}
	g.evaluate(ctx, uid, out.State, today)
}

// evaluate grants any newly-qualified badges and the all-quests-done daily
// bonus, mutating st (XP/level) and logging ledger events. Badge/quest wins are
// pushed to the user via the notifier (WS + APNs/FCM) when configured.
func (g *GamificationService) evaluate(ctx context.Context, uid uuid.UUID, st models.UserGamification, today string) []BadgeStatus {
	favCount, _ := g.repo.CountByType(ctx, uid, EventFavorite)
	alertCount, _ := g.repo.CountByType(ctx, uid, EventAlert)

	var newly []BadgeStatus
	for _, def := range badgeCatalog {
		if !badgeQualifies(def.Code, &st, favCount, alertCount) {
			continue
		}
		created, err := g.repo.AddBadge(ctx, uid, def.Code, time.Now().UTC())
		if err != nil || !created {
			continue
		}
		if def.XP > 0 {
			out, awardErr := g.awardXP(ctx, uid, "badge:"+def.Code, "badge:"+def.Code,
				"badge", def.Code, "Kalıcı başarı rozeti", def.XP, 0)
			if awardErr == nil && out.Created {
				st = out.State
			}
		}
		now := time.Now().UTC()
		newly = append(newly, badgeStatus(def, st, favCount, alertCount, true, &now))
		if g.notifier != nil {
			body := def.Description
			if def.XP > 0 {
				body = fmt.Sprintf("%s · +%d XP kazandın.", def.Description, def.XP)
			}
			g.notifier.Notify(ctx, uid, nil, "", "Yeni rozet: "+def.Name, body, 0)
		}
	}

	// Daily quest completion bonus (once per day).
	todayTypes, _ := g.repo.TodayTypes(ctx, uid, today)
	if !todayTypes[EventDailyComplete] && todayTypes[EventDailyPlan] && todayTypes[EventLesson] && todayTypes[EventDiscipline] {
		out, awardErr := g.awardXP(ctx, uid, "daily-complete:"+today, EventDailyComplete,
			"calendar_day", today, "Üç anlamlı günlük görevi tamamlama", dailyBonusXP, 15)
		if awardErr == nil && out.Created {
			st = out.State
		}
		if g.notifier != nil {
			g.notifier.Notify(ctx, uid, nil, "",
				"Günlük görevler tamam",
				fmt.Sprintf("Bugünkü tüm görevleri tamamladın ve +%d bonus XP kazandın. Yarın görüşürüz!", dailyBonusXP), 0)
		}
	}

	return newly
}

func (g *GamificationService) awardXP(
	ctx context.Context,
	uid uuid.UUID,
	idempotencyKey, eventType, sourceType, sourceID, reason string,
	xp, seasonPoints int,
) (repository.AwardOutcome, error) {
	return g.repo.ApplyAward(ctx, repository.AwardSpec{
		UserID: uid, IdempotencyKey: idempotencyKey, EventType: eventType,
		SourceType: sourceType, SourceID: sourceID, Reason: reason,
		RuleVersion: gamificationRuleVersion, DayKey: g.today(),
		SeasonCode: g.currentSeasonCode(), SeasonPoints: seasonPoints,
	}, func(st *models.UserGamification) int {
		st.XP += int64(xp)
		st.Level = levelForXP(st.XP)
		return xp
	})
}

func (g *GamificationService) currentSeasonCode() string {
	now := time.Now().In(g.loc)
	quarter := (int(now.Month())-1)/3 + 1
	return fmt.Sprintf("%d-Q%d", now.Year(), quarter)
}

func badgeQualifies(code string, st *models.UserGamification, favCount, alertCount int64) bool {
	current, target := badgeProgress(code, st, favCount, alertCount)
	return target > 0 && current >= target
}

// badgeProgress uses the same server-authoritative counters as badge
// qualification. Returning the counters with every badge keeps all clients in
// sync and makes the remaining work transparent before a badge is earned.
func badgeProgress(code string, st *models.UserGamification, favCount, alertCount int64) (int64, int64) {
	switch code {
	case "first_steps":
		return int64(st.TotalCheckIns), 1
	case "streak_3":
		return int64(st.LongestStreak), 3
	case "streak_7":
		return int64(st.LongestStreak), 7
	case "streak_14":
		return int64(st.LongestStreak), 14
	case "streak_30":
		return int64(st.LongestStreak), 30
	case "streak_60":
		return int64(st.LongestStreak), 60
	case "streak_100":
		return int64(st.LongestStreak), 100
	case "streak_180":
		return int64(st.LongestStreak), 180
	case "streak_365":
		return int64(st.LongestStreak), 365
	case "collector_1":
		return favCount, 1
	case "collector":
		return favCount, 5
	case "collector_10":
		return favCount, 10
	case "collector_25":
		return favCount, 25
	case "collector_50":
		return favCount, 50
	case "watchful_1":
		return alertCount, 1
	case "watchful":
		return alertCount, 5
	case "watchful_10":
		return alertCount, 10
	case "watchful_25":
		return alertCount, 25
	case "watchful_50":
		return alertCount, 50
	case "level_3":
		return int64(st.Level), 3
	case "level_5":
		return int64(st.Level), 5
	case "level_10":
		return int64(st.Level), 10
	case "level_15":
		return int64(st.Level), 15
	case "level_25":
		return int64(st.Level), 25
	case "level_50":
		return int64(st.Level), 50
	case "committed_7":
		return int64(st.TotalCheckIns), 7
	case "committed":
		return int64(st.TotalCheckIns), 30
	case "committed_90":
		return int64(st.TotalCheckIns), 90
	case "committed_180":
		return int64(st.TotalCheckIns), 180
	case "committed_365":
		return int64(st.TotalCheckIns), 365
	}
	return 0, 0
}

func badgeStatus(
	def BadgeDef,
	st models.UserGamification,
	favCount, alertCount int64,
	earned bool,
	earnedAt *time.Time,
) BadgeStatus {
	current, target := badgeProgress(def.Code, &st, favCount, alertCount)
	progress := 0.0
	if target > 0 {
		progress = math.Max(0, math.Min(1, float64(current)/float64(target)))
		// An earned badge is permanent. Even if a supporting counter is
		// temporarily unavailable while the profile is built, never regress an
		// earned badge below its completed target in the API response.
		if earned {
			progress = 1
			if current < target {
				current = target
			}
		}
	}
	return BadgeStatus{
		Code:        def.Code,
		Name:        def.Name,
		Description: def.Description,
		Icon:        def.Icon,
		Earned:      earned,
		EarnedAt:    earnedAt,
		Current:     current,
		Target:      target,
		Progress:    progress,
	}
}

// Leaderboard returns the top weekly-XP entries plus the caller's own rank.
func (g *GamificationService) Leaderboard(ctx context.Context, uid uuid.UUID, limit int) ([]LeaderEntry, LeaderEntry, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	since := g.weekStart()
	rows, err := g.repo.Leaderboard(ctx, since, limit)
	if err != nil {
		return nil, LeaderEntry{}, err
	}
	entries := make([]LeaderEntry, 0, len(rows))
	for i, r := range rows {
		name := r.DisplayName
		if name == "" {
			name = "Yatırımcı"
		}
		entries = append(entries, LeaderEntry{
			Rank:        i + 1,
			UserID:      r.UserID.String(),
			DisplayName: name,
			AvatarURL:   r.AvatarURL,
			XP:          r.XP,
			Level:       levelForXP(r.XP),
		})
	}
	myXP, _ := g.repo.WeeklyXP(ctx, uid, since)
	myRank, _ := g.repo.RankOf(ctx, uid, since, myXP)
	me := LeaderEntry{Rank: myRank, UserID: uid.String(), XP: myXP, Level: levelForXP(myXP)}
	return entries, me, nil
}
