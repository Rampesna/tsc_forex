package service

import (
	"context"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/tsc-forex/tsc_forex/internal/models"
	"github.com/tsc-forex/tsc_forex/internal/repository"
)

func newGamificationTestService(t *testing.T) (*GamificationService, *gorm.DB, uuid.UUID) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:gam-v2-"+uuid.NewString()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(
		&models.User{},
		&models.UserGamification{},
		&models.UserBadge{},
		&models.GamificationEvent{},
		&models.XPLedgerEntry{},
		&models.DailyPlan{},
		&models.LearningProgress{},
		&models.LearningAttempt{},
		&models.DisciplineReview{},
		&models.SeasonScore{},
		&models.PaperTrade{},
		&models.GamificationPrivacy{},
		&models.StudyGroup{},
		&models.StudyGroupMember{},
	); err != nil {
		t.Fatal(err)
	}
	uid := uuid.New()
	if err := db.Create(&models.User{
		ID: uid, Email: uuid.NewString() + "@example.com", Role: "user", Provider: "local",
	}).Error; err != nil {
		t.Fatal(err)
	}
	repo := repository.NewGamificationRepository(db, nil)
	return NewGamificationService(repo, nil, zap.NewNop()), db, uid
}

func TestDailyPlanAndQuizAwardsAreIdempotent(t *testing.T) {
	svc, db, uid := newGamificationTestService(t)
	ctx := context.Background()
	input := DailyPlanInput{
		RiskAppetite: "moderate", WatchCodes: []string{"GRAM_GOLD", "USDTRY"},
		LearningGoal: "risk_budget",
	}
	first, err := svc.SaveDailyPlan(ctx, uid, input)
	if err != nil {
		t.Fatal(err)
	}
	if first.XPGained <= 0 {
		t.Fatalf("expected first plan award, got %d", first.XPGained)
	}
	var before models.UserGamification
	if err := db.First(&before, "user_id = ?", uid).Error; err != nil {
		t.Fatal(err)
	}
	second, err := svc.SaveDailyPlan(ctx, uid, input)
	if err != nil {
		t.Fatal(err)
	}
	var after models.UserGamification
	if err := db.First(&after, "user_id = ?", uid).Error; err != nil {
		t.Fatal(err)
	}
	if after.XP != before.XP || !second.AlreadyCheckedIn {
		t.Fatalf("plan retry changed XP: before=%d after=%d", before.XP, after.XP)
	}

	quiz := QuizAttemptInput{
		LessonCode: "risk_budget", QuestionCode: "risk_budget_q1",
		Answer: 1, IdempotencyKey: "attempt-" + uuid.NewString(),
	}
	correct, err := svc.SubmitQuiz(ctx, uid, quiz)
	if err != nil {
		t.Fatal(err)
	}
	if !correct.Correct || correct.XPGained != 20 {
		t.Fatalf("unexpected first quiz result: %+v", correct)
	}
	quiz.IdempotencyKey = "attempt-" + uuid.NewString()
	repeat, err := svc.SubmitQuiz(ctx, uid, quiz)
	if err != nil {
		t.Fatal(err)
	}
	if !repeat.Correct || repeat.XPGained != 0 {
		t.Fatalf("repeat quiz farmed XP: %+v", repeat)
	}

	var duplicateKeys int64
	if err := db.Model(&models.XPLedgerEntry{}).
		Select("idempotency_key").
		Where("user_id = ?", uid).
		Group("idempotency_key").
		Having("COUNT(*) > 1").
		Count(&duplicateKeys).Error; err != nil {
		t.Fatal(err)
	}
	if duplicateKeys != 0 {
		t.Fatalf("found %d duplicate ledger keys", duplicateKeys)
	}
}

func TestPaperScoreAndSocialPrivacyDoNotUseReturn(t *testing.T) {
	svc, _, uid := newGamificationTestService(t)
	ctx := context.Background()
	trade, created, err := svc.CreatePaperTrade(ctx, uid, PaperTradeInput{
		ClientID: "paper-" + uuid.NewString(), Code: "GRAM_GOLD", Side: "buy",
		Quantity: 1, EntryPrice: 5000, RiskBudgetPct: 1,
		Thesis:       "Altın kararını yalnızca belirlenmiş risk bütçesiyle test ediyorum.",
		Invalidation: "Risk sınırı bozulursa tez geçersiz.",
		OccurredAt:   time.Now().UTC(),
	})
	if err != nil || !created {
		t.Fatalf("create paper trade: created=%v err=%v", created, err)
	}
	reviewed, xp, err := svc.ReviewPaperTrade(
		ctx, uid, trade.ID, false,
		"Sonuç ne olursa olsun plan dışına çıktığım noktayı ve düzeltmeyi kaydettim.",
	)
	if err != nil {
		t.Fatal(err)
	}
	if reviewed.ProcessScore != 70 || xp != 30 {
		t.Fatalf("unexpected process scoring: score=%d xp=%d", reviewed.ProcessScore, xp)
	}

	privacy, err := svc.SavePrivacy(ctx, uid, models.GamificationPrivacy{
		Alias: "Disiplinli", SocialOptIn: true, LeagueOptIn: true,
		PersonalizedLearning: true, LearningNotifications: true,
	})
	if err != nil || !privacy.SocialOptIn {
		t.Fatalf("save privacy: %+v %v", privacy, err)
	}
	group, err := svc.CreateStudyGroup(ctx, uid, "Risk Günlüğü")
	if err != nil {
		t.Fatal(err)
	}
	if group.MemberCount != 1 || group.InviteCode == "" {
		t.Fatalf("unexpected private group: %+v", group)
	}
}
