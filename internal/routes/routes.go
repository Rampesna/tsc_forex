package routes

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/tsc-forex/tsc_forex/docs"
	"github.com/tsc-forex/tsc_forex/internal/api"
	"github.com/tsc-forex/tsc_forex/internal/config"
	"github.com/tsc-forex/tsc_forex/internal/metrics"
	"github.com/tsc-forex/tsc_forex/internal/middleware"
	"github.com/tsc-forex/tsc_forex/internal/service"
	ws "github.com/tsc-forex/tsc_forex/internal/websocket"
)

// Deps are the collaborators the HTTP layer needs.
type Deps struct {
	Config  *config.Config
	Hub     *ws.Hub
	WS      *ws.Handler
	Metrics *metrics.Metrics

	Gold         *api.GoldHandler
	Instrument   *api.InstrumentHandler
	System       *api.SystemHandler
	Auth         *api.AuthHandler
	User         *api.UserHandler         // protected per-user features; may be nil
	Passkey      *api.PasskeyHandler      // WebAuthn; may be nil
	Gamification *api.GamificationHandler // streak/XP/badges/leaderboard; may be nil
	Portfolio    *api.PortfolioHandler    // transaction ledger + analytics; may be nil
	AI           *api.MarketAIHandler     // server-side market briefs; may be nil
	Billing      *api.BillingHandler      // App Store plans/entitlements; may be nil
	Ready        func(context.Context) error

	AuthService *service.AuthService // for protected-route middleware; may be nil
}

