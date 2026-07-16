package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/tsc-forex/tsc_forex/internal/domain"
	"github.com/tsc-forex/tsc_forex/internal/middleware"
	"github.com/tsc-forex/tsc_forex/internal/models"
	"github.com/tsc-forex/tsc_forex/internal/repository"
	"github.com/tsc-forex/tsc_forex/internal/service"
)

func TestPortfolioHandlerCreateTransactionAndOverview(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(
		sqlite.Open("file:portfolio-handler-"+uuid.NewString()+"?mode=memory&cache=shared"),
		&gorm.Config{},
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(
		&models.Portfolio{},
		&models.PortfolioPosition{},
		&models.PortfolioTransaction{},
	); err != nil {
		t.Fatal(err)
	}
	prices := handlerPortfolioPrices{
		"USDTRY": {Code: "USDTRY", Price: 32, PriceTRY: 32, Currency: "TRY"},
		"NVDA":   {Code: "NVDA", Price: 100, PriceTRY: 3200, Currency: "USD"},
	}
	handler := NewPortfolioHandler(
		service.NewPortfolioService(repository.NewPortfolioRepository(db, nil), prices),
	)
	userID := uuid.New()
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(middleware.ClaimsKey, &service.Claims{UserID: userID.String()})
		c.Next()
	})
	router.POST("/portfolios", handler.Create)
	router.GET("/portfolios/:portfolioID", handler.Get)
	router.POST("/portfolios/:portfolioID/transactions", handler.CreateTransaction)

	create := performJSON(router, http.MethodPost, "/portfolios", `{"name":"Ana Portföy"}`)
	if create.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", create.Code, create.Body.String())
	}
	var portfolio models.Portfolio
	if err := json.Unmarshal(create.Body.Bytes(), &portfolio); err != nil {
		t.Fatal(err)
	}

	path := "/portfolios/" + portfolio.ID.String() + "/transactions"
	body := `{
		"client_id":"ios-offline-1",
		"type":"buy",
		"code":"NVDA",
		"quantity":"2",
		"unit_price_native":"100",
		"occurred_at":"2026-07-15T12:00:00Z"
	}`
	first := performJSON(router, http.MethodPost, path, body)
	if first.Code != http.StatusCreated {
		t.Fatalf("transaction status=%d body=%s", first.Code, first.Body.String())
	}
	retry := performJSON(router, http.MethodPost, path, body)
	if retry.Code != http.StatusOK {
		t.Fatalf("idempotent retry status=%d body=%s", retry.Code, retry.Body.String())
	}

	overview := performJSON(
		router,
		http.MethodGet,
		"/portfolios/"+portfolio.ID.String(),
		"",
	)
	if overview.Code != http.StatusOK {
		t.Fatalf("overview status=%d body=%s", overview.Code, overview.Body.String())
	}
	var response struct {
		Positions []models.PortfolioPosition `json:"positions"`
	}
	if err := json.Unmarshal(overview.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if len(response.Positions) != 1 || response.Positions[0].Code != "NVDA" {
		t.Fatalf("unexpected positions: %+v", response.Positions)
	}
}

func performJSON(router http.Handler, method, path, body string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	return recorder
}

type handlerPortfolioPrices map[string]domain.InstrumentPrice

func (prices handlerPortfolioPrices) Get(code string) (domain.InstrumentPrice, bool) {
	price, ok := prices[code]
	return price, ok
}
