package websocket

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"

	"github.com/tsc-forex/tsc_forex/internal/config"
	"github.com/tsc-forex/tsc_forex/internal/presence"
)

type PresenceRecorder interface {
	Connected(context.Context, presence.Entry) error
	Heartbeat(context.Context, string) error
	Disconnected(context.Context, string) error
}

// Hub maintains the set of active clients and broadcasts messages.
// A single Run goroutine owns the clients map — no mutex on the hot path
// for register/unregister/broadcast fan-out.
type Hub struct {
	cfg config.WebSocketConfig
	log *zap.Logger

	register   chan registration
	unregister chan *Client
	broadcast  chan []byte
	direct     chan userMessage

	// clients / userIndex are only touched from Run().
	clients   map[*Client]struct{}
	userIndex map[string]map[*Client]struct{}

	clientCount atomic.Int64
	droppedMsgs atomic.Int64
	broadcasts  atomic.Int64
	accepting   atomic.Bool

	done        chan struct{}
	drained     chan struct{}
	closeOnce   sync.Once
	drainedOnce sync.Once
	presence    PresenceRecorder
}

type registration struct {
	client   *Client
	accepted chan bool
}

// userMessage targets a single authenticated user's connections.
type userMessage struct {
	userID string
	msg    []byte
}

// NewHub creates a hub. Call Run in a dedicated goroutine.
func NewHub(cfg config.WebSocketConfig, log *zap.Logger) *Hub {
	if log == nil {
		log = zap.NewNop()
	}
	if cfg.SendBufferSize <= 0 {
		cfg.SendBufferSize = 256
	}
	h := &Hub{
		cfg:        cfg,
		log:        log,
		register:   make(chan registration),
		unregister: make(chan *Client),
		broadcast:  make(chan []byte, 1024),
		direct:     make(chan userMessage, 1024),
		clients:    make(map[*Client]struct{}),
		userIndex:  make(map[string]map[*Client]struct{}),
		done:       make(chan struct{}),
		drained:    make(chan struct{}),
	}
	h.accepting.Store(true)
	return h
}

// Run processes register / unregister / broadcast until Close.
func (h *Hub) Run() {
	for {
		select {
		case <-h.done:
			h.accepting.Store(false)
			clients := make([]*Client, 0, len(h.clients))
			for c := range h.clients {
				clients = append(clients, c)
				h.removeClient(c, false)
				c.requestShutdown(websocketCloseServiceRestart, "service restart")
			}
			h.clients = nil
			h.userIndex = nil
			h.clientCount.Store(0)
			go h.waitForClients(clients)
			return
		case reg := <-h.register:
			c := reg.client
			if c == nil || !h.accepting.Load() {
				reg.accepted <- false
				continue
			}
			h.clients[c] = struct{}{}
			if c.UserID != "" {
				set := h.userIndex[c.UserID]
				if set == nil {
					set = make(map[*Client]struct{})
					h.userIndex[c.UserID] = set
				}
				set[c] = struct{}{}
			}
			h.clientCount.Store(int64(len(h.clients)))
			h.recordPresence(c)
			h.log.Debug("ws client registered",
				zap.String("client_id", c.ID),
				zap.Int64("clients", h.clientCount.Load()),
			)
			reg.accepted <- true
		case c := <-h.unregister:
			if _, ok := h.clients[c]; ok {
				h.closeClient(c)
				h.clientCount.Store(int64(len(h.clients)))
				h.log.Debug("ws client unregistered",
					zap.String("client_id", c.ID),
					zap.Int64("clients", h.clientCount.Load()),
				)
			}
		case dm := <-h.direct:
			for c := range h.userIndex[dm.userID] {
				select {
				case c.send <- dm.msg:
				default:
					h.droppedMsgs.Add(1)
				}
			}
		case msg := <-h.broadcast:
			h.broadcasts.Add(1)
			for c := range h.clients {
				select {
				case c.send <- msg:
				default:
					// Slow consumer: drop this message; do not block the hub.
					h.droppedMsgs.Add(1)
					h.log.Warn("ws message dropped",
						zap.String("client_id", c.ID),
						zap.Int64("dropped_total", h.droppedMsgs.Load()),
					)
				}
			}
		}
	}
}

func (h *Hub) closeClient(c *Client) {
	h.removeClient(c, true)
}

