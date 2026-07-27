package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// GoldPrice is the persisted history record for each calculated gram-gold price.
type GoldPrice struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	Symbol    string    `gorm:"type:varchar(32);index;not null" json:"symbol"`
	Price     float64   `gorm:"not null" json:"price"`
	MidPrice  float64   `gorm:"not null" json:"mid_price"`
	Spread    float64   `gorm:"not null" json:"spread"`
	XAUUSD    float64   `gorm:"column:xauusd;not null" json:"xauusd"`
	USDTRY    float64   `gorm:"column:usdtry;not null" json:"usdtry"`
	Provider  string    `gorm:"type:varchar(64)" json:"provider"`
	CreatedAt time.Time `gorm:"index;not null" json:"created_at"`
}

// TableName pins the table name.
func (GoldPrice) TableName() string { return "gold_prices" }

// BeforeCreate assigns a UUID when missing.
func (g *GoldPrice) BeforeCreate(_ *gorm.DB) error {
	if g.ID == uuid.Nil {
		g.ID = uuid.New()
	}
	return nil
}

// User is an authenticated API consumer (JWT + bcrypt).
type User struct {
	ID           uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	Email        string     `gorm:"type:varchar(255);uniqueIndex;not null" json:"email"`
	PasswordHash string     `gorm:"type:varchar(255)" json:"-"` // empty for social-login users
	Role         string     `gorm:"type:varchar(32);not null;default:'user'" json:"role"`
	LastLoginAt  *time.Time `json:"last_login_at,omitempty"`
	// Provider identifies how the account was created: "local", "google", "apple".
	Provider    string    `gorm:"type:varchar(32);not null;default:'local'" json:"provider"`
	DisplayName string    `gorm:"type:varchar(255)" json:"display_name"`
	AvatarURL   string    `gorm:"type:varchar(512)" json:"avatar_url"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// TableName pins the table name.
func (User) TableName() string { return "users" }

// BeforeCreate assigns a UUID when missing.
func (u *User) BeforeCreate(_ *gorm.DB) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	return nil
}

// Manager is an operations-console identity. It is intentionally isolated
// from User: managers cannot own portfolios, subscriptions, devices, social
// identities, or any other consumer data.
type Manager struct {
	ID                 uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	Email              string     `gorm:"type:varchar(255);uniqueIndex;not null" json:"email"`
	PasswordHash       string     `gorm:"type:varchar(255);not null" json:"-"`
	DisplayName        string     `gorm:"type:varchar(255);not null" json:"display_name"`
	MustChangePassword bool       `gorm:"not null;default:true" json:"-"`
	PasswordChangedAt  *time.Time `json:"-"`
	LastLoginAt        *time.Time `json:"last_login_at,omitempty"`
	CreatedAt          time.Time  `gorm:"not null" json:"created_at"`
	UpdatedAt          time.Time  `gorm:"not null" json:"updated_at"`
}

func (Manager) TableName() string { return "managers" }

func (m *Manager) BeforeCreate(_ *gorm.DB) error {
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
	return nil
}

// OHLC is an aggregated candlestick over an interval.
type OHLC struct {
	Bucket time.Time `json:"bucket"`
	Open   float64   `json:"open"`
	High   float64   `json:"high"`
	Low    float64   `json:"low"`
	Close  float64   `json:"close"`
	Count  int64     `json:"count"`
}

// PriceStats is a rolling window summary (e.g. 24h) for an instrument.
type PriceStats struct {
	Open  float64 `json:"open"`
	High  float64 `json:"high"`
	Low   float64 `json:"low"`
	Last  float64 `json:"last"`
	Count int64   `json:"count"`
}

// Candle is a pre-aggregated OHLC bar for one instrument at one resolution.
// Charts read these directly (indexed, tiny) instead of aggregating millions of
// raw ticks at request time — this is the hot read path for millisecond charts.
// (code, resolution, bucket) is unique so rollovers upsert idempotently.
type Candle struct {
	Code       string    `gorm:"type:varchar(32);not null;uniqueIndex:uq_candle,priority:1" json:"code"`
	Resolution string    `gorm:"type:varchar(8);not null;uniqueIndex:uq_candle,priority:2" json:"resolution"`
	Bucket     time.Time `gorm:"not null;uniqueIndex:uq_candle,priority:3;index:idx_candle_res_bucket,priority:2" json:"bucket"`
	Open       float64   `gorm:"not null" json:"open"`
	High       float64   `gorm:"not null" json:"high"`
	Low        float64   `gorm:"not null" json:"low"`
	Close      float64   `gorm:"not null" json:"close"`
	Count      int64     `gorm:"not null;default:0" json:"count"`
}

func (Candle) TableName() string { return "candles" }

// PriceTick is the generalized history record for EVERY tracked instrument
// (all gold products + currencies). Feeds charts, indicators and AI features.
type PriceTick struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	Code      string    `gorm:"type:varchar(32);index:idx_code_created,priority:1;not null" json:"code"`
	Kind      string    `gorm:"type:varchar(16);not null" json:"kind"`
	Price     float64   `gorm:"not null" json:"price"`
	Bid       float64   `json:"bid"`
	Ask       float64   `json:"ask"`
	Provider  string    `gorm:"type:varchar(64)" json:"provider"`
	CreatedAt time.Time `gorm:"index:idx_code_created,priority:2;not null" json:"created_at"`
}

func (PriceTick) TableName() string { return "price_ticks" }

func (p *PriceTick) BeforeCreate(_ *gorm.DB) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return nil
}

// WatchlistItem is an instrument a user follows.
type WatchlistItem struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	UserID    uuid.UUID `gorm:"type:uuid;index;not null;uniqueIndex:uq_user_code" json:"user_id"`
	Code      string    `gorm:"type:varchar(32);not null;uniqueIndex:uq_user_code" json:"code"`
	CreatedAt time.Time `json:"created_at"`
}

func (WatchlistItem) TableName() string { return "watchlist_items" }

func (w *WatchlistItem) BeforeCreate(_ *gorm.DB) error {
	if w.ID == uuid.Nil {
		w.ID = uuid.New()
	}
	return nil
}

// AlertDirection enumerates threshold-cross directions.
const (
	AlertAbove = "above"
	AlertBelow = "below"
)

// PriceAlert fires a notification when an instrument crosses a threshold.
type PriceAlert struct {
	ID          uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	UserID      uuid.UUID  `gorm:"type:uuid;index;not null" json:"user_id"`
	Code        string     `gorm:"type:varchar(32);index;not null" json:"code"`
	Direction   string     `gorm:"type:varchar(8);not null" json:"direction"`
	Threshold   float64    `gorm:"not null" json:"threshold"`
	Active      bool       `gorm:"not null;default:true" json:"active"`
	LastFiredAt *time.Time `json:"last_fired_at"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

func (PriceAlert) TableName() string { return "price_alerts" }

func (a *PriceAlert) BeforeCreate(_ *gorm.DB) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	return nil
}

