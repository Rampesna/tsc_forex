package service

import (
	"context"
	"strings"
	"unicode/utf8"
)

const (
	ClientPlatformIOS     = "ios"
	ClientPlatformAndroid = "android"
)

// ClientMetadata describes the mobile application that created or refreshed a
// consumer session. It deliberately lives on the refresh session rather than a
// push token: notification permission must never decide whether the operations
// console can identify an iOS or Android user.
type ClientMetadata struct {
	Platform    string
	Locale      string
	AppVersion  string
	DeviceModel string
}

type clientMetadataContextKey struct{}

// NormalizeClientMetadata allow-lists platform values and bounds every
// client-controlled field before it reaches persistence.
func NormalizeClientMetadata(platform, locale, appVersion, deviceModel string) ClientMetadata {
	platform = strings.ToLower(strings.TrimSpace(platform))
	if platform != ClientPlatformIOS && platform != ClientPlatformAndroid {
		platform = ""
	}
	return ClientMetadata{
		Platform:    platform,
		Locale:      truncateRunes(strings.TrimSpace(locale), 16),
		AppVersion:  truncateRunes(strings.TrimSpace(appVersion), 32),
		DeviceModel: truncateRunes(strings.TrimSpace(deviceModel), 96),
	}
}

// WithClientMetadata attaches trusted, normalized mobile metadata to a request
// context so every authentication flow (password, social, passkey and refresh)
// persists it through the same session-issuing path.
func WithClientMetadata(ctx context.Context, metadata ClientMetadata) context.Context {
	if metadata.Platform == "" {
		return ctx
	}
	return context.WithValue(ctx, clientMetadataContextKey{}, metadata)
}

// ClientMetadataFromContext returns mobile metadata when the request declared
// a supported native client platform.
func ClientMetadataFromContext(ctx context.Context) (ClientMetadata, bool) {
	metadata, ok := ctx.Value(clientMetadataContextKey{}).(ClientMetadata)
	return metadata, ok && metadata.Platform != ""
}

func truncateRunes(value string, limit int) string {
	if limit <= 0 || utf8.RuneCountInString(value) <= limit {
		return value
	}
	runes := []rune(value)
	return string(runes[:limit])
}
