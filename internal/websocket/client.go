package websocket

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

// Client is a single WebSocket connection managed by the Hub.
type Client struct {
	ID          string
	UserID      string // set when the connection is authenticated (JWT); "" = anonymous
	Platform    string
	UserAgent   string
	RemoteAddr  string
	ConnectedAt time.Time
	hub         *Hub
	conn        *websocket.Conn
	send        chan []byte
	shutdown    chan closeRequest
	done        chan struct{}
	log         *zap.Logger

	sendOnce sync.Once
	doneOnce sync.Once
}

const websocketCloseServiceRestart = websocket.CloseServiceRestart

type closeRequest struct {
	code   int
	reason string
}

func newClient(hub *Hub, conn *websocket.Conn, log *zap.Logger) *Client {
	if log == nil {
		log = zap.NewNop()
	}
	return &Client{
		ID:          uuid.NewString(),
		ConnectedAt: time.Now().UTC(),
		hub:         hub,
		conn:        conn,
		send:        make(chan []byte, hub.cfg.SendBufferSize),
		shutdown:    make(chan closeRequest, 1),
		done:        make(chan struct{}),
		log:         log,
	}
}

func (c *Client) closeSend() {
	c.sendOnce.Do(func() { close(c.send) })
}

func (c *Client) markDone() {
	c.doneOnce.Do(func() { close(c.done) })
}

func (c *Client) requestShutdown(code int, reason string) {
	select {
	case c.shutdown <- closeRequest{code: code, reason: reason}:
	default:
	}
}

func (c *Client) rejectForRestart() {
	deadline := time.Now().Add(c.hub.cfg.WriteWait)
	_ = c.conn.WriteControl(
		websocket.CloseMessage,
		websocket.FormatCloseMessage(websocketCloseServiceRestart, "service restart"),
		deadline,
	)
	_ = c.conn.Close()
	c.markDone()
}

// readPump consumes inbound frames (pong, optional hello). On exit, unregisters.
func (c *Client) readPump() {
	defer func() {
		c.hub.Unregister(c)
		_ = c.conn.Close()
		c.markDone()
	}()

	cfg := c.hub.cfg
	c.conn.SetReadLimit(cfg.MaxMessageSize)
	_ = c.conn.SetReadDeadline(time.Now().Add(cfg.PongWait))
	c.conn.SetPongHandler(func(string) error {
		c.hub.touchPresence(c)
		return c.conn.SetReadDeadline(time.Now().Add(cfg.PongWait))
	})

	for {
		_, data, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(
				err,
				websocket.CloseGoingAway,
				websocket.CloseNormalClosure,
				websocketCloseServiceRestart,
			) {
				c.log.Debug("ws read error", zap.String("client_id", c.ID), zap.Error(err))
			}
			return
		}
		// Optional client hello / ping text — ignore unknown payloads.
		var hello ClientHello
		if err := json.Unmarshal(data, &hello); err == nil && hello.Type != "" {
			c.log.Debug("ws client message",
				zap.String("client_id", c.ID),
				zap.String("type", hello.Type),
			)
		}
	}
}

// writePump flushes the send queue and emits protocol pings.
func (c *Client) writePump() {
	cfg := c.hub.cfg
	pingPeriod := cfg.PingPeriod
	if pingPeriod <= 0 {
		pingPeriod = (cfg.PongWait * 9) / 10
	}
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		_ = c.conn.Close()
		c.markDone()
	}()

	for {
		// Prefer a service-restart close over queued market updates once drain
		// begins, so clients reconnect to a healthy replacement pod promptly.
		select {
		case req := <-c.shutdown:
			c.writeClose(req)
			return
		default:
		}
		select {
		case req := <-c.shutdown:
			c.writeClose(req)
			return
		case msg, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(cfg.WriteWait))
			if !ok {
				c.writeClose(closeRequest{code: websocket.CloseNormalClosure})
				return
			}
			// One JSON envelope per WebSocket text frame (mobile clients parse simply).
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(cfg.WriteWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (c *Client) writeClose(req closeRequest) {
	_ = c.conn.SetWriteDeadline(time.Now().Add(c.hub.cfg.WriteWait))
	_ = c.conn.WriteMessage(
		websocket.CloseMessage,
		websocket.FormatCloseMessage(req.code, req.reason),
	)
}
