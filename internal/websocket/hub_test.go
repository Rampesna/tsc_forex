package websocket_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"github.com/tsc-forex/tsc_forex/internal/config"
	ws "github.com/tsc-forex/tsc_forex/internal/websocket"
)

func testWSConfig() config.WebSocketConfig {
	return config.WebSocketConfig{
		Path:              "/ws",
		WriteWait:         2 * time.Second,
		PongWait:          3 * time.Second,
		PingPeriod:        1 * time.Second,
		MaxMessageSize:    4096,
		SendBufferSize:    8,
		EnableCompression: false,
		CheckOrigin:       false,
	}
}

func TestHubBroadcastAndDrop(t *testing.T) {
	t.Parallel()

	hub := ws.NewHub(testWSConfig(), nil)
	go hub.Run()
	defer hub.Close()

	// Direct client injection without network: use Register via handler path below.
	// Here we only exercise Broadcast queue accounting with zero clients.
	hub.Broadcast([]byte(`{"type":"heartbeat"}`))
	time.Sleep(20 * time.Millisecond)
	if hub.BroadcastCount() < 1 {
		t.Fatalf("broadcasts = %d", hub.BroadcastCount())
	}
}

func TestWebSocketConnectAndPricePush(t *testing.T) {
	gin.SetMode(gin.TestMode)
	hub := ws.NewHub(testWSConfig(), nil)
	go hub.Run()
	defer hub.Close()

	h := ws.NewHandler(hub, nil)
	r := gin.New()
	r.GET("/ws", h.ServeWS)

	srv := httptest.NewServer(r)
	defer srv.Close()

	url := "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws"
	conn, resp, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()
	if resp.StatusCode != http.StatusSwitchingProtocols {
		t.Fatalf("status = %d", resp.StatusCode)
	}

	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, welcome, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read welcome: %v", err)
	}
	var env ws.Envelope
	if err := json.Unmarshal(welcome, &env); err != nil {
		t.Fatal(err)
	}
	if env.Type != ws.TypeSubscribed {
		t.Fatalf("type = %s", env.Type)
	}

	deadline := time.Now().Add(2 * time.Second)
	for hub.ClientCount() < 1 && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if hub.ClientCount() != 1 {
		t.Fatalf("clients = %d", hub.ClientCount())
	}

	payload, err := ws.EncodePrice(ws.PricePayload{
		Symbol:   "XAU_GRAM_TRY",
		Price:    2600.5,
		MidPrice: 2600.5,
		Spread:   0.1,
		Provider: "test",
	})
	if err != nil {
		t.Fatal(err)
	}
	hub.Broadcast(payload)

	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, msg, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read price: %v", err)
	}
	if err := json.Unmarshal(msg, &env); err != nil {
		t.Fatal(err)
	}
	if env.Type != ws.TypePriceUpdate {
		t.Fatalf("type = %s want price_update", env.Type)
	}

	_ = conn.Close()
	deadline = time.Now().Add(2 * time.Second)
	for hub.ClientCount() > 0 && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if hub.ClientCount() != 0 {
		t.Fatalf("clients after close = %d", hub.ClientCount())
	}
}

func TestHubDrainRejectsNewConnectionsAndClosesExistingWithRestartCode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	hub := ws.NewHub(testWSConfig(), nil)
	go hub.Run()

	h := ws.NewHandler(hub, nil)
	r := gin.New()
	r.GET("/ws", h.ServeWS)
	srv := httptest.NewServer(r)
	defer srv.Close()

	url := "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws"
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("dial existing: %v", err)
	}
	defer conn.Close()
	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	if _, _, err := conn.ReadMessage(); err != nil {
		t.Fatalf("read welcome: %v", err)
	}

	hub.BeginDrain()
	_, resp, err := websocket.DefaultDialer.Dial(url, nil)
	if err == nil {
		t.Fatal("new WebSocket unexpectedly admitted while draining")
	}
	if resp == nil || resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("draining status = %#v", resp)
	}
	if got := resp.Header.Get("Retry-After"); got != "5" {
		t.Fatalf("Retry-After = %q", got)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := hub.Shutdown(shutdownCtx); err != nil {
		t.Fatalf("hub shutdown: %v", err)
	}

	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, _, err = conn.ReadMessage()
	var closeErr *websocket.CloseError
	if !errors.As(err, &closeErr) {
		t.Fatalf("close error = %T %v", err, err)
	}
	if closeErr.Code != websocket.CloseServiceRestart {
		t.Fatalf("close code = %d, want %d", closeErr.Code, websocket.CloseServiceRestart)
	}
	if hub.ClientCount() != 0 {
		t.Fatalf("clients after shutdown = %d", hub.ClientCount())
	}
}

func TestEncodeHelpers(t *testing.T) {
	t.Parallel()
	b, err := ws.EncodeHeartbeat()
	if err != nil || len(b) == 0 {
		t.Fatalf("heartbeat: %v %s", err, b)
	}
	b, err = ws.EncodePrice(ws.PricePayload{Symbol: "X", Price: 1})
	if err != nil {
		t.Fatal(err)
	}
	var env ws.Envelope
	if err := json.Unmarshal(b, &env); err != nil {
		t.Fatal(err)
	}
}
