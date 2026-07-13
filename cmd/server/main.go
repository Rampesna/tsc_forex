package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/tsc-forex/tsc_forex/internal/api"
	"github.com/tsc-forex/tsc_forex/internal/cache"
	"github.com/tsc-forex/tsc_forex/internal/calculator"
	"github.com/tsc-forex/tsc_forex/internal/config"
	"github.com/tsc-forex/tsc_forex/internal/database"
	"github.com/tsc-forex/tsc_forex/internal/domain"
	"github.com/tsc-forex/tsc_forex/internal/logger"
	"github.com/tsc-forex/tsc_forex/internal/metrics"
	"github.com/tsc-forex/tsc_forex/internal/provider"
	"github.com/tsc-forex/tsc_forex/internal/repository"
	"github.com/tsc-forex/tsc_forex/internal/routes"
	"github.com/tsc-forex/tsc_forex/internal/service"
	"github.com/tsc-forex/tsc_forex/internal/storage"
	"github.com/tsc-forex/tsc_forex/internal/utils"
	ws "github.com/tsc-forex/tsc_forex/internal/websocket"
	"github.com/tsc-forex/tsc_forex/pkg/eventbus"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "fatal: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	mgrCfg, err := config.Load(os.Getenv("APP_CONFIG_FILE"))
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}
	mgrCfg.Watch()
	cfg := mgrCfg.Get()

	log, err := logger.New(cfg.Log)
	if err != nil {
		return fmt.Errorf("logger: %w", err)
	}
	defer func() { _ = log.Sync() }()

	bus := eventbus.New()
	defer bus.Close()

	m := metrics.New(cfg.App.Name)

	log.Info("tsc_forex boot",
		zap.String("version", utils.Version),
		zap.String("env", cfg.App.Environment),
		zap.String("listen", cfg.Server.Addr()),
		zap.String("provider", cfg.Provider.Primary),
		zap.String("ws_path", cfg.WebSocket.Path),
		zap.Float64("troy_oz_to_gram", cfg.Gold.TroyOunceToGram),
	)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	instanceID := uuid.NewString()
	log.Info("instance id", zap.String("instance", instanceID))

	// --- Provider manager ---
	pm, err := provider.NewManager(
		cfg.Provider,
		provider.DefaultFactory(cfg.Provider),
		provider.WithBus(bus),
		provider.WithLogger(logger.WithProvider(log, cfg.Provider.Primary)),
		provider.WithSymbols(domain.BaseSymbols()...),
	)
	if err != nil {
		return fmt.Errorf("provider manager: %w", err)
	}
	if err := pm.Run(ctx); err != nil {
		return fmt.Errorf("provider run: %w", err)
	}
	defer func() { _ = pm.Close() }()

	// --- WebSocket hub ---
	hub := ws.NewHub(cfg.WebSocket, log)
	go hub.Run()
	defer hub.Close()

	// --- Calculator ---
	calc := calculator.New(
		calculator.WithBus(bus),
		calculator.WithLogger(log),
		calculator.WithTroyOunceToGram(cfg.Gold.TroyOunceToGram),
	)
	go calc.Run(ctx, pm.PriceChannel())

	// --- Multi-instrument engine (all gold products + currencies) ---
	engine := calculator.NewEngine(bus, log)
	engine.Attach()

	// --- Broadcast (single calculation → many clients) ---
	_ = service.NewGoldBroadcaster(bus, hub, log)
	_ = service.NewInstrumentBroadcaster(bus, hub, log)

	// --- Redis (optional; degrade gracefully) ---
	var redisCache *cache.RedisCache
	if rc, err := cache.NewRedisCache(ctx, cfg.Redis, cfg.Cache, log, m); err != nil {
		log.Warn("redis unavailable — running without cache/pubsub", zap.Error(err))
	} else {
		redisCache = rc
		defer func() { _ = redisCache.Close() }()
		_ = service.NewCacheBridge(ctx, bus, redisCache, instanceID, log)
		// Cross-instance: apply updates from OTHER instances to local hub.
		if sub, cleanup, err := redisCache.Subscribe(ctx); err == nil {
			defer cleanup()
			go fanInRedis(ctx, sub, hub, instanceID, log)
		}
		log.Info("redis connected", zap.String("addr", cfg.Redis.Addr))
	}

	// --- Postgres (optional; degrade gracefully) ---
	var (
		db          *gorm.DB
		goldRepo    repository.GoldRepository
		priceRepo   repository.PriceRepository
		authSvc     *service.AuthService
		authH       *api.AuthHandler
		userH       *api.UserHandler
		passkeyH    *api.PasskeyHandler
		instrRepo   repository.PriceRepository
		histWriter  *service.HistoryWriter
		instrWriter *service.InstrumentWriter
		alertEngine *service.AlertEngine
	)
	if gdb, err := database.New(ctx, cfg.Database); err != nil {
		log.Warn("postgres unavailable — running without history/auth", zap.Error(err))
	} else {
		db = gdb
		if err := database.Migrate(db); err != nil {
			log.Warn("migrate failed", zap.Error(err))
		}
		goldRepo = repository.NewGoldRepository(db, m)
		priceRepo = repository.NewPriceRepository(db, m)
		instrRepo = priceRepo
		userRepo := repository.NewUserRepository(db, m)
		refreshRepo := repository.NewRefreshTokenRepository(db, m)
		watchRepo := repository.NewWatchlistRepository(db, m)
		alertRepo := repository.NewAlertRepository(db, m)
		notifRepo := repository.NewNotificationRepository(db, m)
		deviceRepo := repository.NewDeviceTokenRepository(db, m)

		authSvc = service.NewAuthService(userRepo, cfg.JWT).
			WithRefreshStore(refreshRepo).
			WithOAuth(cfg.OAuth)
		authH = api.NewAuthHandler(authSvc)

		// Object storage (avatars): shared MinIO/S3 so every pod sees uploads.
		var avatarStore storage.Store
		if cfg.Storage.Enabled {
			if s, err := storage.New(cfg.Storage); err != nil {
				log.Warn("object storage unavailable — avatars disabled", zap.Error(err))
			} else {
				avatarStore = s
				log.Info("object storage enabled", zap.String("bucket", cfg.Storage.Bucket))
			}
		}
		userH = api.NewUserHandler(watchRepo, alertRepo, notifRepo, deviceRepo, userRepo, avatarStore, log)

		// Push senders, routed by device platform (ios → APNs, android → FCM).
		senders := map[string]service.PushSender{}
		if s, err := service.NewAPNsSender(cfg.Notify, log); err == nil {
			senders["ios"] = s
			log.Info("apns push enabled", zap.String("topic", cfg.Notify.APNsTopic))
		} else if cfg.Notify.APNsEnabled {
			log.Warn("apns push disabled", zap.Error(err))
		}
		if s, err := service.NewFCMSender(cfg.Notify, log); err == nil {
			senders["android"] = s
			log.Info("fcm push enabled", zap.String("project", cfg.Notify.FCMProjectID))
		} else if cfg.Notify.FCMEnabled {
			log.Warn("fcm push disabled", zap.Error(err))
		}
		notifier := service.NewNotifier(notifRepo, deviceRepo, hub, senders, log)
		alertEngine = service.NewAlertEngine(bus, alertRepo, notifier, cfg.Notify.AlertCooldown, log)
		alertEngine.Start(ctx)

		// Passkey (WebAuthn) — optional; enabled when configured.
		credRepo := repository.NewWebAuthnCredentialRepository(db, m)
		if ps, err := service.NewPasskeyService(cfg.WebAuthn, cfg.JWT.Secret, credRepo, authSvc, log); err != nil {
			if !errors.Is(err, service.ErrPasskeyDisabled) {
				log.Warn("passkey disabled", zap.Error(err))
			}
		} else {
			passkeyH = api.NewPasskeyHandler(ps)
			log.Info("passkey enabled", zap.String("rp_id", cfg.WebAuthn.RPID))
		}

		if cfg.Gold.PersistEvery {
			histWriter = service.NewHistoryWriter(bus, goldRepo, cfg.Gold.HistoryWorkers, cfg.Gold.HistoryQueueSize, log)
			histWriter.Start(ctx)
			instrWriter = service.NewInstrumentWriter(bus, priceRepo, cfg.Gold.HistoryWorkers, cfg.Gold.HistoryQueueSize*2, log)
			instrWriter.Start(ctx)
		}
		log.Info("postgres connected", zap.String("host", cfg.Database.Host))
	}

	// --- Metrics updater ---
	go metricsUpdater(ctx, m, hub, pm, calc)

	// --- HTTP ---
	dbPing := func(ctx context.Context) error {
		if db == nil {
			return fmt.Errorf("db not configured")
		}
		sqlDB, err := db.DB()
		if err != nil {
			return err
		}
		return sqlDB.PingContext(ctx)
	}
	var redisPinger api.Pinger
	if redisCache != nil {
		redisPinger = redisCache
	}

	// Authenticated WS connections (via ?token=) receive targeted notifications.
	var wsOpts []ws.HandlerOption
	if authSvc != nil {
		wsOpts = append(wsOpts, ws.WithVerifier(func(token string) (string, bool) {
			claims, err := authSvc.Verify(token)
			if err != nil {
				return "", false
			}
			return claims.UserID, true
		}))
	}

	engine2 := routes.NewEngine(routes.Deps{
		Config:     cfg,
		Hub:        hub,
		WS:         ws.NewHandler(hub, log, wsOpts...),
		Metrics:    m,
		Gold:       api.NewGoldHandler(calc, redisCache, goldRepo, log),
		Instrument: api.NewInstrumentHandler(engine, instrRepo, log),
		System: api.NewSystemHandler(api.SystemDeps{
			Provider:  pm,
			Calc:      calc,
			Redis:     redisPinger,
			DBPing:    dbPingOrNil(db, dbPing),
			ClientsFn: hub.ClientCount,
		}),
		Auth:        authH,
		User:        userH,
		Passkey:     passkeyH,
		AuthService: authSvc,
	})

	srv := &http.Server{
		Addr:         cfg.Server.Addr(),
		Handler:      engine2,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Info("http listening", zap.String("addr", cfg.Server.Addr()))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case <-ctx.Done():
		log.Info("shutdown signal received")
	case err := <-errCh:
		if err != nil {
			return fmt.Errorf("http server: %w", err)
		}
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("http shutdown: %w", err)
	}
	if histWriter != nil {
		histWriter.Stop()
	}
	if instrWriter != nil {
		instrWriter.Stop()
	}
	if alertEngine != nil {
		alertEngine.Stop()
	}
	if db != nil {
		_ = database.Close(db)
	}

	log.Info("shutdown complete",
		zap.Int64("ws_dropped", hub.DroppedMessages()),
		zap.Int64("calculations", calc.CalculationCount()),
		zap.Int64("gold_publishes", calc.PublishCount()),
	)
	return nil
}

