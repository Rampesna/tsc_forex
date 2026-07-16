package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/tsc-forex/tsc_forex/internal/metrics"
	"github.com/tsc-forex/tsc_forex/internal/models"
)

// LeaderRow is one entry of the weekly XP leaderboard (joined with the user).
type LeaderRow struct {
	UserID      uuid.UUID `gorm:"column:user_id" json:"user_id"`
	DisplayName string    `gorm:"column:display_name" json:"display_name"`
	AvatarURL   string    `gorm:"column:avatar_url" json:"avatar_url"`
	XP          int64     `gorm:"column:xp" json:"xp"`
}

type AwardSpec struct {
	UserID         uuid.UUID
	IdempotencyKey string
	EventType      string
	SourceType     string
	SourceID       string
	Reason         string
	RuleVersion    string
	DayKey         string
	SeasonCode     string
	SeasonPoints   int
}

type AwardOutcome struct {
	State   models.UserGamification
	Ledger  models.XPLedgerEntry
	Created bool
}

type AwardMutation func(*models.UserGamification) int

type StudyGroupOverview struct {
	models.StudyGroup
	MemberCount int64 `json:"member_count"`
}

// GamificationRepository persists engagement state, earned badges and the XP
// event ledger.
type GamificationRepository interface {
	State(ctx context.Context, userID uuid.UUID) (models.UserGamification, error)
	SaveState(ctx context.Context, s *models.UserGamification) error

	Badges(ctx context.Context, userID uuid.UUID) ([]models.UserBadge, error)
	AddBadge(ctx context.Context, userID uuid.UUID, code string, at time.Time) (bool, error)

	AddEvent(ctx context.Context, e *models.GamificationEvent) error
	ApplyAward(ctx context.Context, spec AwardSpec, mutate AwardMutation) (AwardOutcome, error)
	TodayTypes(ctx context.Context, userID uuid.UUID, dayKey string) (map[string]bool, error)
	CountByType(ctx context.Context, userID uuid.UUID, eventType string) (int64, error)

	WeeklyXP(ctx context.Context, userID uuid.UUID, since time.Time) (int64, error)
	Leaderboard(ctx context.Context, since time.Time, limit int) ([]LeaderRow, error)
	RankOf(ctx context.Context, userID uuid.UUID, since time.Time, myXP int64) (int, error)
	SeasonLeaderboard(ctx context.Context, seasonCode string, limit int) ([]LeaderRow, error)
	SeasonScore(ctx context.Context, userID uuid.UUID, seasonCode string) (models.SeasonScore, error)

	SaveDailyPlan(ctx context.Context, plan *models.DailyPlan) error
	DailyPlan(ctx context.Context, userID uuid.UUID, dayKey string) (models.DailyPlan, error)
	LearningProgress(ctx context.Context, userID uuid.UUID) ([]models.LearningProgress, error)
	SaveLearningAttempt(ctx context.Context, attempt *models.LearningAttempt, progress *models.LearningProgress) (bool, error)
	SaveDisciplineReview(ctx context.Context, review *models.DisciplineReview) (bool, error)
	ListPaperTrades(ctx context.Context, userID uuid.UUID, limit int) ([]models.PaperTrade, error)
	CreatePaperTrade(ctx context.Context, trade *models.PaperTrade) (bool, error)
	UpdatePaperTrade(ctx context.Context, trade *models.PaperTrade) error
	PaperTrade(ctx context.Context, userID, tradeID uuid.UUID) (models.PaperTrade, error)
	Privacy(ctx context.Context, userID uuid.UUID) (models.GamificationPrivacy, error)
	SavePrivacy(ctx context.Context, privacy *models.GamificationPrivacy) error
	CreateStudyGroup(ctx context.Context, group *models.StudyGroup, member *models.StudyGroupMember) error
	JoinStudyGroup(ctx context.Context, inviteCode string, member *models.StudyGroupMember) error
	StudyGroups(ctx context.Context, userID uuid.UUID) ([]StudyGroupOverview, error)
}

