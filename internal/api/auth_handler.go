package api

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/tsc-forex/tsc_forex/internal/service"
)

// AuthHandler exposes registration and login endpoints.
type AuthHandler struct {
	auth *service.AuthService
}

// NewAuthHandler builds the auth handler.
func NewAuthHandler(auth *service.AuthService) *AuthHandler {
	return &AuthHandler{auth: auth}
}

// Register godoc
// @Summary  Register a new user
// @Tags     auth
// @Accept   json
// @Produce  json
// @Param    body body RegisterRequest true "Registration"
// @Success  201 {object} map[string]string
// @Failure  400 {object} ErrorResponse
// @Failure  409 {object} ErrorResponse
// @Router   /auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}
	u, err := h.auth.Register(c.Request.Context(), req.Email, req.Password, "user")
	if err != nil {
		if errors.Is(err, service.ErrEmailTaken) {
			c.JSON(http.StatusConflict, ErrorResponse{Error: "email already registered"})
			return
		}
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "registration failed"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": u.ID, "email": u.Email})
}

// Login godoc
// @Summary  Log in and receive JWT tokens
// @Tags     auth
// @Accept   json
// @Produce  json
// @Param    body body LoginRequest true "Credentials"
// @Success  200 {object} TokenResponse
// @Failure  400 {object} ErrorResponse
// @Failure  401 {object} ErrorResponse
// @Router   /auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}
	access, refresh, err := h.auth.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "invalid credentials"})
		return
	}
	c.JSON(http.StatusOK, TokenResponse{
		AccessToken:  access,
		RefreshToken: refresh,
		TokenType:    "Bearer",
	})
}

// Google logs in (or signs up) a user with a verified Google ID token.
// @Summary  Sign in with Google
// @Tags     auth
// @Accept   json
// @Produce  json
// @Param    body body GoogleAuthRequest true "Google ID token"
// @Success  200 {object} TokenResponse
// @Failure  400 {object} ErrorResponse
// @Failure  401 {object} ErrorResponse
// @Router   /auth/google [post]
func (h *AuthHandler) Google(c *gin.Context) {
	var req GoogleAuthRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}
	access, refresh, err := h.auth.LoginWithGoogle(c.Request.Context(), req.IDToken)
	if err != nil {
		if errors.Is(err, service.ErrOAuthNotConfigured) {
			c.JSON(http.StatusServiceUnavailable, ErrorResponse{Error: "google login not configured"})
			return
		}
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "google authentication failed"})
		return
	}
	c.JSON(http.StatusOK, TokenResponse{AccessToken: access, RefreshToken: refresh, TokenType: "Bearer"})
}

// Refresh rotates a refresh token, returning a new access + refresh pair.
// @Summary  Rotate refresh token
// @Tags     auth
// @Router   /auth/refresh [post]
func (h *AuthHandler) Refresh(c *gin.Context) {
	var req RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}
	access, refresh, err := h.auth.Refresh(c.Request.Context(), req.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "invalid or expired refresh token"})
		return
	}
	c.JSON(http.StatusOK, TokenResponse{AccessToken: access, RefreshToken: refresh, TokenType: "Bearer"})
}

// Logout revokes a refresh token.
// @Summary  Log out (revoke refresh token)
// @Tags     auth
// @Router   /auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	var req RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}
	_ = h.auth.Logout(c.Request.Context(), req.RefreshToken)
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
