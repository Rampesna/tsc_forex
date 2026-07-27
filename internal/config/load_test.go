package config

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"
)

func TestLoadDefaults(t *testing.T) {
	// Not parallel: AutomaticEnv reads process environment.
	clearAppEnv(t)
	t.Setenv("APP_SERVER_PORT", "")
	_ = os.Unsetenv("APP_SERVER_PORT")

	// No config file: defaults + validation must succeed.
	m, err := Load("")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	cfg := m.Get()
	if cfg.App.Name != "tsc_forex" {
		t.Fatalf("App.Name = %q", cfg.App.Name)
	}
	if cfg.Server.Port != 8080 {
		t.Fatalf("Server.Port = %d", cfg.Server.Port)
	}
	if cfg.Gold.TroyOunceToGram != 31.1034768 {
		t.Fatalf("TroyOunceToGram = %v", cfg.Gold.TroyOunceToGram)
	}
	if cfg.Provider.Primary != "mock" {
		t.Fatalf("Provider.Primary = %q", cfg.Provider.Primary)
	}
	if cfg.JWT.Secret == "" || len(cfg.JWT.Secret) < 32 {
		t.Fatal("JWT secret must be at least 32 chars")
	}
	if cfg.Server.ReadTimeout != 10*time.Second {
		t.Fatalf("ReadTimeout = %v", cfg.Server.ReadTimeout)
	}
	if cfg.Server.DrainDelay != 5*time.Second {
		t.Fatalf("DrainDelay = %v", cfg.Server.DrainDelay)
	}
	if cfg.Server.ShutdownTimeout != 75*time.Second {
		t.Fatalf("ShutdownTimeout = %v", cfg.Server.ShutdownTimeout)
	}
}

func TestLoadAndroidAssetLinkFingerprintsFromEnv(t *testing.T) {
	clearAppEnv(t)
	t.Setenv("APP_OAUTH_ANDROID_PACKAGE_NAME", "com.talhacan.tscforex")
	t.Setenv("APP_OAUTH_ANDROID_SHA256", "DEBUG_SHA256,PLAY_SHA256")

	m, err := Load("")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	cfg := m.Get()
	if cfg.OAuth.AndroidPackageName != "com.talhacan.tscforex" {
		t.Fatalf("AndroidPackageName = %q", cfg.OAuth.AndroidPackageName)
	}
	if len(cfg.OAuth.AndroidSHA256) != 2 ||
		cfg.OAuth.AndroidSHA256[0] != "DEBUG_SHA256" ||
		cfg.OAuth.AndroidSHA256[1] != "PLAY_SHA256" {
		t.Fatalf("AndroidSHA256 = %#v", cfg.OAuth.AndroidSHA256)
	}
}

func TestLoadFromYAML(t *testing.T) {
	clearAppEnv(t)

	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := `
app:
  name: tsc_forex
  environment: staging
server:
  port: 9090
  read_timeout: 5s
  write_timeout: 5s
  shutdown_timeout: 10s
  mode: release
database:
  host: db
  port: 5432
  user: u
  password: p
  name: n
  ssl_mode: disable
  max_open_conns: 10
redis:
  addr: redis:6379
  pubsub_channel: gold:gram:updates
  pool_size: 10
provider:
  primary: mock
  reconnect_initial: 1s
  reconnect_max: 10s
  health_interval: 5s
  stale_threshold: 30s
jwt:
  secret: "abcdefghijklmnopqrstuvwxyz012345"
  issuer: test
  access_token_ttl: 15m
  refresh_token_ttl: 24h
websocket:
  path: /ws
  write_wait: 10s
  pong_wait: 60s
  ping_period: 54s
  max_message_size: 4096
  send_buffer_size: 64
cache:
  latest_price_ttl: 30s
  latest_price_key: gold:gram:latest
gold:
  troy_ounce_to_gram: 31.1034768
  history_workers: 2
  history_queue_size: 100
log:
  level: debug
  encoding: console
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	m, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	cfg := m.Get()
	if cfg.App.Environment != "staging" {
		t.Fatalf("environment = %q", cfg.App.Environment)
	}
	if cfg.Server.Port != 9090 {
		t.Fatalf("port = %d", cfg.Server.Port)
	}
	if cfg.Server.Addr() != "0.0.0.0:9090" {
		t.Fatalf("Addr = %q", cfg.Server.Addr())
	}
}

func TestLoadRejectsInvalidJWT(t *testing.T) {
	clearAppEnv(t)

	dir := t.TempDir()
	path := filepath.Join(dir, "bad.yaml")
	content := `
app:
  name: tsc_forex
  environment: development
server:
  port: 8080
  read_timeout: 10s
  write_timeout: 10s
  shutdown_timeout: 15s
  mode: debug
database:
  host: localhost
  port: 5432
  user: u
  name: n
  ssl_mode: disable
  max_open_conns: 1
redis:
  addr: localhost:6379
  pubsub_channel: ch
  pool_size: 1
provider:
  primary: mock
  reconnect_initial: 1s
  reconnect_max: 10s
  health_interval: 5s
  stale_threshold: 30s
jwt:
  secret: "short"
  issuer: test
  access_token_ttl: 15m
  refresh_token_ttl: 24h
websocket:
  path: /ws
  write_wait: 10s
  pong_wait: 60s
  ping_period: 54s
  max_message_size: 4096
  send_buffer_size: 64
cache:
  latest_price_ttl: 30s
  latest_price_key: k
gold:
  troy_ounce_to_gram: 31.1034768
  history_workers: 1
  history_queue_size: 10
log:
  level: info
  encoding: json
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := Load(path); err == nil {
		t.Fatal("expected validation error for short JWT secret")
	}
}