type gamificationRepository struct{ gormBase }

// NewGamificationRepository builds the gamification repository.
func NewGamificationRepository(db *gorm.DB, m *metrics.Metrics) GamificationRepository {
	return &gamificationRepository{gormBase{db, m}}
}

func (r *gamificationRepository) State(ctx context.Context, userID uuid.UUID) (models.UserGamification, error) {
	start := time.Now()
	var s models.UserGamification
	now := time.Now().UTC()
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Attrs(models.UserGamification{UserID: userID, Level: 1, CreatedAt: now, UpdatedAt: now}).
		FirstOrCreate(&s).Error
	r.observe(start, err)
	return s, err
}

func (r *gamificationRepository) SaveState(ctx context.Context, s *models.UserGamification) error {
	start := time.Now()
	s.UpdatedAt = time.Now().UTC()
	err := r.db.WithContext(ctx).Save(s).Error
	r.observe(start, err)
	return err
}

func (r *gamificationRepository) Badges(ctx context.Context, userID uuid.UUID) ([]models.UserBadge, error) {
	start := time.Now()
	var out []models.UserBadge
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).Order("earned_at ASC").Find(&out).Error
	r.observe(start, err)
	return out, err
}

func (r *gamificationRepository) AddBadge(ctx context.Context, userID uuid.UUID, code string, at time.Time) (bool, error) {
	start := time.Now()
	var b models.UserBadge
	res := r.db.WithContext(ctx).
		Where("user_id = ? AND code = ?", userID, code).
		Attrs(models.UserBadge{ID: uuid.New(), UserID: userID, Code: code, EarnedAt: at}).
		FirstOrCreate(&b)
	r.observe(start, res.Error)
	return res.RowsAffected > 0, res.Error
}

func (r *gamificationRepository) AddEvent(ctx context.Context, e *models.GamificationEvent) error {
	start := time.Now()
	if e.CreatedAt.IsZero() {
		e.CreatedAt = time.Now().UTC()
	}
	err := r.db.WithContext(ctx).Create(e).Error
	r.observe(start, err)
	return err
}

func (r *gamificationRepository) ApplyAward(
	ctx context.Context,
	spec AwardSpec,
	mutate AwardMutation,
) (AwardOutcome, error) {
	start := time.Now()
	var out AwardOutcome
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		now := time.Now().UTC()
		seed := models.UserGamification{
			UserID: spec.UserID, Level: 1, CreatedAt: now, UpdatedAt: now,
		}
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&seed).Error; err != nil {
			return err
		}
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("user_id = ?", spec.UserID).First(&out.State).Error; err != nil {
			return err
		}

		var existing models.XPLedgerEntry
		err := tx.Where("user_id = ? AND idempotency_key = ?", spec.UserID, spec.IdempotencyKey).
			First(&existing).Error
		if err == nil {
			out.Ledger = existing
			out.Created = false
			return nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		delta := mutate(&out.State)
		out.State.UpdatedAt = now
		out.Ledger = models.XPLedgerEntry{
			UserID: spec.UserID, IdempotencyKey: spec.IdempotencyKey,
			EventType: spec.EventType, SourceType: spec.SourceType, SourceID: spec.SourceID,
			Reason: spec.Reason, RuleVersion: spec.RuleVersion, Delta: delta,
			DayKey: spec.DayKey, CreatedAt: now,
		}
		if err := tx.Create(&out.Ledger).Error; err != nil {
			return err
		}
		if err := tx.Create(&models.GamificationEvent{
			UserID: spec.UserID, Type: spec.EventType, XP: delta, DayKey: spec.DayKey,
			SourceType: spec.SourceType, SourceID: spec.SourceID,
			RuleVersion: spec.RuleVersion, CreatedAt: now,
		}).Error; err != nil {
			return err
		}
		if err := tx.Save(&out.State).Error; err != nil {
			return err
		}
		if spec.SeasonPoints > 0 && spec.SeasonCode != "" {
			var score models.SeasonScore
			res := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
				Where("user_id = ? AND season_code = ?", spec.UserID, spec.SeasonCode).
				First(&score)
			if errors.Is(res.Error, gorm.ErrRecordNotFound) {
				score = models.SeasonScore{
					UserID: spec.UserID, SeasonCode: spec.SeasonCode,
					Points: int64(spec.SeasonPoints), League: leagueForPoints(int64(spec.SeasonPoints)),
					UpdatedAt: now,
				}
				if err := tx.Create(&score).Error; err != nil {
					return err
				}
			} else if res.Error != nil {
				return res.Error
			} else {
				score.Points += int64(spec.SeasonPoints)
				score.League = leagueForPoints(score.Points)
				score.UpdatedAt = now
				if err := tx.Save(&score).Error; err != nil {
					return err
				}
			}
		}
		out.Created = true
		return nil
	})
	r.observe(start, err)
	return out, err
}

