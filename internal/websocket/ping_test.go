package websocket_test

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	gws "github.com/gorilla/websocket"

	"github.com/tsc-forex/tsc_forex/internal/config"
	ws "github.com/tsc-forex/tsc_forex/internal/websocket"
)

func TestWebSocketPingAndClientHello(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := config.WebSocketConfig{
		Path:           "/ws",
		WriteWait:      time.Second,
		PongWait:       2 * time.Second,
		PingPeriod:     150 * time.Millisecond, // fast ping to exercise writePump ticker
		MaxMessageSize: 4096,
		SendBufferSize: 8,
	}
	hub := ws.NewHub(cfg, nil)
	go hub.Run()
	defer hub.Close()

	r := gin.New()
	r.GET("/ws", ws.NewHandler(hub, nil).ServeWS)
	srv := httptest.NewServer(r)
	defer srv.Close()

	url := "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws"
	conn, _, err := gws.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	gotPing := make(chan struct{}, 1)
	conn.SetPingHandler(func(string) error {
		select {
		case gotPing <- struct{}{}:
		default:
		}
		return conn.WriteControl(gws.PongMessage, nil, time.Now().Add(time.Second))
	})

	// Send a client hello (covers readPump hello-decoding branch).
	hello, _ := json.Marshal(ws.ClientHello{Type: "subscribe", Symbols: []string{"XAU_GRAM_TRY"}})
	if err := conn.WriteMessage(gws.TextMessage, hello); err != nil {
		t.Fatal(err)
	}

	// Read welcome + drive the connection to receive a ping.
	deadline := time.Now().Add(3 * time.Second)
	conn.SetReadDeadline(deadline)
	go func() {
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}()

	select {
	case <-gotPing:
	case <-time.After(2 * time.Second):
		t.Fatal("did not receive server ping")
	}
}
