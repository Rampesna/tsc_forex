package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

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

// List returns the current user's registered passkeys.
func (h *PasskeyHandler) List(c *gin.Context) {
	uid, ok := currentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "unauthorized"})
		return
	}
	items, err := h.svc.ListCredentials(c.Request.Context(), uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "list failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"passkeys": items})
}

// Delete removes one of the current user's passkeys by id.
func (h *PasskeyHandler) Delete(c *gin.Context) {
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
	if err := h.svc.DeleteCredential(c.Request.Context(), uid, id); err != nil {
		if errors.Is(err, service.ErrPasskeyNotFound) {
			c.JSON(http.StatusNotFound, ErrorResponse{Error: "not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "delete failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}

// Rename sets a user-friendly label for one of the user's passkeys.
func (h *PasskeyHandler) Rename(c *gin.Context) {
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
	var req struct {
		Name string `json:"name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}
	if err := h.svc.RenameCredential(c.Request.Context(), uid, id, req.Name); err != nil {
		if errors.Is(err, service.ErrPasskeyNotFound) {
			c.JSON(http.StatusNotFound, ErrorResponse{Error: "not found"})
			return
		}
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "renamed"})
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
