package api

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/tsc-forex/tsc_forex/internal/service"
)

// GamificationAwarder lets other handlers (watchlist, alerts) grant XP for
// engagement actions without depending on the concrete service.
type GamificationAwarder interface {
	Award(ctx context.Context, userID uuid.UUID, action string)
}

// GamificationHandler exposes the streak/XP/badge/leaderboard endpoints.
type GamificationHandler struct {
	svc *service.GamificationService
}

// NewGamificationHandler builds the handler.
func NewGamificationHandler(svc *service.GamificationService) *GamificationHandler {
	return &GamificationHandler{svc: svc}
}

// Get returns the caller's full gamification profile.
func (h *GamificationHandler) Get(c *gin.Context) {
	uid, ok := currentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "unauthorized"})
		return
	}
	p, err := h.svc.Profile(c.Request.Context(), uid)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, ErrorResponse{Error: "gamification unavailable"})
		return
	}
	c.JSON(http.StatusOK, p)
}

// CheckIn performs the daily check-in and returns the updated profile.
func (h *GamificationHandler) CheckIn(c *gin.Context) {
	uid, ok := currentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "unauthorized"})
		return
	}
	res, err := h.svc.CheckIn(c.Request.Context(), uid)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, ErrorResponse{Error: "gamification unavailable"})
		return
	}
	c.JSON(http.StatusOK, res)
}

// Leaderboard returns the weekly top entries plus the caller's rank.
func (h *GamificationHandler) Leaderboard(c *gin.Context) {
	uid, ok := currentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "unauthorized"})
		return
	}
	limit := 50
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			limit = n
		}
	}
	entries, me, err := h.svc.Leaderboard(c.Request.Context(), uid, limit)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, ErrorResponse{Error: "gamification unavailable"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"entries": entries, "me": me})
}
