package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"

	"github.com/tsc-forex/tsc_forex/internal/api"
	"github.com/tsc-forex/tsc_forex/internal/cache"
	"github.com/tsc-forex/tsc_forex/internal/config"
	"github.com/tsc-forex/tsc_forex/internal/domain"
)

func TestGoldCurrentFromCacheFallback(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mr, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}
	defer mr.Close()

	rc, err := cache.NewRedisCache(
		context.Background(),
		config.RedisConfig{Addr: mr.Addr(), PoolSize: 2, PubSubChannel: "c"},
		config.CacheConfig{LatestPriceTTL: time.Minute, LatestPriceKey: "gold:latest"},
		nil, nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	defer rc.Close()

	// Seed cache; calculator is nil so handler must fall back to cache.
	g := domain.GramGold{Symbol: domain.SymbolGramGoldTRY, Price: 2888, Timestamp: time.Now().UTC()}
	if err := rc.SetLatest(context.Background(), g); err != nil {
		t.Fatal(err)
	}

	h := api.NewGoldHandler(nil, rc, nil, nil)
	r := gin.New()
	r.GET("/gold/current", h.Current)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/gold/current", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
	var resp api.GoldResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Price != 2888 || resp.Source != "cache" {
		t.Fatalf("resp = %+v", resp)
	}
}
