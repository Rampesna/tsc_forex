package utils

// Build-time variables; overridden via -ldflags in CI/Docker.
var (
	Version   = "0.1.0-dev"
	Commit    = "unknown"
	BuildTime = "unknown"
)
