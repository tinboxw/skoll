package plugin

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
)

var (
	ErrSignatureInvalid    = errors.New("signature verification failed")
	ErrSignatureAlgorithm  = errors.New("unsupported signature algorithm")
	ErrPublicKeyMissing    = errors.New("public key is required for signature verification")
	ErrPublicKeyInvalid    = errors.New("invalid public key format")
	ErrSignatureNotPresent = errors.New("signature not found in manifest")
)

// Verifier validates plugin manifest signatures
type SignatureVerifier interface {
	// Verify validates the signature of a manifest.
	// manifestData: the manifest content (without signature fields) in normalized form
	// sig: the signature structure containing algorithm, value, and public key
	Verify(manifestData []byte, sig *Signature) error
}

// RSAVerifier implements RSA-SHA256 signature verification
type RSAVerifier struct{}

// NewRSAVerifier creates a new RSA signature verifier
func NewRSAVerifier() *RSAVerifier {
	return &RSAVerifier{}
}

// Verify validates the RSA-SHA256 signature of manifest data
// The public key must be in PEM-encoded PKIX format (base64-decoded)
func (v *RSAVerifier) Verify(manifestData []byte, sig *Signature) error {
	if sig == nil {
		return ErrSignatureNotPresent
	}

	if sig.Algorithm != SigAlgoRSASHA256 {
		return ErrSignatureAlgorithm
	}

	if sig.PublicKey == "" {
		return ErrPublicKeyMissing
	}

	// Decode the public key from PEM format
	pubKey, err := parsePublicKey(sig.PublicKey)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrPublicKeyInvalid, err)
	}

	// Decode the signature from base64
	sigBytes, err := base64.StdEncoding.DecodeString(sig.Value)
	if err != nil {
		return fmt.Errorf("decode signature: %w", err)
	}

	// Compute SHA256 hash of the manifest data
	hash := sha256.Sum256(manifestData)

	// Verify the signature
	err = rsa.VerifyPKCS1v15(pubKey, crypto.SHA256, hash[:], sigBytes)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrSignatureInvalid, err)
	}

	return nil
}

// parsePublicKey parses a PEM-encoded RSA public key (base64-encoded)
// The key should be in PKIX format (BEGIN PUBLIC KEY)
func parsePublicKey(pemBase64 string) (*rsa.PublicKey, error) {
	// Decode from base64
	pemBytes, err := base64.StdEncoding.DecodeString(pemBase64)
	if err != nil {
		return nil, fmt.Errorf("decode base64 public key: %w", err)
	}

	// Parse PKIX public key
	pubKeyInterface, err := x509.ParsePKIXPublicKey(pemBytes)
	if err != nil {
		return nil, fmt.Errorf("parse PKIX public key: %w", err)
	}

	pubKey, ok := pubKeyInterface.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("public key is not RSA")
	}

	return pubKey, nil
}

// SignatureChecker validates manifest signatures without updating the store
type SignatureChecker struct {
	verifier SignatureVerifier
	logger   func(format string, args ...interface{})
}

// NewSignatureChecker creates a signature checker with default RSA verifier
func NewSignatureChecker() *SignatureChecker {
	return &SignatureChecker{
		verifier: NewRSAVerifier(),
		logger:   noOpLogger,
	}
}

// WithLogger sets a custom logger for signature verification messages
func (sc *SignatureChecker) WithLogger(logFunc func(format string, args ...interface{})) *SignatureChecker {
	sc.logger = logFunc
	return sc
}

// Check validates the signature of a plugin info object
// Returns:
// - nil: signature valid or not present (only warns)
// - error: signature present but invalid
func (sc *SignatureChecker) Check(info *Info) error {
	if info.Signature == nil {
		// No signature present - this is not an error, just a warning
		sc.logger("WARN: plugin %s has no signature", info.ID)
		return nil
	}

	// Signature present - must be valid
	// For now, we compute a canonical JSON representation of the manifest
	// (excluding signature fields)
	manifestData := sc.canonicalManifestBytes(info)

	err := sc.verifier.Verify(manifestData, info.Signature)
	if err != nil {
		sc.logger("ERROR: signature verification failed for plugin %s: %v", info.ID, err)
		return err
	}

	sc.logger("INFO: signature verified for plugin %s (vendor: %s)", info.ID, info.Vendor)
	return nil
}

// canonicalManifestBytes returns a canonical byte representation of the manifest
// excluding signature and vendor public key fields
func (sc *SignatureChecker) canonicalManifestBytes(info *Info) []byte {
	// Simple canonical form: reconstruct manifest without signature fields
	// In production, this should use a deterministic JSON/YAML format
	buf := &strings.Builder{}
	buf.WriteString("id: " + info.ID + "\n")
	buf.WriteString("name: " + info.Name + "\n")
	buf.WriteString("version: " + info.Version + "\n")
	if info.Description != "" {
		buf.WriteString("description: " + info.Description + "\n")
	}
	if info.Vendor != "" {
		buf.WriteString("vendor: " + info.Vendor + "\n")
	}
	if info.VendorURL != "" {
		buf.WriteString("vendor_url: " + info.VendorURL + "\n")
	}
	if info.Level != "" {
		buf.WriteString("level: " + string(info.Level) + "\n")
	}
	if info.AppID != "" {
		buf.WriteString("app_id: " + info.AppID + "\n")
	}
	if info.MountPolicy != "" {
		buf.WriteString("mount_policy: " + string(info.MountPolicy) + "\n")
	}
	if info.UIMode != "" {
		buf.WriteString("ui_mode: " + string(info.UIMode) + "\n")
	}
	if len(info.Dependencies) > 0 {
		buf.WriteString("dependencies:\n")
		for _, dep := range info.Dependencies {
			buf.WriteString("  - id: " + dep.ID)
			if dep.Version != "" {
				buf.WriteString(", version: " + dep.Version)
			}
			buf.WriteString("\n")
		}
	}
	if len(info.Permissions) > 0 {
		buf.WriteString("permissions:\n")
		for _, perm := range info.Permissions {
			buf.WriteString("  - " + perm + "\n")
		}
	}
	return []byte(buf.String())
}

// noOpLogger is a no-op logger function
func noOpLogger(format string, args ...interface{}) {
}

// GenerateSignatureExample generates an example RSA-SHA256 signature for testing
// Returns: (publicKeyPEM base64, signatureValue base64, error)
func GenerateSignatureExample(manifestData []byte) (string, string, error) {
	// Generate RSA key pair
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", "", err
	}

	// Sign the manifest
	hash := sha256.Sum256(manifestData)
	signature, err := rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.SHA256, hash[:])
	if err != nil {
		return "", "", err
	}

	// Export public key in PKIX format
	pubKeyBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		return "", "", err
	}

	pubKeyBase64 := base64.StdEncoding.EncodeToString(pubKeyBytes)
	sigBase64 := base64.StdEncoding.EncodeToString(signature)

	return pubKeyBase64, sigBase64, nil
}
