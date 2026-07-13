package routes

import (
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

	Gold       *api.GoldHandler
	Instrument *api.InstrumentHandler
	System     *api.SystemHandler
	Auth       *api.AuthHandler
	User       *api.UserHandler // protected per-user features; may be nil

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
	r.GET("/readyz", func(c *gin.Context) { c.String(200, "ready") })

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
		}

		if d.Auth != nil {
			v1.POST("/auth/register", d.Auth.Register)
			v1.POST("/auth/login", d.Auth.Login)
			v1.POST("/auth/google", d.Auth.Google)
			v1.POST("/auth/refresh", d.Auth.Refresh)
			v1.POST("/auth/logout", d.Auth.Logout)
		}

		// Protected per-user features (watchlist, alerts, notifications, devices).
		if d.User != nil && d.AuthService != nil {
			authed := v1.Group("")
			authed.Use(middleware.JWTAuth(d.AuthService))
			{
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
			}
		}
	}

	return r
}
