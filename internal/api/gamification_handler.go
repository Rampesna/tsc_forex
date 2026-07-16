package api

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/tsc-forex/tsc_forex/internal/models"
	"github.com/tsc-forex/tsc_forex/internal/repository"
	"github.com/tsc-forex/tsc_forex/internal/service"
)

// GamificationAwarder lets other handlers (watchlist, alerts) grant XP for
// engagement actions without depending on the concrete service.
type GamificationAwarder interface {
	Award(ctx context.Context, userID uuid.UUID, action string, sourceIDs ...string)
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

// Experience returns the versioned learning, discipline, season, paper-trade
// and privacy dashboard in one request for consistent cross-platform clients.
func (h *GamificationHandler) Experience(c *gin.Context) {
	uid, ok := currentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "unauthorized"})
		return
	}
	out, err := h.svc.Experience(c.Request.Context(), uid)
	if err != nil {
		writeGamificationError(c, err)
		return
	}
	c.JSON(http.StatusOK, out)
}

func (h *GamificationHandler) SaveDailyPlan(c *gin.Context) {
	uid, ok := currentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "unauthorized"})
		return
	}
	var input service.DailyPlanInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}
	out, err := h.svc.SaveDailyPlan(c.Request.Context(), uid, input)
	if err != nil {
		writeGamificationError(c, err)
		return
	}
	c.JSON(http.StatusOK, out)
}

func (h *GamificationHandler) SubmitQuiz(c *gin.Context) {
	uid, ok := currentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "unauthorized"})
		return
	}
	var input service.QuizAttemptInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}
	out, err := h.svc.SubmitQuiz(c.Request.Context(), uid, input)
	if err != nil {
		writeGamificationError(c, err)
		return
	}
	c.JSON(http.StatusOK, out)
}

func (h *GamificationHandler) CompleteDiscipline(c *gin.Context) {
	uid, ok := currentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "unauthorized"})
		return
	}
	var input struct {
		Note string `json:"note"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}
	out, err := h.svc.CompleteDiscipline(c.Request.Context(), uid, c.Param("code"), input.Note)
	if err != nil {
		writeGamificationError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"awarded": out.Created, "xp_gained": out.Ledger.Delta,
		"rule_version": out.Ledger.RuleVersion,
	})
}

func (h *GamificationHandler) Season(c *gin.Context) {
	uid, ok := currentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "unauthorized"})
		return
	}
	out, err := h.svc.Season(c.Request.Context(), uid, 50)
	if err != nil {
		writeGamificationError(c, err)
		return
	}
	c.JSON(http.StatusOK, out)
}

func (h *GamificationHandler) CreatePaperTrade(c *gin.Context) {
	uid, ok := currentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "unauthorized"})
		return
	}
	var input service.PaperTradeInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}
	out, created, err := h.svc.CreatePaperTrade(c.Request.Context(), uid, input)
	if err != nil {
		writeGamificationError(c, err)
		return
	}
	status := http.StatusOK
	if created {
		status = http.StatusCreated
	}
	c.JSON(status, gin.H{"created": created, "trade": out})
}

func (h *GamificationHandler) ReviewPaperTrade(c *gin.Context) {
	uid, ok := currentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "unauthorized"})
		return
	}
	tradeID, err := uuid.Parse(c.Param("tradeID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid trade id"})
		return
	}
	var input struct {
		FollowedPlan  bool   `json:"followed_plan"`
		Retrospective string `json:"retrospective"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}
	out, xp, err := h.svc.ReviewPaperTrade(
		c.Request.Context(), uid, tradeID, input.FollowedPlan, input.Retrospective,
	)
	if err != nil {
		writeGamificationError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"trade": out, "xp_gained": xp})
}

func (h *GamificationHandler) SavePrivacy(c *gin.Context) {
	uid, ok := currentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "unauthorized"})
		return
	}
	var input models.GamificationPrivacy
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}
	out, err := h.svc.SavePrivacy(c.Request.Context(), uid, input)
	if err != nil {
		writeGamificationError(c, err)
		return
	}
	c.JSON(http.StatusOK, out)
}

func (h *GamificationHandler) CreateStudyGroup(c *gin.Context) {
	uid, ok := currentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "unauthorized"})
		return
	}
	var input struct {
		Name string `json:"name"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}
	out, err := h.svc.CreateStudyGroup(c.Request.Context(), uid, input.Name)
	if err != nil {
		writeGamificationError(c, err)
		return
	}
	c.JSON(http.StatusCreated, out)
}

func (h *GamificationHandler) JoinStudyGroup(c *gin.Context) {
	uid, ok := currentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "unauthorized"})
		return
	}
	var input struct {
		InviteCode string `json:"invite_code"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}
	if err := h.svc.JoinStudyGroup(c.Request.Context(), uid, input.InviteCode); err != nil {
		writeGamificationError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"joined": true, "joined_at": time.Now().UTC()})
}

func writeGamificationError(c *gin.Context, err error) {
	status := http.StatusBadRequest
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		status = http.StatusNotFound
	case errors.Is(err, repository.ErrStudyGroupFull):
		status = http.StatusConflict
	}
	c.JSON(status, ErrorResponse{Error: err.Error()})
}