func leagueForPoints(points int64) string {
	switch {
	case points >= 1200:
		return "Ustalık"
	case points >= 600:
		return "Disiplin"
	case points >= 250:
		return "Gelişim"
	default:
		return "Başlangıç"
	}
}

func (r *gamificationRepository) TodayTypes(ctx context.Context, userID uuid.UUID, dayKey string) (map[string]bool, error) {
	start := time.Now()
	var types []string
	err := r.db.WithContext(ctx).Model(&models.GamificationEvent{}).
		Where("user_id = ? AND day_key = ?", userID, dayKey).
		Distinct().Pluck("type", &types).Error
	r.observe(start, err)
	set := make(map[string]bool, len(types))
	for _, t := range types {
		set[t] = true
	}
	return set, err
}

func (r *gamificationRepository) CountByType(ctx context.Context, userID uuid.UUID, eventType string) (int64, error) {
	start := time.Now()
	var n int64
	err := r.db.WithContext(ctx).Model(&models.GamificationEvent{}).
		Where("user_id = ? AND type = ?", userID, eventType).Count(&n).Error
	r.observe(start, err)
	return n, err
}

func (r *gamificationRepository) WeeklyXP(ctx context.Context, userID uuid.UUID, since time.Time) (int64, error) {
	start := time.Now()
	var sum *int64
	err := r.db.WithContext(ctx).Model(&models.GamificationEvent{}).
		Where("user_id = ? AND created_at >= ?", userID, since).
		Select("COALESCE(SUM(xp),0)").Scan(&sum).Error
	r.observe(start, err)
	if sum == nil {
		return 0, err
	}
	return *sum, err
}

func (r *gamificationRepository) Leaderboard(ctx context.Context, since time.Time, limit int) ([]LeaderRow, error) {
	start := time.Now()
	var rows []LeaderRow
	err := r.db.WithContext(ctx).
		Table("gamification_events AS e").
		Select("e.user_id AS user_id, COALESCE(u.display_name, '') AS display_name, COALESCE(u.avatar_url, '') AS avatar_url, SUM(e.xp) AS xp").
		Joins("LEFT JOIN users u ON u.id = e.user_id").
		Where("e.created_at >= ?", since).
		Group("e.user_id, u.display_name, u.avatar_url").
		Having("SUM(e.xp) > 0").
		Order("xp DESC").
		Limit(limit).
		Scan(&rows).Error
	r.observe(start, err)
	return rows, err
}

func (r *gamificationRepository) RankOf(ctx context.Context, userID uuid.UUID, since time.Time, myXP int64) (int, error) {
	start := time.Now()
	// Count users whose weekly XP strictly exceeds mine; rank = that + 1.
	var ahead int64
	err := r.db.WithContext(ctx).
		Table("(?) AS t", r.db.Model(&models.GamificationEvent{}).
			Select("user_id, SUM(xp) AS s").
			Where("created_at >= ?", since).
			Group("user_id")).
		Where("t.s > ?", myXP).
		Count(&ahead).Error
	r.observe(start, err)
	return int(ahead) + 1, err
}

