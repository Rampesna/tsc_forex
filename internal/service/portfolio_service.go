package service

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/tsc-forex/tsc_forex/internal/domain"
	"github.com/tsc-forex/tsc_forex/internal/models"
	"github.com/tsc-forex/tsc_forex/internal/repository"
)

const (
	PortfolioBuy      = "buy"
	PortfolioSell     = "sell"
	PortfolioDividend = "dividend"
	PortfolioFee      = "fee"
	PortfolioDeposit  = "deposit"
	PortfolioWithdraw = "withdraw"
)

var (
	ErrInvalidPortfolio          = errors.New("invalid portfolio")
	ErrInvalidTransaction        = errors.New("invalid portfolio transaction")
	ErrNegativePosition          = errors.New("transaction would create a negative position")
	ErrLivePriceUnavailable      = errors.New("live price unavailable")
	ErrPortfolioPositionNotFound = errors.New("portfolio position not found")
)

type PortfolioPriceSource interface {
	Get(code string) (domain.InstrumentPrice, bool)
}

type PortfolioCandleSource interface {
	Candles(ctx context.Context, code, interval string, from, to time.Time, limit int) ([]models.OHLC, error)
}

// PortfolioService owns validation, exact-decimal position projection and live
// mixed-currency analytics. Persistence serializes mutations per portfolio.
type PortfolioService struct {
	repo    repository.PortfolioRepository
	prices  PortfolioPriceSource
	candles PortfolioCandleSource
}

func NewPortfolioService(repo repository.PortfolioRepository, prices PortfolioPriceSource) *PortfolioService {
	return &PortfolioService{repo: repo, prices: prices}
}

func (s *PortfolioService) WithHistory(candles PortfolioCandleSource) *PortfolioService {
	s.candles = candles
	return s
}

type PortfolioPositionAnalytics struct {
	ID                 uuid.UUID       `json:"id"`
	Code               string          `json:"code"`
	DisplayName        string          `json:"display_name"`
	Kind               string          `json:"kind"`
	Currency           string          `json:"currency"`
	Quantity           decimal.Decimal `json:"quantity"`
	AverageCostNative  decimal.Decimal `json:"average_cost_native"`
	AverageCostTRY     decimal.Decimal `json:"average_cost_try"`
	CurrentPriceNative decimal.Decimal `json:"current_price_native"`
	CurrentPriceTRY    decimal.Decimal `json:"current_price_try"`
	ValueNative        decimal.Decimal `json:"value_native"`
	ValueTRY           decimal.Decimal `json:"value_try"`
	CostBasisTRY       decimal.Decimal `json:"cost_basis_try"`
	UnrealizedPnLTRY   decimal.Decimal `json:"unrealized_pnl_try"`
	UnrealizedPnLPct   decimal.Decimal `json:"unrealized_pnl_pct"`
	RealizedPnLTRY     decimal.Decimal `json:"realized_pnl_try"`
	DayChangeTRY       decimal.Decimal `json:"day_change_try"`
	DayChangePct       decimal.Decimal `json:"day_change_pct"`
	AllocationPct      decimal.Decimal `json:"allocation_pct"`
	UpdatedAt          time.Time       `json:"updated_at"`
}

type PortfolioOverview struct {
	Portfolio       models.Portfolio             `json:"portfolio"`
	Positions       []PortfolioPositionAnalytics `json:"positions"`
	TotalValueTRY   decimal.Decimal              `json:"total_value_try"`
	TotalCostTRY    decimal.Decimal              `json:"total_cost_try"`
	UnrealizedPnL   decimal.Decimal              `json:"unrealized_pnl_try"`
	RealizedPnL     decimal.Decimal              `json:"realized_pnl_try"`
	DayChangeTRY    decimal.Decimal              `json:"day_change_try"`
	DayChangePct    decimal.Decimal              `json:"day_change_pct"`
	NetDepositsTRY  decimal.Decimal              `json:"net_deposits_try"`
	ActivePositions int                          `json:"active_positions"`
}