// Notification is a delivered/queued user notification.
type Notification struct {
	ID                  uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	UserID              uuid.UUID  `gorm:"type:uuid;index;not null" json:"user_id"`
	AlertID             *uuid.UUID `gorm:"type:uuid" json:"alert_id,omitempty"`
	CampaignRecipientID *uuid.UUID `gorm:"type:uuid;uniqueIndex" json:"-"`
	Code                string     `gorm:"type:varchar(32)" json:"code"`
	Title               string     `gorm:"type:varchar(128)" json:"title"`
	Body                string     `gorm:"type:varchar(512)" json:"body"`
	DeepLink            string     `gorm:"type:varchar(256)" json:"deep_link,omitempty"`
	Price               float64    `json:"price"`
	ReadAt              *time.Time `json:"read_at"`
	CreatedAt           time.Time  `gorm:"index" json:"created_at"`
}

func (Notification) TableName() string { return "notifications" }

func (n *Notification) BeforeCreate(_ *gorm.DB) error {
	if n.ID == uuid.Nil {
		n.ID = uuid.New()
	}
	return nil
}

// RefreshToken stores a hashed, rotating refresh token for a user session.
type RefreshToken struct {
	ID          uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	UserID      uuid.UUID  `gorm:"type:uuid;index;not null" json:"user_id"`
	TokenHash   string     `gorm:"type:varchar(128);uniqueIndex;not null" json:"-"`
	Platform    string     `gorm:"type:varchar(16);index" json:"-"` // ios | android; independent of push permission
	Locale      string     `gorm:"type:varchar(16)" json:"-"`
	AppVersion  string     `gorm:"type:varchar(32)" json:"-"`
	DeviceModel string     `gorm:"type:varchar(96)" json:"-"`
	ExpiresAt   time.Time  `gorm:"not null" json:"expires_at"`
	RevokedAt   *time.Time `json:"revoked_at"`
	CreatedAt   time.Time  `json:"created_at"`
}

func (RefreshToken) TableName() string { return "refresh_tokens" }

func (r *RefreshToken) BeforeCreate(_ *gorm.DB) error {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	return nil
}

// ManagerSession stores only operations-console refresh sessions. Keeping this
// separate prevents a consumer refresh token from ever being interpreted as a
// manager session, even if identifiers collide.
type ManagerSession struct {
	ID        uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	ManagerID uuid.UUID  `gorm:"type:uuid;index;not null" json:"manager_id"`
	TokenHash string     `gorm:"type:varchar(128);uniqueIndex;not null" json:"-"`
	ExpiresAt time.Time  `gorm:"not null;index" json:"expires_at"`
	RevokedAt *time.Time `json:"revoked_at,omitempty"`
	CreatedAt time.Time  `gorm:"not null" json:"created_at"`
}

func (ManagerSession) TableName() string { return "manager_sessions" }

func (s *ManagerSession) BeforeCreate(_ *gorm.DB) error {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	return nil
}

// DeviceToken stores a push token (APNs/FCM) for a user's device.
type DeviceToken struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	UserID      uuid.UUID `gorm:"type:uuid;index;not null" json:"user_id"`
	Platform    string    `gorm:"type:varchar(16);not null;index" json:"platform"` // ios | android
	Token       string    `gorm:"type:varchar(256);uniqueIndex;not null" json:"token"`
	Locale      string    `gorm:"type:varchar(16)" json:"locale,omitempty"`
	AppVersion  string    `gorm:"type:varchar(32)" json:"app_version,omitempty"`
	DeviceModel string    `gorm:"type:varchar(96)" json:"device_model,omitempty"`
	LastSeenAt  time.Time `gorm:"index" json:"last_seen_at"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (DeviceToken) TableName() string { return "device_tokens" }

func (d *DeviceToken) BeforeCreate(_ *gorm.DB) error {
	if d.ID == uuid.Nil {
		d.ID = uuid.New()
	}
	if d.LastSeenAt.IsZero() {
		d.LastSeenAt = time.Now().UTC()
	}
	return nil
}

// JSONDocument stores structured, validated campaign/audit data as JSONB in
// Postgres while remaining compatible with the SQLite test suite.
type JSONDocument []byte

func (j JSONDocument) Value() (driver.Value, error) {
	if len(j) == 0 {
		return "{}", nil
	}
	if !json.Valid(j) {
		return nil, fmt.Errorf("invalid JSON document")
	}
	return string(j), nil
}

func (j *JSONDocument) Scan(value any) error {
	switch v := value.(type) {
	case nil:
		*j = JSONDocument(`{}`)
	case []byte:
		*j = append((*j)[:0], v...)
	case string:
		*j = append((*j)[:0], v...)
	default:
		return fmt.Errorf("unsupported JSON document type %T", value)
	}
	return nil
}

func (j JSONDocument) MarshalJSON() ([]byte, error) {
	if len(j) == 0 {
		return []byte("{}"), nil
	}
	if !json.Valid(j) {
		return nil, fmt.Errorf("invalid JSON document")
	}
	return j, nil
}

// AuthIdentity links an external provider identity (Google/Apple subject) to a
// local user, so one account can have MULTIPLE providers bound to it (e.g. sign
// in with Google, then link Apple — including Apple private-relay emails).
type AuthIdentity struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	UserID    uuid.UUID `gorm:"type:uuid;index;not null" json:"user_id"`
	Provider  string    `gorm:"type:varchar(32);not null;uniqueIndex:idx_provider_subject" json:"provider"`
	Subject   string    `gorm:"type:varchar(255);not null;uniqueIndex:idx_provider_subject" json:"subject"`
	Email     string    `gorm:"type:varchar(255)" json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

func (AuthIdentity) TableName() string { return "auth_identities" }

func (a *AuthIdentity) BeforeCreate(_ *gorm.DB) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	return nil
}