func (r *gamificationRepository) SeasonLeaderboard(
	ctx context.Context,
	seasonCode string,
	limit int,
) ([]LeaderRow, error) {
	start := time.Now()
	var rows []LeaderRow
	err := r.db.WithContext(ctx).
		Table("season_scores AS s").
		Select("s.user_id, COALESCE(NULLIF(p.alias, ''), NULLIF(u.display_name, ''), 'Yatırımcı') AS display_name, '' AS avatar_url, s.points AS xp").
		Joins("JOIN users u ON u.id = s.user_id").
		Joins("LEFT JOIN gamification_privacy p ON p.user_id = s.user_id").
		Where("s.season_code = ? AND COALESCE(p.league_opt_in, true) = true", seasonCode).
		Order("s.points DESC, s.updated_at ASC").
		Limit(limit).
		Scan(&rows).Error
	r.observe(start, err)
	return rows, err
}

func (r *gamificationRepository) SeasonScore(
	ctx context.Context,
	userID uuid.UUID,
	seasonCode string,
) (models.SeasonScore, error) {
	start := time.Now()
	var score models.SeasonScore
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND season_code = ?", userID, seasonCode).
		First(&score).Error
	r.observe(start, err)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return models.SeasonScore{
			UserID: userID, SeasonCode: seasonCode, League: "Başlangıç",
		}, nil
	}
	return score, err
}

func (r *gamificationRepository) SaveDailyPlan(ctx context.Context, plan *models.DailyPlan) error {
	start := time.Now()
	err := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "user_id"}, {Name: "day_key"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"risk_appetite", "watch_codes", "learning_goal", "reflection", "updated_at",
		}),
	}).Create(plan).Error
	r.observe(start, err)
	return err
}

func (r *gamificationRepository) DailyPlan(
	ctx context.Context,
	userID uuid.UUID,
	dayKey string,
) (models.DailyPlan, error) {
	start := time.Now()
	var plan models.DailyPlan
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND day_key = ?", userID, dayKey).
		First(&plan).Error
	r.observe(start, err)
	return plan, err
}

func (r *gamificationRepository) LearningProgress(
	ctx context.Context,
	userID uuid.UUID,
) ([]models.LearningProgress, error) {
	start := time.Now()
	var rows []models.LearningProgress
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).Find(&rows).Error
	r.observe(start, err)
	return rows, err
}

func (r *gamificationRepository) SaveLearningAttempt(
	ctx context.Context,
	attempt *models.LearningAttempt,
	progress *models.LearningProgress,
) (bool, error) {
	start := time.Now()
	created := false
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing models.LearningAttempt
		err := tx.Where("user_id = ? AND idempotency_key = ?", attempt.UserID, attempt.IdempotencyKey).
			First(&existing).Error
		if err == nil {
			*attempt = existing
			return nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if err := tx.Create(attempt).Error; err != nil {
			return err
		}
		if err := tx.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "user_id"}, {Name: "lesson_code"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"mastery", "attempts", "correct", "completed_at", "next_review_at", "updated_at",
			}),
		}).Create(progress).Error; err != nil {
			return err
		}
		created = true
		return nil
	})
	r.observe(start, err)
	return created, err
}

func (r *gamificationRepository) SaveDisciplineReview(
	ctx context.Context,
	review *models.DisciplineReview,
) (bool, error) {
	start := time.Now()
	result := r.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(review)
	r.observe(start, result.Error)
	return result.RowsAffected > 0, result.Error
}

func (r *gamificationRepository) ListPaperTrades(
	ctx context.Context,
	userID uuid.UUID,
	limit int,
) ([]models.PaperTrade, error) {
	start := time.Now()
	var rows []models.PaperTrade
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).
		Order("occurred_at DESC, id DESC").Limit(limit).Find(&rows).Error
	r.observe(start, err)
	return rows, err
}

