package service

import (
	"context"
	"strings"
	"testing"
)

func TestNormalizeClientMetadata(t *testing.T) {
	t.Parallel()
	metadata := NormalizeClientMetadata(
		" Android ",
		"tr-TR",
		strings.Repeat("1", 40),
		strings.Repeat("telefon", 20),
	)
	if metadata.Platform != ClientPlatformAndroid || metadata.Locale != "tr-TR" {
		t.Fatalf("metadata=%+v", metadata)
	}
	if len([]rune(metadata.AppVersion)) != 32 || len([]rune(metadata.DeviceModel)) != 96 {
		t.Fatalf("metadata was not bounded: %+v", metadata)
	}
	if invalid := NormalizeClientMetadata("web", "", "", ""); invalid.Platform != "" {
		t.Fatalf("unsupported platform accepted: %+v", invalid)
	}
}

func TestClientMetadataContext(t *testing.T) {
	t.Parallel()
	want := NormalizeClientMetadata("ios", "en-US", "1.2.0", "iPhone")
	ctx := WithClientMetadata(context.Background(), want)
	got, ok := ClientMetadataFromContext(ctx)
	if !ok || got != want {
		t.Fatalf("got=%+v ok=%v want=%+v", got, ok, want)
	}
	if _, ok := ClientMetadataFromContext(context.Background()); ok {
		t.Fatal("empty context unexpectedly contains client metadata")
	}
}
