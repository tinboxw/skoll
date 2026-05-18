package plugin

import (
	"encoding/base64"
	"testing"
	"time"
)

func TestSignatureVerification(t *testing.T) {
	// Generate test data
	manifestData := []byte("id: test-plugin\nname: Test Plugin\nversion: 1.0.0\n")

	pubKey, sigValue, err := GenerateSignatureExample(manifestData)
	if err != nil {
		t.Fatalf("failed to generate signature: %v", err)
	}

	// Create signature structure
	sig := &Signature{
		Algorithm: SigAlgoRSASHA256,
		Timestamp: time.Now(),
		Value:     sigValue,
		PublicKey: pubKey,
		VendorID:  "test-vendor",
	}

	// Verify signature
	verifier := NewRSAVerifier()
	err = verifier.Verify(manifestData, sig)
	if err != nil {
		t.Fatalf("signature verification failed: %v", err)
	}
}

func TestSignatureVerificationWithInvalidSignature(t *testing.T) {
	manifestData := []byte("id: test-plugin\nname: Test Plugin\nversion: 1.0.0\n")

	pubKey, _, err := GenerateSignatureExample(manifestData)
	if err != nil {
		t.Fatalf("failed to generate signature: %v", err)
	}

	// Create signature with invalid value
	sig := &Signature{
		Algorithm: SigAlgoRSASHA256,
		Timestamp: time.Now(),
		Value:     base64.StdEncoding.EncodeToString([]byte("invalid signature")),
		PublicKey: pubKey,
		VendorID:  "test-vendor",
	}

	// Verify should fail
	verifier := NewRSAVerifier()
	err = verifier.Verify(manifestData, sig)
	if err == nil {
		t.Errorf("expected signature verification to fail, but it succeeded")
	}
}

func TestSignatureValidationInManifest(t *testing.T) {
	// Valid signature
	sig := &Signature{
		Algorithm: SigAlgoRSASHA256,
		Timestamp: time.Now(),
		Value:     "base64-signature",
		PublicKey: "base64-pubkey",
	}

	info := Info{
		ID:        "test-plugin",
		Name:      "Test",
		Version:   "1.0.0",
		Signature: sig,
	}

	err := info.ValidateManifest()
	if err != nil {
		t.Errorf("expected valid manifest, got error: %v", err)
	}

	// Invalid signature (empty value)
	invalidSig := &Signature{
		Algorithm: SigAlgoRSASHA256,
		Timestamp: time.Now(),
		Value:     "",
		PublicKey: "base64-pubkey",
	}

	info.Signature = invalidSig
	err = info.ValidateManifest()
	if err != ErrPluginManifestBroken {
		t.Errorf("expected ErrPluginManifestBroken, got: %v", err)
	}

	// Invalid signature (zero timestamp)
	invalidSig2 := &Signature{
		Algorithm: SigAlgoRSASHA256,
		Timestamp: time.Time{},
		Value:     "base64-signature",
		PublicKey: "base64-pubkey",
	}

	info.Signature = invalidSig2
	err = info.ValidateManifest()
	if err != ErrPluginManifestBroken {
		t.Errorf("expected ErrPluginManifestBroken, got: %v", err)
	}

	// Invalid algorithm
	invalidSig3 := &Signature{
		Algorithm: SignatureAlgorithm("unsupported"),
		Timestamp: time.Now(),
		Value:     "base64-signature",
		PublicKey: "base64-pubkey",
	}

	info.Signature = invalidSig3
	err = info.ValidateManifest()
	if err != ErrPluginManifestBroken {
		t.Errorf("expected ErrPluginManifestBroken, got: %v", err)
	}
}

func TestSignatureCheckerWithoutSignature(t *testing.T) {
	checker := NewSignatureChecker()

	// Create a log buffer to capture output
	logs := []string{}
	checker.WithLogger(func(format string, args ...interface{}) {
		logs = append(logs, format)
	})

	info := &Info{
		ID:      "test-plugin",
		Name:    "Test",
		Version: "1.0.0",
	}

	// Should not error, but should warn
	err := checker.Check(info)
	if err != nil {
		t.Errorf("expected no error for plugin without signature, got: %v", err)
	}

	if len(logs) == 0 || logs[0] != "WARN: plugin %s has no signature" {
		t.Errorf("expected warning log, got logs: %v", logs)
	}
}

