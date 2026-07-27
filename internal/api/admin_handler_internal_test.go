package api

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"

	"github.com/tsc-forex/tsc_forex/internal/models"
)

func TestAdminSessionResponseUsesManagerEnvelope(t *testing.T) {
	manager := models.Manager{
		ID:                 uuid.New(),
		Email:              "manager@example.com",
		DisplayName:        "Manager",
		MustChangePassword: true,
	}
	body, err := json.Marshal(newAdminSessionResponse(manager, "access"))
	if err != nil {
		t.Fatal(err)
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatal(err)
	}
	if _, exists := payload["user"]; exists {
		t.Fatal("manager session leaked the legacy user envelope")
	}
	managerPayload, ok := payload["manager"].(map[string]any)
	if !ok || managerPayload["email"] != manager.Email {
		t.Fatalf("manager envelope=%v", payload["manager"])
	}
}