// WebAuthnCredential stores a registered passkey for a user. CredentialID and
// PublicKey are the WebAuthn credential bytes; SignCount guards against cloned
// authenticators.
type WebAuthnCredential struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	UserID       uuid.UUID `gorm:"type:uuid;index;not null" json:"user_id"`
	CredentialID []byte    `gorm:"type:bytea;uniqueIndex;not null" json:"-"`
	PublicKey    []byte    `gorm:"type:bytea;not null" json:"-"`
	AAGUID       []byte    `gorm:"type:bytea" json:"-"`
	SignCount    uint32    `gorm:"not null;default:0" json:"-"`
	Transports   string    `gorm:"type:varchar(128)" json:"-"` // comma-separated
	// WebAuthn credential flags. BackupEligible/BackupState MUST be persisted and
	// replayed at login: synced (iCloud/Google) passkeys set BE=1, and go-webauthn
	// rejects the assertion if the stored BE differs from the response ("Backup
	// Eligible flag inconsistency").
	BackupEligible bool `gorm:"not null;default:false" json:"-"`
	BackupState    bool `gorm:"not null;default:false" json:"-"`
	// Name is a user-editable label for the passkey (e.g. "iPhone'um").
	Name      string    `gorm:"type:varchar(80)" json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

func (WebAuthnCredential) TableName() string { return "webauthn_credentials" }

func (w *WebAuthnCredential) BeforeCreate(_ *gorm.DB) error {
	if w.ID == uuid.Nil {
		w.ID = uuid.New()
	}
	return nil
}

// UserGamification is the per-user engagement state that powers streaks, XP and
// levels. One row per user (UserID is the primary key). LastCheckInDay is the
// Istanbul calendar day ("2006-01-02") of the last daily check-in, so streaks
// use a stable local-midnight boundary.
type UserGamification struct {
	UserID         uuid.UUID `gorm:"type:uuid;primaryKey" json:"user_id"`
	XP             int64     `gorm:"not null;default:0" json:"xp"`
	Level          int       `gorm:"not null;default:1" json:"level"`
	CurrentStreak  int       `gorm:"not null;default:0" json:"current_streak"`
	LongestStreak  int       `gorm:"not null;default:0" json:"longest_streak"`
	LastCheckInDay string    `gorm:"type:varchar(10)" json:"last_check_in_day"`
	TotalCheckIns  int       `gorm:"not null;default:0" json:"total_check_ins"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

func (UserGamification) TableName() string { return "user_gamification" }

// UserBadge is an earned achievement (one row per user+badge code).
type UserBadge struct {
	ID       uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	UserID   uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:uq_user_badge" json:"user_id"`
	Code     string    `gorm:"type:varchar(48);not null;uniqueIndex:uq_user_badge" json:"code"`
	EarnedAt time.Time `json:"earned_at"`
}

func (UserBadge) TableName() string { return "user_badges" }

func (b *UserBadge) BeforeCreate(_ *gorm.DB) error {
	if b.ID == uuid.Nil {
		b.ID = uuid.New()
	}
	return nil
}

// GamificationEvent is an XP ledger entry. It drives daily quests (today's
// events) and the weekly leaderboard (sum of XP since the ISO week start).
type GamificationEvent struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	UserID      uuid.UUID `gorm:"type:uuid;index:idx_gam_user_created,priority:1;not null" json:"user_id"`
	Type        string    `gorm:"type:varchar(48);not null" json:"type"`
	XP          int       `gorm:"not null;default:0" json:"xp"`
	DayKey      string    `gorm:"type:varchar(10);index" json:"day_key"`
	SourceType  string    `gorm:"type:varchar(32);index" json:"source_type"`
	SourceID    string    `gorm:"type:varchar(96);index" json:"source_id"`
	RuleVersion string    `gorm:"type:varchar(24);not null;default:'2026.1'" json:"rule_version"`
	CreatedAt   time.Time `gorm:"index:idx_gam_user_created,priority:2;not null" json:"created_at"`
}

func (GamificationEvent) TableName() string { return "gamification_events" }

func (e *GamificationEvent) BeforeCreate(_ *gorm.DB) error {
	if e.ID == uuid.Nil {
		e.ID = uuid.New()
	}
	return nil
}

// XPLedgerEntry is the immutable, idempotent source of truth for every XP
// mutation. Projections such as UserGamification and SeasonScore are updated in
// the same DB transaction and can be rebuilt from this ledger.
type XPLedgerEntry struct {
	ID             uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	UserID         uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:uq_xp_ledger_user_key,priority:1;index:idx_xp_ledger_user_created,priority:1" json:"user_id"`
	IdempotencyKey string    `gorm:"type:varchar(120);not null;uniqueIndex:uq_xp_ledger_user_key,priority:2" json:"idempotency_key"`
	EventType      string    `gorm:"type:varchar(48);not null;index" json:"event_type"`
	SourceType     string    `gorm:"type:varchar(32);index" json:"source_type"`
	SourceID       string    `gorm:"type:varchar(96);index" json:"source_id"`
	Reason         string    `gorm:"type:varchar(160);not null" json:"reason"`
	RuleVersion    string    `gorm:"type:varchar(24);not null" json:"rule_version"`
	Delta          int       `gorm:"not null" json:"delta"`
	DayKey         string    `gorm:"type:varchar(10);index" json:"day_key"`
	CreatedAt      time.Time `gorm:"index:idx_xp_ledger_user_created,priority:2;not null" json:"created_at"`
}

func (XPLedgerEntry) TableName() string { return "xp_ledger" }

func (e *XPLedgerEntry) BeforeCreate(_ *gorm.DB) error {
	if e.ID == uuid.Nil {
		e.ID = uuid.New()
	}
	if e.CreatedAt.IsZero() {
		e.CreatedAt = time.Now().UTC()
	}
	return nil
}

// DailyPlan turns check-in into a short planning ritual. WatchCodes stores a
// comma-separated canonical list (maximum three) to remain portable across
// Postgres and SQLite tests.
type DailyPlan struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	UserID       uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:uq_daily_plan_user_day,priority:1;index" json:"user_id"`
	DayKey       string    `gorm:"type:varchar(10);not null;uniqueIndex:uq_daily_plan_user_day,priority:2" json:"day_key"`
	RiskAppetite string    `gorm:"type:varchar(16);not null" json:"risk_appetite"`
	WatchCodes   string    `gorm:"type:varchar(128)" json:"-"`
	LearningGoal string    `gorm:"type:varchar(64);not null" json:"learning_goal"`
	Reflection   string    `gorm:"type:varchar(500)" json:"reflection"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (DailyPlan) TableName() string { return "daily_plans" }

func (p *DailyPlan) BeforeCreate(_ *gorm.DB) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return nil
}

type LearningProgress struct {
	ID           uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	UserID       uuid.UUID  `gorm:"type:uuid;not null;uniqueIndex:uq_learning_progress_user_lesson,priority:1;index" json:"user_id"`
	LessonCode   string     `gorm:"type:varchar(48);not null;uniqueIndex:uq_learning_progress_user_lesson,priority:2" json:"lesson_code"`
	Mastery      int        `gorm:"not null;default:0" json:"mastery"`
	Attempts     int        `gorm:"not null;default:0" json:"attempts"`
	Correct      int        `gorm:"not null;default:0" json:"correct"`
	CompletedAt  *time.Time `json:"completed_at,omitempty"`
	NextReviewAt *time.Time `gorm:"index" json:"next_review_at,omitempty"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