func dbPingOrNil(db *gorm.DB, fn func(ctx context.Context) error) func(ctx context.Context) error {
	if db == nil {
		return nil
	}
	return fn
}

// fanInRedis applies price updates published by OTHER instances to this hub.
// Own messages (matching instanceID) are skipped since they were already
// broadcast locally by the GoldBroadcaster.
func fanInRedis(ctx context.Context, sub <-chan cache.PriceMessage, hub *ws.Hub, instanceID string, log *zap.Logger) {
	for {
		select {
		case <-ctx.Done():
			return
		case pm, ok := <-sub:
			if !ok {
				return
			}
			if pm.Instance == instanceID {
				continue
			}
			msg, err := ws.EncodePrice(ws.PricePayload{
				Symbol:   pm.Gold.Symbol,
				Price:    pm.Gold.Price,
				MidPrice: pm.Gold.MidPrice,
				Spread:   pm.Gold.Spread,
				XAUUSD:   pm.Gold.XAUUSD,
				USDTRY:   pm.Gold.USDTRY,
				Provider: pm.Gold.Provider,
			})
			if err != nil {
				log.Warn("fan-in encode failed", zap.Error(err))
				continue
			}
			hub.Broadcast(msg)
		}
	}
}

// metricsUpdater periodically syncs gauge/counter values from runtime state.
func metricsUpdater(ctx context.Context, m *metrics.Metrics, hub *ws.Hub, pm *provider.Manager, calc *calculator.Calculator) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	var lastReconnects, lastSwitches, lastCalcs, lastBroadcasts, lastDropped int64
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.WSClients.Set(float64(hub.ClientCount()))

			if b := hub.BroadcastCount(); b > lastBroadcasts {
				m.WSBroadcastTotal.Add(float64(b - lastBroadcasts))
				lastBroadcasts = b
			}
			if d := hub.DroppedMessages(); d > lastDropped {
				m.WSDroppedTotal.Add(float64(d - lastDropped))
				lastDropped = d
			}
			if r := pm.ReconnectCount(); r > lastReconnects {
				m.ProviderReconnects.WithLabelValues(pm.ActiveName()).Add(float64(r - lastReconnects))
				lastReconnects = r
			}
			if s := pm.SwitchCount(); s > lastSwitches {
				m.ProviderSwitches.Add(float64(s - lastSwitches))
				lastSwitches = s
			}
			if calc != nil {
				if cc := calc.CalculationCount(); cc > lastCalcs {
					m.CalculationsTotal.Add(float64(cc - lastCalcs))
					lastCalcs = cc
				}
			}
		}
	}
}
