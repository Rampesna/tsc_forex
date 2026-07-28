package localization_test

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/tsc-forex/tsc_forex/internal/domain"
	"github.com/tsc-forex/tsc_forex/internal/localization"
	"github.com/tsc-forex/tsc_forex/internal/models"
)

func localizationDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(
		sqlite.Open("file:localization-"+uuid.NewString()+"?mode=memory&cache=shared"),
		&gorm.Config{Logger: logger.Default.LogMode(logger.Silent)},
	)
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := db.AutoMigrate(
		&models.SupportedLocale{},
		&models.InstrumentDefinition{},
		&models.InstrumentTranslation{},
	); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestCatalogSeedIsCompleteAndIdempotent(t *testing.T) {
	db := localizationDB(t)
	for range 2 {
		if err := localization.SeedCatalog(context.Background(), db); err != nil {
			t.Fatal(err)
		}
	}

	var localeCount, instrumentCount, translationCount int64
	if err := db.Model(&models.SupportedLocale{}).Count(&localeCount).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&models.InstrumentDefinition{}).Count(&instrumentCount).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&models.InstrumentTranslation{}).Count(&translationCount).Error; err != nil {
		t.Fatal(err)
	}
	if localeCount != 6 {
		t.Fatalf("locale count = %d, want 6", localeCount)
	}
	if instrumentCount != int64(len(domain.Catalog)) {
		t.Fatalf("instrument count = %d, want %d", instrumentCount, len(domain.Catalog))
	}
	wantTranslations := int64(len(domain.Catalog) * len(localization.SupportedLocales()))
	if translationCount != wantTranslations {
		t.Fatalf("translation count = %d, want %d", translationCount, wantTranslations)
	}
}

func TestInstrumentLocalizerResolvesAllProductKinds(t *testing.T) {
	db := localizationDB(t)
	if err := localization.SeedCatalog(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	names, err := localization.NewInstrumentLocalizer(db)
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		code, locale, want string
	}{
		{"USDTRY", "en-US", "US Dollar"},
		{"USDTRY", "ar-SA", "دولار أمريكي"},
		{"GRAM_GOLD", "de-DE", "Feingold (1 Gramm)"},
		{"CEYREK_GOLD", "es", "Moneda de oro de cuarto"},
		{"AAPL", "it-IT", "Apple"},
		{"BTC", "ar", "Bitcoin"},
		{"THYAO", "en", "Türk Hava Yolları"},
	}
	for _, test := range cases {
		if got := names.DisplayName(test.code, test.locale); got != test.want {
			t.Errorf("%s/%s = %q, want %q", test.code, test.locale, got, test.want)
		}
	}
}

func TestRequestLocaleNegotiation(t *testing.T) {
	request := httptest.NewRequest("GET", "/ws?locale=ar-SA", nil)
	request.Header.Set(localization.ClientLocaleHeader, "de-DE")
	if got := localization.RequestLocale(request); got != "ar" {
		t.Fatalf("query locale = %q, want ar", got)
	}

	request = httptest.NewRequest("GET", "/", nil)
	request.Header.Set("Accept-Language", "fr-FR;q=1, es-ES;q=0.8")
	if got := localization.RequestLocale(request); got != "es" {
		t.Fatalf("negotiated locale = %q, want es", got)
	}

	request = httptest.NewRequest("GET", "/", nil)
	request.Header.Set("Accept-Language", "es-ES;q=0.4, de-DE;q=0.9, en;q=0")
	if got := localization.RequestLocale(request); got != "de" {
		t.Fatalf("quality-negotiated locale = %q, want de", got)
	}

	request = httptest.NewRequest("GET", "/", nil)
	request.Header.Set(localization.ClientLocaleHeader, "fr-FR")
	if got := localization.RequestLocale(request); got != localization.DefaultLocale {
		t.Fatalf("unsupported locale = %q, want %q", got, localization.DefaultLocale)
	}
}