func (LearningProgress) TableName() string { return "learning_progress" }

func (p *LearningProgress) BeforeCreate(_ *gorm.DB) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return nil
}

type LearningAttempt struct {
	ID             uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	UserID         uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:uq_learning_attempt_user_key,priority:1;index" json:"user_id"`
	IdempotencyKey string    `gorm:"type:varchar(120);not null;uniqueIndex:uq_learning_attempt_user_key,priority:2" json:"idempotency_key"`
	LessonCode     string    `gorm:"type:varchar(48);not null;index" json:"lesson_code"`
	QuestionCode   string    `gorm:"type:varchar(64);not null" json:"question_code"`
	Answer         int       `gorm:"not null" json:"answer"`
	Correct        bool      `gorm:"not null" json:"correct"`
	XPAwarded      int       `gorm:"not null;default:0" json:"xp_awarded"`
	CreatedAt      time.Time `gorm:"index;not null" json:"created_at"`
}

func (LearningAttempt) TableName() string { return "learning_attempts" }

func (a *LearningAttempt) BeforeCreate(_ *gorm.DB) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	if a.CreatedAt.IsZero() {
		a.CreatedAt = time.Now().UTC()
	}
	return nil
}

type DisciplineReview struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	UserID    uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:uq_discipline_review_user_key,priority:1;index" json:"user_id"`
	DayKey    string    `gorm:"type:varchar(10);not null;uniqueIndex:uq_discipline_review_user_key,priority:2" json:"day_key"`
	Code      string    `gorm:"type:varchar(48);not null;uniqueIndex:uq_discipline_review_user_key,priority:3" json:"code"`
	Note      string    `gorm:"type:varchar(1000);not null" json:"note"`
	CreatedAt time.Time `json:"created_at"`
}

func (DisciplineReview) TableName() string { return "discipline_reviews" }

func (r *DisciplineReview) BeforeCreate(_ *gorm.DB) error {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	return nil
}

type SeasonScore struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	UserID     uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:uq_season_score_user,priority:1;index" json:"user_id"`
	SeasonCode string    `gorm:"type:varchar(32);not null;uniqueIndex:uq_season_score_user,priority:2;index" json:"season_code"`
	Points     int64     `gorm:"not null;default:0;index" json:"points"`
	League     string    `gorm:"type:varchar(24);not null;default:'Başlangıç'" json:"league"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func (SeasonScore) TableName() string { return "season_scores" }

func (s *SeasonScore) BeforeCreate(_ *gorm.DB) error {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	return nil
}

// PaperTrade records the decision process separately from the real portfolio.
// ProcessScore never depends on return; it rewards thesis/risk planning and a
// completed retrospective.
type PaperTrade struct {
	ID            uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	UserID        uuid.UUID  `gorm:"type:uuid;not null;uniqueIndex:uq_paper_trade_user_client,priority:1;index" json:"user_id"`
	ClientID      string     `gorm:"type:varchar(80);not null;uniqueIndex:uq_paper_trade_user_client,priority:2" json:"client_id"`
	Code          string     `gorm:"type:varchar(32);not null;index" json:"code"`
	Side          string     `gorm:"type:varchar(8);not null" json:"side"`
	Quantity      float64    `gorm:"not null" json:"quantity"`
	EntryPrice    float64    `gorm:"not null" json:"entry_price"`
	Thesis        string     `gorm:"type:varchar(1000);not null" json:"thesis"`
	RiskBudgetPct float64    `gorm:"not null" json:"risk_budget_pct"`
	Invalidation  string     `gorm:"type:varchar(500);not null" json:"invalidation"`
	OccurredAt    time.Time  `gorm:"index;not null" json:"occurred_at"`
	FollowedPlan  *bool      `json:"followed_plan,omitempty"`
	Retrospective string     `gorm:"type:varchar(1000)" json:"retrospective"`
	ProcessScore  int        `gorm:"not null;default:0" json:"process_score"`
	ReviewedAt    *time.Time `json:"reviewed_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

func (PaperTrade) TableName() string { return "paper_trades" }

func (p *PaperTrade) BeforeCreate(_ *gorm.DB) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return nil
}

type GamificationPrivacy struct {
	UserID                uuid.UUID `gorm:"type:uuid;primaryKey" json:"user_id"`
	Alias                 string    `gorm:"type:varchar(40)" json:"alias"`
	LeagueOptIn           bool      `gorm:"not null;default:true" json:"league_opt_in"`
	PersonalizedLearning  bool      `gorm:"not null;default:true" json:"personalized_learning"`
	SocialOptIn           bool      `gorm:"not null;default:false" json:"social_opt_in"`
	LearningNotifications bool      `gorm:"not null;default:true" json:"learning_notifications"`
	UpdatedAt             time.Time `json:"updated_at"`
}

func (GamificationPrivacy) TableName() string { return "gamification_privacy" }

type StudyGroup struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	OwnerID    uuid.UUID `gorm:"type:uuid;not null;index" json:"owner_id"`
	Name       string    `gorm:"type:varchar(60);not null" json:"name"`
	InviteCode string    `gorm:"type:varchar(12);not null;uniqueIndex" json:"invite_code"`
	CreatedAt  time.Time `json:"created_at"`
}

func (StudyGroup) TableName() string { return "study_groups" }

func (g *StudyGroup) BeforeCreate(_ *gorm.DB) error {
	if g.ID == uuid.Nil {
		g.ID = uuid.New()
	}
	return nil
}

type StudyGroupMember struct {
	ID       uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	GroupID  uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:uq_study_group_member,priority:1;index" json:"group_id"`
	UserID   uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:uq_study_group_member,priority:2;index" json:"user_id"`
	Alias    string    `gorm:"type:varchar(40);not null" json:"alias"`
	JoinedAt time.Time `json:"joined_at"`
}

func (StudyGroupMember) TableName() string { return "study_group_members" }

