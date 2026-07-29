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
	_ "time/tzdata" // embed the IANA tz database (Europe/Istanbul) for change % day boundaries

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/tsc-forex/tsc_forex/internal/api"
	"github.com/tsc-forex/tsc_forex/internal/cache"
	"github.com/tsc-forex/tsc_forex/internal/calculator"
	"github.com/tsc-forex/tsc_forex/internal/config"
	"github.com/tsc-forex/tsc_forex/internal/database"
	"github.com/tsc-forex/tsc_forex/internal/domain"
	"github.com/tsc-forex/tsc_forex/internal/lifecycle"
	"github.com/tsc-forex/tsc_forex/internal/localization"
	"github.com/tsc-forex/tsc_forex/internal/logger"
	"github.com/tsc-forex/tsc_forex/internal/metrics"
	"github.com/tsc-forex/tsc_forex/internal/presence"
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

	ctx, stopSignals := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stopSignals()
	drainer := lifecycle.NewDrainer()

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
	var presenceStore *presence.Store
	if cfg.Admin.Enabled {
		if store, presenceErr := presence.New(ctx, cfg.Redis, cfg.Admin.PresenceTTL); presenceErr != nil {
			log.Warn("cross-pod presence unavailable", zap.Error(presenceErr))
		} else {
			presenceStore = store
			hub.SetPresenceRecorder(store)
			defer func() { _ = store.Close() }()
		}
	}
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

	// --- Candle store (hot path for charts/sparklines/stats) ---
	// Created early so it captures instrument updates from the first tick; the DB
	// repository is attached later once Postgres is ready.
	candleStore := service.NewCandleStore(bus, nil, log)

	// --- Broadcast (single calculation → many clients) ---
	_ = service.NewGoldBroadcaster(bus, hub, log)

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
		db              *gorm.DB
		dbStartupErr    error
		goldRepo        repository.GoldRepository
		priceRepo       repository.PriceRepository
		authSvc         *service.AuthService
		managerAuthSvc  *service.ManagerAuthService
		authH           *api.AuthHandler
		userH           *api.UserHandler
		passkeyH        *api.PasskeyHandler
		gamH            *api.GamificationHandler
		portfolioH      *api.PortfolioHandler
		billingH        *api.BillingHandler
		adminH          *api.AdminHandler
		subscriptionSvc *service.SubscriptionService
		notifier        *service.Notifier
		instrRepo       repository.PriceRepository
		histWriter      *service.HistoryWriter
		instrWriter     *service.InstrumentWriter
		alertEngine     *service.AlertEngine
		instrumentNames *localization.InstrumentLocalizer
		persistenceLead *database.AdvisoryLeader
	)
	if gdb, err := database.New(ctx, cfg.Database); err != nil {
		dbStartupErr = fmt.Errorf("postgres startup: %w", err)
		log.Error("postgres unavailable — pod will remain unready", zap.Error(err))
	} else if err := database.Migrate(gdb); err != nil {
		dbStartupErr = fmt.Errorf("database migration: %w", err)
		log.Error("database migration failed — pod will remain unready", zap.Error(err))
		_ = database.Close(gdb)
	} else {
		db = gdb
		instrumentNames, err = localization.NewInstrumentLocalizer(db)
		if err != nil {
			return fmt.Errorf("instrument localization: %w", err)
		}
		goldRepo = repository.NewGoldRepository(db, m)
		priceRepo = repository.NewPriceRepository(db, m)
		instrRepo = priceRepo
		persistenceLead, err = database.NewAdvisoryLeader(
			db,
			database.PersistenceLeaderLockKey,
			log,
		)
		if err != nil {
			return fmt.Errorf("persistence leader: %w", err)
		}
		persistenceLead.Start(ctx)

		// Attach the candle DB and bootstrap chart history from existing ticks.
		candleRepo := repository.NewCandleRepository(db, m)
		candleStore.SetRepo(candleRepo)
		candleStore.SetWriteGate(persistenceLead.IsLeader)
		candleStore.Bootstrap(ctx)
		// Seed day-open baselines from the compact candles table (fast: one
		// indexed row per instrument) so change % is correct right after a
		// restart, without a multi-second scan of a day of raw ticks.
		if opens := candleStore.SeedDayOpens(ctx, engine.DayStart()); len(opens) > 0 {
			engine.SeedDayOpen(opens, engine.DayKey())
			log.Info("seeded day-open baselines", zap.Int("instruments", len(opens)))
		}
		startTimeSeriesRetention(
			ctx,
			priceRepo,
			goldRepo,
			persistenceLead.IsLeader,
			log,
		)
		userRepo := repository.NewUserRepository(db, m)
		refreshRepo := repository.NewRefreshTokenRepository(db, m)
		watchRepo := repository.NewWatchlistRepository(db, m)
		alertRepo := repository.NewAlertRepository(db, m)
		notifRepo := repository.NewNotificationRepository(db, m)
		deviceRepo := repository.NewDeviceTokenRepository(db, m)

		identityRepo := repository.NewAuthIdentityRepository(db, m)
		authSvc = service.NewAuthService(userRepo, cfg.JWT).
			WithRefreshStore(refreshRepo).
			WithOAuth(cfg.OAuth).
			WithIdentities(identityRepo)
		authH = api.NewAuthHandler(authSvc)
		if cfg.Admin.Enabled {
			managerRepo := repository.NewManagerRepository(db, m)
			managerSessionRepo := repository.NewManagerSessionRepository(db, m)
			if err := service.EnsureBootstrapManager(ctx, db, managerRepo, cfg.Admin); err != nil {
				return fmt.Errorf("manager bootstrap: %w", err)
			}
			managerAuthSvc = service.NewManagerAuthService(managerRepo, managerSessionRepo, cfg.JWT)
		}

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

		var appleVerifier service.AppleSignedDataVerifier
		if cfg.Billing.Enabled {
			verifier, verifierErr := service.NewAppleJWSVerifier()
			if verifierErr != nil {
				log.Error("app store JWS verifier unavailable", zap.Error(verifierErr))
			} else {
				appleVerifier = verifier
			}
		}
		subscriptionSvc = service.NewSubscriptionService(db, cfg.Billing, appleVerifier)
		if googleVerifier, googleErr := service.NewGooglePlayVerifier(cfg.Billing); googleErr != nil {
			log.Warn("google play billing verifier unavailable", zap.Error(googleErr))
		} else if googleVerifier != nil {
			subscriptionSvc.WithGooglePlay(googleVerifier)
			log.Info("google play billing verifier enabled", zap.String("package", cfg.Billing.GooglePackageName))
		}
		billingH = api.NewBillingHandler(subscriptionSvc)

		portfolioRepo := repository.NewPortfolioRepository(db, m)
		portfolioSvc := service.NewPortfolioService(portfolioRepo, engine).WithHistory(candleStore)
		if cfg.Billing.Enabled {
			portfolioSvc.WithSubscriptions(subscriptionSvc)
		}
		portfolioH = api.NewPortfolioHandler(portfolioSvc).WithLocalizer(instrumentNames)
		userH.WithAuthService(authSvc)
		if cfg.Billing.Enabled {
			userH.WithSubscriptions(subscriptionSvc)
		}

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
		notifier = service.NewNotifier(notifRepo, deviceRepo, hub, senders, log)
		alertEngine = service.NewAlertEngine(bus, alertRepo, notifier, cfg.Notify.AlertCooldown, log).
			WithWriteGate(persistenceLead.IsLeader)
		alertEngine.Start(ctx)

		// Gamification: streaks, XP/levels, badges, weekly leaderboard. Reuses
		// the notifier so badge/quest wins arrive as WS + push notifications,
		// and awards XP when the user adds favorites / creates alerts.
		gamRepo := repository.NewGamificationRepository(db, m)
		gamSvc := service.NewGamificationService(gamRepo, notifier, log)
		gamH = api.NewGamificationHandler(gamSvc)
		userH.WithGamification(gamSvc)

		if cfg.Admin.Enabled {
			adminSvc := service.NewAdminService(db, presenceStore, notifier, cfg.Admin, cfg.AI, log)
			adminSvc.StartCampaignWorker(ctx)
			adminH = api.NewAdminHandler(managerAuthSvc, adminSvc, cfg.Admin)
			log.Info("admin console enabled", zap.String("email", cfg.Admin.BootstrapEmail))
		}

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
			histWriter = service.NewHistoryWriter(bus, goldRepo, cfg.Gold.HistoryWorkers, cfg.Gold.HistoryQueueSize, log).
				WithWriteGate(persistenceLead.IsLeader)
			histWriter.Start(ctx)
			instrWriter = service.NewInstrumentWriter(bus, priceRepo, cfg.Gold.HistoryWorkers, cfg.Gold.HistoryQueueSize*2, log).
				WithWriteGate(persistenceLead.IsLeader)
			instrWriter.Start(ctx)
		}
		log.Info("postgres connected", zap.String("host", cfg.Database.Host))
	}
	if dbStartupErr != nil && cfg.App.Environment == "production" {
		// There is no in-process reconnect path for a database that was absent
		// during boot. Exit so Kubernetes restarts the container until Postgres
		// and the serialized migration are actually available.
		return dbStartupErr
	}
	_ = service.NewInstrumentBroadcaster(bus, hub, log, instrumentNames)

	// Candle flush + retention loops (no-op without a DB, but the in-memory
	// aggregation for live charts/sparklines/stats runs regardless).
	candleStore.Start(ctx)

	// --- Metrics updater ---
	go metricsUpdater(ctx, m, hub, pm, calc)

	// --- HTTP ---
	readinessCheck := func(ctx context.Context) error {
		if err := drainer.ReadinessError(); err != nil {
			return err
		}
		if dbStartupErr != nil {
			return dbStartupErr
		}
		if err := database.CheckReady(ctx, db); err != nil {
			return err
		}
		if subscriptionSvc != nil {
			return subscriptionSvc.ReadinessError()
		}
		return nil
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

	aiHandler := api.NewMarketAIHandler(cfg.AI, engine, candleStore)
	if subscriptionSvc != nil && cfg.Billing.Enabled {
		aiHandler.WithSubscriptions(subscriptionSvc)
	}
	engine2 := routes.NewEngine(routes.Deps{
		Config:  cfg,
		Hub:     hub,
		WS:      ws.NewHandler(hub, log, wsOpts...),
		Metrics: m,
		Gold:    api.NewGoldHandler(calc, redisCache, goldRepo, log),
		Instrument: api.NewInstrumentHandler(
			engine, instrRepo, candleStore, cfg.Provider.REST.CryptoBaseURL, log,
		).WithLocalizer(instrumentNames),
		System: api.NewSystemHandler(api.SystemDeps{
			Provider:  pm,
			Calc:      calc,
			Redis:     redisPinger,
			DBPing:    readinessCheck,
			ClientsFn: hub.ClientCount,
		}),
		Auth:               authH,
		User:               userH,
		Passkey:            passkeyH,
		Gamification:       gamH,
		Portfolio:          portfolioH,
		AI:                 aiHandler,
		Billing:            billingH,
		Admin:              adminH,
		AuthService:        authSvc,
		ManagerAuthService: managerAuthSvc,
		Ready:              readinessCheck,
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

	var serverErr error
	select {
	case <-ctx.Done():
		log.Info("shutdown signal received", zap.Error(ctx.Err()))
	case err, ok := <-errCh:
		if !ok || err == nil {
			err = errors.New("listener stopped without shutdown request")
		}
		log.Error("http server stopped unexpectedly", zap.Error(err))
		stopSignals()
		serverErr = fmt.Errorf("http server: %w", err)
	}

	// Kubernetes-aware phase 1: fail readiness and reject new long-lived
	// sessions before closing listeners. Existing requests/connections remain
	// available during the short propagation window.
	drainer.BeginDrain()
	hub.BeginDrain()
	srv.SetKeepAlivesEnabled(false)
	log.Info("shutdown drain started",
		zap.Duration("drain_delay", cfg.Server.DrainDelay),
		zap.Duration("shutdown_timeout", cfg.Server.ShutdownTimeout),
		zap.Int64("ws_clients", hub.ClientCount()),
	)

	// Restore the default signal behavior after the first signal: a second
	// SIGTERM/SIGINT is an explicit request to force termination.
	stopSignals()

	if serverErr == nil {
		if err := lifecycle.WaitDrainDelay(context.Background(), cfg.Server.DrainDelay); err != nil {
			log.Warn("traffic drain delay interrupted", zap.Error(err))
		}
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer shutdownCancel()

	httpShutdown := make(chan error, 1)
	wsShutdown := make(chan error, 1)
	go func() { httpShutdown <- srv.Shutdown(shutdownCtx) }()
	go func() { wsShutdown <- hub.Shutdown(shutdownCtx) }()

	var shutdownErr error
	if err := <-httpShutdown; err != nil {
		log.Error("http graceful shutdown exceeded budget; forcing close", zap.Error(err))
		if closeErr := srv.Close(); closeErr != nil {
			err = errors.Join(err, closeErr)
		}
		shutdownErr = errors.Join(shutdownErr, fmt.Errorf("http shutdown: %w", err))
	}
	if err := <-wsShutdown; err != nil {
		log.Error("websocket graceful shutdown exceeded budget", zap.Error(err))
		shutdownErr = errors.Join(shutdownErr, fmt.Errorf("websocket shutdown: %w", err))
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
	cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
	if err := candleStore.Wait(cleanupCtx); err != nil {
		log.Error("candle workers did not stop before cleanup deadline", zap.Error(err))
		shutdownErr = errors.Join(shutdownErr, fmt.Errorf("candle shutdown: %w", err))
	}
	if persistenceLead != nil {
		if err := persistenceLead.Wait(cleanupCtx); err != nil {
			log.Error("persistence leader did not stop before cleanup deadline", zap.Error(err))
			shutdownErr = errors.Join(shutdownErr, fmt.Errorf("persistence leader shutdown: %w", err))
		}
	}
	cleanupCancel()
	if db != nil {
		if err := database.Close(db); err != nil {
			shutdownErr = errors.Join(shutdownErr, fmt.Errorf("database close: %w", err))
		}
	}

	log.Info("shutdown complete",
		zap.Int64("ws_dropped", hub.DroppedMessages()),
		zap.Int64("calculations", calc.CalculationCount()),
		zap.Int64("gold_publishes", calc.PublishCount()),
	)
	return errors.Join(serverErr, shutdownErr)
}

const (
	liveTickRetention = 72 * time.Hour
	goldRawRetention  = 30 * 24 * time.Hour
	retentionInterval = 30 * time.Minute
)

// startTimeSeriesRetention smooths deletion work into bounded, frequent batches.
// Only the elected persistence leader performs it, so five API replicas never
// launch competing multi-million-row DELETE transactions.
func startTimeSeriesRetention(
	ctx context.Context,
	prices repository.PriceRepository,
	gold repository.GoldRepository,
	canWrite func() bool,
	log *zap.Logger,
) {
	if prices == nil || gold == nil || canWrite == nil {
		return
	}
	go func() {
		// Give candle bootstrap and the initial readiness wave priority.
		timer := time.NewTimer(30 * time.Minute)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
		}

		ticker := time.NewTicker(retentionInterval)
		defer ticker.Stop()
		run := func() {
			if !canWrite() {
				return
			}
			cctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
			defer cancel()
			if n, err := prices.PruneOlderThan(cctx, time.Now().Add(-liveTickRetention)); err != nil {
				log.Warn("price tick retention failed", zap.Error(err))
			} else if n > 0 {
				log.Info("pruned old price ticks", zap.Int64("rows", n))
			}
			if n, err := gold.PruneOlderThan(cctx, time.Now().Add(-goldRawRetention)); err != nil {
				log.Warn("gold history retention failed", zap.Error(err))
			} else if n > 0 {
				log.Info("pruned old gold prices", zap.Int64("rows", n))
			}
		}

		run()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				run()
			}
		}
	}()
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
