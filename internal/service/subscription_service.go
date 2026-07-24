package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/tsc-forex/tsc_forex/internal/config"
	"github.com/tsc-forex/tsc_forex/internal/models"
)

const (
	FeatureAIResearch      = "ai_research"
	FeatureActiveAlerts    = "active_alerts"
	FeatureWatchlist       = "watchlist"
	FeaturePortfolios      = "portfolios"
	FeaturePortfolioHealth = "portfolio_health"
	FeatureExports         = "exports"
)

const (
	PlanFree   = "free"
	PlanPlus   = "plus"
	PlanPro    = "pro"
	PlanExpert = "expert"
	PlanElite  = "elite"
)

var (
	ErrBillingDisabled              = errors.New("billing disabled")
	ErrSubscriptionRequired         = errors.New("subscription required")
	ErrFreeResearchUnavailable      = errors.New("free research unavailable")
	ErrInvalidDeviceID              = errors.New("invalid device id")
	ErrRequestConflict              = errors.New("request key reused for a different operation")
	ErrOperationInProgress          = errors.New("idempotent operation is still in progress")
	ErrTransactionAccount           = errors.New("app store transaction belongs to another account")
	ErrInvalidAppleTransaction      = errors.New("invalid apple transaction")
	ErrInvalidGooglePurchase        = errors.New("invalid google play purchase")
	ErrGoogleBillingUnavailable     = errors.New("google play billing unavailable")
	ErrDeviceAttestationNeeded      = errors.New("verified iOS App Attest assertion required")
	ErrDeviceAttestationUnavailable = errors.New("iOS App Attest verification unavailable")
	ErrInvalidDeviceAttestation     = errors.New("invalid iOS App Attest assertion")
	ErrDeviceCheckUnavailable       = errors.New("Apple DeviceCheck verification unavailable")
	ErrInvalidDeviceCheckToken      = errors.New("invalid Apple DeviceCheck token")
)

// PlanLimits is intentionally shared by API clients. Capacity limits are
// enforced on current rows; monthly limits are consumed in FeatureUsage.
type PlanLimits struct {
	AIResearch          int `json:"ai_research"`
	ActiveAlerts        int `json:"active_alerts"`
	Watchlist           int `json:"watchlist"`
	Portfolios          int `json:"portfolios"`
	PortfolioHealth     int `json:"portfolio_health"`
	Exports             int `json:"exports"`
	ResearchArchiveDays int `json:"research_archive_days"`
}

type SubscriptionPlan struct {
	Code         string     `json:"code"`
	Name         string     `json:"name"`
	ProductID    string     `json:"product_id,omitempty"`
	PriceHintTRY float64    `json:"price_hint_try"`
	Limits       PlanLimits `json:"limits"`
}

var planCatalog = []SubscriptionPlan{
	{Code: PlanPlus, Name: "Başlangıç", ProductID: "talhacan.tscforex.plus.monthly", PriceHintTRY: 149.99,
		Limits: PlanLimits{AIResearch: 15, ActiveAlerts: 10, Watchlist: 30, Portfolios: 2, PortfolioHealth: 2, Exports: 2, ResearchArchiveDays: 30}},
	{Code: PlanPro, Name: "Pro", ProductID: "talhacan.tscforex.pro.monthly", PriceHintTRY: 249.99,
		Limits: PlanLimits{AIResearch: 25, ActiveAlerts: 25, Watchlist: 75, Portfolios: 5, PortfolioHealth: 5, Exports: 10, ResearchArchiveDays: 90}},
	{Code: PlanExpert, Name: "Uzman", ProductID: "talhacan.tscforex.expert.monthly", PriceHintTRY: 499.99,
		Limits: PlanLimits{AIResearch: 50, ActiveAlerts: 60, Watchlist: 200, Portfolios: 15, PortfolioHealth: 15, Exports: 30, ResearchArchiveDays: 180}},
	{Code: PlanElite, Name: "Elite", ProductID: "talhacan.tscforex.elite.monthly", PriceHintTRY: 999.99,
		Limits: PlanLimits{AIResearch: 100, ActiveAlerts: 150, Watchlist: 500, Portfolios: 50, PortfolioHealth: 40, Exports: 100, ResearchArchiveDays: 365}},
}

var freePlan = SubscriptionPlan{
	Code: PlanFree, Name: "Ücretsiz",
	Limits: PlanLimits{AIResearch: 1, ActiveAlerts: 2, Watchlist: 10, Portfolios: 1},
}

