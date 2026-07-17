package service

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/sha256"
	"crypto/x509"
	"embed"
	"encoding/asn1"
	"encoding/base64"
	"encoding/binary"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/fxamacker/cbor/v2"

	"github.com/tsc-forex/tsc_forex/internal/config"
)

const (
	maxAppAttestObjectBytes = 64 << 10
	maxAppAttestReceipt     = 32 << 10
	maxAppAttestCertBytes   = 8 << 10
)

var (
	errInvalidAppAttestation = errors.New("invalid app attest object")
	appAttestNonceOID        = asn1.ObjectIdentifier{1, 2, 840, 113635, 100, 8, 2}
)

//go:embed apple_app_attestation_root.pem
var embeddedAppAttestRoot embed.FS

type AppAttestResult struct {
	PublicKeyDER []byte
	Receipt      []byte
	Environment  string
}

// AppAttestVerifier is injectable for deterministic challenge/counter tests.
// The production implementation follows Apple's attestation-object validation
// guide and pins Apple's dedicated App Attestation root CA.
type AppAttestVerifier interface {
	Ready() bool
	VerifyAttestation(attestationObject, clientDataHash []byte, keyID string) (AppAttestResult, error)
	VerifyAssertion(assertionObject, clientData, publicKeyDER []byte, previousCounter uint32) (uint32, error)
}

type AppleAppAttestVerifier struct {
	appID          string
	environment    string
	bundleVersions map[string]struct{}
	roots          *x509.CertPool
	loadErr        error
	now            func() time.Time
}

func NewAppleAppAttestVerifier(cfg config.BillingConfig) *AppleAppAttestVerifier {
	v := &AppleAppAttestVerifier{
		appID:          strings.TrimSpace(cfg.AppAttestTeamID) + "." + strings.TrimSpace(cfg.BundleID),
		environment:    strings.ToLower(strings.TrimSpace(cfg.AppAttestEnvironment)),
		bundleVersions: make(map[string]struct{}), now: time.Now,
	}
	for _, version := range cfg.AppAttestBundleVersions {
		if version = strings.TrimSpace(version); version != "" {
			v.bundleVersions[version] = struct{}{}
		}
	}
	rootPEM, err := embeddedAppAttestRoot.ReadFile("apple_app_attestation_root.pem")
	if path := strings.TrimSpace(cfg.AppAttestRootCAPath); path != "" {
		rootPEM, err = os.ReadFile(path)
	}
	if err != nil {
		v.loadErr = err
		return v
	}
	block, _ := pem.Decode(rootPEM)
	if block == nil || block.Type != "CERTIFICATE" {
		v.loadErr = errors.New("invalid App Attest root PEM")
		return v
	}
	root, err := x509.ParseCertificate(block.Bytes)
	if err != nil || root.Subject.CommonName != "Apple App Attestation Root CA" ||
		!appleRootOrgContains(root.Subject.Organization, "Apple Inc.") {
		v.loadErr = errors.New("unexpected App Attest root certificate")
		return v
	}
	roots := x509.NewCertPool()
	roots.AddCert(root)
	v.roots = roots
	return v
}

func (v *AppleAppAttestVerifier) Ready() bool {
	if v == nil || v.loadErr != nil || v.roots == nil || strings.TrimSpace(v.appID) == "." {
		return false
	}
	if v.environment != "development" && v.environment != "production" {
		return false
	}
	return len(v.bundleVersions) > 0
}

type appAttestationObject struct {
	Format    string `cbor:"fmt"`
	AuthData  []byte `cbor:"authData"`
	Statement struct {
		Certificates [][]byte `cbor:"x5c"`
		Receipt      []byte   `cbor:"receipt"`
	} `cbor:"attStmt"`
}

type appAssertionObject struct {
	Signature []byte `cbor:"signature"`
	AuthData  []byte `cbor:"authenticatorData"`
}

type parsedAttestAuthData struct {
	rpIDHash     []byte
	flags        byte
	counter      uint32
	aaguid       []byte
	credentialID []byte
	extensions   map[string]any
}

