package database_test

import (
	"context"
	"testing"
	"time"

	"github.com/tsc-forex/tsc_forex/internal/config"
	"github.com/tsc-forex/tsc_forex/internal/database"
)

// TestNewUnreachable covers the connection-error path (no live Postgres needed).
func TestNewUnreachable(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
	defer cancel()
	_, err := database.New(ctx, config.DatabaseConfig{
		Host:         "127.0.0.1",
		Port:         1, // refused
		User:         "u",
		Password:     "p",
		Name:         "n",
		SSLMode:      "disable",
		MaxOpenConns: 1,
		MaxIdleConns: 1,
	})
	if err == nil {
		t.Fatal("expected connection error")
	}
}