// NewEngine builds the Gin engine with all middleware and routes.
func NewEngine(d Deps) *gin.Engine {
	gin.SetMode(d.Config.Server.Mode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.RequestID())
	r.Use(middleware.SecurityHeaders())
	r.Use(middleware.CORS(d.Config.CORS.AllowedOrigins))
	if d.Metrics != nil {
		r.Use(middleware.Metrics(d.Metrics))
	}

	// WebSocket endpoint (no-cache, no rate limit — long-lived connection).
	path := d.Config.WebSocket.Path
	if path == "" {
		path = "/ws"
	}
	r.GET(path, middleware.NoCache(), d.WS.ServeWS)

	// Kubernetes probes (unauthenticated, no rate limit).
	r.GET("/livez", func(c *gin.Context) { c.String(200, "ok") })
	r.GET("/readyz", func(c *gin.Context) {
		if d.Ready == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not_ready"})
			return
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()
		if err := d.Ready(ctx); err != nil {
			c.Header("Retry-After", "5")
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not_ready"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ready"})
	})

	// Well-known association files for passkeys / Google credential sharing.
	registerWellKnown(r, d.Config)

	// Swagger UI + OpenAPI spec.
	docs.Register(r)

	// Prometheus metrics.
	if d.Metrics != nil && d.Config.Metrics.Enabled {
		mp := d.Config.Metrics.Path
		if mp == "" {
			mp = "/metrics"
		}
		r.GET(mp, gin.WrapH(promhttp.HandlerFor(d.Metrics.Registry(), promhttp.HandlerOpts{})))
	}

	v1 := r.Group("/api/v1")
	v1.Use(middleware.NoCache())
	if d.Config.RateLimit.Enabled {
		v1.Use(middleware.RateLimit(d.Config.RateLimit.Requests, d.Config.RateLimit.WindowSec))
	}
	{
		v1.GET("/health", d.System.Health)
		v1.GET("/version", d.System.Version)
		v1.GET("/providers", d.System.Providers)
		if d.Billing != nil {
			v1.GET("/billing/plans", d.Billing.Plans)
			v1.POST("/billing/apple/notifications", d.Billing.AppleNotifications)
		}

		v1.GET("/gold/current", d.Gold.Current)
		v1.GET("/gold/history", d.Gold.History)

		// Multi-instrument catalog (all gold products + currencies).
		if d.Instrument != nil {
			v1.GET("/instruments", d.Instrument.Catalog)
			v1.GET("/instruments/prices", d.Instrument.Prices)
			v1.GET("/instruments/sparklines", d.Instrument.Sparklines)
			v1.GET("/instruments/:code", d.Instrument.Get)
			v1.GET("/instruments/:code/history", d.Instrument.History)
			v1.GET("/instruments/:code/stats", d.Instrument.Stats)
			v1.GET("/instruments/:code/detail", d.Instrument.Detail)
		}

		if d.Auth != nil {
			v1.POST("/auth/register", d.Auth.Register)
			v1.POST("/auth/login", d.Auth.Login)
			v1.POST("/auth/google", d.Auth.Google)
			v1.POST("/auth/apple", d.Auth.Apple)
			v1.POST("/auth/refresh", d.Auth.Refresh)
			v1.POST("/auth/logout", d.Auth.Logout)
		}

		// Passkey (WebAuthn) usernameless login (public begin/finish).
		if d.Passkey != nil {
			v1.POST("/auth/passkey/login/begin", d.Passkey.LoginBegin)
			v1.POST("/auth/passkey/login/finish", d.Passkey.LoginFinish)
		}

		// Protected per-user features (watchlist, alerts, notifications, devices).
		if d.User != nil && d.AuthService != nil {
			authed := v1.Group("")
			authed.Use(middleware.JWTAuth(d.AuthService))
			{
				authed.GET("/me/profile", d.User.Profile)
				authed.PATCH("/me/profile", d.User.UpdateProfile)
				authed.POST("/me/avatar", d.User.UploadAvatar)
				authed.DELETE("/me/account", d.User.DeleteAccount)
				authed.GET("/me/account/deletion-readiness", d.User.AccountDeletionReadiness)

				// Multi-provider account linking (Google + Apple on one account).
				authed.GET("/me/providers", d.Auth.LinkedProviders)
				authed.POST("/me/link/google", d.Auth.LinkGoogle)
				authed.POST("/me/link/apple", d.Auth.LinkApple)

				authed.GET("/me/watchlist", d.User.ListWatchlist)
				authed.POST("/me/watchlist", d.User.AddWatchlist)
				authed.DELETE("/me/watchlist/:code", d.User.RemoveWatchlist)

				authed.GET("/me/alerts", d.User.ListAlerts)
				authed.POST("/me/alerts", d.User.CreateAlert)
				authed.DELETE("/me/alerts/:id", d.User.DeleteAlert)

				authed.GET("/me/notifications", d.User.ListNotifications)
				authed.POST("/me/notifications/:id/read", d.User.MarkNotificationRead)
				authed.DELETE("/me/notifications/:id", d.User.DeleteNotification)
				authed.DELETE("/me/notifications", d.User.ClearNotifications)

				authed.POST("/me/devices", d.User.RegisterDevice)

				if d.Billing != nil {
					authed.GET("/me/billing/status", d.Billing.Status)
					authed.POST("/me/billing/apple/sync", d.Billing.SyncApple)
					authed.POST("/me/device-attestation/challenge", d.Billing.AppAttestChallenge)
					authed.POST("/me/device-attestation/register", d.Billing.RegisterAppAttest)
					authed.GET("/me/ai/research-history", d.Billing.ResearchHistory)
				}

				if d.AI != nil {
					authed.POST("/me/ai/market-insight", d.AI.MarketInsight)
				}

				if d.Portfolio != nil {
					authed.GET("/me/portfolios", d.Portfolio.List)
					authed.POST("/me/portfolios", d.Portfolio.Create)
					authed.GET("/me/portfolios/:portfolioID", d.Portfolio.Get)
					authed.PATCH("/me/portfolios/:portfolioID", d.Portfolio.Update)
					authed.DELETE("/me/portfolios/:portfolioID", d.Portfolio.Delete)
					authed.GET("/me/portfolios/:portfolioID/positions/:code", d.Portfolio.Position)
					authed.GET("/me/portfolios/:portfolioID/transactions", d.Portfolio.Transactions)
					authed.POST("/me/portfolios/:portfolioID/transactions", d.Portfolio.CreateTransaction)
					authed.PATCH("/me/portfolios/:portfolioID/transactions/:transactionID", d.Portfolio.UpdateTransaction)
					authed.DELETE("/me/portfolios/:portfolioID/transactions/:transactionID", d.Portfolio.DeleteTransaction)
					authed.GET("/me/portfolios/:portfolioID/performance", d.Portfolio.Performance)
					authed.POST("/me/portfolios/:portfolioID/health-check", d.Portfolio.HealthCheck)
					authed.POST("/me/portfolios/:portfolioID/export", d.Portfolio.Export)
				}

				// Gamification: daily streak, XP/levels, badges, leaderboard.
				if d.Gamification != nil {
					authed.GET("/me/gamification", d.Gamification.Get)
					authed.POST("/me/gamification/checkin", d.Gamification.CheckIn)
					authed.GET("/me/gamification/leaderboard", d.Gamification.Leaderboard)
					authed.GET("/me/gamification/experience", d.Gamification.Experience)
					authed.POST("/me/gamification/daily-plan", d.Gamification.SaveDailyPlan)
					authed.POST("/me/gamification/quiz-attempts", d.Gamification.SubmitQuiz)
					authed.POST("/me/gamification/discipline/:code", d.Gamification.CompleteDiscipline)
					authed.GET("/me/gamification/season", d.Gamification.Season)
					authed.POST("/me/gamification/paper-trades", d.Gamification.CreatePaperTrade)
					authed.PATCH("/me/gamification/paper-trades/:tradeID/review", d.Gamification.ReviewPaperTrade)
					authed.PATCH("/me/gamification/privacy", d.Gamification.SavePrivacy)
					authed.POST("/me/gamification/study-groups", d.Gamification.CreateStudyGroup)
					authed.POST("/me/gamification/study-groups/join", d.Gamification.JoinStudyGroup)
				}

				// Passkey registration + management for the signed-in user.
				if d.Passkey != nil {
					authed.POST("/me/passkey/register/begin", d.Passkey.RegisterBegin)
					authed.POST("/me/passkey/register/finish", d.Passkey.RegisterFinish)
					authed.GET("/me/passkeys", d.Passkey.List)
					authed.PATCH("/me/passkeys/:id", d.Passkey.Rename)
					authed.DELETE("/me/passkeys/:id", d.Passkey.Delete)
				}
			}
		}
	}

	return r
}