func (s *PortfolioService) Create(
	ctx context.Context,
	userID uuid.UUID,
	name string,
) (models.Portfolio, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "Ana Portföy"
	}
	if len([]rune(name)) > 80 {
		return models.Portfolio{}, ErrInvalidPortfolio
	}
	portfolio := models.Portfolio{
		ID:           uuid.New(),
		UserID:       userID,
		Name:         name,
		BaseCurrency: "TRY",
		Version:      1,
	}
	if err := s.repo.Create(ctx, &portfolio); err != nil {
		return models.Portfolio{}, err
	}
	return portfolio, nil
}

func (s *PortfolioService) List(ctx context.Context, userID uuid.UUID) ([]PortfolioOverview, error) {
	portfolios, err := s.repo.List(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]PortfolioOverview, 0, len(portfolios))
	for _, portfolio := range portfolios {
		overview, err := s.overview(ctx, portfolio)
		if err != nil {
			return nil, err
		}
		out = append(out, overview)
	}
	return out, nil
}

func (s *PortfolioService) Get(ctx context.Context, userID, portfolioID uuid.UUID) (PortfolioOverview, error) {
	portfolio, err := s.repo.Get(ctx, userID, portfolioID)
	if err != nil {
		return PortfolioOverview{}, err
	}
	return s.overview(ctx, portfolio)
}

func (s *PortfolioService) Update(
	ctx context.Context,
	userID, portfolioID uuid.UUID,
	name string,
) (models.Portfolio, error) {
	name = strings.TrimSpace(name)
	if name == "" || len([]rune(name)) > 80 {
		return models.Portfolio{}, ErrInvalidPortfolio
	}
	portfolio, err := s.repo.Get(ctx, userID, portfolioID)
	if err != nil {
		return models.Portfolio{}, err
	}
	portfolio.Name = name
	if err := s.repo.Update(ctx, &portfolio); err != nil {
		return models.Portfolio{}, err
	}
	return s.repo.Get(ctx, userID, portfolioID)
}

func (s *PortfolioService) Delete(ctx context.Context, userID, portfolioID uuid.UUID) error {
	return s.repo.Delete(ctx, userID, portfolioID)
}

func (s *PortfolioService) Transactions(
	ctx context.Context,
	userID, portfolioID uuid.UUID,
	limit int,
	before time.Time,
) ([]models.PortfolioTransaction, error) {
	return s.repo.Transactions(ctx, userID, portfolioID, limit, before)
}

func (s *PortfolioService) CreateTransaction(
	ctx context.Context,
	userID, portfolioID uuid.UUID,
	transaction models.PortfolioTransaction,
) (models.PortfolioTransaction, bool, error) {
	if err := s.normalizeTransaction(&transaction); err != nil {
		return models.PortfolioTransaction{}, false, err
	}
	return s.repo.CreateTransaction(ctx, userID, portfolioID, &transaction, ProjectPortfolioLedger)
}

func (s *PortfolioService) UpdateTransaction(
	ctx context.Context,
	userID, portfolioID, transactionID uuid.UUID,
	transaction models.PortfolioTransaction,
) (models.PortfolioTransaction, error) {
	existing, err := s.repo.Transaction(ctx, userID, portfolioID, transactionID)
	if err != nil {
		return models.PortfolioTransaction{}, err
	}
	transaction.ID = transactionID
	if transaction.ClientID == "" {
		transaction.ClientID = existing.ClientID
	}
	if err := s.normalizeTransaction(&transaction); err != nil {
		return models.PortfolioTransaction{}, err
	}
	return s.repo.UpdateTransaction(ctx, userID, portfolioID, &transaction, ProjectPortfolioLedger)
}

func (s *PortfolioService) DeleteTransaction(
	ctx context.Context,
	userID, portfolioID, transactionID uuid.UUID,
) error {
	return s.repo.DeleteTransaction(ctx, userID, portfolioID, transactionID, ProjectPortfolioLedger)
}