func (v *AppleAppAttestVerifier) VerifyAttestation(attestationObject, clientDataHash []byte, keyID string) (AppAttestResult, error) {
	if !v.Ready() || len(attestationObject) == 0 || len(attestationObject) > maxAppAttestObjectBytes || len(clientDataHash) != sha256.Size {
		return AppAttestResult{}, errInvalidAppAttestation
	}
	var object appAttestationObject
	if err := cbor.Unmarshal(attestationObject, &object); err != nil || object.Format != "apple-appattest" ||
		len(object.Statement.Certificates) < 2 || len(object.Statement.Certificates) > 3 ||
		len(object.Statement.Receipt) == 0 || len(object.Statement.Receipt) > maxAppAttestReceipt {
		return AppAttestResult{}, errInvalidAppAttestation
	}
	certificates := make([]*x509.Certificate, 0, len(object.Statement.Certificates))
	for _, der := range object.Statement.Certificates {
		if len(der) == 0 || len(der) > maxAppAttestCertBytes {
			return AppAttestResult{}, errInvalidAppAttestation
		}
		certificate, err := x509.ParseCertificate(der)
		if err != nil {
			return AppAttestResult{}, errInvalidAppAttestation
		}
		certificates = append(certificates, certificate)
	}
	intermediates := x509.NewCertPool()
	for _, certificate := range certificates[1:] {
		intermediates.AddCert(certificate)
	}
	if _, err := certificates[0].Verify(x509.VerifyOptions{
		Roots: v.roots, Intermediates: intermediates, CurrentTime: v.now(),
		KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
	}); err != nil {
		return AppAttestResult{}, errInvalidAppAttestation
	}

	parsed, err := parseAttestationAuthData(object.AuthData)
	if err != nil || parsed.counter != 0 || !v.validRPID(parsed.rpIDHash) || !v.validAAGUID(parsed.aaguid) ||
		!v.validExtensions(parsed.extensions) {
		return AppAttestResult{}, errInvalidAppAttestation
	}
	keyBytes, err := decodeAppAttestKeyID(keyID)
	if err != nil || !bytes.Equal(parsed.credentialID, keyBytes) {
		return AppAttestResult{}, errInvalidAppAttestation
	}
	publicKey, ok := certificates[0].PublicKey.(*ecdsa.PublicKey)
	if !ok || publicKey.Curve != elliptic.P256() {
		return AppAttestResult{}, errInvalidAppAttestation
	}
	point := elliptic.Marshal(elliptic.P256(), publicKey.X, publicKey.Y)
	pointHash := sha256.Sum256(point)
	if !bytes.Equal(pointHash[:], keyBytes) {
		return AppAttestResult{}, errInvalidAppAttestation
	}
	nonceInput := make([]byte, 0, len(object.AuthData)+len(clientDataHash))
	nonceInput = append(nonceInput, object.AuthData...)
	nonceInput = append(nonceInput, clientDataHash...)
	nonce := sha256.Sum256(nonceInput)
	certificateNonce, err := appAttestCertificateNonce(certificates[0])
	if err != nil || !bytes.Equal(certificateNonce, nonce[:]) {
		return AppAttestResult{}, errInvalidAppAttestation
	}
	publicKeyDER, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		return AppAttestResult{}, errInvalidAppAttestation
	}
	return AppAttestResult{
		PublicKeyDER: publicKeyDER,
		Receipt:      append([]byte(nil), object.Statement.Receipt...),
		Environment:  v.environment,
	}, nil
}

func (v *AppleAppAttestVerifier) VerifyAssertion(assertionObject, clientData, publicKeyDER []byte, previousCounter uint32) (uint32, error) {
	if !v.Ready() || len(assertionObject) == 0 || len(assertionObject) > maxAppAttestObjectBytes || len(clientData) == 0 || len(clientData) > 4096 {
		return 0, errInvalidAppAttestation
	}
	var object appAssertionObject
	if err := cbor.Unmarshal(assertionObject, &object); err != nil || len(object.Signature) == 0 || len(object.Signature) > 128 {
		return 0, errInvalidAppAttestation
	}
	parsed, err := parseAssertionAuthData(object.AuthData)
	if err != nil || !v.validRPID(parsed.rpIDHash) || parsed.counter == 0 || parsed.counter <= previousCounter ||
		!v.validExtensions(parsed.extensions) {
		return 0, errInvalidAppAttestation
	}
	parsedKey, err := x509.ParsePKIXPublicKey(publicKeyDER)
	if err != nil {
		return 0, errInvalidAppAttestation
	}
	publicKey, ok := parsedKey.(*ecdsa.PublicKey)
	if !ok || publicKey.Curve != elliptic.P256() {
		return 0, errInvalidAppAttestation
	}
	clientDataHash := sha256.Sum256(clientData)
	nonceInput := make([]byte, 0, len(object.AuthData)+len(clientDataHash))
	nonceInput = append(nonceInput, object.AuthData...)
	nonceInput = append(nonceInput, clientDataHash[:]...)
	nonce := sha256.Sum256(nonceInput)
	if !ecdsa.VerifyASN1(publicKey, nonce[:], object.Signature) {
		return 0, errInvalidAppAttestation
	}
	return parsed.counter, nil
}

