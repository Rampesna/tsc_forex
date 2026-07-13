package websocket

import (
	"sync"
	"sync/atomic"

	"go.uber.org/zap"

	"github.com/tsc-forex/tsc_forex/internal/config"
)

// Hub maintains the set of active clients and broadcasts messages.
// A single Run goroutine owns the clients map — no mutex on the hot path
// for register/unregister/broadcast fan-out.
type Hub struct {
	cfg config.WebSocketConfig
	log *zap.Logger

	register   chan *Client
	unregister chan *Client
	broadcast  chan []byte
	direct     chan userMessage

	// clients / userIndex are only touched from Run().
	clients   map[*Client]struct{}
	userIndex map[string]map[*Client]struct{}

	clientCount atomic.Int64
	droppedMsgs atomic.Int64
	broadcasts  atomic.Int64

	done chan struct{}
	once sync.Once
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
	return &Hub{
		cfg:        cfg,
		log:        log,
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan []byte, 1024),
		direct:     make(chan userMessage, 1024),
		clients:    make(map[*Client]struct{}),
		userIndex:  make(map[string]map[*Client]struct{}),
		done:       make(chan struct{}),
	}
}

// Run processes register / unregister / broadcast until Close.
func (h *Hub) Run() {
	for {
		select {
		case <-h.done:
			for c := range h.clients {
				h.closeClient(c)
			}
			h.clients = nil
			return
		case c := <-h.register:
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
			h.log.Debug("ws client registered",
				zap.String("client_id", c.ID),
				zap.Int64("clients", h.clientCount.Load()),
			)
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
	delete(h.clients, c)
	if c.UserID != "" {
		if set := h.userIndex[c.UserID]; set != nil {
			delete(set, c)
			if len(set) == 0 {
				delete(h.userIndex, c.UserID)
			}
		}
	}
	c.closeSend()
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

// Register admits a client into the hub.
func (h *Hub) Register(c *Client) {
	select {
	case h.register <- c:
	case <-h.done:
		c.closeSend()
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

// Close stops the hub and disconnects all clients. Idempotent.
func (h *Hub) Close() {
	h.once.Do(func() { close(h.done) })
}

// Config exposes websocket knobs for the upgrader / client pumps.
func (h *Hub) Config() config.WebSocketConfig { return h.cfg }