func (s *PortfolioService) normalizeTransaction(transaction *models.PortfolioTransaction) error {
	transaction.Type = strings.ToLower(strings.TrimSpace(transaction.Type))
	transaction.Code = strings.ToUpper(strings.TrimSpace(transaction.Code))
	transaction.ClientID = strings.TrimSpace(transaction.ClientID)
	transaction.Note = strings.TrimSpace(transaction.Note)
	if transaction.ClientID == "" || len(transaction.ClientID) > 80 || len([]rune(transaction.Note)) > 500 {
		return ErrInvalidTransaction
	}
	if transaction.OccurredAt.IsZero() {
		transaction.OccurredAt = time.Now().UTC()
	} else {
		transaction.OccurredAt = transaction.OccurredAt.UTC()
	}
	if transaction.OccurredAt.After(time.Now().Add(5 * time.Minute)) {
		return ErrInvalidTransaction
	}
	if transaction.Quantity.IsNegative() || transaction.UnitPriceNative.IsNegative() ||
		transaction.AmountNative.IsNegative() || transaction.FeeNative.IsNegative() {
		return ErrInvalidTransaction
	}

	switch transaction.Type {
	case PortfolioBuy, PortfolioSell:
		instrument, ok := domain.CatalogByCode[transaction.Code]
		if !ok || !transaction.Quantity.IsPositive() || !transaction.UnitPriceNative.IsPositive() {
			return ErrInvalidTransaction
		}
		transaction.Currency = instrumentCurrency(instrument)
	case PortfolioDividend, PortfolioFee:
		instrument, ok := domain.CatalogByCode[transaction.Code]
		if !ok || !transaction.AmountNative.IsPositive() {
			return ErrInvalidTransaction
		}
		transaction.Currency = instrumentCurrency(instrument)
	case PortfolioDeposit, PortfolioWithdraw:
		transaction.Code = ""
		transaction.Quantity = decimal.Zero
		transaction.UnitPriceNative = decimal.Zero
		transaction.FeeNative = decimal.Zero
		if !transaction.AmountNative.IsPositive() {
			return ErrInvalidTransaction
		}
		transaction.Currency = strings.ToUpper(strings.TrimSpace(transaction.Currency))
		if transaction.Currency == "" {
			transaction.Currency = "TRY"
		}
		if transaction.Currency != "TRY" && transaction.Currency != "USD" {
			return ErrInvalidTransaction
		}
	default:
		return ErrInvalidTransaction
	}

	if transaction.Currency == "TRY" {
		transaction.USDTRY = decimal.NewFromInt(1)
	} else if !transaction.USDTRY.IsPositive() {
		if s.prices == nil {
			return ErrLivePriceUnavailable
		}
		price, ok := s.prices.Get("USDTRY")
		if !ok || price.Price <= 0 {
			return ErrLivePriceUnavailable
		}
		transaction.USDTRY = decimal.NewFromFloat(price.Price)
	}
	return nil
}

func instrumentCurrency(instrument domain.Instrument) string {
	if instrument.Currency != "" {
		return instrument.Currency
	}
	return "TRY"
}

type projectedPosition struct {
	currency    string
	quantity    decimal.Decimal
	avgNative   decimal.Decimal
	avgTRY      decimal.Decimal
	realizedTRY decimal.Decimal
}

