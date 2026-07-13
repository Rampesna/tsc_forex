package models

import (
	"time"

	"github.com/google/uuid"
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
	ID           uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	Email        string    `gorm:"type:varchar(255);uniqueIndex;not null" json:"email"`
	PasswordHash string    `gorm:"type:varchar(255)" json:"-"` // empty for social-login users
	Role         string    `gorm:"type:varchar(32);not null;default:'user'" json:"role"`
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
	ID        uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	UserID    uuid.UUID  `gorm:"type:uuid;index;not null" json:"user_id"`
	AlertID   *uuid.UUID `gorm:"type:uuid" json:"alert_id,omitempty"`
	Code      string     `gorm:"type:varchar(32)" json:"code"`
	Title     string     `gorm:"type:varchar(128)" json:"title"`
	Body      string     `gorm:"type:varchar(512)" json:"body"`
	Price     float64    `json:"price"`
	ReadAt    *time.Time `json:"read_at"`
	CreatedAt time.Time  `gorm:"index" json:"created_at"`
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
	ID        uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	UserID    uuid.UUID  `gorm:"type:uuid;index;not null" json:"user_id"`
	TokenHash string     `gorm:"type:varchar(128);uniqueIndex;not null" json:"-"`
	ExpiresAt time.Time  `gorm:"not null" json:"expires_at"`
	RevokedAt *time.Time `json:"revoked_at"`
	CreatedAt time.Time  `json:"created_at"`
}

func (RefreshToken) TableName() string { return "refresh_tokens" }

func (r *RefreshToken) BeforeCreate(_ *gorm.DB) error {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	return nil
}

// DeviceToken stores a push token (APNs/FCM) for a user's device.
type DeviceToken struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	UserID    uuid.UUID `gorm:"type:uuid;index;not null" json:"user_id"`
	Platform  string    `gorm:"type:varchar(16);not null" json:"platform"` // ios | android
	Token     string    `gorm:"type:varchar(256);uniqueIndex;not null" json:"token"`
	CreatedAt time.Time `json:"created_at"`
}

func (DeviceToken) TableName() string { return "device_tokens" }

func (d *DeviceToken) BeforeCreate(_ *gorm.DB) error {
	if d.ID == uuid.Nil {
		d.ID = uuid.New()
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
	CreatedAt    time.Time `json:"created_at"`
}

func (WebAuthnCredential) TableName() string { return "webauthn_credentials" }

func (w *WebAuthnCredential) BeforeCreate(_ *gorm.DB) error {
	if w.ID == uuid.Nil {
		w.ID = uuid.New()
	}
	return nil
}