func SubscriptionPlans() []SubscriptionPlan {
	out := make([]SubscriptionPlan, len(planCatalog))
	copy(out, planCatalog)
	return out
}

func planForProduct(productID string) (SubscriptionPlan, bool) {
	for _, plan := range planCatalog {
		if plan.ProductID == productID {
			return plan, true
		}
	}
	return SubscriptionPlan{}, false
}

func planForCode(code string) SubscriptionPlan {
	for _, plan := range planCatalog {
		if plan.Code == code {
			return plan
		}
	}
	return freePlan
}

type FeatureQuota struct {
	Limit     int        `json:"limit"`
	Used      int        `json:"used"`
	Remaining int        `json:"remaining"`
	PeriodEnd *time.Time `json:"period_end,omitempty"`
}

type EntitlementView struct {
	ProductID   string    `json:"product_id"`
	Status      string    `json:"status"`
	Environment string    `json:"environment"`
	ExpiresAt   time.Time `json:"expires_at"`
}

type BillingStatus struct {
	Plan                  SubscriptionPlan        `json:"plan"`
	Entitlement           *EntitlementView        `json:"entitlement,omitempty"`
	Usage                 map[string]FeatureQuota `json:"usage"`
	FreeResearchAvailable bool                    `json:"free_research_available"`
	PurchaseChannel       string                  `json:"purchase_channel"`
	ManageURL             string                  `json:"manage_url"`
}

// QuotaError provides enough structured context for mobile/web to present a
// paywall without parsing localized error strings.
type QuotaError struct {
	Feature string
	Limit   int
	Used    int
	Code    string
}

func (e *QuotaError) Error() string {
	if e.Code == "subscription_required" {
		return "bu özellik ücretli paket gerektiriyor"
	}
	return "aylık kullanım hakkı tükendi"
}

type Consumption struct {
	Plan      SubscriptionPlan
	Limit     int
	Used      int
	Remaining int
	FreeTrial bool
}

type ResearchArchiveItem struct {
	ID          uuid.UUID       `json:"id"`
	Code        string          `json:"code"`
	Payload     json.RawMessage `json:"payload"`
	GeneratedAt time.Time       `json:"generated_at"`
	ExpiresAt   time.Time       `json:"expires_at"`
}

// SubscriptionService owns entitlement, abuse-prevention and quota state.
// It deliberately uses one database transaction for each consumption so a
// concurrent retry cannot overspend the same allowance.
type SubscriptionService struct {
	db             *gorm.DB
	cfg            config.BillingConfig
	verifier       AppleSignedDataVerifier
	googleVerifier GooglePlayPurchaseVerifier
	appAttest      AppAttestVerifier
	deviceCheck    DeviceCheckVerifier
	now            func() time.Time
}

func NewSubscriptionService(db *gorm.DB, cfg config.BillingConfig, verifier AppleSignedDataVerifier, appAttest ...AppAttestVerifier) *SubscriptionService {
	attestationVerifier := AppAttestVerifier(NewAppleAppAttestVerifier(cfg))
	if len(appAttest) > 0 {
		attestationVerifier = appAttest[0]
	}
	return &SubscriptionService{
		db: db, cfg: cfg, verifier: verifier, appAttest: attestationVerifier,
		deviceCheck: NewAppleDeviceCheckVerifier(cfg), now: func() time.Time { return time.Now().UTC() },
	}
}

// WithGooglePlay attaches an optional Play purchase verifier. Apple paths are
// unchanged when this is nil or not ready.
func (s *SubscriptionService) WithGooglePlay(verifier GooglePlayPurchaseVerifier) *SubscriptionService {
	if s != nil {
		s.googleVerifier = verifier
	}
	return s
}

func (s *SubscriptionService) Enabled() bool { return s != nil && s.db != nil && s.cfg.Enabled }

func (s *SubscriptionService) AppAttestReady() bool { return s.appAttestReady() }
func (s *SubscriptionService) FreeTrialReady() bool { return s.freeTrialReady() }

// ReadinessError validates locally checkable billing prerequisites. It does
// not contact Apple, but it prevents a production pod from advertising ready
// when an enabled verifier/key cannot even be loaded and used for ES256.
func (s *SubscriptionService) ReadinessError() error {
	if s == nil || !s.cfg.Enabled {
		return nil
	}
	if s.verifier == nil {
		return errors.New("app store signed-data verifier unavailable")
	}
	if s.cfg.AppAttestEnabled && (s.appAttest == nil || !s.appAttest.Ready()) {
		return errors.New("app attest verifier unavailable")
	}
	if s.cfg.DeviceCheckEnabled && (s.deviceCheck == nil || !s.deviceCheck.Ready()) {
		return errors.New("device check verifier unavailable")
	}
	return nil
}

