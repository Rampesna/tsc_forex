package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/tsc-forex/tsc_forex/internal/api"
	"github.com/tsc-forex/tsc_forex/internal/localization"
	"github.com/tsc-forex/tsc_forex/internal/models"
)

func TestInstrumentCatalogUsesRequestedDatabaseLocale(t *testing.T) {
	db, err := gorm.Open(
		sqlite.Open("file:api-localization-"+uuid.NewString()+"?mode=memory&cache=shared"),
		&gorm.Config{Logger: logger.Default.LogMode(logger.Silent)},
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(
		&models.SupportedLocale{},
		&models.InstrumentDefinition{},
		&models.InstrumentTranslation{},
	); err != nil {
		t.Fatal(err)
	}
	if err := localization.SeedCatalog(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	names, err := localization.NewInstrumentLocalizer(db)
	if err != nil {
		t.Fatal(err)
	}

	gin.SetMode(gin.TestMode)
	handler := api.NewInstrumentHandler(nil, nil, nil, "", nil).WithLocalizer(names)
	router := gin.New()
	router.GET("/instruments", handler.Catalog)
	request := httptest.NewRequest(http.MethodGet, "/instruments", nil)
	request.Header.Set(localization.ClientLocaleHeader, "it-IT")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d", response.Code)
	}

	var body struct {
		Instruments []struct {
			Code        string `json:"code"`
			DisplayName string `json:"display_name"`
		} `json:"instruments"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	for _, instrument := range body.Instruments {
		if instrument.Code == "USDTRY" {
			if instrument.DisplayName != "dollaro statunitense" {
				t.Fatalf("USDTRY name = %q", instrument.DisplayName)
			}
			return
		}
	}
	t.Fatal("USDTRY was not returned")
}
