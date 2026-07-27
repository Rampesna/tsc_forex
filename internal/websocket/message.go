package websocket

import "time"

// MessageType identifies outbound / inbound WS frames.
type MessageType string

const (
	TypePriceUpdate      MessageType = "price_update"
	TypeInstrumentUpdate MessageType = "instrument_update"
	TypeNotification     MessageType = "notification"
	TypeHeartbeat        MessageType = "heartbeat"
	TypeError            MessageType = "error"
	TypeSubscribed       MessageType = "subscribed"
)

// Envelope is the JSON wrapper pushed to clients.
type Envelope struct {
	Type      MessageType `json:"type"`
	RequestID string      `json:"request_id,omitempty"`
	Payload   any         `json:"payload,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
}

// PricePayload is the gram-gold (or future symbol) update body.
type PricePayload struct {
	Symbol   string  `json:"symbol"`
	Price    float64 `json:"price"`
	MidPrice float64 `json:"mid_price"`
	Spread   float64 `json:"spread"`
	XAUUSD   float64 `json:"xauusd,omitempty"`
	USDTRY   float64 `json:"usdtry,omitempty"`
	Provider string  `json:"provider,omitempty"`
}

// InstrumentPayload is a single computed instrument price pushed to clients.
type InstrumentPayload struct {
	Code        string  `json:"code"`
	DisplayName string  `json:"display_name"`
	Kind        string  `json:"kind"`
	Price       float64 `json:"price"`
	Bid         float64 `json:"bid"`
	Ask         float64 `json:"ask"`
	Change      float64 `json:"change"`
	ChangePct   float64 `json:"change_pct"`
	Currency    string  `json:"currency,omitempty"`
	PriceTRY    float64 `json:"price_try,omitempty"`
	Provider    string  `json:"provider,omitempty"`
}

// NotificationPayload is a targeted alert/notification pushed to one user.
type NotificationPayload struct {
	ID       string  `json:"id,omitempty"`
	Code     string  `json:"code"`
	Title    string  `json:"title"`
	Body     string  `json:"body"`
	DeepLink string  `json:"deep_link,omitempty"`
	Price    float64 `json:"price"`
}

// ClientHello is an optional first message from the client (subscribe hints).
type ClientHello struct {
	Type    string   `json:"type"`
	Symbols []string `json:"symbols,omitempty"`
}
