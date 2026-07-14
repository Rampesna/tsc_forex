package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

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

// GamificationRepository persists engagement state, earned badges and the XP
// event ledger.
type GamificationRepository interface {
	State(ctx context.Context, userID uuid.UUID) (models.UserGamification, error)
	SaveState(ctx context.Context, s *models.UserGamification) error

	Badges(ctx context.Context, userID uuid.UUID) ([]models.UserBadge, error)
	AddBadge(ctx context.Context, userID uuid.UUID, code string, at time.Time) (bool, error)

	AddEvent(ctx context.Context, e *models.GamificationEvent) error
	TodayTypes(ctx context.Context, userID uuid.UUID, dayKey string) (map[string]bool, error)
	CountByType(ctx context.Context, userID uuid.UUID, eventType string) (int64, error)

	WeeklyXP(ctx context.Context, userID uuid.UUID, since time.Time) (int64, error)
	Leaderboard(ctx context.Context, since time.Time, limit int) ([]LeaderRow, error)
	RankOf(ctx context.Context, userID uuid.UUID, since time.Time, myXP int64) (int, error)
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
