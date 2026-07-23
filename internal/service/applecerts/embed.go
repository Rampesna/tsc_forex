package applecerts

import "embed"

// RootPEMs are Apple's published root CAs from https://www.apple.com/certificateauthority/.
// Alpine/Mozilla trust stores do not include these, so App Store Server API /
// StoreKit 2 JWS chains must be verified against an explicitly embedded pool.
//
//go:embed AppleIncRootCertificate.pem AppleRootCA-G2.pem AppleRootCA-G3.pem
var RootPEMs embed.FS
