package config

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/fsnotify/fsnotify"
	"github.com/go-playground/validator/v10"
	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

// ChangeHandler is invoked after a successful hot-reload.
type ChangeHandler func(old, new *Config)

// Manager owns a validated, thread-safe configuration snapshot.
type Manager struct {
	v        *viper.Viper
	validate *validator.Validate
	current  atomic.Pointer[Config]
	mu       sync.Mutex
	onChange []ChangeHandler
}

// Load reads configuration from (in order of precedence, highest last):
//  1. defaults
//  2. config file (yaml/json/toml)
//  3. .env file (optional)
//  4. environment variables (APP_*, nested via underscore)
func Load(configPath string) (*Manager, error) {
	_ = godotenv.Load() // ignore missing .env

	v := viper.New()
	setDefaults(v)

	v.SetEnvPrefix("APP")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		v.SetConfigName("config")
		v.SetConfigType("yaml")
		v.AddConfigPath(".")
		v.AddConfigPath("./configs")
		v.AddConfigPath("/etc/tsc_forex")
	}

	if err := v.ReadInConfig(); err != nil {
		// Config file is optional when env vars supply everything.
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			if !os.IsNotExist(err) {
				// Explicit path that failed to open is fatal.
				if configPath != "" {
					return nil, fmt.Errorf("read config file: %w", err)
				}
			}
		}
	}

	m := &Manager{
		v:        v,
		validate: validator.New(),
	}

	cfg, err := m.unmarshalAndValidate()
	if err != nil {
		return nil, err
	}
	m.current.Store(cfg)

	return m, nil
}

// Get returns the current immutable snapshot. Safe for concurrent use.
func (m *Manager) Get() *Config {
	return m.current.Load()
}

// OnChange registers a handler for successful reloads.
func (m *Manager) OnChange(h ChangeHandler) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.onChange = append(m.onChange, h)
}

// Watch enables fsnotify-based hot reload of the config file.
// Non-secret runtime knobs (timeouts, rate limits, log level) can change
// without restart. Secrets (JWT, DB password) should still prefer restart
// or secret-manager rotation in production.
func (m *Manager) Watch() {
	m.v.OnConfigChange(func(_ fsnotify.Event) {
		old := m.Get()
		cfg, err := m.unmarshalAndValidate()
		if err != nil {
			// Keep previous snapshot; invalid reload must not crash the process.
			fmt.Fprintf(os.Stderr, "config reload rejected: %v\n", err)
			return
		}
		m.current.Store(cfg)

		m.mu.Lock()
		handlers := append([]ChangeHandler(nil), m.onChange...)
		m.mu.Unlock()
		for _, h := range handlers {
			h(old, cfg)
		}
	})
	m.v.WatchConfig()
}

// Reload forces a re-read (useful in tests).
func (m *Manager) Reload() error {
	if err := m.v.ReadInConfig(); err != nil {
		return fmt.Errorf("reload config: %w", err)
	}
	old := m.Get()
	cfg, err := m.unmarshalAndValidate()
	if err != nil {
		return err
	}
	m.current.Store(cfg)

	m.mu.Lock()
	handlers := append([]ChangeHandler(nil), m.onChange...)
	m.mu.Unlock()
	for _, h := range handlers {
		h(old, cfg)
	}
	return nil
}

