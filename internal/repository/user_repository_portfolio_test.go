package repository

import (
	"context"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/tsc-forex/tsc_forex/internal/models"
)

func TestDeleteUserRemovesPortfolioLedger(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:delete-user-"+uuid.NewString()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(
		&models.User{},
		&models.Portfolio{},
		&models.PortfolioPosition{},
		&models.PortfolioTransaction{},
		&models.WatchlistItem{},
		&models.PriceAlert{},
		&models.Notification{},
		&models.DeviceToken{},
		&models.RefreshToken{},
		&models.AuthIdentity{},
		&models.WebAuthnCredential{},
		&models.UserBadge{},
		&models.GamificationEvent{},
		&models.XPLedgerEntry{},
		&models.UserGamification{},
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
	user := models.User{ID: uuid.New(), Email: uuid.NewString() + "@example.com", Role: "user", Provider: "local"}
	portfolio := models.Portfolio{ID: uuid.New(), UserID: user.ID, Name: "Test", BaseCurrency: "TRY"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&portfolio).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.PortfolioPosition{
		ID: uuid.New(), PortfolioID: portfolio.ID, Code: "NVDA", Currency: "USD",
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.PortfolioTransaction{
		ID: uuid.New(), PortfolioID: portfolio.ID, UserID: user.ID,
		ClientID: "delete-test", Type: "buy", Code: "NVDA", Currency: "USD",
	}).Error; err != nil {
		t.Fatal(err)
	}

	if err := NewUserRepository(db, nil).DeleteUser(context.Background(), user.ID); err != nil {
		t.Fatal(err)
	}
	for name, model := range map[string]any{
		"portfolios":   &models.Portfolio{},
		"positions":    &models.PortfolioPosition{},
		"transactions": &models.PortfolioTransaction{},
	} {
		var count int64
		if err := db.Model(model).Count(&count).Error; err != nil {
			t.Fatal(err)
		}
		if count != 0 {
			t.Fatalf("%s left %d rows", name, count)
		}
	}
}