// registerWellKnown serves the Apple App Site Association (passkeys / Sign in
// with Apple) and Android Digital Asset Links (Google credential sharing).
func registerWellKnown(r *gin.Engine, cfg *config.Config) {
	// apple-app-site-association MUST be served without a file extension and as
	// application/json (no redirects) at the RP domain.
	r.GET("/.well-known/apple-app-site-association", func(c *gin.Context) {
		apps := []string{}
		if cfg.OAuth.AppleTeamID != "" {
			for _, bundle := range cfg.OAuth.AppleBundleIDs {
				apps = append(apps, cfg.OAuth.AppleTeamID+"."+bundle)
			}
		}
		c.Header("Content-Type", "application/json")
		c.JSON(http.StatusOK, gin.H{
			"webcredentials": gin.H{"apps": apps},
			"applinks":       gin.H{"apps": []string{}, "details": []any{}},
		})
	})

	// Sign in with Apple (web) domain verification. Apple fetches this exact
	// file from the Services ID's registered domain. Content comes from config
	// (paste the file Apple issues in the Services ID configuration screen).
	r.GET("/.well-known/apple-developer-domain-association.txt", func(c *gin.Context) {
		if cfg.OAuth.AppleDomainAssociation == "" {
			c.String(http.StatusNotFound, "not configured")
			return
		}
		c.Header("Content-Type", "text/plain; charset=utf-8")
		c.String(http.StatusOK, cfg.OAuth.AppleDomainAssociation)
	})

	r.GET("/.well-known/assetlinks.json", func(c *gin.Context) {
		out := []any{}
		if cfg.OAuth.AndroidPackageName != "" && len(cfg.OAuth.AndroidSHA256) > 0 {
			out = append(out, gin.H{
				"relation": []string{"delegate_permission/common.get_login_creds"},
				"target": gin.H{
					"namespace":                "android_app",
					"package_name":             cfg.OAuth.AndroidPackageName,
					"sha256_cert_fingerprints": cfg.OAuth.AndroidSHA256,
				},
			})
		}
		c.JSON(http.StatusOK, out)
	})
}
