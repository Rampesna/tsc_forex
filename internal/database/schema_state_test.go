package database

import (
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestSchemaVersionBeforeOnlySelectsPendingMigration(t *testing.T) {
	t.Parallel()

	db, err := gorm.Open(sqlite.Open("file:schema-version-state?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&schemaState{}); err != nil {
		t.Fatal(err)
	}

	pending, err := schemaVersionBefore(db, clientPlatformBackfillVersion)
	if err != nil {
		t.Fatal(err)
	}
	if !pending {
		t.Fatal("a database without a readiness marker must run the backfill")
	}

	state := schemaState{
		Component: "core",
		Version:   clientPlatformBackfillVersion - 1,
	}
	if err := db.Create(&state).Error; err != nil {
		t.Fatal(err)
	}
	pending, err = schemaVersionBefore(db, clientPlatformBackfillVersion)
	if err != nil {
		t.Fatal(err)
	}
	if !pending {
		t.Fatal("an older schema must run the backfill")
	}

	if err := db.Model(&state).Update("version", clientPlatformBackfillVersion).Error; err != nil {
		t.Fatal(err)
	}
	pending, err = schemaVersionBefore(db, clientPlatformBackfillVersion)
	if err != nil {
		t.Fatal(err)
	}
	if pending {
		t.Fatal("a completed schema must never rerun the legacy backfill")
	}
}

func TestRecordSchemaReadyNeverDowngradesNewerMarker(t *testing.T) {
	t.Parallel()

	db, err := gorm.Open(sqlite.Open("file:schema-version-monotonic?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&schemaState{}); err != nil {
		t.Fatal(err)
	}

	newerUpdatedAt := time.Now().UTC().Add(-time.Hour).Truncate(time.Millisecond)
	newer := schemaState{
		Component: "core",
		Version:   currentSchemaVersion + 1,
		UpdatedAt: newerUpdatedAt,
	}
	if err := db.Create(&newer).Error; err != nil {
		t.Fatal(err)
	}
	if err := recordSchemaReady(db); err != nil {
		t.Fatal(err)
	}

	var got schemaState
	if err := db.Where("component = ?", "core").Take(&got).Error; err != nil {
		t.Fatal(err)
	}
	if got.Version != newer.Version {
		t.Fatalf("schema marker downgraded: got %d, want %d", got.Version, newer.Version)
	}
	if !got.UpdatedAt.Equal(newerUpdatedAt) {
		t.Fatalf("newer marker timestamp changed: got %s, want %s", got.UpdatedAt, newerUpdatedAt)
	}
}