func (r *gamificationRepository) CreatePaperTrade(
	ctx context.Context,
	trade *models.PaperTrade,
) (bool, error) {
	start := time.Now()
	var existing models.PaperTrade
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND client_id = ?", trade.UserID, trade.ClientID).
		First(&existing).Error
	if err == nil {
		*trade = existing
		r.observe(start, nil)
		return false, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		r.observe(start, err)
		return false, err
	}
	err = r.db.WithContext(ctx).Create(trade).Error
	r.observe(start, err)
	return err == nil, err
}

func (r *gamificationRepository) UpdatePaperTrade(ctx context.Context, trade *models.PaperTrade) error {
	start := time.Now()
	err := r.db.WithContext(ctx).Save(trade).Error
	r.observe(start, err)
	return err
}

func (r *gamificationRepository) PaperTrade(
	ctx context.Context,
	userID, tradeID uuid.UUID,
) (models.PaperTrade, error) {
	start := time.Now()
	var trade models.PaperTrade
	err := r.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", tradeID, userID).First(&trade).Error
	r.observe(start, err)
	return trade, err
}

func (r *gamificationRepository) Privacy(
	ctx context.Context,
	userID uuid.UUID,
) (models.GamificationPrivacy, error) {
	start := time.Now()
	privacy := models.GamificationPrivacy{
		UserID: userID, LeagueOptIn: true, PersonalizedLearning: true,
		LearningNotifications: true,
	}
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).
		Attrs(privacy).FirstOrCreate(&privacy).Error
	r.observe(start, err)
	return privacy, err
}

func (r *gamificationRepository) SavePrivacy(
	ctx context.Context,
	privacy *models.GamificationPrivacy,
) error {
	start := time.Now()
	privacy.UpdatedAt = time.Now().UTC()
	err := r.db.WithContext(ctx).Save(privacy).Error
	r.observe(start, err)
	return err
}

func (r *gamificationRepository) CreateStudyGroup(
	ctx context.Context,
	group *models.StudyGroup,
	member *models.StudyGroupMember,
) error {
	start := time.Now()
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(group).Error; err != nil {
			return err
		}
		member.GroupID = group.ID
		return tx.Create(member).Error
	})
	r.observe(start, err)
	return err
}

func (r *gamificationRepository) JoinStudyGroup(
	ctx context.Context,
	inviteCode string,
	member *models.StudyGroupMember,
) error {
	start := time.Now()
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var group models.StudyGroup
		if err := tx.Where("invite_code = ?", inviteCode).First(&group).Error; err != nil {
			return err
		}
		var count int64
		if err := tx.Model(&models.StudyGroupMember{}).
			Where("group_id = ?", group.ID).Count(&count).Error; err != nil {
			return err
		}
		if count >= 8 {
			return ErrStudyGroupFull
		}
		member.GroupID = group.ID
		return tx.Clauses(clause.OnConflict{DoNothing: true}).Create(member).Error
	})
	r.observe(start, err)
	return err
}

func (r *gamificationRepository) StudyGroups(
	ctx context.Context,
	userID uuid.UUID,
) ([]StudyGroupOverview, error) {
	start := time.Now()
	var rows []StudyGroupOverview
	err := r.db.WithContext(ctx).
		Table("study_groups AS g").
		Select("g.*, COUNT(all_members.id) AS member_count").
		Joins("JOIN study_group_members mine ON mine.group_id = g.id AND mine.user_id = ?", userID).
		Joins("LEFT JOIN study_group_members all_members ON all_members.group_id = g.id").
		Group("g.id").
		Order("g.created_at DESC").
		Scan(&rows).Error
	r.observe(start, err)
	return rows, err
}

var ErrStudyGroupFull = errors.New("study group is full")
