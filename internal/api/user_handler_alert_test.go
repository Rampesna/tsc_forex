package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/tsc-forex/tsc_forex/internal/middleware"
	"github.com/tsc-forex/tsc_forex/internal/models"
	"github.com/tsc-forex/tsc_forex/internal/repository"
	"github.com/tsc-forex/tsc_forex/internal/service"
)

type alertHandlerRepo struct {
	updatedUser      uuid.UUID
	updatedID        uuid.UUID
	updatedDirection string
	updatedThreshold float64
	updateResult     models.PriceAlert
	updateErr        error
}

func (*alertHandlerRepo) Create(context.Context, *models.PriceAlert) error { return nil }
func (r *alertHandlerRepo) Update(_ context.Context, userID, id uuid.UUID, direction string, threshold float64) (models.PriceAlert, error) {
	r.updatedUser = userID
	r.updatedID = id
	r.updatedDirection = direction
	r.updatedThreshold = threshold
	return r.updateResult, r.updateErr
}
func (*alertHandlerRepo) Delete(context.Context, uuid.UUID, uuid.UUID) error { return nil }
func (*alertHandlerRepo) ListByUser(context.Context, uuid.UUID) ([]models.PriceAlert, error) {
	return nil, nil
}
func (*alertHandlerRepo) ActiveByCode(context.Context, string) ([]models.PriceAlert, error) {
	return nil, nil
}
func (*alertHandlerRepo) ClaimTriggered(context.Context, models.PriceAlert, *models.Notification, time.Time) (bool, error) {
	return false, nil
}

func updateAlertTestRouter(userID uuid.UUID, repo repository.AlertRepository) *gin.Engine {
	gin.SetMode(gin.TestMode)
	handler := NewUserHandler(nil, repo, nil, nil, nil, nil, nil)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(middleware.ClaimsKey, &service.Claims{UserID: userID.String()})
		c.Next()
	})
	router.PATCH("/me/alerts/:id", handler.UpdateAlert)
	return router
}

func TestUpdateAlert(t *testing.T) {
	userID := uuid.New()
	alertID := uuid.New()
	createdAt := time.Now().UTC().Truncate(time.Second)
	repo := &alertHandlerRepo{updateResult: models.PriceAlert{
		ID: alertID, UserID: userID, Code: "GRAM_GOLD", Direction: models.AlertBelow,
		Threshold: 2450.5, Active: true, CreatedAt: createdAt,
	}}
	router := updateAlertTestRouter(userID, repo)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPatch, "/me/alerts/"+alertID.String(),
		strings.NewReader(`{"direction":"below","threshold":2450.5}`))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if repo.updatedUser != userID || repo.updatedID != alertID ||
		repo.updatedDirection != models.AlertBelow || repo.updatedThreshold != 2450.5 {
		t.Fatalf("repository update args: user=%s id=%s direction=%q threshold=%v",
			repo.updatedUser, repo.updatedID, repo.updatedDirection, repo.updatedThreshold)
	}
	var response AlertResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.ID != alertID.String() || response.Direction != models.AlertBelow ||
		response.Threshold != 2450.5 || !response.Active {
		t.Fatalf("response = %+v", response)
	}
}

func TestUpdateAlertRejectsInvalidThreshold(t *testing.T) {
	userID := uuid.New()
	alertID := uuid.New()
	repo := &alertHandlerRepo{}
	router := updateAlertTestRouter(userID, repo)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPatch, "/me/alerts/"+alertID.String(),
		strings.NewReader(`{"direction":"above","threshold":-1}`))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if repo.updatedID != uuid.Nil {
		t.Fatal("invalid request reached repository")
	}
}

func TestUpdateAlertHidesMissingOrTriggeredAlert(t *testing.T) {
	userID := uuid.New()
	alertID := uuid.New()
	repo := &alertHandlerRepo{updateErr: repository.ErrNotFound}
	router := updateAlertTestRouter(userID, repo)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPatch, "/me/alerts/"+alertID.String(),
		strings.NewReader(`{"direction":"above","threshold":2700}`))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}
