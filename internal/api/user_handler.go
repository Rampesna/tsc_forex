package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/tsc-forex/tsc_forex/internal/domain"
	"github.com/tsc-forex/tsc_forex/internal/middleware"
	"github.com/tsc-forex/tsc_forex/internal/models"
	"github.com/tsc-forex/tsc_forex/internal/repository"
	"github.com/tsc-forex/tsc_forex/internal/service"
)

// UserHandler exposes authenticated per-user features: watchlist, price
// alerts, notifications and device (push) token registration.
type UserHandler struct {
	watch   repository.WatchlistRepository
	alerts  repository.AlertRepository
	notifs  repository.NotificationRepository
	devices repository.DeviceTokenRepository
	log     *zap.Logger
}

// NewUserHandler builds the user handler.
func NewUserHandler(
	watch repository.WatchlistRepository,
	alerts repository.AlertRepository,
	notifs repository.NotificationRepository,
	devices repository.DeviceTokenRepository,
	log *zap.Logger,
) *UserHandler {
	if log == nil {
		log = zap.NewNop()
	}
	return &UserHandler{watch: watch, alerts: alerts, notifs: notifs, devices: devices, log: log}
}

// currentUserID extracts the authenticated user's UUID from JWT claims.
func currentUserID(c *gin.Context) (uuid.UUID, bool) {
	v, ok := c.Get(middleware.ClaimsKey)
	if !ok {
		return uuid.Nil, false
	}
	claims, ok := v.(*service.Claims)
	if !ok {
		return uuid.Nil, false
	}
	id, err := uuid.Parse(claims.UserID)
	if err != nil {
		return uuid.Nil, false
	}
	return id, true
}

func validCode(code string) bool {
	_, ok := domain.CatalogByCode[code]
	return ok
}

// --- Watchlist ---

// ListWatchlist returns the user's tracked instruments.
func (h *UserHandler) ListWatchlist(c *gin.Context) {
	uid, ok := currentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "unauthorized"})
		return
	}
	items, err := h.watch.List(c.Request.Context(), uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "list failed"})
		return
	}
	codes := make([]string, 0, len(items))
	for _, it := range items {
		codes = append(codes, it.Code)
	}
	c.JSON(http.StatusOK, gin.H{"count": len(codes), "codes": codes})
}

// AddWatchlist adds an instrument to the user's watchlist.
func (h *UserHandler) AddWatchlist(c *gin.Context) {
	uid, ok := currentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "unauthorized"})
		return
	}
	var req WatchlistRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}
	if !validCode(req.Code) {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "unknown instrument code"})
		return
	}
	if _, err := h.watch.Add(c.Request.Context(), uid, req.Code); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "add failed"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"code": req.Code})
}

// RemoveWatchlist removes an instrument from the user's watchlist.
func (h *UserHandler) RemoveWatchlist(c *gin.Context) {
	uid, ok := currentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "unauthorized"})
		return
	}
	code := c.Param("code")
	if err := h.watch.Remove(c.Request.Context(), uid, code); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "remove failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "removed", "code": code})
}

// --- Alerts ---

// ListAlerts returns the user's price alerts.
func (h *UserHandler) ListAlerts(c *gin.Context) {
	uid, ok := currentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "unauthorized"})
		return
	}
	rows, err := h.alerts.ListByUser(c.Request.Context(), uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "list failed"})
		return
	}
	out := make([]AlertResponse, 0, len(rows))
	for _, a := range rows {
		out = append(out, AlertResponse{
			ID:          a.ID.String(),
			Code:        a.Code,
			Direction:   a.Direction,
			Threshold:   a.Threshold,
			Active:      a.Active,
			LastFiredAt: a.LastFiredAt,
			CreatedAt:   a.CreatedAt,
		})
	}
	c.JSON(http.StatusOK, gin.H{"count": len(out), "alerts": out})
}

// CreateAlert creates a threshold alert for a tracked instrument.
func (h *UserHandler) CreateAlert(c *gin.Context) {
	uid, ok := currentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "unauthorized"})
		return
	}
	var req AlertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}
	if !validCode(req.Code) {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "unknown instrument code"})
		return
	}
	alert := &models.PriceAlert{
		ID:        uuid.New(),
		UserID:    uid,
		Code:      req.Code,
		Direction: req.Direction,
		Threshold: req.Threshold,
		Active:    true,
	}
	if err := h.alerts.Create(c.Request.Context(), alert); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "create failed"})
		return
	}
	c.JSON(http.StatusCreated, AlertResponse{
		ID:        alert.ID.String(),
		Code:      alert.Code,
		Direction: alert.Direction,
		Threshold: alert.Threshold,
		Active:    alert.Active,
		CreatedAt: alert.CreatedAt,
	})
}

// DeleteAlert removes a user's alert.
func (h *UserHandler) DeleteAlert(c *gin.Context) {
	uid, ok := currentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "unauthorized"})
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid id"})
		return
	}
	if err := h.alerts.Delete(c.Request.Context(), uid, id); err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "alert not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}

// --- Notifications ---

// ListNotifications returns the user's recent notifications.
func (h *UserHandler) ListNotifications(c *gin.Context) {
	uid, ok := currentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "unauthorized"})
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))
	rows, err := h.notifs.ListByUser(c.Request.Context(), uid, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "list failed"})
		return
	}
	out := make([]NotificationResponse, 0, len(rows))
	for _, n := range rows {
		out = append(out, NotificationResponse{
			ID:        n.ID.String(),
			Code:      n.Code,
			Title:     n.Title,
			Body:      n.Body,
			Price:     n.Price,
			ReadAt:    n.ReadAt,
			CreatedAt: n.CreatedAt,
		})
	}
	c.JSON(http.StatusOK, gin.H{"count": len(out), "notifications": out})
}

// MarkNotificationRead marks a notification as read.
func (h *UserHandler) MarkNotificationRead(c *gin.Context) {
	uid, ok := currentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "unauthorized"})
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid id"})
		return
	}
	if err := h.notifs.MarkRead(c.Request.Context(), uid, id); err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "notification not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "read"})
}

// DeleteNotification removes a single notification owned by the user.
func (h *UserHandler) DeleteNotification(c *gin.Context) {
	uid, ok := currentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "unauthorized"})
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid id"})
		return
	}
	if err := h.notifs.Delete(c.Request.Context(), uid, id); err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "notification not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}

// ClearNotifications removes all notifications for the user.
func (h *UserHandler) ClearNotifications(c *gin.Context) {
	uid, ok := currentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "unauthorized"})
		return
	}
	n, err := h.notifs.DeleteAll(c.Request.Context(), uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "clear failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "cleared", "deleted": n})
}

// --- Devices (push tokens) ---

// RegisterDevice stores an APNs/FCM token for the current user.
func (h *UserHandler) RegisterDevice(c *gin.Context) {
	uid, ok := currentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "unauthorized"})
		return
	}
	var req DeviceTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}
	dt := &models.DeviceToken{
		ID:       uuid.New(),
		UserID:   uid,
		Platform: req.Platform,
		Token:    req.Token,
	}
	if err := h.devices.Upsert(c.Request.Context(), dt); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "register failed"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"status": "registered"})
}
