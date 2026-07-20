package service

import (
	"context"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/tsc-forex/tsc_forex/internal/domain"
	"github.com/tsc-forex/tsc_forex/internal/models"
)

type PortfolioPositionDetail struct {
	Position     PortfolioPositionAnalytics    `json:"position"`
	Transactions []models.PortfolioTransaction `json:"transactions"`
}

func (s *PortfolioService) Position(
	ctx context.Context,
	userID, portfolioID uuid.UUID,
	code string,
) (PortfolioPositionDetail, error) {
	overview, err := s.Get(ctx, userID, portfolioID)
	if err != nil {
		return PortfolioPositionDetail{}, err
	}
	var selected *PortfolioPositionAnalytics
	for index := range overview.Positions {
		if overview.Positions[index].Code == code {
			selected = &overview.Positions[index]
			break
		}
	}
	if selected == nil {
		return PortfolioPositionDetail{}, ErrPortfolioPositionNotFound
	}
	transactions, err := s.repo.Transactions(ctx, userID, portfolioID, 5000, time.Time{})
	if err != nil {
		return PortfolioPositionDetail{}, err
	}
	filtered := make([]models.PortfolioTransaction, 0)
	for _, transaction := range transactions {
		if transaction.Code == code {
			filtered = append(filtered, transaction)
		}
	}
	return PortfolioPositionDetail{Position: *selected, Transactions: filtered}, nil
}

type PortfolioPerformancePoint struct {
	Date           string          `json:"date"`
	ValueTRY       decimal.Decimal `json:"value_try"`
	CostBasisTRY   decimal.Decimal `json:"cost_basis_try"`
	RealizedPnLTRY decimal.Decimal `json:"realized_pnl_try"`
	TotalPnLTRY    decimal.Decimal `json:"total_pnl_try"`
	NetDepositsTRY decimal.Decimal `json:"net_deposits_try"`
}

type performancePosition struct {
	quantity decimal.Decimal
	avgTRY   decimal.Decimal
	realized decimal.Decimal
}

// Performance reconstructs daily holdings from the ledger and values them from
// pre-aggregated 1d candles. It performs one indexed history read per held code,
// never scans raw ticks, and caps the range to five years/2k points.
func (s *PortfolioService) Performance(
	ctx context.Context,
	userID, portfolioID uuid.UUID,
	from, to time.Time,
	maxPoints int,
) ([]PortfolioPerformancePoint, error) {
	if s.candles == nil {
		return nil, ErrLivePriceUnavailable
	}
	if _, err := s.repo.Get(ctx, userID, portfolioID); err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	if to.IsZero() || to.After(now) {
		to = now
	}
	to = dayStartUTC(to)
	if from.IsZero() {
		from = to.AddDate(-1, 0, 0)
	}
	from = dayStartUTC(from)
	earliest := to.AddDate(-5, 0, 0)
	if from.Before(earliest) {
		from = earliest
	}
	if from.After(to) {
		return nil, ErrInvalidPortfolio
	}
	if maxPoints <= 0 {
		maxPoints = 365
	}
	if maxPoints > 2000 {
		maxPoints = 2000
	}

	transactions, err := s.repo.Transactions(ctx, userID, portfolioID, 5000, time.Time{})
	if err != nil {
		return nil, err
	}
	sort.SliceStable(transactions, func(i, j int) bool {
		return transactions[i].OccurredAt.Before(transactions[j].OccurredAt)
	})
	codeSet := make(map[string]struct{})
	for _, transaction := range transactions {
		if transaction.Code != "" {
			codeSet[transaction.Code] = struct{}{}
		}
	}

	historyFrom := from.AddDate(0, 0, -10)
	closeByCode := make(map[string]map[string]decimal.Decimal, len(codeSet))
	lastClose := make(map[string]decimal.Decimal, len(codeSet))
	for code := range codeSet {
		candles, err := s.candles.Candles(ctx, code, "1d", historyFrom, to.AddDate(0, 0, 1), 2000)
		if err != nil {
			return nil, err
		}
		closeByCode[code] = dailyCloses(candles)
		lastClose[code] = latestCloseBefore(candles, from)
	}
	usdCandles, err := s.candles.Candles(ctx, "USDTRY", "1d", historyFrom, to.AddDate(0, 0, 1), 2000)
	if err != nil {
		return nil, err
	}
	usdCloses := dailyCloses(usdCandles)
	lastUSD := latestCloseBefore(usdCandles, from)

	states := make(map[string]*performancePosition)
	netDeposits := decimal.Zero
	txIndex := 0
	points := make([]PortfolioPerformancePoint, 0, int(to.Sub(from).Hours()/24)+1)
	for day := from; !day.After(to); day = day.AddDate(0, 0, 1) {
		dayEnd := day.AddDate(0, 0, 1)
		for txIndex < len(transactions) && transactions[txIndex].OccurredAt.Before(dayEnd) {
			applyPerformanceTransaction(states, &netDeposits, transactions[txIndex])
			txIndex++
		}
		dayKey := day.Format("2006-01-02")
		if value, ok := usdCloses[dayKey]; ok {
			lastUSD = value
		}
		valueTRY := decimal.Zero
		costTRY := decimal.Zero
		realizedTRY := decimal.Zero
		for code, state := range states {
			if value, ok := closeByCode[code][dayKey]; ok {
				lastClose[code] = value
			}
			price := lastClose[code]
			instrument, ok := domain.CatalogByCode[code]
			if ok && instrumentCurrency(instrument) == "USD" && lastUSD.IsPositive() {
				price = price.Mul(lastUSD)
			}
			valueTRY = valueTRY.Add(state.quantity.Mul(price))
			costTRY = costTRY.Add(state.quantity.Mul(state.avgTRY))
			realizedTRY = realizedTRY.Add(state.realized)
		}
		points = append(points, PortfolioPerformancePoint{
			Date:           dayKey,
			ValueTRY:       valueTRY,
			CostBasisTRY:   costTRY,
			RealizedPnLTRY: realizedTRY,
			TotalPnLTRY:    valueTRY.Sub(costTRY).Add(realizedTRY),
			NetDepositsTRY: netDeposits,
		})
	}
	if len(points) <= maxPoints {
		return points, nil
	}
	step := (len(points) + maxPoints - 1) / maxPoints
	sampled := make([]PortfolioPerformancePoint, 0, maxPoints+1)
	for index := 0; index < len(points); index += step {
		sampled = append(sampled, points[index])
	}
	if sampled[len(sampled)-1].Date != points[len(points)-1].Date {
		sampled = append(sampled, points[len(points)-1])
	}
	return sampled, nil
}

