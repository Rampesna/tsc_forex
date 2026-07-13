package api

import "time"

// LoginRequest is the login payload.
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
}

// RegisterRequest is the registration payload.
type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
}

// TokenResponse carries issued JWT tokens.
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
}

// RefreshRequest carries a refresh token for rotation.
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// GoogleAuthRequest carries a Google ID token obtained by the mobile client
// via the Google Sign-In SDK.
type GoogleAuthRequest struct {
	IDToken string `json:"id_token" binding:"required"`
}

// AppleAuthRequest carries an Apple identity token from Sign in with Apple.
// FullName is only provided by Apple on the very first authorization.
type AppleAuthRequest struct {
	IDToken  string `json:"id_token" binding:"required"`
	FullName string `json:"full_name"`
}

// ProfileResponse is the authenticated user's profile.
type ProfileResponse struct {
	ID          string    `json:"id"`
	Email       string    `json:"email"`
	DisplayName string    `json:"display_name"`
	AvatarURL   string    `json:"avatar_url"`
	Provider    string    `json:"provider"`
	Role        string    `json:"role"`
	CreatedAt   time.Time `json:"created_at"`
}

// UpdateProfileRequest updates mutable profile fields. Pointers distinguish
// "not provided" from "set to empty".
type UpdateProfileRequest struct {
	DisplayName *string `json:"display_name"`
}

// InstrumentResponse is a single computed instrument price.
type InstrumentResponse struct {
	Code        string    `json:"code"`
	DisplayName string    `json:"display_name"`
	Kind        string    `json:"kind"`
	Price       float64   `json:"price"`
	Bid         float64   `json:"bid"`
	Ask         float64   `json:"ask"`
	Change      float64   `json:"change"`
	ChangePct   float64   `json:"change_pct"`
	Provider    string    `json:"provider,omitempty"`
	Timestamp   time.Time `json:"timestamp"`
}

// WatchlistRequest adds/removes a tracked instrument.
type WatchlistRequest struct {
	Code string `json:"code" binding:"required"`
}

// AlertRequest creates a price alert.
type AlertRequest struct {
	Code      string  `json:"code" binding:"required"`
	Direction string  `json:"direction" binding:"required,oneof=above below"`
	Threshold float64 `json:"threshold" binding:"required"`
}

// AlertResponse describes a user alert.
type AlertResponse struct {
	ID          string     `json:"id"`
	Code        string     `json:"code"`
	Direction   string     `json:"direction"`
	Threshold   float64    `json:"threshold"`
	Active      bool       `json:"active"`
	LastFiredAt *time.Time `json:"last_fired_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

// NotificationResponse describes a delivered notification.
type NotificationResponse struct {
	ID        string     `json:"id"`
	Code      string     `json:"code"`
	Title     string     `json:"title"`
	Body      string     `json:"body"`
	Price     float64    `json:"price"`
	ReadAt    *time.Time `json:"read_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

// DeviceTokenRequest registers a push token for the current user.
type DeviceTokenRequest struct {
	Platform string `json:"platform" binding:"required,oneof=ios android"`
	Token    string `json:"token" binding:"required"`
}

// GoldResponse is the current gram-gold price payload.
type GoldResponse struct {
	Symbol    string    `json:"symbol"`
	Price     float64   `json:"price"`
	MidPrice  float64   `json:"mid_price"`
	Spread    float64   `json:"spread"`
	XAUUSD    float64   `json:"xauusd"`
	USDTRY    float64   `json:"usdtry"`
	Provider  string    `json:"provider"`
	Timestamp time.Time `json:"timestamp"`
	Source    string    `json:"source"`
}

// HistoryItem is a single historical record.
type HistoryItem struct {
	Price     float64   `json:"price"`
	MidPrice  float64   `json:"mid_price"`
	Spread    float64   `json:"spread"`
	XAUUSD    float64   `json:"xauusd"`
	USDTRY    float64   `json:"usdtry"`
	Provider  string    `json:"provider"`
	CreatedAt time.Time `json:"created_at"`
}

// ProviderStatusResponse describes a provider's health.
type ProviderStatusResponse struct {
	Name          string    `json:"name"`
	Status        string    `json:"status"`
	LastTickAt    time.Time `json:"last_tick_at"`
	LastError     string    `json:"last_error,omitempty"`
	Reconnects    int64     `json:"reconnects"`
	Subscribed    []string  `json:"subscribed"`
	LatencyMillis int64     `json:"latency_ms"`
	Active        bool      `json:"active"`
}

// ErrorResponse is the standard error envelope.
type ErrorResponse struct {
	Error string `json:"error"`
}