func (h *Hub) removeClient(c *Client, closeSend bool) {
	delete(h.clients, c)
	if c.UserID != "" {
		if set := h.userIndex[c.UserID]; set != nil {
			delete(set, c)
			if len(set) == 0 {
				delete(h.userIndex, c.UserID)
			}
		}
	}
	if closeSend {
		c.closeSend()
	}
	if h.presence != nil && c.UserID != "" {
		go func(id string) {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			_ = h.presence.Disconnected(ctx, id)
		}(c.ID)
	}
}

func (h *Hub) waitForClients(clients []*Client) {
	for _, c := range clients {
		<-c.done
	}
	h.drainedOnce.Do(func() { close(h.drained) })
}

func (h *Hub) recordPresence(c *Client) {
	if h.presence == nil || c == nil || c.UserID == "" {
		return
	}
	entry := presence.Entry{
		ConnectionID: c.ID, UserID: c.UserID, Platform: c.Platform,
		UserAgent: c.UserAgent, RemoteAddr: c.RemoteAddr, ConnectedAt: c.ConnectedAt,
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = h.presence.Connected(ctx, entry)
	}()
}

func (h *Hub) touchPresence(c *Client) {
	if h.presence == nil || c == nil || c.UserID == "" {
		return
	}
	go func(id string) {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = h.presence.Heartbeat(ctx, id)
	}(c.ID)
}

// SetPresenceRecorder enables cross-pod online-user tracking after Redis is
// available. Existing WebSocket behavior is unchanged when it is nil.
func (h *Hub) SetPresenceRecorder(recorder PresenceRecorder) {
	h.presence = recorder
}

// SendToUser queues a message for all connections belonging to a user.
// Used for targeted price alerts / notifications.
func (h *Hub) SendToUser(userID string, msg []byte) {
	if userID == "" {
		return
	}
	select {
	case h.direct <- userMessage{userID: userID, msg: msg}:
	case <-h.done:
	default:
		h.droppedMsgs.Add(1)
	}
}

// Broadcast queues a pre-encoded JSON payload for all clients.
// Non-blocking for the caller relative to slow clients (hub drops per-client).
func (h *Hub) Broadcast(msg []byte) {
	select {
	case h.broadcast <- msg:
	case <-h.done:
	default:
		// Hub broadcast queue full — count as dropped at hub level.
		h.droppedMsgs.Add(1)
	}
}

// Register admits a client into the hub. It returns false after draining starts,
// including the narrow race between the HTTP pre-check and WebSocket upgrade.
func (h *Hub) Register(c *Client) bool {
	if c == nil || !h.accepting.Load() {
		return false
	}
	accepted := make(chan bool, 1)
	select {
	case h.register <- registration{client: c, accepted: accepted}:
	case <-h.done:
		return false
	}
	select {
	case ok := <-accepted:
		return ok
	case <-h.done:
		return false
	}
}

// Unregister removes a client (also invoked from client pumps on disconnect).
func (h *Hub) Unregister(c *Client) {
	select {
	case h.unregister <- c:
	case <-h.done:
	}
}

// ClientCount returns the number of connected clients.
func (h *Hub) ClientCount() int64 { return h.clientCount.Load() }

// DroppedMessages returns messages dropped due to slow clients or full queues.
func (h *Hub) DroppedMessages() int64 { return h.droppedMsgs.Load() }

// BroadcastCount returns how many fan-outs completed.
func (h *Hub) BroadcastCount() int64 { return h.broadcasts.Load() }

// Accepting reports whether new WebSocket sessions may be admitted.
func (h *Hub) Accepting() bool { return h.accepting.Load() }

// BeginDrain rejects new WebSocket sessions before active sessions are closed.
// It is safe to call more than once.
func (h *Hub) BeginDrain() bool {
	return h.accepting.CompareAndSwap(true, false)
}

// Close stops the hub and disconnects all clients. Idempotent.
func (h *Hub) Close() {
	h.BeginDrain()
	h.closeOnce.Do(func() { close(h.done) })
}

// Shutdown sends active clients RFC 6455 close code 1012 (service restart) and
// waits for their pumps to release the hijacked connections. net/http.Shutdown
// intentionally does not manage WebSockets, so the application must do this.
func (h *Hub) Shutdown(ctx context.Context) error {
	h.Close()
	select {
	case <-h.drained:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Config exposes websocket knobs for the upgrader / client pumps.
func (h *Hub) Config() config.WebSocketConfig { return h.cfg }