func (s *SubscriptionService) Status(ctx context.Context, userID uuid.UUID, deviceID string) (BillingStatus, error) {
	if s == nil || s.db == nil {
		return BillingStatus{}, ErrBillingDisabled
	}
	now := s.now()
	ent, active := s.activeEntitlement(ctx, s.db, userID, now)
	plan := freePlan
	if active {
		plan = planForCode(ent.PlanCode)
	}
	status := BillingStatus{
		Plan: plan, Usage: make(map[string]FeatureQuota), PurchaseChannel: "ios_app_store",
		ManageURL: "https://apps.apple.com/account/subscriptions",
	}
	if ent.UserID != uuid.Nil {
		state := ent.Status
		if ent.RevokedAt != nil {
			state = "revoked"
		} else if !ent.ExpiresAt.After(now) {
			state = "expired"
		}
		status.Entitlement = &EntitlementView{ProductID: ent.ProductID, Status: state, Environment: ent.Environment, ExpiresAt: ent.ExpiresAt}
		if strings.EqualFold(ent.Provider, "google") {
			status.PurchaseChannel = "google_play"
			status.ManageURL = "https://play.google.com/store/account/subscriptions?package=com.talhacan.tscforex"
		}
	}

	periodKey, periodEnd := quotaPeriod(ent, active, now)
	for _, feature := range []string{FeatureAIResearch, FeatureActiveAlerts, FeatureWatchlist, FeaturePortfolios, FeaturePortfolioHealth, FeatureExports} {
		limit := limitForFeature(plan.Limits, feature)
		used := 0
		var err error
		switch feature {
		case FeatureActiveAlerts, FeatureWatchlist, FeaturePortfolios:
			used, err = s.resourceCount(ctx, s.db, userID, feature)
		case FeatureAIResearch:
			if active {
				used, err = s.usageCount(ctx, s.db, userID, feature, periodKey)
			} else {
				used, err = s.freeClaimCount(ctx, s.db, userID, deviceID)
			}
		default:
			used, err = s.usageCount(ctx, s.db, userID, feature, periodKey)
		}
		if err != nil {
			return BillingStatus{}, err
		}
		remaining := max(0, limit-used)
		q := FeatureQuota{Limit: limit, Used: used, Remaining: remaining}
		if active && feature != FeatureActiveAlerts && feature != FeatureWatchlist && feature != FeaturePortfolios {
			end := periodEnd
			q.PeriodEnd = &end
		}
		status.Usage[feature] = q
	}
	status.FreeResearchAvailable = !active && status.Usage[FeatureAIResearch].Remaining > 0 && s.hasAttestedDevice(ctx, deviceID)
	return status, nil
}

func (s *SubscriptionService) hasAttestedDevice(ctx context.Context, deviceID string) bool {
	if s == nil || !s.freeTrialReady() {
		return false
	}
	digest, err := deviceDigest(deviceID)
	if err != nil {
		return false
	}
	var count int64
	return s.db.WithContext(ctx).Model(&models.AppAttestKey{}).Where("device_hash = ?", digest).Count(&count).Error == nil && count == 1
}

func (s *SubscriptionService) activeEntitlement(ctx context.Context, db *gorm.DB, userID uuid.UUID, now time.Time) (models.SubscriptionEntitlement, bool) {
	var ent models.SubscriptionEntitlement
	err := db.WithContext(ctx).Where("user_id = ?", userID).First(&ent).Error
	if err != nil {
		return models.SubscriptionEntitlement{}, false
	}
	active := ent.Status == "active" && ent.RevokedAt == nil && ent.ExpiresAt.After(now)
	return ent, active
}

func quotaPeriod(ent models.SubscriptionEntitlement, active bool, now time.Time) (string, time.Time) {
	if active {
		return ent.OriginalTransactionID + ":" + ent.PeriodStart.UTC().Format(time.RFC3339Nano), ent.ExpiresAt.UTC()
	}
	start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	return "free:" + start.Format("2006-01"), start.AddDate(0, 1, 0)
}

