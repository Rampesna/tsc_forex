package websocket

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

// TokenVerifier validates a JWT and returns the user id (subject) when valid.
type TokenVerifier func(token string) (userID string, ok bool)

// Handler upgrades HTTP connections and attaches clients to the Hub.
type Handler struct {
	hub      *Hub
	log      *zap.Logger
	upgrader websocket.Upgrader
	verify   TokenVerifier
}

// HandlerOption configures optional Handler behavior.
type HandlerOption func(*Handler)

// WithVerifier attaches a token verifier so that authenticated connections
// (via ?token=... query param) can receive targeted notifications.
func WithVerifier(v TokenVerifier) HandlerOption {
	return func(h *Handler) { h.verify = v }
}

// NewHandler builds a Gin-compatible WebSocket handler.
func NewHandler(hub *Hub, log *zap.Logger, opts ...HandlerOption) *Handler {
	if log == nil {
		log = zap.NewNop()
	}
	cfg := hub.Config()
	upgrader := websocket.Upgrader{
		ReadBufferSize:    1024,
		WriteBufferSize:   1024,
		EnableCompression: cfg.EnableCompression,
		CheckOrigin: func(r *http.Request) bool {
			if !cfg.CheckOrigin {
				return true
			}
			// Strict mode: same-origin only until CORS allow-list is wired in step 9.
			origin := r.Header.Get("Origin")
			return origin == "" || origin == "http://"+r.Host || origin == "https://"+r.Host
		},
	}
	h := &Handler{hub: hub, log: log, upgrader: upgrader}
	for _, opt := range opts {
		opt(h)
	}
	return h
}

// ServeWS is the Gin handler for GET /ws (or configured path).
func (h *Handler) ServeWS(c *gin.Context) {
	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		h.log.Warn("ws upgrade failed", zap.Error(err))
		return
	}
	if h.hub.Config().EnableCompression {
		_ = conn.SetCompressionLevel(1) // fast compression for low latency
	}

	client := newClient(h.hub, conn, h.log)
	if h.verify != nil {
		if tok := c.Query("token"); tok != "" {
			if uid, ok := h.verify(tok); ok {
				client.UserID = uid
			}
		}
	}
	h.hub.Register(client)

	// Welcome frame so clients know the session is live (reconnect-friendly).
	welcome, _ := json.Marshal(Envelope{
		Type:      TypeSubscribed,
		RequestID: c.GetString("request_id"),
		Payload:   gin.H{"client_id": client.ID, "path": h.hub.Config().Path},
		Timestamp: time.Now().UTC(),
	})
	select {
	case client.send <- welcome:
	default:
		h.hub.droppedMsgs.Add(1)
	}

	go client.writePump()
	go client.readPump()
}

// EncodePrice marshals a price update envelope (used by calculator / redis bridge).
func EncodePrice(p PricePayload) ([]byte, error) {
	return json.Marshal(Envelope{
		Type:      TypePriceUpdate,
		Payload:   p,
		Timestamp: time.Now().UTC(),
	})
}

// EncodeInstrument marshals an instrument price update envelope.
func EncodeInstrument(p InstrumentPayload) ([]byte, error) {
	return json.Marshal(Envelope{
		Type:      TypeInstrumentUpdate,
		Payload:   p,
		Timestamp: time.Now().UTC(),
	})
}

// EncodeNotification marshals a targeted notification envelope.
func EncodeNotification(p NotificationPayload) ([]byte, error) {
	return json.Marshal(Envelope{
		Type:      TypeNotification,
		Payload:   p,
		Timestamp: time.Now().UTC(),
	})
}

// EncodeHeartbeat marshals a heartbeat envelope.
func EncodeHeartbeat() ([]byte, error) {
	return json.Marshal(Envelope{
		Type:      TypeHeartbeat,
		Timestamp: time.Now().UTC(),
	})
}