func (m *StudyGroupMember) BeforeCreate(_ *gorm.DB) error {
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
	if m.JoinedAt.IsZero() {
		m.JoinedAt = time.Now().UTC()
	}
	return nil
}

// Portfolio is a user-owned, cloud-synchronised investment account. Version is
// incremented on every ledger mutation and lets clients invalidate offline
// caches/outboxes without relying on wall-clock timestamps.
type Portfolio struct {
	ID             uuid.UUID       `gorm:"type:uuid;primaryKey" json:"id"`
	UserID         uuid.UUID       `gorm:"type:uuid;not null;index:idx_portfolios_user_updated,priority:1" json:"user_id"`
	Name           string          `gorm:"type:varchar(80);not null" json:"name"`
	BaseCurrency   string          `gorm:"type:varchar(3);not null;default:'TRY'" json:"base_currency"`
	NetDepositsTRY decimal.Decimal `gorm:"type:numeric(36,12);not null;default:0" json:"net_deposits_try"`
	Version        int64           `gorm:"not null;default:1" json:"version"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `gorm:"index:idx_portfolios_user_updated,priority:2" json:"updated_at"`
}

func (Portfolio) TableName() string { return "portfolios" }

func (p *Portfolio) BeforeCreate(_ *gorm.DB) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	if p.BaseCurrency == "" {
		p.BaseCurrency = "TRY"
	}
	if p.Version == 0 {
		p.Version = 1
	}
	return nil
}

// PortfolioPosition is the materialized projection of the immutable-style
// transaction ledger. It is rebuilt inside the same DB transaction whenever
// the ledger changes, so reads stay fast while edits/deletes remain correct.
type PortfolioPosition struct {
	ID                uuid.UUID       `gorm:"type:uuid;primaryKey" json:"id"`
	PortfolioID       uuid.UUID       `gorm:"type:uuid;not null;uniqueIndex:uq_portfolio_position,priority:1;index" json:"portfolio_id"`
	Code              string          `gorm:"type:varchar(32);not null;uniqueIndex:uq_portfolio_position,priority:2" json:"code"`
	Currency          string          `gorm:"type:varchar(3);not null" json:"currency"`
	Quantity          decimal.Decimal `gorm:"type:numeric(36,18);not null;default:0" json:"quantity"`
	AverageCostNative decimal.Decimal `gorm:"type:numeric(36,12);not null;default:0" json:"average_cost_native"`
	AverageCostTRY    decimal.Decimal `gorm:"type:numeric(36,12);not null;default:0" json:"average_cost_try"`
	RealizedPnLTRY    decimal.Decimal `gorm:"type:numeric(36,12);not null;default:0" json:"realized_pnl_try"`
	CreatedAt         time.Time       `json:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at"`
}

func (PortfolioPosition) TableName() string { return "portfolio_positions" }

func (p *PortfolioPosition) BeforeCreate(_ *gorm.DB) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return nil
}

// PortfolioTransaction is the source of truth for holdings and P&L. ClientID
// is generated by mobile/web outboxes and unique per portfolio, making retries
// after offline/network failures exactly-once from the user's perspective.
type PortfolioTransaction struct {
	ID              uuid.UUID       `gorm:"type:uuid;primaryKey" json:"id"`
	PortfolioID     uuid.UUID       `gorm:"type:uuid;not null;index:idx_portfolio_tx_time,priority:1;uniqueIndex:uq_portfolio_client,priority:1" json:"portfolio_id"`
	UserID          uuid.UUID       `gorm:"type:uuid;not null;index" json:"user_id"`
	ClientID        string          `gorm:"type:varchar(80);not null;uniqueIndex:uq_portfolio_client,priority:2" json:"client_id"`
	Code            string          `gorm:"type:varchar(32);index" json:"code"`
	Type            string          `gorm:"type:varchar(16);not null" json:"type"`
	Quantity        decimal.Decimal `gorm:"type:numeric(36,18);not null;default:0" json:"quantity"`
	UnitPriceNative decimal.Decimal `gorm:"type:numeric(36,12);not null;default:0" json:"unit_price_native"`
	AmountNative    decimal.Decimal `gorm:"type:numeric(36,12);not null;default:0" json:"amount_native"`
	FeeNative       decimal.Decimal `gorm:"type:numeric(36,12);not null;default:0" json:"fee_native"`
	Currency        string          `gorm:"type:varchar(3);not null" json:"currency"`
	USDTRY          decimal.Decimal `gorm:"type:numeric(24,12);not null;default:1" json:"usdtry"`
	OccurredAt      time.Time       `gorm:"not null;index:idx_portfolio_tx_time,priority:2" json:"occurred_at"`
	Note            string          `gorm:"type:varchar(500)" json:"note"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

func (PortfolioTransaction) TableName() string { return "portfolio_transactions" }

func (t *PortfolioTransaction) BeforeCreate(_ *gorm.DB) error {
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}
	return nil
}

// SubscriptionEntitlement is the server-authoritative snapshot of a user's
// auto-renewable store subscription. The client never grants itself a plan:
// this row is only updated after Apple JWS or Google Play purchase verification.
type SubscriptionEntitlement struct {
	UserID                uuid.UUID  `gorm:"type:uuid;primaryKey" json:"user_id"`
	ProductID             string     `gorm:"type:varchar(160);not null" json:"product_id"`
	PlanCode              string     `gorm:"type:varchar(32);not null;index" json:"plan_code"`
	Provider              string     `gorm:"type:varchar(24);not null;default:apple;index" json:"provider"`
	OriginalTransactionID string     `gorm:"type:varchar(96);not null;uniqueIndex" json:"original_transaction_id"`
	LatestTransactionID   string     `gorm:"type:varchar(96);not null" json:"latest_transaction_id"`
	Environment           string     `gorm:"type:varchar(24);not null" json:"environment"`
	Status                string     `gorm:"type:varchar(24);not null;index" json:"status"`
	PeriodStart           time.Time  `gorm:"not null" json:"period_start"`
	ExpiresAt             time.Time  `gorm:"not null;index" json:"expires_at"`
	RevokedAt             *time.Time `json:"revoked_at,omitempty"`
	CreatedAt             time.Time  `json:"created_at"`
	UpdatedAt             time.Time  `json:"updated_at"`
}

func (SubscriptionEntitlement) TableName() string { return "subscription_entitlements" }

// AppleTransaction is an immutable audit/fraud record for a verified StoreKit
// transaction. UserID is nullable so account deletion can remove the personal
// link while retaining the minimum transaction record needed for restore,
// refunds and transaction-sharing prevention.
type AppleTransaction struct {
	TransactionID         string     `gorm:"type:varchar(96);primaryKey" json:"transaction_id"`
	OriginalTransactionID string     `gorm:"type:varchar(96);not null;index" json:"original_transaction_id"`
	UserID                *uuid.UUID `gorm:"type:uuid;index" json:"user_id,omitempty"`
	ProductID             string     `gorm:"type:varchar(160);not null" json:"product_id"`
	AppAccountToken       string     `gorm:"type:varchar(64)" json:"app_account_token,omitempty"`
	Environment           string     `gorm:"type:varchar(24);not null" json:"environment"`
	PurchasedAt           time.Time  `gorm:"not null" json:"purchased_at"`
	ExpiresAt             time.Time  `gorm:"not null;index" json:"expires_at"`
	RevokedAt             *time.Time `json:"revoked_at,omitempty"`
	JWSSHA256             string     `gorm:"column:jws_sha256;type:char(64);not null" json:"-"`
	CreatedAt             time.Time  `json:"created_at"`
	UpdatedAt             time.Time  `json:"updated_at"`
}

func (AppleTransaction) TableName() string { return "apple_transactions" }

// GooglePlayTransaction is an audit/fraud record for a verified Play Billing
// subscription purchase. UserID is nullable so account deletion can remove the
// personal link while retaining the purchase-token shell for restores/refunds.
type GooglePlayTransaction struct {
	// Purchase tokens from Play can exceed 255 chars; keep them as text.
	PurchaseToken       string     `gorm:"type:text;primaryKey" json:"purchase_token"`
	OrderID             string     `gorm:"type:varchar(160);not null;index" json:"order_id"`
	LinkedPurchaseToken string     `gorm:"type:text;index" json:"linked_purchase_token,omitempty"`
	UserID              *uuid.UUID `gorm:"type:uuid;index" json:"user_id,omitempty"`
	PackageName         string     `gorm:"type:varchar(160);not null" json:"package_name"`
	ProductID           string     `gorm:"type:varchar(160);not null" json:"product_id"`
	ObfuscatedAccountID string     `gorm:"type:varchar(64)" json:"obfuscated_account_id,omitempty"`
	Environment         string     `gorm:"type:varchar(24);not null" json:"environment"`
	PurchasedAt         time.Time  `gorm:"not null" json:"purchased_at"`
	ExpiresAt           time.Time  `gorm:"not null;index" json:"expires_at"`
	RevokedAt           *time.Time `json:"revoked_at,omitempty"`
	Acknowledged        bool       `gorm:"not null;default:false" json:"acknowledged"`
	RawSHA256           string     `gorm:"column:raw_sha256;type:char(64);not null" json:"-"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
}