func (m *Manager) unmarshalAndValidate() (*Config, error) {
	var cfg Config
	if err := m.v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}
	if err := m.validate.Struct(&cfg); err != nil {
		return nil, fmt.Errorf("validate config: %w", err)
	}
	return &cfg, nil
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("app.name", "tsc_forex")
	v.SetDefault("app.environment", "development")

	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("server.port", 8080)
	v.SetDefault("server.read_timeout", "10s")
	v.SetDefault("server.write_timeout", "70s")
	v.SetDefault("server.shutdown_timeout", "15s")
	v.SetDefault("server.mode", "debug")

	v.SetDefault("database.host", "localhost")
	v.SetDefault("database.port", 5432)
	v.SetDefault("database.user", "tsc_forex")
	v.SetDefault("database.password", "tsc_forex")
	v.SetDefault("database.name", "tsc_forex")
	v.SetDefault("database.ssl_mode", "disable")
	v.SetDefault("database.max_open_conns", 25)
	v.SetDefault("database.max_idle_conns", 10)
	v.SetDefault("database.conn_max_lifetime", "30m")
	v.SetDefault("database.conn_max_idle_time", "5m")

	v.SetDefault("redis.addr", "localhost:6379")
	v.SetDefault("redis.password", "")
	v.SetDefault("redis.db", 0)
	v.SetDefault("redis.pool_size", 20)
	v.SetDefault("redis.min_idle_conns", 5)
	v.SetDefault("redis.dial_timeout", "3s")
	v.SetDefault("redis.read_timeout", "2s")
	v.SetDefault("redis.write_timeout", "2s")
	v.SetDefault("redis.pubsub_channel", "gold:gram:updates")

	v.SetDefault("nats.enabled", false)
	v.SetDefault("nats.url", "nats://localhost:4222")
	v.SetDefault("nats.subject", "gold.gram.updates")

	// Social login (optional). Empty by default; production sets the accepted
	// Google/Apple client IDs via config file or APP_OAUTH_* env vars.
	v.SetDefault("oauth.google_client_ids", []string{})
	v.SetDefault("oauth.apple_bundle_ids", []string{})
	v.SetDefault("oauth.apple_domain_association", "")
	v.SetDefault("oauth.apple_revoke_client_id", "talhacan.tscforex")
	v.SetDefault("oauth.apple_key_id", "")
	v.SetDefault("oauth.apple_private_key_path", "")

	// Safe default when no config file/env is present (tests, bare runs): the
	// mock provider needs no network. Production sets yahoo via config/ConfigMap.
	v.SetDefault("provider.primary", "mock")
	v.SetDefault("provider.fallbacks", []string{"rest", "synthetic"})
	v.SetDefault("provider.reconnect_initial", "1s")
	v.SetDefault("provider.reconnect_max", "30s")
	v.SetDefault("provider.health_interval", "5s")
	v.SetDefault("provider.stale_threshold", "30s")
	v.SetDefault("provider.mock.enabled", true)
	v.SetDefault("provider.mock.tick_interval", "500ms")
	v.SetDefault("provider.mock.xauusd_base", 4060.0)
	v.SetDefault("provider.mock.usdtry_base", 47.0)
	v.SetDefault("provider.mock.jitter_percent", 0.05)

	v.SetDefault("provider.synthetic.enabled", true)
	v.SetDefault("provider.synthetic.tick_interval", "5s")
	v.SetDefault("provider.synthetic.xauusd_base", 4060.0)
	v.SetDefault("provider.synthetic.usdtry_base", 47.0)
	v.SetDefault("provider.synthetic.eurtry_base", 53.6)
	v.SetDefault("provider.synthetic.gbptry_base", 63.0)
	v.SetDefault("provider.synthetic.jpytry_base", 0.29)
	v.SetDefault("provider.synthetic.jitter_percent", 0.03)

	v.SetDefault("provider.rest.enabled", true)
	v.SetDefault("provider.rest.poll_interval", "2s")
	v.SetDefault("provider.rest.fx_base_url", "https://open.er-api.com/v6/latest/USD")
	v.SetDefault("provider.rest.gold_url", "https://api.gold-api.com/price/XAU")
	v.SetDefault("provider.rest.crypto_base_url", "https://api.binance.com/api/v3/ticker/price")
	v.SetDefault("provider.rest.timeout", "4s")

	v.SetDefault("jwt.secret", "CHANGE_ME_TO_A_LONG_RANDOM_SECRET_KEY_32+")
	v.SetDefault("jwt.issuer", "tsc_forex")
	v.SetDefault("jwt.access_token_ttl", "15m")
	v.SetDefault("jwt.refresh_token_ttl", "168h")

	v.SetDefault("websocket.path", "/ws")
	v.SetDefault("websocket.write_wait", "10s")
	v.SetDefault("websocket.pong_wait", "60s")
	v.SetDefault("websocket.ping_period", "54s")
	v.SetDefault("websocket.max_message_size", 4096)
	v.SetDefault("websocket.send_buffer_size", 256)
	v.SetDefault("websocket.enable_compression", true)
	v.SetDefault("websocket.check_origin", false)

	v.SetDefault("cache.latest_price_ttl", "60s")
	v.SetDefault("cache.latest_price_key", "gold:gram:latest")

	v.SetDefault("gold.troy_ounce_to_gram", 31.1034768)
	v.SetDefault("gold.persist_every", true)
	v.SetDefault("gold.history_workers", 4)
	v.SetDefault("gold.history_queue_size", 4096)

	v.SetDefault("metrics.enabled", true)
	v.SetDefault("metrics.path", "/metrics")

	v.SetDefault("tracing.enabled", false)
	v.SetDefault("tracing.endpoint", "localhost:4317")
	v.SetDefault("tracing.service_name", "tsc_forex")
	v.SetDefault("tracing.sample_rate", 0.1)

	v.SetDefault("cors.allowed_origins", []string{"*"})
	v.SetDefault("cors.allowed_methods", []string{"GET", "POST", "OPTIONS"})
	v.SetDefault("cors.allowed_headers", []string{"Authorization", "Content-Type", "X-Request-ID"})
	v.SetDefault("cors.allow_credentials", false)
	v.SetDefault("cors.max_age", 86400)

	v.SetDefault("rate_limit.enabled", true)
	v.SetDefault("rate_limit.requests", 100)
	v.SetDefault("rate_limit.window_sec", 60)

	v.SetDefault("notify.alert_cooldown", "60s")
	v.SetDefault("notify.apns_enabled", false)
	v.SetDefault("notify.apns_topic", "talhacan.tscforex")
	v.SetDefault("notify.apns_key_id", "")
	v.SetDefault("notify.apns_team_id", "")
	v.SetDefault("notify.apns_key_path", "")
	v.SetDefault("notify.apns_sandbox", true)
	v.SetDefault("notify.fcm_enabled", false)
	// These must be registered so AutomaticEnv binds APP_NOTIFY_FCM_* even when
	// they are absent from config.yaml (e.g. in the container image).
	v.SetDefault("notify.fcm_project_id", "")
	v.SetDefault("notify.fcm_credential_path", "")

	v.SetDefault("storage.enabled", false)
	v.SetDefault("storage.endpoint", "minio:9000")
	v.SetDefault("storage.bucket", "avatars")
	v.SetDefault("storage.use_ssl", false)
	v.SetDefault("storage.region", "us-east-1")

	v.SetDefault("ai.enabled", false)
	v.SetDefault("ai.api_key", "")
	v.SetDefault("ai.model", "gpt-5.6-sol")
	v.SetDefault("ai.base_url", "https://api.openai.com/v1")
	v.SetDefault("ai.timeout", "55s")
	v.SetDefault("ai.max_output_tokens", 8000)
	v.SetDefault("ai.reasoning_effort", "high")
	v.SetDefault("ai.web_search", true)
	v.SetDefault("ai.search_context", "high")
	v.SetDefault("ai.max_tool_calls", 6)

	v.SetDefault("billing.enabled", false)
	v.SetDefault("billing.bundle_id", "talhacan.tscforex")
	// TestFlight and App Review transactions use Apple's Sandbox environment.
	// They remain isolated by the verified environment carried in the JWS.
	v.SetDefault("billing.allow_sandbox", true)
	// Sandbox/TestFlight grants are intentionally short lived even when the
	// signed transaction carries a much later test expiry. A fresh Apple JWS
	// extends the window; copied stale Sandbox JWS cannot unlock production
	// indefinitely.
	v.SetDefault("billing.sandbox_max_lifetime", "24h")
	v.SetDefault("billing.app_attest_enabled", false)
	v.SetDefault("billing.app_attest_team_id", "")
	v.SetDefault("billing.app_attest_environment", "development")
	v.SetDefault("billing.app_attest_root_ca_path", "")
	v.SetDefault("billing.app_attest_bundle_versions", []string{"3", "4"})
	v.SetDefault("billing.app_attest_challenge_ttl", "5m")
	v.SetDefault("billing.device_check_enabled", false)
	v.SetDefault("billing.device_check_key_id", "")
	v.SetDefault("billing.device_check_private_key_path", "")
	v.SetDefault("billing.google_package_name", "com.talhacan.tscforex")
	v.SetDefault("billing.google_credentials_path", "")

	v.SetDefault("webauthn.enabled", false)
	v.SetDefault("webauthn.rp_display_name", "TSC Forex")

	v.SetDefault("log.level", "info")
	v.SetDefault("log.encoding", "json")
	v.SetDefault("log.development", false)
}
