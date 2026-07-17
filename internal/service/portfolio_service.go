package service

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"math"
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
	billing *SubscriptionService
}

func NewPortfolioService(repo repository.PortfolioRepository, prices PortfolioPriceSource) *PortfolioService {
	return &PortfolioService{repo: repo, prices: prices}
}

func (s *PortfolioService) WithHistory(candles PortfolioCandleSource) *PortfolioService {
	s.candles = candles
	return s
}

func (s *PortfolioService) WithSubscriptions(billing *SubscriptionService) *PortfolioService {
	s.billing = billing
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
	var createErr error
	if s.billing != nil {
		createErr = s.billing.CreatePortfolioWithinCapacity(ctx, &portfolio)
	} else {
		createErr = s.repo.Create(ctx, &portfolio)
	}
	if createErr != nil {
		return models.Portfolio{}, createErr
	}
	return portfolio, nil
}

type PortfolioHealthFinding struct {
	Title    string `json:"title"`
	Detail   string `json:"detail"`
	Severity string `json:"severity"`
}

type PortfolioHealth struct {
	Score       int                      `json:"score"`
	Grade       string                   `json:"grade"`
	Summary     string                   `json:"summary"`
	Findings    []PortfolioHealthFinding `json:"findings"`
	GeneratedAt time.Time                `json:"generated_at"`
}

// HealthCheck is a deterministic educational risk overview. It avoids price
// predictions and buy/sell language; the paid allowance covers the scheduled
// server-side analysis/report generation feature.
func (s *PortfolioService) HealthCheck(ctx context.Context, userID, portfolioID uuid.UUID, requestKey string) (PortfolioHealth, error) {
	overview, err := s.Get(ctx, userID, portfolioID)
	if err != nil {
		return PortfolioHealth{}, err
	}
	artifact := FeatureArtifactReservation{Created: true}
	if s.billing != nil {
		artifact, err = s.billing.BeginFeatureArtifact(ctx, userID, FeaturePortfolioHealth, requestKey, portfolioID.String())
		if err != nil {
			return PortfolioHealth{}, err
		}
		if !artifact.Created {
			var replay PortfolioHealth
			if json.Unmarshal(artifact.Payload, &replay) != nil {
				return PortfolioHealth{}, ErrRequestConflict
			}
			return replay, nil
		}
	}
	score := 100
	findings := make([]PortfolioHealthFinding, 0, 5)
	if overview.ActivePositions == 0 {
		score = 35
		findings = append(findings, PortfolioHealthFinding{Title: "Henüz pozisyon yok", Detail: "Dağılım analizi için portföye en az bir işlem ekleyin.", Severity: "medium"})
	} else {
		if overview.ActivePositions < 3 {
			score -= 15
			findings = append(findings, PortfolioHealthFinding{Title: "Sınırlı çeşitlilik", Detail: "Portföy üçten az aktif pozisyona dayanıyor; tekil hareketlerin etkisi artabilir.", Severity: "medium"})
		}
		largest := 0.0
		kindWeights := map[string]float64{}
		for _, position := range overview.Positions {
			allocation := position.AllocationPct.InexactFloat64()
			largest = math.Max(largest, allocation)
			kindWeights[position.Kind] += allocation
		}
		if largest > 50 {
			score -= 25
			findings = append(findings, PortfolioHealthFinding{Title: "Yüksek tek-varlık yoğunluğu", Detail: fmt.Sprintf("En büyük pozisyon portföyün yaklaşık %%%.0f'sini oluşturuyor.", largest), Severity: "high"})
		} else if largest > 35 {
			score -= 12
			findings = append(findings, PortfolioHealthFinding{Title: "Belirgin pozisyon yoğunluğu", Detail: fmt.Sprintf("En büyük pozisyonun ağırlığı %%%.0f seviyesinde.", largest), Severity: "medium"})
		}
		for kind, weight := range kindWeights {
			if weight > 75 {
				score -= 12
				findings = append(findings, PortfolioHealthFinding{Title: "Varlık sınıfı yoğunluğu", Detail: fmt.Sprintf("%s sınıfı toplam dağılımın %%%.0f'sini oluşturuyor.", portfolioKindLabel(kind), weight), Severity: "medium"})
				break
			}
		}
		if overview.TotalCostTRY.IsPositive() {
			pnlPct := overview.UnrealizedPnL.Div(overview.TotalCostTRY).Mul(decimal.NewFromInt(100)).InexactFloat64()
			if pnlPct < -20 {
				score -= 15
				findings = append(findings, PortfolioHealthFinding{Title: "Yüksek gerçekleşmemiş kayıp", Detail: fmt.Sprintf("Toplam maliyete göre gerçekleşmemiş değişim %%%.1f.", pnlPct), Severity: "high"})
			}
		}
		dayPct := math.Abs(overview.DayChangePct.InexactFloat64())
		if dayPct > 5 {
			score -= 10
			findings = append(findings, PortfolioHealthFinding{Title: "Yüksek günlük oynaklık", Detail: fmt.Sprintf("Portföyün gün içi hareketi yaklaşık %%%.1f.", dayPct), Severity: "medium"})
		}
	}
	score = max(0, min(100, score))
	grade := "Dengeli"
	summary := "Dağılım ve güncel risk göstergeleri genel olarak izlenebilir seviyede."
	switch {
	case score >= 85:
		grade, summary = "Güçlü", "Dağılım ve yoğunluk göstergeleri mevcut kurallar içinde güçlü görünüyor."
	case score < 50:
		grade, summary = "Dikkat", "Yoğunluk veya oynaklık göstergelerinden bazıları yakından izlenmeli."
	case score < 70:
		grade, summary = "İzlenmeli", "Portföyde iyileştirilebilecek birkaç dağılım göstergesi bulunuyor."
	}
	if len(findings) == 0 {
		findings = append(findings, PortfolioHealthFinding{Title: "Belirgin yoğunluk uyarısı yok", Detail: "Mevcut dağılım eşikleri önemli bir uyarı üretmedi.", Severity: "low"})
	}
	result := PortfolioHealth{Score: score, Grade: grade, Summary: summary, Findings: findings, GeneratedAt: time.Now().UTC()}
	if s.billing != nil {
		payload, marshalErr := json.Marshal(result)
		if marshalErr != nil {
			_ = s.billing.ReleaseFeatureArtifact(ctx, userID, FeaturePortfolioHealth, requestKey, artifact.OwnerToken)
			return PortfolioHealth{}, marshalErr
		}
		if err := s.billing.CompleteFeatureArtifact(ctx, userID, FeaturePortfolioHealth, requestKey, artifact.OwnerToken, payload, "application/json", ""); err != nil {
			_ = s.billing.ReleaseFeatureArtifact(ctx, userID, FeaturePortfolioHealth, requestKey, artifact.OwnerToken)
			return PortfolioHealth{}, err
		}
	}
	return result, nil
}

