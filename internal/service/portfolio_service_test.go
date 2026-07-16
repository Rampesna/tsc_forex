package service

import (
	"context"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"

	"github.com/tsc-forex/tsc_forex/internal/domain"
	"github.com/tsc-forex/tsc_forex/internal/models"
	"github.com/tsc-forex/tsc_forex/internal/repository"
)

func TestProjectPortfolioLedgerMixedCurrency(t *testing.T) {
	portfolioID := uuid.New()
	at := time.Date(2026, 1, 2, 12, 0, 0, 0, time.UTC)
	transactions := []models.PortfolioTransaction{
		tx("deposit", "", "TRY", "0", "0", "20000", "0", "1", at),
		tx("buy", "NVDA", "USD", "10", "100", "0", "1", "30", at.Add(time.Minute)),
		tx("buy", "THYAO", "TRY", "5", "200", "0", "0", "1", at.Add(2*time.Minute)),
		tx("sell", "NVDA", "USD", "2", "120", "0", "1", "32", at.Add(3*time.Minute)),
		tx("dividend", "NVDA", "USD", "0", "0", "10", "0", "32", at.Add(4*time.Minute)),
		tx("buy", "GRAM_GOLD", "TRY", "3", "5000", "0", "3", "1", at.Add(5*time.Minute)),
		tx("buy", "BTC", "TRY", "0.005", "3000000", "0", "15", "1", at.Add(6*time.Minute)),
	}
	positions, netDeposits, err := ProjectPortfolioLedger(portfolioID, transactions)
	if err != nil {
		t.Fatal(err)
	}
	if !netDeposits.Equal(decimal.RequireFromString("20000")) {
		t.Fatalf("net deposits = %s", netDeposits)
	}
	byCode := map[string]models.PortfolioPosition{}
	for _, position := range positions {
		byCode[position.Code] = position
	}
	nvda := byCode["NVDA"]
	if !nvda.Quantity.Equal(decimal.NewFromInt(8)) {
		t.Fatalf("NVDA quantity = %s", nvda.Quantity)
	}
	if !nvda.AverageCostNative.Equal(decimal.RequireFromString("100.1")) {
		t.Fatalf("NVDA average native = %s", nvda.AverageCostNative)
	}
	if !nvda.AverageCostTRY.Equal(decimal.RequireFromString("3003")) {
		t.Fatalf("NVDA average TRY = %s", nvda.AverageCostTRY)
	}
	if !nvda.RealizedPnLTRY.Equal(decimal.RequireFromString("1962")) {
		t.Fatalf("NVDA realized = %s", nvda.RealizedPnLTRY)
	}
	thyao := byCode["THYAO"]
	if !thyao.AverageCostTRY.Equal(decimal.NewFromInt(200)) {
		t.Fatalf("THYAO average TRY = %s", thyao.AverageCostTRY)
	}
	if !byCode["GRAM_GOLD"].AverageCostTRY.Equal(decimal.NewFromInt(5001)) {
		t.Fatalf("GRAM_GOLD average TRY = %s", byCode["GRAM_GOLD"].AverageCostTRY)
	}
	if !byCode["BTC"].AverageCostTRY.Equal(decimal.NewFromInt(3003000)) {
		t.Fatalf("BTC average TRY = %s", byCode["BTC"].AverageCostTRY)
	}
}

func TestProjectPortfolioLedgerRejectsNegativePosition(t *testing.T) {
	transactions := []models.PortfolioTransaction{
		tx("sell", "NVDA", "USD", "1", "100", "0", "0", "30", time.Now()),
	}
	if _, _, err := ProjectPortfolioLedger(uuid.New(), transactions); err != ErrNegativePosition {
		t.Fatalf("error = %v, want ErrNegativePosition", err)
	}
}

func TestPortfolioCreateTransactionIsIdempotent(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:portfolio-"+uuid.NewString()+"?mode=memory&cache=shared"), &gorm.Config{})
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
	repo := repository.NewPortfolioRepository(db, nil)
	prices := fixedPortfolioPrices{
		"USDTRY": {Code: "USDTRY", Price: 32, PriceTRY: 32, Currency: "TRY"},
	}
	svc := NewPortfolioService(repo, prices)
	userID := uuid.New()
	portfolio, err := svc.Create(context.Background(), userID, "Test")
	if err != nil {
		t.Fatal(err)
	}
	request := models.PortfolioTransaction{
		ClientID:        "offline-1",
		Code:            "NVDA",
		Type:            PortfolioBuy,
		Quantity:        decimal.NewFromInt(2),
		UnitPriceNative: decimal.NewFromInt(100),
		OccurredAt:      time.Now().UTC(),
	}
	first, created, err := svc.CreateTransaction(context.Background(), userID, portfolio.ID, request)
	if err != nil || !created {
		t.Fatalf("first create: created=%v err=%v", created, err)
	}
	second, created, err := svc.CreateTransaction(context.Background(), userID, portfolio.ID, request)
	if err != nil || created {
		t.Fatalf("retry create: created=%v err=%v", created, err)
	}
	if first.ID != second.ID {
		t.Fatalf("idempotent retry IDs differ: %s vs %s", first.ID, second.ID)
	}
	rows, err := svc.Transactions(context.Background(), userID, portfolio.ID, 100, time.Time{})
	if err != nil || len(rows) != 1 {
		t.Fatalf("transactions=%d err=%v", len(rows), err)
	}
}