func parseAttestationAuthData(data []byte) (parsedAttestAuthData, error) {
	if len(data) < 37+16+2 || data[32]&0x40 == 0 {
		return parsedAttestAuthData{}, errInvalidAppAttestation
	}
	parsed := parsedAttestAuthData{
		rpIDHash: append([]byte(nil), data[:32]...), flags: data[32], counter: binary.BigEndian.Uint32(data[33:37]),
		aaguid: append([]byte(nil), data[37:53]...),
	}
	credentialLength := int(binary.BigEndian.Uint16(data[53:55]))
	if credentialLength != sha256.Size || len(data) < 55+credentialLength+1 {
		return parsedAttestAuthData{}, errInvalidAppAttestation
	}
	parsed.credentialID = append([]byte(nil), data[55:55+credentialLength]...)
	rest := data[55+credentialLength:]
	var coseKey map[int64]any
	rest, err := cbor.UnmarshalFirst(rest, &coseKey)
	if err != nil || !validCOSEP256Key(coseKey) {
		return parsedAttestAuthData{}, errInvalidAppAttestation
	}
	if parsed.flags&0x80 == 0 {
		return parsedAttestAuthData{}, errInvalidAppAttestation
	}
	if err := cbor.Unmarshal(rest, &parsed.extensions); err != nil {
		return parsedAttestAuthData{}, errInvalidAppAttestation
	}
	return parsed, nil
}

func parseAssertionAuthData(data []byte) (parsedAttestAuthData, error) {
	if len(data) < 37 {
		return parsedAttestAuthData{}, errInvalidAppAttestation
	}
	parsed := parsedAttestAuthData{
		rpIDHash: append([]byte(nil), data[:32]...), flags: data[32], counter: binary.BigEndian.Uint32(data[33:37]),
	}
	if parsed.flags&0x80 == 0 || len(data) == 37 {
		return parsedAttestAuthData{}, errInvalidAppAttestation
	}
	if err := cbor.Unmarshal(data[37:], &parsed.extensions); err != nil {
		return parsedAttestAuthData{}, errInvalidAppAttestation
	}
	return parsed, nil
}

func validCOSEP256Key(key map[int64]any) bool {
	integer := func(value any) (int64, bool) {
		switch n := value.(type) {
		case int64:
			return n, true
		case uint64:
			return int64(n), true
		default:
			return 0, false
		}
	}
	kty, ok1 := integer(key[1])
	alg, ok2 := integer(key[3])
	curve, ok3 := integer(key[-1])
	x, ok4 := key[-2].([]byte)
	y, ok5 := key[-3].([]byte)
	return ok1 && ok2 && ok3 && ok4 && ok5 && kty == 2 && alg == -7 && curve == 1 && len(x) == 32 && len(y) == 32
}

func (v *AppleAppAttestVerifier) validRPID(value []byte) bool {
	want := sha256.Sum256([]byte(v.appID))
	return bytes.Equal(value, want[:])
}

func (v *AppleAppAttestVerifier) validAAGUID(value []byte) bool {
	want := []byte("appattestdevelop")
	if v.environment == "production" {
		want = append([]byte("appattest"), make([]byte, 7)...)
	}
	return bytes.Equal(value, want)
}

func (v *AppleAppAttestVerifier) validExtensions(extensions map[string]any) bool {
	if len(extensions) == 0 {
		return false
	}
	category, ok := numericExtension(extensions["apple_validation_category_01"])
	if !ok {
		return false
	}
	if v.environment == "development" {
		if category != 3 {
			return false
		}
	} else if category != 2 && category != 4 {
		return false
	}
	version, ok := extensions["apple_bundle_version_01"].(string)
	if !ok {
		return false
	}
	_, ok = v.bundleVersions[version]
	return ok
}

func numericExtension(value any) (uint64, bool) {
	switch number := value.(type) {
	case uint64:
		return number, true
	case int64:
		return uint64(number), number >= 0
	case uint32:
		return uint64(number), true
	case int:
		return uint64(number), number >= 0
	default:
		return 0, false
	}
}

func decodeAppAttestKeyID(keyID string) ([]byte, error) {
	keyID = strings.TrimSpace(keyID)
	for _, encoding := range []*base64.Encoding{base64.StdEncoding, base64.RawStdEncoding, base64.RawURLEncoding} {
		decoded, err := encoding.DecodeString(keyID)
		if err == nil && len(decoded) == sha256.Size {
			return decoded, nil
		}
	}
	return nil, errInvalidAppAttestation
}

func appAttestCertificateNonce(certificate *x509.Certificate) ([]byte, error) {
	for _, extension := range certificate.Extensions {
		if !extension.Id.Equal(appAttestNonceOID) {
			continue
		}
		var nonce struct {
			Value []byte `asn1:"tag:1,explicit"`
		}
		if rest, err := asn1.Unmarshal(extension.Value, &nonce); err == nil && len(rest) == 0 && len(nonce.Value) == sha256.Size {
			return nonce.Value, nil
		}
		return nil, errInvalidAppAttestation
	}
	return nil, fmt.Errorf("%w: nonce extension missing", errInvalidAppAttestation)
}
