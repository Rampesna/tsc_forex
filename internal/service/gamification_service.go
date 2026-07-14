package service

import (
	"context"
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
	EventDailyComplete = "daily_complete"

	checkInBaseXP = 20
	favoriteXP    = 10
	alertXP       = 15
	dailyBonusXP  = 25
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
	{"streak_30", "Aylık Disiplin", "30 gün üst üste giriş yaptın.", "⚡️", 300},
	{"streak_100", "Efsane Seri", "100 gün üst üste giriş yaptın.", "🏆", 1000},
	{"collector", "Koleksiyoncu", "5 varlığı favorilere ekledin.", "⭐️", 50},
	{"watchful", "Tetikte", "5 fiyat alarmı kurdun.", "🔔", 60},
	{"level_5", "Yükselen", "5. seviyeye ulaştın.", "🥉", 0},
	{"level_10", "Deneyimli", "10. seviyeye ulaştın.", "🥈", 0},
	{"level_25", "Usta Yatırımcı", "25. seviyeye ulaştın.", "🥇", 0},
	{"committed", "Sadık Takipçi", "Toplam 30 gün giriş yaptın.", "📅", 200},
}

// QuestDef is a daily task that resets each Istanbul day.
type QuestDef struct {
	Code        string
	Title       string
	Description string
	XP          int
}

var dailyQuests = []QuestDef{
	{EventCheckIn, "Günlük Giriş", "Bugün uygulamaya giriş yap.", checkInBaseXP},
	{EventFavorite, "Bir Favori Ekle", "İzleme listene bir varlık ekle.", favoriteXP},
	{EventAlert, "Alarm Kur", "Bir fiyat alarmı oluştur.", alertXP},
}

// BadgeStatus is a badge plus whether the current user has earned it.
type BadgeStatus struct {
	Code        string     `json:"code"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Icon        string     `json:"icon"`
	Earned      bool       `json:"earned"`
	EarnedAt    *time.Time `json:"earned_at,omitempty"`
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
		bs = append(bs, BadgeStatus{def.Code, def.Name, def.Description, def.Icon, ok, atp})
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
	st, err := g.repo.State(ctx, uid)
	if err != nil {
		return CheckInResult{}, err
	}
	today := g.today()
	if st.LastCheckInDay == today {
		return CheckInResult{Profile: g.buildProfile(ctx, uid, st), AlreadyCheckedIn: true}, nil
	}

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

	gained := checkInBaseXP + min(st.CurrentStreak-1, 10)*3
	st.XP += int64(gained)
	st.Level = levelForXP(st.XP)
	_ = g.repo.AddEvent(ctx, &models.GamificationEvent{UserID: uid, Type: EventCheckIn, XP: gained, DayKey: today})

	newBadges := g.evaluate(ctx, uid, &st, today)
	_ = g.repo.SaveState(ctx, &st)

	return CheckInResult{
		Profile:   g.buildProfile(ctx, uid, st),
		XPGained:  gained,
		NewBadges: newBadges,
	}, nil
}

// Award grants XP for a server-verified engagement action (favorite/alert).
// XP is capped to the first occurrence of each action per day; the event is
// always logged so daily quests and count-based badges still progress.
func (g *GamificationService) Award(ctx context.Context, uid uuid.UUID, action string) {
	if action != EventFavorite && action != EventAlert {
		return
	}
	st, err := g.repo.State(ctx, uid)
	if err != nil {
		return
	}
	today := g.today()
	todayTypes, _ := g.repo.TodayTypes(ctx, uid, today)
	xp := 0
	if !todayTypes[action] {
		switch action {
		case EventFavorite:
			xp = favoriteXP
		case EventAlert:
			xp = alertXP
		}
	}
	st.XP += int64(xp)
	st.Level = levelForXP(st.XP)
	_ = g.repo.AddEvent(ctx, &models.GamificationEvent{UserID: uid, Type: action, XP: xp, DayKey: today})

	g.evaluate(ctx, uid, &st, today)
	_ = g.repo.SaveState(ctx, &st)
}

// evaluate grants any newly-qualified badges and the all-quests-done daily
// bonus, mutating st (XP/level) and logging ledger events. Badge/quest wins are
// pushed to the user via the notifier (WS + APNs/FCM) when configured.
func (g *GamificationService) evaluate(ctx context.Context, uid uuid.UUID, st *models.UserGamification, today string) []BadgeStatus {
	favCount, _ := g.repo.CountByType(ctx, uid, EventFavorite)
	alertCount, _ := g.repo.CountByType(ctx, uid, EventAlert)

	var newly []BadgeStatus
	for _, def := range badgeCatalog {
		if !badgeQualifies(def.Code, st, favCount, alertCount) {
			continue
		}
		created, err := g.repo.AddBadge(ctx, uid, def.Code, time.Now().UTC())
		if err != nil || !created {
			continue
		}
		if def.XP > 0 {
			st.XP += int64(def.XP)
			_ = g.repo.AddEvent(ctx, &models.GamificationEvent{UserID: uid, Type: "badge:" + def.Code, XP: def.XP, DayKey: today})
		}
		now := time.Now().UTC()
		newly = append(newly, BadgeStatus{def.Code, def.Name, def.Description, def.Icon, true, &now})
		if g.notifier != nil {
			g.notifier.Notify(ctx, uid, nil, "", "🏅 Yeni Rozet: "+def.Name, def.Description, 0)
		}
	}

	// Daily quest completion bonus (once per day).
	todayTypes, _ := g.repo.TodayTypes(ctx, uid, today)
	if !todayTypes[EventDailyComplete] && todayTypes[EventCheckIn] && todayTypes[EventFavorite] && todayTypes[EventAlert] {
		st.XP += int64(dailyBonusXP)
		_ = g.repo.AddEvent(ctx, &models.GamificationEvent{UserID: uid, Type: EventDailyComplete, XP: dailyBonusXP, DayKey: today})
		if g.notifier != nil {
			g.notifier.Notify(ctx, uid, nil, "", "✅ Günlük Görevler Tamamlandı", "Bugünkü tüm görevleri bitirdin, bonus XP kazandın!", 0)
		}
	}

	st.Level = levelForXP(st.XP)
	return newly
}

func badgeQualifies(code string, st *models.UserGamification, favCount, alertCount int64) bool {
	switch code {
	case "first_steps":
		return st.TotalCheckIns >= 1
	case "streak_3":
		return st.LongestStreak >= 3
	case "streak_7":
		return st.LongestStreak >= 7
	case "streak_30":
		return st.LongestStreak >= 30
	case "streak_100":
		return st.LongestStreak >= 100
	case "collector":
		return favCount >= 5
	case "watchful":
		return alertCount >= 5
	case "level_5":
		return st.Level >= 5
	case "level_10":
		return st.Level >= 10
	case "level_25":
		return st.Level >= 25
	case "committed":
		return st.TotalCheckIns >= 30
	}
	return false
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