func TestPortfolioPerformanceValuesMixedAssetClasses(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:performance-"+uuid.NewString()+"?mode=memory&cache=shared"), &gorm.Config{})
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
	repo := repository.NewPortfolioRepository(db, nil)
	svc := NewPortfolioService(repo, fixedPortfolioPrices{
		"USDTRY": {Code: "USDTRY", Price: 31, PriceTRY: 31, Currency: "TRY"},
	}).WithHistory(fixedPortfolioCandles{})
	userID := uuid.New()
	portfolio, err := svc.Create(context.Background(), userID, "Mixed")
	if err != nil {
		t.Fatal(err)
	}
	at := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	inputs := []models.PortfolioTransaction{
		tx("buy", "NVDA", "USD", "2", "100", "0", "0", "30", at),
		tx("buy", "THYAO", "TRY", "5", "200", "0", "0", "1", at.Add(time.Minute)),
		tx("buy", "GRAM_GOLD", "TRY", "1", "3000", "0", "0", "1", at.Add(2*time.Minute)),
		tx("buy", "BTC", "TRY", "0.001", "2000000", "0", "0", "1", at.Add(3*time.Minute)),
	}
	for index := range inputs {
		inputs[index].ClientID = "mixed-" + inputs[index].Code
		if _, _, err := svc.CreateTransaction(
			context.Background(),
			userID,
			portfolio.ID,
			inputs[index],
		); err != nil {
			t.Fatal(err)
		}
	}
	day := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
	svc.candles = fixedPortfolioCandles{
		"USDTRY":    {{Bucket: day, Close: 31}},
		"NVDA":      {{Bucket: day, Close: 110}},
		"THYAO":     {{Bucket: day, Close: 210}},
		"GRAM_GOLD": {{Bucket: day, Close: 3100}},
		"BTC":       {{Bucket: day, Close: 2200000}},
	}
	points, err := svc.Performance(context.Background(), userID, portfolio.ID, day, day, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(points) != 1 {
		t.Fatalf("points = %d", len(points))
	}
	if !points[0].ValueTRY.Equal(decimal.NewFromInt(13170)) {
		t.Fatalf("value TRY = %s", points[0].ValueTRY)
	}
	if !points[0].CostBasisTRY.Equal(decimal.NewFromInt(12000)) {
		t.Fatalf("cost TRY = %s", points[0].CostBasisTRY)
	}
	if !points[0].TotalPnLTRY.Equal(decimal.NewFromInt(1170)) {
		t.Fatalf("total P/L TRY = %s", points[0].TotalPnLTRY)
	}
}

type fixedPortfolioPrices map[string]domain.InstrumentPrice

func (prices fixedPortfolioPrices) Get(code string) (domain.InstrumentPrice, bool) {
	price, ok := prices[code]
	return price, ok
}

type fixedPortfolioCandles map[string][]models.OHLC

func (candles fixedPortfolioCandles) Candles(
	_ context.Context,
	code, _ string,
	_, _ time.Time,
	_ int,
) ([]models.OHLC, error) {
	return candles[code], nil
}

func tx(
	kind, code, currency, quantity, unitPrice, amount, fee, usdtry string,
	at time.Time,
) models.PortfolioTransaction {
	return models.PortfolioTransaction{
		ID:              uuid.New(),
		ClientID:        uuid.NewString(),
		Type:            kind,
		Code:            code,
		Currency:        currency,
		Quantity:        decimal.RequireFromString(quantity),
		UnitPriceNative: decimal.RequireFromString(unitPrice),
		AmountNative:    decimal.RequireFromString(amount),
		FeeNative:       decimal.RequireFromString(fee),
		USDTRY:          decimal.RequireFromString(usdtry),
		OccurredAt:      at,
		CreatedAt:       at,
	}
}