func limitForFeature(l PlanLimits, feature string) int {
	switch feature {
	case FeatureAIResearch:
		return l.AIResearch
	case FeatureActiveAlerts:
		return l.ActiveAlerts
	case FeatureWatchlist:
		return l.Watchlist
	case FeaturePortfolios:
		return l.Portfolios
	case FeaturePortfolioHealth:
		return l.PortfolioHealth
	case FeatureExports:
		return l.Exports
	default:
		return 0
	}
}

func normalizeRequest(requestKey, scope string) (string, string, error) {
	requestKey = strings.TrimSpace(requestKey)
	scope = strings.TrimSpace(scope)
	if requestKey == "" || len(requestKey) > 80 || scope == "" || len(scope) > 160 {
		return "", "", ErrRequestConflict
	}
	return requestKey, scope, nil
}

func deviceDigest(deviceID string) (string, error) {
	deviceID = strings.TrimSpace(deviceID)
	if len(deviceID) < 16 || len(deviceID) > 256 {
		return "", ErrInvalidDeviceID
	}
	digest := sha256.Sum256([]byte("tsc-free-research-v1:" + deviceID))
	return hex.EncodeToString(digest[:]), nil
}

// ConsumeAIResearch spends either the one-lifetime user+device trial or one
// active subscription allowance. Retries with the same request/scope are free.
func (s *SubscriptionService) ConsumeAIResearch(ctx context.Context, userID uuid.UUID, deviceID, requestKey, scope string) (Consumption, error) {
	requestKey, scope, err := normalizeRequest(requestKey, scope)
	if err != nil {
		return Consumption{}, err
	}
	if s == nil || s.db == nil {
		return Consumption{}, ErrBillingDisabled
	}
	now := s.now()
	var result Consumption
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var consumeErr error
		result, consumeErr = s.consumeAIResearchTx(ctx, tx, userID, deviceID, requestKey, scope, now, false)
		return consumeErr
	})
	return result, err
}

func (s *SubscriptionService) consumeAIResearchTx(
	ctx context.Context,
	tx *gorm.DB,
	userID uuid.UUID,
	deviceID, requestKey, scope string,
	now time.Time,
	requireAttestation bool,
) (Consumption, error) {
	ent, active := s.lockedEntitlement(ctx, tx, userID, now)
	if active {
		return s.consumePaid(ctx, tx, userID, ent, FeatureAIResearch, requestKey, scope, now)
	}
	if requireAttestation {
		if err := s.requireFreeAuthorizationTx(ctx, tx, userID, deviceID, requestKey, scope); err != nil {
			return Consumption{}, err
		}
	}
	digest, digestErr := deviceDigest(deviceID)
	if digestErr != nil {
		return Consumption{}, digestErr
	}
	var existing models.FreeResearchClaim
	if findErr := tx.WithContext(ctx).Where("request_key = ?", requestKey).First(&existing).Error; findErr == nil {
		if existing.UserID != nil && *existing.UserID == userID && existing.DeviceHash == digest && existing.Scope == scope {
			return Consumption{Plan: freePlan, Limit: 1, Used: 1, Remaining: 0, FreeTrial: true}, nil
		}
		return Consumption{}, ErrRequestConflict
	} else if !errors.Is(findErr, gorm.ErrRecordNotFound) {
		return Consumption{}, findErr
	}
	var count int64
	if err := tx.WithContext(ctx).Model(&models.FreeResearchClaim{}).
		Where("user_id = ? OR device_hash = ?", userID, digest).Count(&count).Error; err != nil {
		return Consumption{}, err
	}
	if count > 0 {
		return Consumption{}, ErrFreeResearchUnavailable
	}
	claim := models.FreeResearchClaim{UserID: &userID, DeviceHash: digest, RequestKey: requestKey, Scope: scope, ClaimedAt: now}
	created := tx.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&claim)
	if created.Error != nil {
		return Consumption{}, created.Error
	}
	if created.RowsAffected == 0 {
		var winner models.FreeResearchClaim
		if err := tx.WithContext(ctx).Where("request_key = ?", requestKey).First(&winner).Error; err == nil &&
			winner.UserID != nil && *winner.UserID == userID && winner.DeviceHash == digest && winner.Scope == scope {
			return Consumption{Plan: freePlan, Limit: 1, Used: 1, Remaining: 0, FreeTrial: true}, nil
		}
		return Consumption{}, ErrFreeResearchUnavailable
	}
	return Consumption{Plan: freePlan, Limit: 1, Used: 1, Remaining: 0, FreeTrial: true}, nil
}