func portfolioKindLabel(kind string) string {
	switch kind {
	case "gold":
		return "Altın"
	case "currency":
		return "Döviz"
	case "crypto":
		return "Kripto"
	case "stock":
		return "Hisse"
	default:
		return "Aynı varlık"
	}
}

// ExportCSV creates a portable ledger export after consuming the monthly
// export allowance. Values are exact decimal strings to avoid rounding money.
func (s *PortfolioService) ExportCSV(ctx context.Context, userID, portfolioID uuid.UUID, requestKey string) ([]byte, string, error) {
	portfolio, err := s.repo.Get(ctx, userID, portfolioID)
	if err != nil {
		return nil, "", err
	}
	transactions, err := s.repo.Transactions(ctx, userID, portfolioID, 5000, time.Time{})
	if err != nil {
		return nil, "", err
	}
	artifact := FeatureArtifactReservation{Created: true}
	if s.billing != nil {
		artifact, err = s.billing.BeginFeatureArtifact(ctx, userID, FeatureExports, requestKey, portfolioID.String())
		if err != nil {
			return nil, "", err
		}
		if !artifact.Created {
			return artifact.Payload, artifact.Filename, nil
		}
	}
	var buffer bytes.Buffer
	buffer.Write([]byte{0xEF, 0xBB, 0xBF}) // Excel-friendly UTF-8 BOM.
	writer := csv.NewWriter(&buffer)
	rows := [][]string{{"Portföy", "İşlem ID", "Tarih", "Tür", "Varlık", "Miktar", "Birim fiyat", "Tutar", "Masraf", "Para birimi", "USD/TRY", "Not"}}
	for _, transaction := range transactions {
		rows = append(rows, []string{
			portfolio.Name, transaction.ID.String(), transaction.OccurredAt.UTC().Format(time.RFC3339),
			transaction.Type, transaction.Code, transaction.Quantity.String(), transaction.UnitPriceNative.String(),
			transaction.AmountNative.String(), transaction.FeeNative.String(), transaction.Currency,
			transaction.USDTRY.String(), transaction.Note,
		})
	}
	if err := writer.WriteAll(rows); err != nil {
		return nil, "", err
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, "", err
	}
	filename := fmt.Sprintf("portfoy-%s-%s.csv", portfolio.ID.String()[:8], time.Now().UTC().Format("2006-01-02"))
	if s.billing != nil {
		payload := buffer.Bytes()
		if err := s.billing.CompleteFeatureArtifact(ctx, userID, FeatureExports, requestKey, artifact.OwnerToken, payload, "text/csv; charset=utf-8", filename); err != nil {
			_ = s.billing.ReleaseFeatureArtifact(ctx, userID, FeatureExports, requestKey, artifact.OwnerToken)
			return nil, "", err
		}
	}
	return buffer.Bytes(), filename, nil
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