// ProjectPortfolioLedger is deterministic and side-effect free. Replaying the
// same ledger always produces the same positions, which makes edits, deletes,
// retries and concurrency safe.
func ProjectPortfolioLedger(
	portfolioID uuid.UUID,
	transactions []models.PortfolioTransaction,
) ([]models.PortfolioPosition, decimal.Decimal, error) {
	ordered := append([]models.PortfolioTransaction(nil), transactions...)
	sort.SliceStable(ordered, func(i, j int) bool {
		if ordered[i].OccurredAt.Equal(ordered[j].OccurredAt) {
			if ordered[i].CreatedAt.Equal(ordered[j].CreatedAt) {
				return ordered[i].ID.String() < ordered[j].ID.String()
			}
			return ordered[i].CreatedAt.Before(ordered[j].CreatedAt)
		}
		return ordered[i].OccurredAt.Before(ordered[j].OccurredAt)
	})

	states := make(map[string]*projectedPosition)
	netDeposits := decimal.Zero
	for _, transaction := range ordered {
		fx := transaction.USDTRY
		if transaction.Currency == "TRY" {
			fx = decimal.NewFromInt(1)
		}
		if !fx.IsPositive() {
			return nil, decimal.Zero, ErrInvalidTransaction
		}
		switch transaction.Type {
		case PortfolioDeposit:
			netDeposits = netDeposits.Add(transaction.AmountNative.Mul(fx))
			continue
		case PortfolioWithdraw:
			netDeposits = netDeposits.Sub(transaction.AmountNative.Mul(fx))
			continue
		}

		if _, ok := domain.CatalogByCode[transaction.Code]; !ok {
			return nil, decimal.Zero, ErrInvalidTransaction
		}
		state := states[transaction.Code]
		if state == nil {
			state = &projectedPosition{currency: transaction.Currency}
			states[transaction.Code] = state
		}
		switch transaction.Type {
		case PortfolioBuy:
			if !transaction.Quantity.IsPositive() || !transaction.UnitPriceNative.IsPositive() {
				return nil, decimal.Zero, ErrInvalidTransaction
			}
			newQuantity := state.quantity.Add(transaction.Quantity)
			nativeCost := state.avgNative.Mul(state.quantity).
				Add(transaction.UnitPriceNative.Mul(transaction.Quantity)).
				Add(transaction.FeeNative)
			tryCost := state.avgTRY.Mul(state.quantity).
				Add(transaction.UnitPriceNative.Mul(transaction.Quantity).Mul(fx)).
				Add(transaction.FeeNative.Mul(fx))
			state.quantity = newQuantity
			state.avgNative = nativeCost.Div(newQuantity)
			state.avgTRY = tryCost.Div(newQuantity)
		case PortfolioSell:
			if !transaction.Quantity.IsPositive() || transaction.Quantity.GreaterThan(state.quantity) {
				return nil, decimal.Zero, ErrNegativePosition
			}
			proceedsTRY := transaction.UnitPriceNative.Mul(transaction.Quantity).Mul(fx).
				Sub(transaction.FeeNative.Mul(fx))
			costTRY := state.avgTRY.Mul(transaction.Quantity)
			state.realizedTRY = state.realizedTRY.Add(proceedsTRY.Sub(costTRY))
			state.quantity = state.quantity.Sub(transaction.Quantity)
			if state.quantity.IsZero() {
				state.avgNative = decimal.Zero
				state.avgTRY = decimal.Zero
			}
		case PortfolioDividend:
			if !transaction.AmountNative.IsPositive() {
				return nil, decimal.Zero, ErrInvalidTransaction
			}
			state.realizedTRY = state.realizedTRY.Add(transaction.AmountNative.Mul(fx))
		case PortfolioFee:
			if !transaction.AmountNative.IsPositive() {
				return nil, decimal.Zero, ErrInvalidTransaction
			}
			state.realizedTRY = state.realizedTRY.Sub(transaction.AmountNative.Mul(fx))
		default:
			return nil, decimal.Zero, ErrInvalidTransaction
		}
	}

	now := time.Now().UTC()
	codes := make([]string, 0, len(states))
	for code := range states {
		codes = append(codes, code)
	}
	sort.Strings(codes)
	positions := make([]models.PortfolioPosition, 0, len(codes))
	for _, code := range codes {
		state := states[code]
		if state.quantity.IsZero() && state.realizedTRY.IsZero() {
			continue
		}
		positions = append(positions, models.PortfolioPosition{
			ID:                uuid.NewSHA1(portfolioID, []byte(code)),
			PortfolioID:       portfolioID,
			Code:              code,
			Currency:          state.currency,
			Quantity:          state.quantity,
			AverageCostNative: state.avgNative,
			AverageCostTRY:    state.avgTRY,
			RealizedPnLTRY:    state.realizedTRY,
			CreatedAt:         now,
			UpdatedAt:         now,
		})
	}
	return positions, netDeposits, nil
}