func dayStartUTC(value time.Time) time.Time {
	value = value.UTC()
	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, time.UTC)
}

func dailyCloses(candles []models.OHLC) map[string]decimal.Decimal {
	out := make(map[string]decimal.Decimal, len(candles))
	for _, candle := range candles {
		if candle.Close > 0 {
			out[candle.Bucket.UTC().Format("2006-01-02")] = decimal.NewFromFloat(candle.Close)
		}
	}
	return out
}

func latestCloseBefore(candles []models.OHLC, before time.Time) decimal.Decimal {
	latestAt := time.Time{}
	latest := decimal.Zero
	for _, candle := range candles {
		at := candle.Bucket.UTC()
		if candle.Close <= 0 || !at.Before(before) || (!latestAt.IsZero() && !at.After(latestAt)) {
			continue
		}
		latestAt = at
		latest = decimal.NewFromFloat(candle.Close)
	}
	return latest
}

func applyPerformanceTransaction(
	states map[string]*performancePosition,
	netDeposits *decimal.Decimal,
	transaction models.PortfolioTransaction,
) {
	fx := transaction.USDTRY
	if transaction.Currency == "TRY" {
		fx = decimal.NewFromInt(1)
	}
	switch transaction.Type {
	case PortfolioDeposit:
		*netDeposits = netDeposits.Add(transaction.AmountNative.Mul(fx))
		return
	case PortfolioWithdraw:
		*netDeposits = netDeposits.Sub(transaction.AmountNative.Mul(fx))
		return
	}
	state := states[transaction.Code]
	if state == nil {
		state = &performancePosition{}
		states[transaction.Code] = state
	}
	switch transaction.Type {
	case PortfolioBuy:
		newQuantity := state.quantity.Add(transaction.Quantity)
		totalCost := state.avgTRY.Mul(state.quantity).
			Add(transaction.UnitPriceNative.Mul(transaction.Quantity).Mul(fx)).
			Add(transaction.FeeNative.Mul(fx))
		state.quantity = newQuantity
		if newQuantity.IsPositive() {
			state.avgTRY = totalCost.Div(newQuantity)
		}
	case PortfolioSell:
		proceeds := transaction.UnitPriceNative.Mul(transaction.Quantity).Mul(fx).
			Sub(transaction.FeeNative.Mul(fx))
		state.realized = state.realized.Add(proceeds.Sub(state.avgTRY.Mul(transaction.Quantity)))
		state.quantity = state.quantity.Sub(transaction.Quantity)
		if state.quantity.IsZero() {
			state.avgTRY = decimal.Zero
		}
	case PortfolioDividend:
		state.realized = state.realized.Add(transaction.AmountNative.Mul(fx))
	case PortfolioFee:
		state.realized = state.realized.Sub(transaction.AmountNative.Mul(fx))
	}
}