func TestSignatureCheckerWithValidSignature(t *testing.T) {
	// Create a simple info object
	info := &Info{
		ID:      "test-plugin",
		Name:    "Test",
		Version: "1.0.0",
		Vendor:  "test-vendor",
	}

	checker := NewSignatureChecker()

	// Generate canonical manifest bytes from the info
	manifestData := checker.canonicalManifestBytes(info)

	// Generate signature for this canonical data
	pubKey, sigValue, err := GenerateSignatureExample(manifestData)
	if err != nil {
		t.Fatalf("failed to generate signature: %v", err)
	}

	// Create signature structure
	info.Signature = &Signature{
		Algorithm: SigAlgoRSASHA256,
		Timestamp: time.Now(),
		Value:     sigValue,
		PublicKey: pubKey,
	}

	// Create a log buffer to capture output
	logs := []string{}
	checker.WithLogger(func(format string, args ...interface{}) {
		logs = append(logs, format)
	})
	// Should verify successfully
	err = checker.Check(info)
	if err != nil {
		t.Errorf("expected signature verification to succeed, got error: %v", err)
	}

	if len(logs) == 0 || logs[0] != "INFO: signature verified for plugin %s (vendor: %s)" {
		t.Errorf("expected INFO log, got logs: %v", logs)
	}
}

func TestCanonicalManifestBytes(t *testing.T) {
	checker := NewSignatureChecker()

	info := &Info{
		ID:          "test-plugin",
		Name:        "Test",
		Version:     "1.0.0",
		Description: "A test plugin",
		Vendor:      "test-vendor",
		Level:       LevelApp,
		AppID:       "myapp",
		MountPolicy: MountPolicyAdmin,
		UIMode:      UIModeBackendOnly,
	}

	bytes := checker.canonicalManifestBytes(info)
	str := string(bytes)

	// Verify canonical form contains expected fields
	checks := map[string]bool{
		"id: test-plugin":            false,
		"name: Test":                 false,
		"version: 1.0.0":             false,
		"description: A test plugin": false,
		"vendor: test-vendor":        false,
		"level: app":                 false,
		"app_id: myapp":              false,
		"mount_policy: admin":        false,
		"ui_mode: backend_only":      false,
	}

	for expected := range checks {
		if _, found := checks[expected]; found && !stringContains(str, expected) {
			t.Errorf("expected canonical form to contain %q, got:\n%s", expected, str)
		}
	}
}

func TestHashConsistency(t *testing.T) {
	// Generate two signatures from same data
	manifestData := []byte("id: test-plugin\nname: Test\nversion: 1.0.0\n")

	pubKey1, sigValue1, err := GenerateSignatureExample(manifestData)
	if err != nil {
		t.Fatalf("failed to generate signature: %v", err)
	}

	pubKey2, sigValue2, err := GenerateSignatureExample(manifestData)
	if err != nil {
		t.Fatalf("failed to generate second signature: %v", err)
	}

	verifier := NewRSAVerifier()

	// Verify both signatures against their respective public keys
	sig1 := &Signature{
		Algorithm: SigAlgoRSASHA256,
		Timestamp: time.Now(),
		Value:     sigValue1,
		PublicKey: pubKey1,
	}

	sig2 := &Signature{
		Algorithm: SigAlgoRSASHA256,
		Timestamp: time.Now(),
		Value:     sigValue2,
		PublicKey: pubKey2,
	}

	if err := verifier.Verify(manifestData, sig1); err != nil {
		t.Errorf("first signature verification failed: %v", err)
	}

	if err := verifier.Verify(manifestData, sig2); err != nil {
		t.Errorf("second signature verification failed: %v", err)
	}

	// Cross-verify should fail (different public keys)
	if err := verifier.Verify(manifestData, &Signature{
		Algorithm: SigAlgoRSASHA256,
		Timestamp: time.Now(),
		Value:     sigValue1,
		PublicKey: pubKey2, // Wrong key
	}); err == nil {
		t.Errorf("cross-verification with wrong key should have failed")
	}
}

func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
