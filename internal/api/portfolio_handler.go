package api

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/tsc-forex/tsc_forex/internal/models"
	"github.com/tsc-forex/tsc_forex/internal/repository"
	"github.com/tsc-forex/tsc_forex/internal/service"
)

type PortfolioHandler struct {
	service *service.PortfolioService
}

func NewPortfolioHandler(portfolios *service.PortfolioService) *PortfolioHandler {
	return &PortfolioHandler{service: portfolios}
}

type portfolioRequest struct {
	Name string `json:"name" binding:"required,max=80"`
}

type portfolioTransactionRequest struct {
	ClientID        string          `json:"client_id" binding:"required,max=80"`
	Code            string          `json:"code"`
	Type            string          `json:"type" binding:"required"`
	Quantity        decimal.Decimal `json:"quantity"`
	UnitPriceNative decimal.Decimal `json:"unit_price_native"`
	AmountNative    decimal.Decimal `json:"amount_native"`
	FeeNative       decimal.Decimal `json:"fee_native"`
	Currency        string          `json:"currency"`
	USDTRY          decimal.Decimal `json:"usdtry"`
	OccurredAt      time.Time       `json:"occurred_at"`
	Note            string          `json:"note"`
}

func (request portfolioTransactionRequest) model() models.PortfolioTransaction {
	return models.PortfolioTransaction{
		ClientID:        request.ClientID,
		Code:            request.Code,
		Type:            request.Type,
		Quantity:        request.Quantity,
		UnitPriceNative: request.UnitPriceNative,
		AmountNative:    request.AmountNative,
		FeeNative:       request.FeeNative,
		Currency:        request.Currency,
		USDTRY:          request.USDTRY,
		OccurredAt:      request.OccurredAt,
		Note:            request.Note,
	}
}

func (h *PortfolioHandler) List(c *gin.Context) {
	userID, ok := requirePortfolioUser(c)
	if !ok {
		return
	}
	rows, err := h.service.List(c.Request.Context(), userID)
	if err != nil {
		writePortfolioError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"count": len(rows), "portfolios": rows})
}

func (h *PortfolioHandler) Create(c *gin.Context) {
	userID, ok := requirePortfolioUser(c)
	if !ok {
		return
	}
	var request portfolioRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "geçerli bir portföy adı gerekli"})
		return
	}
	portfolio, err := h.service.Create(c.Request.Context(), userID, request.Name)
	if err != nil {
		writePortfolioError(c, err)
		return
	}
	c.JSON(http.StatusCreated, portfolio)
}

func (h *PortfolioHandler) Get(c *gin.Context) {
	userID, portfolioID, ok := portfolioIDs(c)
	if !ok {
		return
	}
	overview, err := h.service.Get(c.Request.Context(), userID, portfolioID)
	if err != nil {
		writePortfolioError(c, err)
		return
	}
	c.JSON(http.StatusOK, overview)
}

func (h *PortfolioHandler) Update(c *gin.Context) {
	userID, portfolioID, ok := portfolioIDs(c)
	if !ok {
		return
	}
	var request portfolioRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "geçerli bir portföy adı gerekli"})
		return
	}
	portfolio, err := h.service.Update(c.Request.Context(), userID, portfolioID, request.Name)
	if err != nil {
		writePortfolioError(c, err)
		return
	}
	c.JSON(http.StatusOK, portfolio)
}

func (h *PortfolioHandler) Delete(c *gin.Context) {
	userID, portfolioID, ok := portfolioIDs(c)
	if !ok {
		return
	}
	if err := h.service.Delete(c.Request.Context(), userID, portfolioID); err != nil {
		writePortfolioError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *PortfolioHandler) Position(c *gin.Context) {
	userID, portfolioID, ok := portfolioIDs(c)
	if !ok {
		return
	}
	detail, err := h.service.Position(
		c.Request.Context(),
		userID,
		portfolioID,
		strings.ToUpper(c.Param("code")),
	)
	if err != nil {
		writePortfolioError(c, err)
		return
	}
	c.JSON(http.StatusOK, detail)
}

func (h *PortfolioHandler) Transactions(c *gin.Context) {
	userID, portfolioID, ok := portfolioIDs(c)
	if !ok {
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))
	if limit > 500 {
		limit = 500
	}
	var before time.Time
	if raw := c.Query("before"); raw != "" {
		parsed, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "before RFC3339 olmalı"})
			return
		}
		before = parsed
	}
	rows, err := h.service.Transactions(c.Request.Context(), userID, portfolioID, limit, before)
	if err != nil {
		writePortfolioError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"count": len(rows), "transactions": rows})
}