func (GooglePlayTransaction) TableName() string { return "google_play_transactions" }

// FeatureUsage is an idempotent monthly quota ledger. RequestKey is generated
// once by the client and Scope identifies the concrete resource (for example
// the researched instrument), preventing request-key reuse across operations.
type FeatureUsage struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	UserID     uuid.UUID `gorm:"type:uuid;not null;index:idx_feature_usage_period,priority:1;uniqueIndex:uq_feature_request,priority:1" json:"user_id"`
	Feature    string    `gorm:"type:varchar(48);not null;index:idx_feature_usage_period,priority:2;uniqueIndex:uq_feature_request,priority:2" json:"feature"`
	PeriodKey  string    `gorm:"type:varchar(48);not null;index:idx_feature_usage_period,priority:3" json:"period_key"`
	RequestKey string    `gorm:"type:varchar(80);not null;uniqueIndex:uq_feature_request,priority:3" json:"request_key"`
	Scope      string    `gorm:"type:varchar(160);not null" json:"scope"`
	Quantity   int       `gorm:"not null;default:1" json:"quantity"`
	CreatedAt  time.Time `json:"created_at"`
}

func (FeatureUsage) TableName() string { return "feature_usages" }

func (u *FeatureUsage) BeforeCreate(_ *gorm.DB) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	return nil
}

// FreeResearchClaim enforces the one-lifetime deep-research allowance on both
// account and device. DeviceHash is a one-way SHA-256 digest of an app-generated
// Keychain identifier; no hardware identifier or IDFA is collected.
type FreeResearchClaim struct {
	ID               uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	UserID           *uuid.UUID `gorm:"type:uuid;uniqueIndex" json:"user_id,omitempty"`
	DeviceHash       string     `gorm:"type:char(64);not null;uniqueIndex" json:"-"`
	RequestKey       string     `gorm:"type:varchar(80);not null;uniqueIndex" json:"request_key"`
	Scope            string     `gorm:"type:varchar(160);not null" json:"scope"`
	ClaimedAt        time.Time  `gorm:"not null" json:"claimed_at"`
	AccountDeletedAt *time.Time `json:"account_deleted_at,omitempty"`
}

func (FreeResearchClaim) TableName() string { return "free_research_claims" }

func (c *FreeResearchClaim) BeforeCreate(_ *gorm.DB) error {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	return nil
}

// MarketResearchArchive stores the structured research response for the
// retention window included in the active plan. Payload is JSON generated by
// this backend and never rendered as executable HTML.
type MarketResearchArchive struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	UserID      uuid.UUID `gorm:"type:uuid;not null;index:idx_research_archive_user_generated,priority:1;uniqueIndex:uq_research_archive_request,priority:1" json:"user_id"`
	RequestKey  string    `gorm:"type:varchar(80);not null;uniqueIndex:uq_research_archive_request,priority:2" json:"request_key"`
	Code        string    `gorm:"type:varchar(32);not null" json:"code"`
	Payload     string    `gorm:"type:jsonb;not null" json:"payload"`
	GeneratedAt time.Time `gorm:"not null;index:idx_research_archive_user_generated,priority:2" json:"generated_at"`
	ExpiresAt   time.Time `gorm:"not null;index" json:"expires_at"`
}

func (MarketResearchArchive) TableName() string { return "market_research_archives" }

func (a *MarketResearchArchive) BeforeCreate(_ *gorm.DB) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	return nil
}