func (s *SubscriptionService) lockedEntitlement(ctx context.Context, tx *gorm.DB, userID uuid.UUID, now time.Time) (models.SubscriptionEntitlement, bool) {
	var ent models.SubscriptionEntitlement
	err := tx.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).Where("user_id = ?", userID).First(&ent).Error
	if err != nil {
		return models.SubscriptionEntitlement{}, false
	}
	return ent, ent.Status == "active" && ent.RevokedAt == nil && ent.ExpiresAt.After(now)
}

func (s *SubscriptionService) consumePaid(ctx context.Context, tx *gorm.DB, userID uuid.UUID, ent models.SubscriptionEntitlement, feature, requestKey, scope string, now time.Time) (Consumption, error) {
	plan := planForCode(ent.PlanCode)
	limit := limitForFeature(plan.Limits, feature)
	if limit <= 0 {
		return Consumption{}, &QuotaError{Feature: feature, Code: "subscription_required"}
	}
	var existing models.FeatureUsage
	findErr := tx.WithContext(ctx).Where("user_id = ? AND feature = ? AND request_key = ?", userID, feature, requestKey).First(&existing).Error
	if findErr == nil {
		if existing.Scope != scope {
			return Consumption{}, ErrRequestConflict
		}
		periodKey, _ := quotaPeriod(ent, true, now)
		used, err := s.usageCount(ctx, tx, userID, feature, periodKey)
		if err != nil {
			return Consumption{}, err
		}
		return Consumption{Plan: plan, Limit: limit, Used: used, Remaining: max(0, limit-used)}, nil
	}
	if !errors.Is(findErr, gorm.ErrRecordNotFound) {
		return Consumption{}, findErr
	}
	periodKey, _ := quotaPeriod(ent, true, now)
	used, err := s.usageCount(ctx, tx, userID, feature, periodKey)
	if err != nil {
		return Consumption{}, err
	}
	if used >= limit {
		return Consumption{}, &QuotaError{Feature: feature, Limit: limit, Used: used, Code: "quota_exhausted"}
	}
	usage := models.FeatureUsage{UserID: userID, Feature: feature, PeriodKey: periodKey, RequestKey: requestKey, Scope: scope, Quantity: 1, CreatedAt: now}
	if err := tx.WithContext(ctx).Create(&usage).Error; err != nil {
		return Consumption{}, err
	}
	used++
	return Consumption{Plan: plan, Limit: limit, Used: used, Remaining: max(0, limit-used)}, nil
}

func (s *SubscriptionService) ConsumePaidFeature(ctx context.Context, userID uuid.UUID, feature, requestKey, scope string) (Consumption, error) {
	requestKey, scope, err := normalizeRequest(requestKey, scope)
	if err != nil {
		return Consumption{}, err
	}
	if s == nil || s.db == nil {
		return Consumption{}, ErrBillingDisabled
	}
	var result Consumption
	now := s.now()
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		ent, active := s.lockedEntitlement(ctx, tx, userID, now)
		if !active {
			return &QuotaError{Feature: feature, Code: "subscription_required"}
		}
		var consumeErr error
		result, consumeErr = s.consumePaid(ctx, tx, userID, ent, feature, requestKey, scope, now)
		return consumeErr
	})
	return result, err
}

// EnsureCapacity gates durable resources such as watchlist rows, active alerts
// and portfolios. Existing rows remain accessible after a downgrade; only new
// creation is blocked until usage is below the current plan's capacity.
func (s *SubscriptionService) EnsureCapacity(ctx context.Context, userID uuid.UUID, feature string) error {
	if s == nil || s.db == nil {
		return nil
	}
	ent, active := s.activeEntitlement(ctx, s.db, userID, s.now())
	plan := freePlan
	if active {
		plan = planForCode(ent.PlanCode)
	}
	limit := limitForFeature(plan.Limits, feature)
	used, err := s.resourceCount(ctx, s.db, userID, feature)
	if err != nil {
		return err
	}
	if used >= limit {
		return &QuotaError{Feature: feature, Limit: limit, Used: used, Code: "quota_exhausted"}
	}
	return nil
}

func (s *SubscriptionService) resourceCount(ctx context.Context, db *gorm.DB, userID uuid.UUID, feature string) (int, error) {
	var count int64
	query := db.WithContext(ctx)
	switch feature {
	case FeatureWatchlist:
		query = query.Model(&models.WatchlistItem{}).Where("user_id = ?", userID)
	case FeatureActiveAlerts:
		query = query.Model(&models.PriceAlert{}).Where("user_id = ? AND active = ?", userID, true)
	case FeaturePortfolios:
		query = query.Model(&models.Portfolio{}).Where("user_id = ?", userID)
	default:
		return 0, fmt.Errorf("unsupported capacity feature %q", feature)
	}
	if err := query.Count(&count).Error; err != nil {
		return 0, err
	}
	return int(count), nil
}

