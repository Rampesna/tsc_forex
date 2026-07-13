package api

import (
	"bytes"
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/tsc-forex/tsc_forex/internal/service"
)

// PasskeyHandler exposes WebAuthn (passkey) registration and login.
type PasskeyHandler struct {
	svc *service.PasskeyService
}

// NewPasskeyHandler builds the handler. svc may be nil (feature disabled).
func NewPasskeyHandler(svc *service.PasskeyService) *PasskeyHandler {
	return &PasskeyHandler{svc: svc}
}

// passkeyFinishRequest carries the signed session plus the raw WebAuthn
// credential produced by the authenticator.
type passkeyFinishRequest struct {
	Session    string          `json:"session"`
	Credential json.RawMessage `json:"credential"`
}

// RegisterBegin returns credential-creation options for the current user.
func (h *PasskeyHandler) RegisterBegin(c *gin.Context) {
	uid, ok := currentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "unauthorized"})
		return
	}
	options, session, err := h.svc.BeginRegistration(c.Request.Context(), uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "passkey begin failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"options": options, "session": session})
}

// RegisterFinish verifies and stores the new passkey.
func (h *PasskeyHandler) RegisterFinish(c *gin.Context) {
	uid, ok := currentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "unauthorized"})
		return
	}
	var req passkeyFinishRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}
	if err := h.svc.FinishRegistration(c.Request.Context(), uid, req.Session, bytes.NewReader(req.Credential)); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "passkey registration failed"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"status": "registered"})
}

// LoginBegin returns assertion options for usernameless passkey login.
func (h *PasskeyHandler) LoginBegin(c *gin.Context) {
	options, session, err := h.svc.BeginLogin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "passkey begin failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"options": options, "session": session})
}

// LoginFinish verifies the assertion and issues our session tokens.
func (h *PasskeyHandler) LoginFinish(c *gin.Context) {
	var req passkeyFinishRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}
	access, refresh, err := h.svc.FinishLogin(c.Request.Context(), req.Session, bytes.NewReader(req.Credential))
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "passkey login failed"})
		return
	}
	c.JSON(http.StatusOK, TokenResponse{AccessToken: access, RefreshToken: refresh, TokenType: "Bearer"})
}
