package config

import (
	"strconv"
	"time"
)

// Config is the root application configuration.
// Loaded once at boot; hot-reload updates a snapshot via Manager.
type Config struct {
	App       AppConfig       `mapstructure:"app" validate:"required"`
	Server    ServerConfig    `mapstructure:"server" validate:"required"`
	Database  DatabaseConfig  `mapstructure:"database" validate:"required"`
	Redis     RedisConfig     `mapstructure:"redis" validate:"required"`
	NATS      NATSConfig      `mapstructure:"nats"`
	Provider  ProviderConfig  `mapstructure:"provider" validate:"required"`
	JWT       JWTConfig       `mapstructure:"jwt" validate:"required"`
	WebSocket WebSocketConfig `mapstructure:"websocket" validate:"required"`
	Cache     CacheConfig     `mapstructure:"cache" validate:"required"`
	Gold      GoldConfig      `mapstructure:"gold" validate:"required"`
	Metrics   MetricsConfig   `mapstructure:"metrics"`
	Tracing   TracingConfig   `mapstructure:"tracing"`
	CORS      CORSConfig      `mapstructure:"cors"`
	RateLimit RateLimitConfig `mapstructure:"rate_limit"`
	Notify    NotifyConfig    `mapstructure:"notify"`
	Log       LogConfig       `mapstructure:"log" validate:"required"`
}

// NotifyConfig controls the alert/notification subsystem.
type NotifyConfig struct {
	// AlertCooldown is the minimum time between two firings of the same alert.
	AlertCooldown time.Duration `mapstructure:"alert_cooldown"`
	APNsEnabled   bool          `mapstructure:"apns_enabled"`
	APNsTopic     string        `mapstructure:"apns_topic"` // iOS bundle id
	APNsKeyID     string        `mapstructure:"apns_key_id"`
	APNsTeamID    string        `mapstructure:"apns_team_id"`
	APNsKeyPath   string        `mapstructure:"apns_key_path"`
	APNsSandbox   bool          `mapstructure:"apns_sandbox"`
}

type AppConfig struct {
	Name        string `mapstructure:"name" validate:"required"`
	Environment string `mapstructure:"environment" validate:"required,oneof=development staging production"`
}

type ServerConfig struct {
	Host            string        `mapstructure:"host"`
	Port            int           `mapstructure:"port" validate:"required,min=1,max=65535"`
	ReadTimeout     time.Duration `mapstructure:"read_timeout" validate:"required"`
	WriteTimeout    time.Duration `mapstructure:"write_timeout" validate:"required"`
	ShutdownTimeout time.Duration `mapstructure:"shutdown_timeout" validate:"required"`
	Mode            string        `mapstructure:"mode" validate:"oneof=debug release test"`
}