func (s *SubscriptionService) usageCount(ctx context.Context, db *gorm.DB, userID uuid.UUID, feature, periodKey string) (int, error) {
	var total int64
	err := db.WithContext(ctx).Model(&models.FeatureUsage{}).
		Where("user_id = ? AND feature = ? AND period_key = ?", userID, feature, periodKey).
		Select("COALESCE(SUM(quantity), 0)").Scan(&total).Error
	return int(total), err
}

func (s *SubscriptionService) freeClaimCount(ctx context.Context, db *gorm.DB, userID uuid.UUID, deviceID string) (int, error) {
	var count int64
	deviceHash, _ := deviceDigest(deviceID)
	authorization := db.WithContext(ctx).Model(&models.DeviceCheckRedemption{}).
		Where("status IN ?", []string{deviceRedemptionUpdating, deviceRedemptionVerified})
	if deviceHash != "" {
		authorization = authorization.Where("user_id = ? OR device_hash = ?", userID, deviceHash)
	} else {
		authorization = authorization.Where("user_id = ?", userID)
	}
	if err := authorization.Count(&count).Error; err != nil {
		return 0, err
	}
	if count > 0 {
		return 1, nil
	}
	query := db.WithContext(ctx).Model(&models.FreeResearchClaim{}).Where("user_id = ?", userID)
	if deviceHash != "" {
		query = db.WithContext(ctx).Model(&models.FreeResearchClaim{}).Where("user_id = ? OR device_hash = ?", userID, deviceHash)
	}
	if err := query.Count(&count).Error; err != nil {
		return 0, err
	}
	if count > 0 {
		return 1, nil
	}
	return 0, nil
}

func (s *SubscriptionService) ArchiveResearch(ctx context.Context, userID uuid.UUID, requestKey, code string, payload []byte) error {
	if s == nil || s.db == nil || !json.Valid(payload) {
		return nil
	}
	payload = sanitizePublicResearchPayload(payload)
	ent, active := s.activeEntitlement(ctx, s.db, userID, s.now())
	if !active {
		return nil
	}
	plan := planForCode(ent.PlanCode)
	if plan.Limits.ResearchArchiveDays <= 0 {
		return nil
	}
	now := s.now()
	row := models.MarketResearchArchive{
		UserID: userID, RequestKey: requestKey, Code: strings.ToUpper(code), Payload: string(payload),
		GeneratedAt: now, ExpiresAt: now.AddDate(0, 0, plan.Limits.ResearchArchiveDays),
	}
	return s.db.WithContext(ctx).Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "user_id"}, {Name: "request_key"}}, DoNothing: true}).Create(&row).Error
}

func (s *SubscriptionService) ResearchHistory(ctx context.Context, userID uuid.UUID, limit int) ([]ResearchArchiveItem, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	var rows []models.MarketResearchArchive
	err := s.db.WithContext(ctx).Where("user_id = ? AND expires_at > ?", userID, s.now()).
		Order("generated_at DESC").Limit(limit).Find(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]ResearchArchiveItem, 0, len(rows))
	for _, row := range rows {
		payload := sanitizePublicResearchPayload([]byte(row.Payload))
		if !json.Valid(payload) {
			continue
		}
		out = append(out, ResearchArchiveItem{ID: row.ID, Code: row.Code, Payload: payload, GeneratedAt: row.GeneratedAt, ExpiresAt: row.ExpiresAt})
	}
	return out, nil
}

// sanitizePublicResearchPayload keeps provider implementation details out of
// both newly archived and legacy reports. Unknown analysis fields remain
// intact for forward-compatible clients.
func sanitizePublicResearchPayload(payload []byte) json.RawMessage {
	if !json.Valid(payload) {
		return nil
	}
	var object map[string]json.RawMessage
	if err := json.Unmarshal(payload, &object); err != nil || object == nil {
		return append(json.RawMessage(nil), payload...)
	}
	delete(object, "model")
	delete(object, "provider")
	delete(object, "provider_model")
	clean, err := json.Marshal(object)
	if err != nil {
		return append(json.RawMessage(nil), payload...)
	}
	return clean
}
