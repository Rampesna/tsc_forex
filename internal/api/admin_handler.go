package api

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/tsc-forex/tsc_forex/internal/config"
	"github.com/tsc-forex/tsc_forex/internal/models"
	"github.com/tsc-forex/tsc_forex/internal/service"
)

const adminRefreshCookie = "tsc_admin_refresh"

type AdminHandler struct {
	auth  *service.AuthService
	admin *service.AdminService
	cfg   config.AdminConfig
}

func NewAdminHandler(auth *service.AuthService, admin *service.AdminService, cfg config.AdminConfig) *AdminHandler {
	return &AdminHandler{auth: auth, admin: admin, cfg: cfg}
}

type adminLoginRequest struct {
	Email    string `json:"email" binding:"required,email,max=255"`
	Password string `json:"password" binding:"required,min=8,max=128"`
}

type adminSessionResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	User        struct {
		ID                 uuid.UUID `json:"id"`
		Email              string    `json:"email"`
		DisplayName        string    `json:"display_name"`
		MustChangePassword bool      `json:"must_change_password"`
	} `json:"user"`
}

func newAdminSessionResponse(user models.User, access string) adminSessionResponse {
	out := adminSessionResponse{AccessToken: access, TokenType: "Bearer"}
	out.User.ID = user.ID
	out.User.Email = user.Email
	out.User.DisplayName = user.DisplayName
	out.User.MustChangePassword = user.MustChangePassword
	return out
}

func (h *AdminHandler) Login(c *gin.Context) {
	var req adminLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "geçerli e-posta ve şifre gerekli"})
		return
	}
	user, access, refresh, err := h.auth.LoginAdmin(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		h.admin.Audit(c.Request.Context(), nil, "admin.login_failed", "session", "", c.ClientIP(), c.GetHeader("User-Agent"), map[string]any{"email": strings.ToLower(strings.TrimSpace(req.Email))})
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "e-posta veya şifre hatalı"})
		return
	}
	h.setRefreshCookie(c, refresh)
	h.admin.Audit(c.Request.Context(), &user.ID, "admin.login", "session", "", c.ClientIP(), c.GetHeader("User-Agent"), map[string]any{"status": "success"})
	c.JSON(http.StatusOK, newAdminSessionResponse(user, access))
}

func (h *AdminHandler) Refresh(c *gin.Context) {
	raw, err := c.Cookie(adminRefreshCookie)
	if err != nil || raw == "" {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "oturum süresi doldu"})
		return
	}
	user, access, refresh, err := h.auth.RefreshAdmin(c.Request.Context(), raw)
	if err != nil {
		h.clearRefreshCookie(c)
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "oturum süresi doldu"})
		return
	}
	h.setRefreshCookie(c, refresh)
	c.JSON(http.StatusOK, newAdminSessionResponse(user, access))
}

func (h *AdminHandler) Logout(c *gin.Context) {
	if raw, err := c.Cookie(adminRefreshCookie); err == nil && raw != "" {
		_ = h.auth.Logout(c.Request.Context(), raw)
	}
	h.clearRefreshCookie(c)
	c.Status(http.StatusNoContent)
}

func (h *AdminHandler) Me(c *gin.Context) {
	uid, ok := currentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "unauthorized"})
		return
	}
	user, err := h.auth.AdminUser(c.Request.Context(), uid)
	if err != nil {
		c.JSON(http.StatusForbidden, ErrorResponse{Error: "admin access required"})
		return
	}
	c.JSON(http.StatusOK, newAdminSessionResponse(user, ""))
}

type adminPasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required,max=128"`
	NewPassword     string `json:"new_password" binding:"required,min=12,max=128"`
}

func (h *AdminHandler) ChangePassword(c *gin.Context) {
	uid, ok := currentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "unauthorized"})
		return
	}
	var req adminPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "yeni şifre en az 12 karakter olmalı"})
		return
	}
	user, err := h.auth.ChangeAdminPassword(c.Request.Context(), uid, req.CurrentPassword, req.NewPassword)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidCredentials):
			c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "mevcut şifre hatalı"})
		case errors.Is(err, service.ErrWeakAdminPassword):
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "şifre büyük/küçük harf, rakam ve özel karakter içermeli"})
		case errors.Is(err, service.ErrPasswordUnchanged):
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "yeni şifre eskisinden farklı olmalı"})
		default:
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "şifre güncellenemedi"})
		}
		return
	}
	access, refresh, err := h.auth.IssueSession(c.Request.Context(), user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "yeni oturum oluşturulamadı"})
		return
	}
	h.setRefreshCookie(c, refresh)
	h.admin.Audit(c.Request.Context(), &uid, "admin.password_changed", "admin", uid.String(), c.ClientIP(), c.GetHeader("User-Agent"), map[string]any{"revoked_other_sessions": true})
	c.JSON(http.StatusOK, newAdminSessionResponse(user, access))
}

func (h *AdminHandler) Dashboard(c *gin.Context) {
	days, _ := strconv.Atoi(c.DefaultQuery("days", "30"))
	view, err := h.admin.Dashboard(c.Request.Context(), days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "dashboard verileri alınamadı"})
		return
	}
	c.JSON(http.StatusOK, view)
}

func (h *AdminHandler) Users(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	view, err := h.admin.Users(c.Request.Context(), service.AdminUserQuery{
		Search: c.Query("search"), Plan: c.Query("plan"), Platform: c.Query("platform"),
		Page: page, PageSize: pageSize,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "kullanıcılar alınamadı"})
		return
	}
	c.JSON(http.StatusOK, view)
}

