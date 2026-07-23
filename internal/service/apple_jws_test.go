package service

import (
	"strings"
	"testing"
)

func TestAppleRootCertPoolLoadsPublishedRoots(t *testing.T) {
	roots, err := appleRootCertPool()
	if err != nil {
		t.Fatalf("appleRootCertPool: %v", err)
	}
	if roots == nil {
		t.Fatal("expected non-nil Apple root pool")
	}
	// Subjects are checked at load time; ensure the helper still rejects empty pools
	// by verifying NewAppleJWSVerifier succeeds with the embedded PEMs.
	verifier, err := NewAppleJWSVerifier()
	if err != nil {
		t.Fatalf("NewAppleJWSVerifier: %v", err)
	}
	if _, err := verifier.VerifyTransaction("not-a-jws"); err == nil || !strings.Contains(err.Error(), "malformed") {
		t.Fatalf("expected malformed JWS error, got %v", err)
	}
}
