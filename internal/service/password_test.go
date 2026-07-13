package service_test

import (
	"testing"

	"golang.org/x/crypto/bcrypt"

	"github.com/tsc-forex/tsc_forex/internal/service"
)

func TestArgon2idHashVerify(t *testing.T) {
	t.Parallel()
	hash, err := service.HashPassword("s3cret-passw0rd")
	if err != nil {
		t.Fatal(err)
	}
	if len(hash) < 20 || hash[:10] != "$argon2id$" {
		t.Fatalf("unexpected hash format: %s", hash)
	}
	if err := service.VerifyPassword(hash, "s3cret-passw0rd"); err != nil {
		t.Fatalf("verify valid failed: %v", err)
	}
	if err := service.VerifyPassword(hash, "wrong"); err == nil {
		t.Fatal("verify wrong should fail")
	}
}

func TestVerifyBcryptLegacy(t *testing.T) {
	t.Parallel()
	// A hash produced by the previous bcrypt scheme must still verify.
	legacy, _ := bcrypt.GenerateFromPassword([]byte("legacy-pass"), bcrypt.MinCost)
	if err := service.VerifyPassword(string(legacy), "legacy-pass"); err != nil {
		t.Fatalf("bcrypt legacy verify failed: %v", err)
	}
	if err := service.VerifyPassword(string(legacy), "nope"); err == nil {
		t.Fatal("bcrypt wrong should fail")
	}
}

func TestVerifyInvalidHash(t *testing.T) {
	t.Parallel()
	if err := service.VerifyPassword("$argon2id$broken", "x"); err == nil {
		t.Fatal("expected invalid hash error")
	}
}