type DatabaseConfig struct {
	Host            string        `mapstructure:"host" validate:"required"`
	Port            int           `mapstructure:"port" validate:"required,min=1,max=65535"`
	User            string        `mapstructure:"user" validate:"required"`
	Password        string        `mapstructure:"password"`
	Name            string        `mapstructure:"name" validate:"required"`
	SSLMode         string        `mapstructure:"ssl_mode" validate:"oneof=disable require verify-full verify-ca"`
	MaxOpenConns    int           `mapstructure:"max_open_conns" validate:"min=1"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns" validate:"min=0"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
	ConnMaxIdleTime time.Duration `mapstructure:"conn_max_idle_time"`
}

type RedisConfig struct {
	Addr          string        `mapstructure:"addr" validate:"required"`
	Password      string        `mapstructure:"password"`
	DB            int           `mapstructure:"db" validate:"min=0"`
	PoolSize      int           `mapstructure:"pool_size" validate:"min=1"`
	MinIdleConns  int           `mapstructure:"min_idle_conns" validate:"min=0"`
	DialTimeout   time.Duration `mapstructure:"dial_timeout"`
	ReadTimeout   time.Duration `mapstructure:"read_timeout"`
	WriteTimeout  time.Duration `mapstructure:"write_timeout"`
	PubSubChannel string        `mapstructure:"pubsub_channel" validate:"required"`
}

type NATSConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	URL     string `mapstructure:"url"`
	Subject string `mapstructure:"subject"`
}

type ProviderConfig struct {
	Primary          string             `mapstructure:"primary" validate:"required"`
	Fallbacks        []string           `mapstructure:"fallbacks"`
	ReconnectInitial time.Duration      `mapstructure:"reconnect_initial" validate:"required"`
	ReconnectMax     time.Duration      `mapstructure:"reconnect_max" validate:"required"`
	HealthInterval   time.Duration      `mapstructure:"health_interval" validate:"required"`
	StaleThreshold   time.Duration      `mapstructure:"stale_threshold" validate:"required"`
	Mock             MockProviderConfig `mapstructure:"mock"`
	Synthetic        SyntheticConfig    `mapstructure:"synthetic"`
	REST             RESTProviderConfig `mapstructure:"rest"`
	TwelveData       APIProviderConfig  `mapstructure:"twelvedata"`
	Finnhub          APIProviderConfig  `mapstructure:"finnhub"`
	Polygon          APIProviderConfig  `mapstructure:"polygon"`
	AlphaVantage     APIProviderConfig  `mapstructure:"alphavantage"`
}

// SyntheticConfig configures the offline multi-leg provider that emits all
// base legs (XAUUSD, USDTRY, EURTRY, GBPTRY, JPYTRY) — used as a always-on
// default so the full instrument catalog is populated without external APIs.
type SyntheticConfig struct {
	Enabled       bool          `mapstructure:"enabled"`
	TickInterval  time.Duration `mapstructure:"tick_interval"`
	XAUUSDBase    float64       `mapstructure:"xauusd_base"`
	USDTRYBase    float64       `mapstructure:"usdtry_base"`
	EURTRYBase    float64       `mapstructure:"eurtry_base"`
	GBPTRYBase    float64       `mapstructure:"gbptry_base"`
	JPYTRYBase    float64       `mapstructure:"jpytry_base"`
	JitterPercent float64       `mapstructure:"jitter_percent"`
}

// RESTProviderConfig configures the real REST polling provider. It polls FX
// (USD/EUR/GBP/JPY vs TRY) and a gold source every PollInterval (default 5s).
type RESTProviderConfig struct {
	Enabled      bool          `mapstructure:"enabled"`
	PollInterval time.Duration `mapstructure:"poll_interval"`
	FXBaseURL    string        `mapstructure:"fx_base_url"`  // e.g. https://open.er-api.com/v6/latest/USD
	GoldURL      string        `mapstructure:"gold_url"`     // XAUUSD source (optional)
	APIKey       string        `mapstructure:"api_key"`
	Timeout      time.Duration `mapstructure:"timeout"`
}

type MockProviderConfig struct {
	Enabled       bool          `mapstructure:"enabled"`
	TickInterval  time.Duration `mapstructure:"tick_interval"`
	XAUUSDBase    float64       `mapstructure:"xauusd_base"`
	USDTRYBase    float64       `mapstructure:"usdtry_base"`
	JitterPercent float64       `mapstructure:"jitter_percent"`
}

type APIProviderConfig struct {
	Enabled bool     `mapstructure:"enabled"`
	APIKey  string   `mapstructure:"api_key"`
	WSURL   string   `mapstructure:"ws_url"`
	RESTURL string   `mapstructure:"rest_url"`
	Symbols []string `mapstructure:"symbols"`
}

type JWTConfig struct {
	Secret          string        `mapstructure:"secret" validate:"required,min=32"`
	Issuer          string        `mapstructure:"issuer" validate:"required"`
	AccessTokenTTL  time.Duration `mapstructure:"access_token_ttl" validate:"required"`
	RefreshTokenTTL time.Duration `mapstructure:"refresh_token_ttl" validate:"required"`
}

type WebSocketConfig struct {
	Path              string        `mapstructure:"path" validate:"required"`
	WriteWait         time.Duration `mapstructure:"write_wait" validate:"required"`
	PongWait          time.Duration `mapstructure:"pong_wait" validate:"required"`
	PingPeriod        time.Duration `mapstructure:"ping_period" validate:"required"`
	MaxMessageSize    int64         `mapstructure:"max_message_size" validate:"min=512"`
	SendBufferSize    int           `mapstructure:"send_buffer_size" validate:"min=1"`
	EnableCompression bool          `mapstructure:"enable_compression"`
	CheckOrigin       bool          `mapstructure:"check_origin"`
}

type CacheConfig struct {
	LatestPriceTTL time.Duration `mapstructure:"latest_price_ttl" validate:"required"`
	LatestPriceKey string        `mapstructure:"latest_price_key" validate:"required"`
}

type GoldConfig struct {
	TroyOunceToGram  float64 `mapstructure:"troy_ounce_to_gram" validate:"required,gt=0"`
	PersistEvery     bool    `mapstructure:"persist_every"`
	HistoryWorkers   int     `mapstructure:"history_workers" validate:"min=1"`
	HistoryQueueSize int     `mapstructure:"history_queue_size" validate:"min=1"`
}

type MetricsConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	Path    string `mapstructure:"path"`
}

type TracingConfig struct {
	Enabled     bool    `mapstructure:"enabled"`
	Endpoint    string  `mapstructure:"endpoint"`
	ServiceName string  `mapstructure:"service_name"`
	SampleRate  float64 `mapstructure:"sample_rate"`
}

type CORSConfig struct {
	AllowedOrigins   []string `mapstructure:"allowed_origins"`
	AllowedMethods   []string `mapstructure:"allowed_methods"`
	AllowedHeaders   []string `mapstructure:"allowed_headers"`
	AllowCredentials bool     `mapstructure:"allow_credentials"`
	MaxAge           int      `mapstructure:"max_age"`
}

type RateLimitConfig struct {
	Enabled   bool `mapstructure:"enabled"`
	Requests  int  `mapstructure:"requests" validate:"min=1"`
	WindowSec int  `mapstructure:"window_sec" validate:"min=1"`
}

type LogConfig struct {
	Level       string `mapstructure:"level" validate:"required,oneof=debug info warn error"`
	Encoding    string `mapstructure:"encoding" validate:"oneof=json console"`
	Development bool   `mapstructure:"development"`
}

// Addr returns host:port for the HTTP server.
func (s ServerConfig) Addr() string {
	host := s.Host
	if host == "" {
		host = "0.0.0.0"
	}
	return host + ":" + strconv.Itoa(s.Port)
}