// MarketResearchJob is the durable idempotency record for one deep-research
// request. The unique (user, request) key means concurrent HTTP retries share
// one provider job and one quota reservation. OwnerTokenHash is an unguessable
// capability held only by the request that created the row; it is required to
// refund a provider job that could not be started.
type MarketResearchJob struct {
	ID                 uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	UserID             uuid.UUID  `gorm:"type:uuid;not null;uniqueIndex:uq_research_job_request,priority:1;index" json:"user_id"`
	RequestKey         string     `gorm:"type:varchar(80);not null;uniqueIndex:uq_research_job_request,priority:2" json:"request_key"`
	Code               string     `gorm:"type:varchar(32);not null" json:"code"`
	Status             string     `gorm:"type:varchar(24);not null;index" json:"status"`
	ProviderResponseID string     `gorm:"type:varchar(160);index" json:"provider_response_id,omitempty"`
	ResultPayload      string     `gorm:"type:text" json:"result_payload,omitempty"`
	OwnerTokenHash     string     `gorm:"type:char(64);not null" json:"-"`
	LeaseExpiresAt     *time.Time `gorm:"index" json:"lease_expires_at,omitempty"`
	AttemptCount       int        `gorm:"not null;default:0" json:"attempt_count"`
	FailureCode        string     `gorm:"type:varchar(64)" json:"failure_code,omitempty"`
	CreatedAt          time.Time  `gorm:"not null" json:"created_at"`
	UpdatedAt          time.Time  `gorm:"not null" json:"updated_at"`
	CompletedAt        *time.Time `json:"completed_at,omitempty"`
}

func (MarketResearchJob) TableName() string { return "market_research_jobs" }

func (j *MarketResearchJob) BeforeCreate(_ *gorm.DB) error {
	if j.ID == uuid.Nil {
		j.ID = uuid.New()
	}
	return nil
}

// FeatureArtifact persists the output of quota-metered deterministic features
// (portfolio health reports and exports). An idempotent replay returns the
// original bytes instead of recomputing against newer portfolio state.
type FeatureArtifact struct {
	ID             uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	UserID         uuid.UUID  `gorm:"type:uuid;not null;uniqueIndex:uq_feature_artifact_request,priority:1;index" json:"user_id"`
	Feature        string     `gorm:"type:varchar(48);not null;uniqueIndex:uq_feature_artifact_request,priority:2" json:"feature"`
	RequestKey     string     `gorm:"type:varchar(80);not null;uniqueIndex:uq_feature_artifact_request,priority:3" json:"request_key"`
	Scope          string     `gorm:"type:varchar(160);not null" json:"scope"`
	Status         string     `gorm:"type:varchar(24);not null" json:"status"`
	Payload        []byte     `gorm:"type:bytea" json:"-"`
	ContentType    string     `gorm:"type:varchar(96)" json:"content_type,omitempty"`
	Filename       string     `gorm:"type:varchar(180)" json:"filename,omitempty"`
	OwnerTokenHash string     `gorm:"type:char(64);not null" json:"-"`
	LeaseExpiresAt *time.Time `gorm:"index" json:"lease_expires_at,omitempty"`
	AttemptCount   int        `gorm:"not null;default:0" json:"attempt_count"`
	CreatedAt      time.Time  `gorm:"not null" json:"created_at"`
	UpdatedAt      time.Time  `gorm:"not null" json:"updated_at"`
}

func (FeatureArtifact) TableName() string { return "feature_artifacts" }

func (a *FeatureArtifact) BeforeCreate(_ *gorm.DB) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	return nil
}

// BillingCapacityLock is a stable row locked inside resource-creation
// transactions. It serializes count+insert per user/feature across API pods.
type BillingCapacityLock struct {
	UserID    uuid.UUID `gorm:"type:uuid;primaryKey" json:"user_id"`
	Feature   string    `gorm:"type:varchar(48);primaryKey" json:"feature"`
	UpdatedAt time.Time `gorm:"not null" json:"updated_at"`
}

func (BillingCapacityLock) TableName() string { return "billing_capacity_locks" }

// AppAttestChallenge is a one-use server nonce already bound to the
// authenticated account and concrete research request. Only its SHA-256 hash
// is retained; the random challenge itself is returned once to the app.
type AppAttestChallenge struct {
	ID         uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	UserID     uuid.UUID  `gorm:"type:uuid;not null;index" json:"user_id"`
	DeviceHash string     `gorm:"type:char(64);not null;index" json:"-"`
	RequestKey string     `gorm:"type:varchar(80);not null" json:"request_key"`
	Scope      string     `gorm:"type:varchar(160);not null" json:"scope"`
	Purpose    string     `gorm:"type:varchar(32);not null" json:"purpose"`
	NonceHash  string     `gorm:"type:char(64);not null" json:"-"`
	ClientData []byte     `gorm:"type:bytea;not null" json:"-"`
	ExpiresAt  time.Time  `gorm:"not null;index" json:"expires_at"`
	UsedAt     *time.Time `json:"used_at,omitempty"`
	CreatedAt  time.Time  `gorm:"not null" json:"created_at"`
}

func (AppAttestChallenge) TableName() string { return "app_attest_challenges" }

func (c *AppAttestChallenge) BeforeCreate(_ *gorm.DB) error {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	return nil
}

// AppAttestKey is the Apple-certified P-256 key for one app installation.
// DeviceHash is the privacy-preserving local installation binding while the
// public key and monotonically increasing counter authenticate assertions.
type AppAttestKey struct {
	KeyID            string     `gorm:"type:varchar(128);primaryKey" json:"key_id"`
	DeviceHash       string     `gorm:"type:char(64);not null;uniqueIndex" json:"-"`
	RegisteredUserID *uuid.UUID `gorm:"type:uuid;index" json:"registered_user_id,omitempty"`
	PublicKeyDER     []byte     `gorm:"type:bytea;not null" json:"-"`
	Receipt          []byte     `gorm:"type:bytea;not null" json:"-"`
	Environment      string     `gorm:"type:varchar(24);not null" json:"environment"`
	Counter          uint32     `gorm:"not null;default:0" json:"counter"`
	CreatedAt        time.Time  `gorm:"not null" json:"created_at"`
	UpdatedAt        time.Time  `gorm:"not null" json:"updated_at"`
}

func (AppAttestKey) TableName() string { return "app_attest_keys" }

// DeviceCheckRedemption is the crash-recovery journal around Apple's
// reinstall-resistant DeviceCheck bit. "updating" is committed before the
// irreversible Apple call, so a retry can query bit 0 and reconcile a response
// lost between Apple and the database commit.
type DeviceCheckRedemption struct {
	ID             uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	UserID         uuid.UUID  `gorm:"type:uuid;not null;uniqueIndex:uq_device_check_request,priority:1;index" json:"user_id"`
	RequestKey     string     `gorm:"type:varchar(80);not null;uniqueIndex:uq_device_check_request,priority:2" json:"request_key"`
	DeviceHash     string     `gorm:"type:char(64);not null;index" json:"-"`
	Scope          string     `gorm:"type:varchar(160);not null" json:"scope"`
	TokenHash      string     `gorm:"type:char(64);not null;uniqueIndex" json:"-"`
	Status         string     `gorm:"type:varchar(24);not null;index" json:"status"`
	OwnerTokenHash string     `gorm:"type:char(64)" json:"-"`
	LeaseExpiresAt *time.Time `gorm:"index" json:"lease_expires_at,omitempty"`
	AttemptCount   int        `gorm:"not null;default:0" json:"attempt_count"`
	CreatedAt      time.Time  `gorm:"not null" json:"created_at"`
	UpdatedAt      time.Time  `gorm:"not null" json:"updated_at"`
}

