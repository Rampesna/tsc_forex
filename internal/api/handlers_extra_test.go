package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/tsc-forex/tsc_forex/internal/api"
	"github.com/tsc-forex/tsc_forex/internal/config"
	"github.com/tsc-forex/tsc_forex/internal/domain"
	"github.com/tsc-forex/tsc_forex/internal/models"
	"github.com/tsc-forex/tsc_forex/internal/provider"
	"github.com/tsc-forex/tsc_forex/internal/repository"
	"github.com/tsc-forex/tsc_forex/internal/service"
)

// fakeRepo implements repository.GoldRepository.
type fakeRepo struct {
	rows    []models.GoldPrice
	candles []models.OHLC
}

func (f *fakeRepo) Save(context.Context, domain.GramGold) error { return nil }
func (f *fakeRepo) History(context.Context, string, time.Time, time.Time, int) ([]models.GoldPrice, error) {
	return f.rows, nil
}
func (f *fakeRepo) Aggregate(context.Context, string, string, time.Time, time.Time) ([]models.OHLC, error) {
	return f.candles, nil
}
func (f *fakeRepo) Latest(context.Context, string) (models.GoldPrice, error) {
	if len(f.rows) == 0 {
		return models.GoldPrice{}, nil
	}
	return f.rows[0], nil
}

func TestHistoryRawAndAggregate(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &fakeRepo{
		rows: []models.GoldPrice{
			{Symbol: domain.SymbolGramGoldTRY, Price: 2500, CreatedAt: time.Now().UTC()},
			{Symbol: domain.SymbolGramGoldTRY, Price: 2501, CreatedAt: time.Now().UTC()},
		},
		candles: []models.OHLC{{Bucket: time.Now().UTC(), Open: 2500, High: 2510, Low: 2495, Close: 2505, Count: 10}},
	}
	h := api.NewGoldHandler(nil, nil, repo, nil)
	r := gin.New()
	r.GET("/gold/history", h.History)

	// Raw with RFC3339 from/to and unix — exercises parseTime branches.
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/gold/history?limit=10&from=2020-01-01T00:00:00Z&to=1893456000", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("raw status = %d", w.Code)
	}
	// Aggregate
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/gold/history?interval=1m", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("agg status = %d", w.Code)
	}
	// Bad interval
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/gold/history?interval=7x", nil))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("bad interval status = %d", w.Code)
	}
}

func TestProvidersAndHealthHandlers(t *testing.T) {
	gin.SetMode(gin.TestMode)
	pmCfg := config.ProviderConfig{
		Primary: "mock", ReconnectInitial: 10 * time.Millisecond, ReconnectMax: 20 * time.Millisecond,
		HealthInterval: time.Second, StaleThreshold: time.Hour,
		Mock: config.MockProviderConfig{TickInterval: 10 * time.Millisecond},
	}
	pm, err := provider.NewManager(pmCfg, provider.DefaultFactory(pmCfg))
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_ = pm.Run(ctx)
	defer pm.Close()

	sys := api.NewSystemHandler(api.SystemDeps{Provider: pm, ClientsFn: func() int64 { return 7 }})
	r := gin.New()
	r.GET("/providers", sys.Providers)
	r.GET("/health", sys.Health)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/providers", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("providers status = %d", w.Code)
	}

	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/health", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("health status = %d", w.Code)
	}
	var body map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	if body["ws_clients"] == nil {
		t.Fatal("missing ws_clients")
	}
}

// memRepo implements repository.UserRepository for auth handler tests.
type memRepo struct{ m map[string]models.User }

func (r *memRepo) Create(_ context.Context, u *models.User) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	r.m[u.Email] = *u
	return nil
}
func (r *memRepo) ByEmail(_ context.Context, e string) (models.User, error) {
	u, ok := r.m[e]
	if !ok {
		return models.User{}, repository.ErrUserNotFound
	}
	return u, nil
}

func TestAuthHandlerRegisterLogin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	auth := service.NewAuthService(&memRepo{m: map[string]models.User{}}, config.JWTConfig{
		Secret: "abcdefghijklmnopqrstuvwxyz012345", Issuer: "t",
		AccessTokenTTL: time.Minute, RefreshTokenTTL: time.Hour,
	})
	h := api.NewAuthHandler(auth)
	r := gin.New()
	r.POST("/register", h.Register)
	r.POST("/login", h.Login)

	// Register
	body, _ := json.Marshal(api.RegisterRequest{Email: "u@e.com", Password: "password123"})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/register", bytes.NewReader(body)))
	if w.Code != http.StatusCreated {
		t.Fatalf("register status = %d body=%s", w.Code, w.Body.String())
	}

	// Login
	body, _ = json.Marshal(api.LoginRequest{Email: "u@e.com", Password: "password123"})
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(body)))
	if w.Code != http.StatusOK {
		t.Fatalf("login status = %d", w.Code)
	}
	var tok api.TokenResponse
	if err := json.Unmarshal(w.Body.Bytes(), &tok); err != nil {
		t.Fatal(err)
	}
	if tok.AccessToken == "" || tok.TokenType != "Bearer" {
		t.Fatalf("token resp = %+v", tok)
	}

	// Bad login
	body, _ = json.Marshal(api.LoginRequest{Email: "u@e.com", Password: "wrongpass1"})
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(body)))
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("bad login status = %d", w.Code)
	}

	// Invalid payload
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/register", bytes.NewReader([]byte(`{"email":"x"}`))))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("invalid register status = %d", w.Code)
	}

	// Duplicate registration → 409 (covers ErrEmailTaken branch).
	body, _ = json.Marshal(api.RegisterRequest{Email: "u@e.com", Password: "password123"})
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/register", bytes.NewReader(body)))
	if w.Code != http.StatusConflict {
		t.Fatalf("duplicate register status = %d", w.Code)
	}
}

type failingPinger struct{}

func (failingPinger) Ping(context.Context) error { return context.DeadlineExceeded }

func TestHealthDegraded(t *testing.T) {
	gin.SetMode(gin.TestMode)
	sys := api.NewSystemHandler(api.SystemDeps{
		Redis:  failingPinger{},
		DBPing: func(context.Context) error { return context.DeadlineExceeded },
	})
	r := gin.New()
	r.GET("/health", sys.Health)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/health", nil))
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("degraded status = %d", w.Code)
	}
	var body map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	if body["status"] != "degraded" {
		t.Fatalf("status = %v", body["status"])
	}
	deps, _ := body["deps"].(map[string]interface{})
	if deps["redis"] != "down" || deps["database"] != "down" {
		t.Fatalf("deps = %v", deps)
	}
}