func TestEnvOverride(t *testing.T) {
	clearAppEnv(t)
	t.Setenv("APP_SERVER_PORT", "18080")
	t.Setenv("APP_PROVIDER_PRIMARY", "mock")

	m, err := Load("")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if m.Get().Server.Port != 18080 {
		t.Fatalf("port = %d, want 18080 from env", m.Get().Server.Port)
	}
}

func TestLoadRejectsShutdownBudgetShorterThanRequestBudget(t *testing.T) {
	clearAppEnv(t)
	t.Setenv("APP_SERVER_WRITE_TIMEOUT", "80s")
	t.Setenv("APP_SERVER_SHUTDOWN_TIMEOUT", "75s")

	if _, err := Load(""); err == nil {
		t.Fatal("expected validation error for undersized shutdown budget")
	}
}

func TestOnChangeReload(t *testing.T) {
	clearAppEnv(t)

	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	write := func(port int) {
		content := `
app:
  name: tsc_forex
  environment: development
server:
  port: ` + strconv.Itoa(port) + `
  read_timeout: 10s
  write_timeout: 10s
  shutdown_timeout: 15s
  mode: debug
database:
  host: localhost
  port: 5432
  user: u
  name: n
  ssl_mode: disable
  max_open_conns: 1
redis:
  addr: localhost:6379
  pubsub_channel: ch
  pool_size: 1
provider:
  primary: mock
  reconnect_initial: 1s
  reconnect_max: 10s
  health_interval: 5s
  stale_threshold: 30s
jwt:
  secret: "abcdefghijklmnopqrstuvwxyz012345"
  issuer: test
  access_token_ttl: 15m
  refresh_token_ttl: 24h
websocket:
  path: /ws
  write_wait: 10s
  pong_wait: 60s
  ping_period: 54s
  max_message_size: 4096
  send_buffer_size: 64
cache:
  latest_price_ttl: 30s
  latest_price_key: k
gold:
  troy_ounce_to_gram: 31.1034768
  history_workers: 1
  history_queue_size: 10
log:
  level: info
  encoding: json
`
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	write(8080)
	m, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	changed := make(chan int, 1)
	m.OnChange(func(_, newCfg *Config) {
		changed <- newCfg.Server.Port
	})

	write(9091)
	if err := m.Reload(); err != nil {
		t.Fatalf("Reload: %v", err)
	}
	select {
	case port := <-changed:
		if port != 9091 {
			t.Fatalf("onChange port = %d", port)
		}
	default:
		t.Fatal("expected OnChange callback")
	}
	if m.Get().Server.Port != 9091 {
		t.Fatalf("after reload port = %d", m.Get().Server.Port)
	}
}

// clearAppEnv removes APP_* overrides that would leak across tests via AutomaticEnv.
func clearAppEnv(t *testing.T) {
	t.Helper()
	for _, e := range os.Environ() {
		if len(e) > 4 && e[:4] == "APP_" {
			key := e
			for i := 0; i < len(e); i++ {
				if e[i] == '=' {
					key = e[:i]
					break
				}
			}
			t.Setenv(key, "")
			_ = os.Unsetenv(key)
		}
	}
}