func (h *PortfolioHandler) CreateTransaction(c *gin.Context) {
	userID, portfolioID, ok := portfolioIDs(c)
	if !ok {
		return
	}
	var request portfolioTransactionRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "geçersiz işlem"})
		return
	}
	transaction, created, err := h.service.CreateTransaction(
		c.Request.Context(),
		userID,
		portfolioID,
		request.model(),
	)
	if err != nil {
		writePortfolioError(c, err)
		return
	}
	status := http.StatusOK
	if created {
		status = http.StatusCreated
	}
	c.JSON(status, gin.H{"created": created, "transaction": transaction})
}

func (h *PortfolioHandler) UpdateTransaction(c *gin.Context) {
	userID, portfolioID, ok := portfolioIDs(c)
	if !ok {
		return
	}
	transactionID, err := uuid.Parse(c.Param("transactionID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "geçersiz işlem kimliği"})
		return
	}
	var request portfolioTransactionRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "geçersiz işlem"})
		return
	}
	transaction, err := h.service.UpdateTransaction(
		c.Request.Context(),
		userID,
		portfolioID,
		transactionID,
		request.model(),
	)
	if err != nil {
		writePortfolioError(c, err)
		return
	}
	c.JSON(http.StatusOK, transaction)
}

func (h *PortfolioHandler) DeleteTransaction(c *gin.Context) {
	userID, portfolioID, ok := portfolioIDs(c)
	if !ok {
		return
	}
	transactionID, err := uuid.Parse(c.Param("transactionID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "geçersiz işlem kimliği"})
		return
	}
	if err := h.service.DeleteTransaction(
		c.Request.Context(),
		userID,
		portfolioID,
		transactionID,
	); err != nil {
		writePortfolioError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *PortfolioHandler) Performance(c *gin.Context) {
	userID, portfolioID, ok := portfolioIDs(c)
	if !ok {
		return
	}
	to := time.Now().UTC()
	from := performanceStart(c.DefaultQuery("range", "1y"), to)
	if raw := c.Query("from"); raw != "" {
		parsed, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "from RFC3339 olmalı"})
			return
		}
		from = parsed
	}
	if raw := c.Query("to"); raw != "" {
		parsed, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "to RFC3339 olmalı"})
			return
		}
		to = parsed
	}
	points, _ := strconv.Atoi(c.DefaultQuery("points", "365"))
	series, err := h.service.Performance(
		c.Request.Context(),
		userID,
		portfolioID,
		from,
		to,
		points,
	)
	if err != nil {
		writePortfolioError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"count": len(series), "points": series})
}

func performanceStart(value string, to time.Time) time.Time {
	switch value {
	case "1m":
		return to.AddDate(0, -1, 0)
	case "3m":
		return to.AddDate(0, -3, 0)
	case "6m":
		return to.AddDate(0, -6, 0)
	case "5y", "max":
		return to.AddDate(-5, 0, 0)
	default:
		return to.AddDate(-1, 0, 0)
	}
}

func requirePortfolioUser(c *gin.Context) (uuid.UUID, bool) {
	userID, ok := currentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "unauthorized"})
	}
	return userID, ok
}

func portfolioIDs(c *gin.Context) (uuid.UUID, uuid.UUID, bool) {
	userID, ok := requirePortfolioUser(c)
	if !ok {
		return uuid.Nil, uuid.Nil, false
	}
	portfolioID, err := uuid.Parse(c.Param("portfolioID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "geçersiz portföy kimliği"})
		return uuid.Nil, uuid.Nil, false
	}
	return userID, portfolioID, true
}

func writePortfolioError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, repository.ErrPortfolioNotFound),
		errors.Is(err, repository.ErrTransactionNotFound),
		errors.Is(err, service.ErrPortfolioPositionNotFound):
		c.JSON(http.StatusNotFound, ErrorResponse{Error: err.Error()})
	case errors.Is(err, service.ErrNegativePosition):
		c.JSON(http.StatusConflict, ErrorResponse{Error: "işlem bakiyeyi negatife düşürüyor"})
	case errors.Is(err, service.ErrInvalidPortfolio),
		errors.Is(err, service.ErrInvalidTransaction):
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
	case errors.Is(err, service.ErrLivePriceUnavailable):
		c.JSON(http.StatusServiceUnavailable, ErrorResponse{Error: err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "portföy işlemi başarısız"})
	}
}