func (DeviceCheckRedemption) TableName() string { return "device_check_redemptions" }

func (r *DeviceCheckRedemption) BeforeCreate(_ *gorm.DB) error {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	return nil
}

// SubscriptionRestoreTombstone retains only one-way identity/token bindings
// after account deletion. A recreated account may restore a still-billed
// subscription only after proving the same Sign in with Apple subject; a
// copied StoreKit transaction alone is insufficient.
type SubscriptionRestoreTombstone struct {
	OriginalTransactionID       string     `gorm:"type:varchar(96);primaryKey" json:"original_transaction_id"`
	PreviousAppAccountTokenHash string     `gorm:"type:char(64);not null" json:"-"`
	IdentityProvider            string     `gorm:"type:varchar(24)" json:"identity_provider,omitempty"`
	ProviderSubjectHash         string     `gorm:"type:char(64)" json:"-"`
	RecoveryTokenHash           string     `gorm:"type:char(64)" json:"-"`
	DeletedAt                   time.Time  `gorm:"not null" json:"deleted_at"`
	ReboundUserID               *uuid.UUID `gorm:"type:uuid;index" json:"rebound_user_id,omitempty"`
	ReboundAt                   *time.Time `json:"rebound_at,omitempty"`
}

func (SubscriptionRestoreTombstone) TableName() string { return "subscription_restore_tombstones" }

// NotificationCampaign is an admin-created, audited push campaign. The AI
// produces only a constrained TargetingPlan JSON document; it never stores or
// executes model-authored SQL.
type NotificationCampaign struct {
	ID                  uuid.UUID    `gorm:"type:uuid;primaryKey" json:"id"`
	ManagerID           uuid.UUID    `gorm:"type:uuid;not null;index" json:"manager_id"`
	Name                string       `gorm:"type:varchar(120);not null" json:"name"`
	Prompt              string       `gorm:"type:text;not null" json:"prompt"`
	Status              string       `gorm:"type:varchar(24);not null;index" json:"status"`
	TargetingPlan       JSONDocument `gorm:"type:jsonb;not null" json:"targeting_plan"`
	Preview             JSONDocument `gorm:"type:jsonb;not null" json:"preview"`
	Title               string       `gorm:"type:varchar(128);not null" json:"title"`
	Body                string       `gorm:"type:varchar(512);not null" json:"body"`
	DeepLink            string       `gorm:"type:varchar(256)" json:"deep_link,omitempty"`
	EstimatedRecipients int          `gorm:"not null;default:0" json:"estimated_recipients"`
	ResolvedRecipients  int          `gorm:"not null;default:0" json:"resolved_recipients"`
	ProcessedRecipients int          `gorm:"not null;default:0" json:"processed_recipients"`
	FailedRecipients    int          `gorm:"not null;default:0" json:"failed_recipients"`
	ApprovalFingerprint string       `gorm:"type:char(64);not null" json:"-"`
	ApprovedAt          *time.Time   `json:"approved_at,omitempty"`
	StartedAt           *time.Time   `json:"started_at,omitempty"`
	CompletedAt         *time.Time   `json:"completed_at,omitempty"`
	CreatedAt           time.Time    `gorm:"not null;index" json:"created_at"`
	UpdatedAt           time.Time    `gorm:"not null" json:"updated_at"`
}

func (NotificationCampaign) TableName() string { return "notification_campaigns" }

func (c *NotificationCampaign) BeforeCreate(_ *gorm.DB) error {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	return nil
}

// NotificationCampaignRecipient is the durable work queue. A unique
// campaign+user key makes approval and worker retries idempotent across pods.
type NotificationCampaignRecipient struct {
	ID              uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	CampaignID      uuid.UUID  `gorm:"type:uuid;not null;uniqueIndex:uq_campaign_recipient,priority:1;index" json:"campaign_id"`
	UserID          uuid.UUID  `gorm:"type:uuid;not null;uniqueIndex:uq_campaign_recipient,priority:2;index" json:"user_id"`
	PlatformSummary string     `gorm:"type:varchar(32)" json:"platform_summary"`
	Status          string     `gorm:"type:varchar(24);not null;index" json:"status"`
	Attempts        int        `gorm:"not null;default:0" json:"attempts"`
	LastError       string     `gorm:"type:varchar(512)" json:"last_error,omitempty"`
	ProcessedAt     *time.Time `json:"processed_at,omitempty"`
	CreatedAt       time.Time  `gorm:"not null" json:"created_at"`
	UpdatedAt       time.Time  `gorm:"not null" json:"updated_at"`
}

func (NotificationCampaignRecipient) TableName() string {
	return "notification_campaign_recipients"
}

func (r *NotificationCampaignRecipient) BeforeCreate(_ *gorm.DB) error {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	return nil
}

// AdminAuditLog is append-only evidence for privileged reads and mutations.
// Metadata deliberately stores structured IDs/counts rather than credentials.
type AdminAuditLog struct {
	ID           uuid.UUID    `gorm:"type:uuid;primaryKey" json:"id"`
	ManagerID    *uuid.UUID   `gorm:"type:uuid;index" json:"manager_id,omitempty"`
	Action       string       `gorm:"type:varchar(80);not null;index" json:"action"`
	ResourceType string       `gorm:"type:varchar(64);not null;index" json:"resource_type"`
	ResourceID   string       `gorm:"type:varchar(128);index" json:"resource_id,omitempty"`
	Metadata     JSONDocument `gorm:"type:jsonb;not null" json:"metadata"`
	IPAddress    string       `gorm:"type:varchar(64)" json:"ip_address,omitempty"`
	UserAgent    string       `gorm:"type:varchar(320)" json:"user_agent,omitempty"`
	CreatedAt    time.Time    `gorm:"not null;index" json:"created_at"`
}

func (AdminAuditLog) TableName() string { return "admin_audit_logs" }

func (a *AdminAuditLog) BeforeCreate(_ *gorm.DB) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	return nil
}