func (s *PortfolioService) overview(
	ctx context.Context,
	portfolio models.Portfolio,
) (PortfolioOverview, error) {
	positions, err := s.repo.Positions(ctx, portfolio.UserID, portfolio.ID)
	if err != nil {
		return PortfolioOverview{}, err
	}
	overview := PortfolioOverview{
		Portfolio:      portfolio,
		NetDepositsTRY: portfolio.NetDepositsTRY,
		Positions:      make([]PortfolioPositionAnalytics, 0, len(positions)),
	}
	usdtry, _ := s.live("USDTRY")
	for _, position := range positions {
		instrument, ok := domain.CatalogByCode[position.Code]
		if !ok {
			continue
		}
		item := PortfolioPositionAnalytics{
			ID:                position.ID,
			Code:              position.Code,
			DisplayName:       instrument.DisplayName,
			Kind:              string(instrument.Kind),
			Currency:          position.Currency,
			Quantity:          position.Quantity,
			AverageCostNative: position.AverageCostNative,
			AverageCostTRY:    position.AverageCostTRY,
			CostBasisTRY:      position.Quantity.Mul(position.AverageCostTRY),
			RealizedPnLTRY:    position.RealizedPnLTRY,
			UpdatedAt:         position.UpdatedAt,
		}
		if live, ok := s.live(position.Code); ok {
			item.CurrentPriceNative = decimal.NewFromFloat(live.Price)
			tryPrice := live.PriceTRY
			if tryPrice <= 0 {
				tryPrice = live.Price
			}
			item.CurrentPriceTRY = decimal.NewFromFloat(tryPrice)
			item.ValueNative = position.Quantity.Mul(item.CurrentPriceNative)
			item.ValueTRY = position.Quantity.Mul(item.CurrentPriceTRY)
			item.UnrealizedPnLTRY = item.ValueTRY.Sub(item.CostBasisTRY)
			if item.CostBasisTRY.IsPositive() {
				item.UnrealizedPnLPct = item.UnrealizedPnLTRY.Div(item.CostBasisTRY).
					Mul(decimal.NewFromInt(100))
			}
			item.DayChangeTRY = position.Quantity.Mul(dayChangeTRY(live, usdtry))
			item.DayChangePct = decimal.NewFromFloat(live.ChangePct)
		}
		if position.Quantity.IsPositive() {
			overview.ActivePositions++
			overview.TotalValueTRY = overview.TotalValueTRY.Add(item.ValueTRY)
			overview.TotalCostTRY = overview.TotalCostTRY.Add(item.CostBasisTRY)
			overview.UnrealizedPnL = overview.UnrealizedPnL.Add(item.UnrealizedPnLTRY)
			overview.DayChangeTRY = overview.DayChangeTRY.Add(item.DayChangeTRY)
		}
		overview.RealizedPnL = overview.RealizedPnL.Add(item.RealizedPnLTRY)
		overview.Positions = append(overview.Positions, item)
	}
	if previous := overview.TotalValueTRY.Sub(overview.DayChangeTRY); previous.IsPositive() {
		overview.DayChangePct = overview.DayChangeTRY.Div(previous).Mul(decimal.NewFromInt(100))
	}
	if overview.TotalValueTRY.IsPositive() {
		for index := range overview.Positions {
			if overview.Positions[index].ValueTRY.IsPositive() {
				overview.Positions[index].AllocationPct = overview.Positions[index].ValueTRY.
					Div(overview.TotalValueTRY).
					Mul(decimal.NewFromInt(100))
			}
		}
	}
	return overview, nil
}

func (s *PortfolioService) live(code string) (domain.InstrumentPrice, bool) {
	if s.prices == nil {
		return domain.InstrumentPrice{}, false
	}
	return s.prices.Get(code)
}

func dayChangeTRY(price, usdtry domain.InstrumentPrice) decimal.Decimal {
	if price.Currency != "USD" {
		return decimal.NewFromFloat(price.Change)
	}
	currentFX := usdtry.Price
	if currentFX <= 0 {
		if price.Price > 0 && price.PriceTRY > 0 {
			currentFX = price.PriceTRY / price.Price
		}
	}
	previousPrice := price.Price - price.Change
	previousFX := currentFX - usdtry.Change
	if previousPrice <= 0 || previousFX <= 0 {
		return decimal.NewFromFloat(price.Change * currentFX)
	}
	return decimal.NewFromFloat(price.Price*currentFX - previousPrice*previousFX)
}