func (h *AdminHandler) UserDetail(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "geçersiz kullanıcı"})
		return
	}
	view, err := h.admin.UserDetail(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, ErrorResponse{Error: "kullanıcı bulunamadı"})
			return
		}
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "kullanıcı detayı alınamadı"})
		return
	}
	actor, _ := currentUserID(c)
	h.admin.Audit(c.Request.Context(), &actor, "admin.user_viewed", "user", id.String(), c.ClientIP(), c.GetHeader("User-Agent"), map[string]any{})
	c.JSON(http.StatusOK, view)
}

func (h *AdminHandler) Online(c *gin.Context) {
	view, err := h.admin.OnlineUsers(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, ErrorResponse{Error: "anlık kullanıcı verisi alınamadı"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": view, "count": len(view), "generated_at": time.Now().UTC()})
}

type campaignPreviewRequest struct {
	Prompt string `json:"prompt" binding:"required,min=8,max=2000"`
}

func (h *AdminHandler) PreviewCampaign(c *gin.Context) {
	uid, ok := currentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "unauthorized"})
		return
	}
	var req campaignPreviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "bildirim talebi en az 8 karakter olmalı"})
		return
	}
	preview, err := h.admin.PreviewCampaign(c.Request.Context(), uid, req.Prompt)
	if err != nil {
		if errors.Is(err, service.ErrInvalidTargetingPlan) {
			c.JSON(http.StatusUnprocessableEntity, ErrorResponse{Error: "talep güvenli bir hedefleme planına dönüştürülemedi"})
			return
		}
		c.JSON(http.StatusServiceUnavailable, ErrorResponse{Error: "kampanya önizlemesi oluşturulamadı"})
		return
	}
	h.admin.Audit(c.Request.Context(), &uid, "campaign.previewed", "campaign", preview.CampaignID.String(), c.ClientIP(), c.GetHeader("User-Agent"), map[string]any{
		"recipient_count": preview.RecipientCount, "asset_count": len(preview.MatchedAssets),
	})
	c.JSON(http.StatusCreated, preview)
}

func (h *AdminHandler) ApproveCampaign(c *gin.Context) {
	uid, ok := currentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "unauthorized"})
		return
	}
	campaignID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "geçersiz kampanya"})
		return
	}
	var req service.AdminCampaignApproval
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "onay bilgileri eksik"})
		return
	}
	campaign, err := h.admin.ApproveCampaign(c.Request.Context(), uid, campaignID, req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrAudienceChanged):
			c.JSON(http.StatusConflict, ErrorResponse{Error: "hedef kitle önizlemeden sonra değişti; yeniden önizleyin"})
		case errors.Is(err, service.ErrCampaignNotFound):
			c.JSON(http.StatusNotFound, ErrorResponse{Error: "kampanya bulunamadı"})
		case errors.Is(err, service.ErrCampaignState):
			c.JSON(http.StatusConflict, ErrorResponse{Error: "kampanya daha önce onaylandı veya iptal edildi"})
		default:
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "kampanya kuyruğa alınamadı"})
		}
		return
	}
	h.admin.Audit(c.Request.Context(), &uid, "campaign.approved", "campaign", campaignID.String(), c.ClientIP(), c.GetHeader("User-Agent"), map[string]any{"recipients": campaign.ResolvedRecipients})
	c.JSON(http.StatusAccepted, campaign)
}

func (h *AdminHandler) Campaigns(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	items, total, err := h.admin.Campaigns(c.Request.Context(), page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "kampanyalar alınamadı"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "total": total, "page": page, "page_size": pageSize})
}

func (h *AdminHandler) CancelCampaign(c *gin.Context) {
	uid, ok := currentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "unauthorized"})
		return
	}
	campaignID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "geçersiz kampanya"})
		return
	}
	if err := h.admin.CancelCampaign(c.Request.Context(), uid, campaignID); err != nil {
		if errors.Is(err, service.ErrCampaignNotFound) {
			c.JSON(http.StatusNotFound, ErrorResponse{Error: "kampanya bulunamadı"})
		} else {
			c.JSON(http.StatusConflict, ErrorResponse{Error: "kampanya iptal edilemiyor"})
		}
		return
	}
	h.admin.Audit(c.Request.Context(), &uid, "campaign.cancelled", "campaign", campaignID.String(), c.ClientIP(), c.GetHeader("User-Agent"), map[string]any{})
	c.Status(http.StatusNoContent)
}

func (h *AdminHandler) AuditLogs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "30"))
	view, err := h.admin.AuditLogs(c.Request.Context(), page, pageSize, c.Query("action"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "denetim günlüğü alınamadı"})
		return
	}
	c.JSON(http.StatusOK, view)
}

func (h *AdminHandler) setRefreshCookie(c *gin.Context, value string) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name: adminRefreshCookie, Value: value, Path: "/api/v1/admin/auth",
		MaxAge: int((7 * 24 * time.Hour).Seconds()), HttpOnly: true,
		Secure: h.cfg.CookieSecure, SameSite: http.SameSiteStrictMode,
	})
}

func (h *AdminHandler) clearRefreshCookie(c *gin.Context) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name: adminRefreshCookie, Value: "", Path: "/api/v1/admin/auth",
		MaxAge: -1, Expires: time.Unix(1, 0), HttpOnly: true,
		Secure: h.cfg.CookieSecure, SameSite: http.SameSiteStrictMode,
	})
}
