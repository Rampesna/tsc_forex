package api

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/tsc-forex/tsc_forex/internal/domain"
	"github.com/tsc-forex/tsc_forex/internal/middleware"
	"github.com/tsc-forex/tsc_forex/internal/models"
	"github.com/tsc-forex/tsc_forex/internal/repository"
	"github.com/tsc-forex/tsc_forex/internal/service"
	"github.com/tsc-forex/tsc_forex/internal/storage"
)

// UserHandler exposes authenticated per-user features: profile, watchlist,
// price alerts, notifications and device (push) token registration.
type UserHandler struct {
	watch   repository.WatchlistRepository
	alerts  repository.AlertRepository
	notifs  repository.NotificationRepository
	devices repository.DeviceTokenRepository
	users   repository.UserRepository
	store   storage.Store       // may be nil when object storage is disabled
	gam     GamificationAwarder // may be nil when gamification is disabled
	log     *zap.Logger
}

// WithGamification wires the XP awarder so favorites/alerts grant engagement
// XP. Returns the handler for chaining. Safe to skip (nil-guarded at call site).
func (h *UserHandler) WithGamification(g GamificationAwarder) *UserHandler {
	h.gam = g
	return h
}

// award grants XP asynchronously (fire-and-forget) so it never delays the API
// response or gets cancelled with the request context.
func (h *UserHandler) award(userID uuid.UUID, action string) {
	if h.gam == nil {
		return
	}
	go h.gam.Award(context.Background(), userID, action)
}

// DeleteAccount permanently deletes the authenticated user and all of their
// data. Exposed as DELETE /me/account for App Store / Play Store in-app
// account-deletion compliance.
func (h *UserHandler) DeleteAccount(c *gin.Context) {
	uid, ok := currentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "unauthorized"})
		return
	}
	if err := h.users.DeleteUser(c.Request.Context(), uid); err != nil {
		h.log.Error("account deletion failed", zap.String("user", uid.String()), zap.Error(err))
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "delete failed"})
		return
	}
	h.log.Info("account deleted", zap.String("user", uid.String()))
	c.JSON(http.StatusOK, gin.H{"deleted": true})
}

// NewUserHandler builds the user handler.
func NewUserHandler(
	watch repository.WatchlistRepository,
	alerts repository.AlertRepository,
	notifs repository.NotificationRepository,
	devices repository.DeviceTokenRepository,
	users repository.UserRepository,
	store storage.Store,
	log *zap.Logger,
) *UserHandler {
	if log == nil {
		log = zap.NewNop()
	}
	return &UserHandler{watch: watch, alerts: alerts, notifs: notifs, devices: devices, users: users, store: store, log: log}
}

// --- Profile ---

// maxAvatarBytes caps avatar uploads at 5 MB.
const maxAvatarBytes = 5 << 20

// Profile returns the authenticated user's profile.
func (h *UserHandler) Profile(c *gin.Context) {
	uid, ok := currentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "unauthorized"})
		return
	}
	u, err := h.users.ByID(c.Request.Context(), uid)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "user not found"})
		return
	}
	c.JSON(http.StatusOK, ProfileResponse{
		ID:          u.ID.String(),
		Email:       u.Email,
		DisplayName: u.DisplayName,
		AvatarURL:   u.AvatarURL,
		Provider:    u.Provider,
		Role:        u.Role,
		CreatedAt:   u.CreatedAt,
	})
}

// UpdateProfile updates mutable profile fields (currently display name).
func (h *UserHandler) UpdateProfile(c *gin.Context) {
	uid, ok := currentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "unauthorized"})
		return
	}
	var req UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}
	u, err := h.users.ByID(c.Request.Context(), uid)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "user not found"})
		return
	}
	if req.DisplayName != nil {
		u.DisplayName = strings.TrimSpace(*req.DisplayName)
	}
	if err := h.users.Update(c.Request.Context(), &u); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "update failed"})
		return
	}
	c.JSON(http.StatusOK, ProfileResponse{
		ID:          u.ID.String(),
		Email:       u.Email,
		DisplayName: u.DisplayName,
		AvatarURL:   u.AvatarURL,
		Provider:    u.Provider,
		Role:        u.Role,
		CreatedAt:   u.CreatedAt,
	})
}

// UploadAvatar accepts a multipart "avatar" image, stores it in object storage
// and saves the resulting public URL on the user's profile.
func (h *UserHandler) UploadAvatar(c *gin.Context) {
	uid, ok := currentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "unauthorized"})
		return
	}
	if h.store == nil {
		c.JSON(http.StatusServiceUnavailable, ErrorResponse{Error: "avatar storage not configured"})
		return
	}
	fileHeader, err := c.FormFile("avatar")
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "avatar file required"})
		return
	}
	if fileHeader.Size > maxAvatarBytes {
		c.JSON(http.StatusRequestEntityTooLarge, ErrorResponse{Error: "avatar too large (max 5MB)"})
		return
	}
	ext := avatarExtension(fileHeader.Filename, fileHeader.Header.Get("Content-Type"))
	if ext == "" {
		c.JSON(http.StatusUnsupportedMediaType, ErrorResponse{Error: "only jpg/png/webp allowed"})
		return
	}
	f, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "read failed"})
		return
	}
	defer func() { _ = f.Close() }()
	data, err := io.ReadAll(io.LimitReader(f, maxAvatarBytes))
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "read failed"})
		return
	}
	// Unique key per upload to bust CDN caches on change.
	key := fmt.Sprintf("users/%s/%d.%s", uid.String(), time.Now().UnixNano(), ext)
	url, err := h.store.Put(c.Request.Context(), key, contentTypeForExt(ext), data)
	if err != nil {
		h.log.Warn("avatar upload failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "upload failed"})
		return
	}
	u, err := h.users.ByID(c.Request.Context(), uid)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "user not found"})
		return
	}
	u.AvatarURL = url
	if err := h.users.Update(c.Request.Context(), &u); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "profile update failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"avatar_url": url})
}

func avatarExtension(filename, contentType string) string {
	switch {
	case strings.Contains(contentType, "png") || strings.HasSuffix(strings.ToLower(filename), ".png"):
		return "png"
	case strings.Contains(contentType, "webp") || strings.HasSuffix(strings.ToLower(filename), ".webp"):
		return "webp"
	case strings.Contains(contentType, "jpeg") || strings.Contains(contentType, "jpg") ||
		strings.HasSuffix(strings.ToLower(filename), ".jpg") || strings.HasSuffix(strings.ToLower(filename), ".jpeg"):
		return "jpg"
	default:
		return ""
	}
}

func contentTypeForExt(ext string) string {
	switch ext {
	case "png":
		return "image/png"
	case "webp":
		return "image/webp"
	default:
		return "image/jpeg"
	}
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
	h.award(uid, service.EventFavorite)
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
	h.award(uid, service.EventAlert)
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
