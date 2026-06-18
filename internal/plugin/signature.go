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
	"sort"
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
	if info.UINavPosition != "" {
		buf.WriteString("ui_nav_position: " + string(info.UINavPosition) + "\n")
	}
	if info.UIOpenMode != "" {
		buf.WriteString("ui_open_mode: " + string(info.UIOpenMode) + "\n")
	}
	if info.UITabMode != "" {
		buf.WriteString("ui_tab_mode: " + string(info.UITabMode) + "\n")
	}
	if info.UIMenu != nil {
		buf.WriteString("ui_menu:\n")
		if info.UIMenu.Key != "" {
			buf.WriteString("  key: " + strings.TrimSpace(info.UIMenu.Key) + "\n")
		}
		if info.UIMenu.ParentKey != "" {
			buf.WriteString("  parent_key: " + strings.TrimSpace(info.UIMenu.ParentKey) + "\n")
		}
		if info.UIMenu.Label != "" {
			buf.WriteString("  label: " + strings.TrimSpace(info.UIMenu.Label) + "\n")
		}
		if info.UIMenu.LabelZhCN != "" {
			buf.WriteString("  label_zh_cn: " + strings.TrimSpace(info.UIMenu.LabelZhCN) + "\n")
		}
		if info.UIMenu.LabelEnUS != "" {
			buf.WriteString("  label_en_us: " + strings.TrimSpace(info.UIMenu.LabelEnUS) + "\n")
		}
		if info.UIMenu.Path != "" {
			buf.WriteString("  path: " + strings.TrimSpace(info.UIMenu.Path) + "\n")
		}
		if info.UIMenu.Component != "" {
			buf.WriteString("  component: " + strings.TrimSpace(info.UIMenu.Component) + "\n")
		}
		if info.UIMenu.Icon != "" {
			buf.WriteString("  icon: " + strings.TrimSpace(info.UIMenu.Icon) + "\n")
		}
		if info.UIMenu.Order != 0 {
			buf.WriteString(fmt.Sprintf("  order: %d\n", info.UIMenu.Order))
		}
		if info.UIMenu.Visible != nil {
			buf.WriteString(fmt.Sprintf("  visible: %t\n", *info.UIMenu.Visible))
		}
		if len(info.UIMenu.RequiredRoles) > 0 {
			buf.WriteString("  required_roles:\n")
			for _, role := range info.UIMenu.RequiredRoles {
				buf.WriteString("    - " + strings.TrimSpace(role) + "\n")
			}
		}
		if len(info.UIMenu.RequiredPermissions) > 0 {
			buf.WriteString("  required_permissions:\n")
			for _, permission := range info.UIMenu.RequiredPermissions {
				buf.WriteString("    - " + strings.TrimSpace(permission) + "\n")
			}
		}
	}
	if info.ConfigSchema != nil {
		buf.WriteString("config_schema:\n")
		if info.ConfigSchema.Title != "" {
			buf.WriteString("  title: " + strings.TrimSpace(info.ConfigSchema.Title) + "\n")
		}
		if info.ConfigSchema.TitleZhCN != "" {
			buf.WriteString("  title_zh_cn: " + strings.TrimSpace(info.ConfigSchema.TitleZhCN) + "\n")
		}
		if info.ConfigSchema.TitleEnUS != "" {
			buf.WriteString("  title_en_us: " + strings.TrimSpace(info.ConfigSchema.TitleEnUS) + "\n")
		}
		if info.ConfigSchema.Description != "" {
			buf.WriteString("  description: " + strings.TrimSpace(info.ConfigSchema.Description) + "\n")
		}
		if len(info.ConfigSchema.Fields) > 0 {
			buf.WriteString("  fields:\n")
			for _, field := range info.ConfigSchema.Fields {
				buf.WriteString("    - key: " + strings.TrimSpace(field.Key) + "\n")
				if field.Label != "" {
					buf.WriteString("      label: " + strings.TrimSpace(field.Label) + "\n")
				}
				if field.LabelZhCN != "" {
					buf.WriteString("      label_zh_cn: " + strings.TrimSpace(field.LabelZhCN) + "\n")
				}
				if field.LabelEnUS != "" {
					buf.WriteString("      label_en_us: " + strings.TrimSpace(field.LabelEnUS) + "\n")
				}
				if field.Type != "" {
					buf.WriteString("      type: " + strings.TrimSpace(field.Type) + "\n")
				}
				if field.Required {
					buf.WriteString("      required: true\n")
				}
				if field.Default != "" {
					buf.WriteString("      default: " + strings.TrimSpace(field.Default) + "\n")
				}
				if field.Placeholder != "" {
					buf.WriteString("      placeholder: " + strings.TrimSpace(field.Placeholder) + "\n")
				}
				if field.Help != "" {
					buf.WriteString("      help: " + strings.TrimSpace(field.Help) + "\n")
				}
				if field.Min != nil {
					buf.WriteString(fmt.Sprintf("      min: %g\n", *field.Min))
				}
				if field.Max != nil {
					buf.WriteString(fmt.Sprintf("      max: %g\n", *field.Max))
				}
				if field.MinLength != nil {
					buf.WriteString(fmt.Sprintf("      min_length: %d\n", *field.MinLength))
				}
				if field.MaxLength != nil {
					buf.WriteString(fmt.Sprintf("      max_length: %d\n", *field.MaxLength))
				}
				if field.Pattern != "" {
					buf.WriteString("      pattern: " + strings.TrimSpace(field.Pattern) + "\n")
				}
				if len(field.Options) > 0 {
					buf.WriteString("      options:\n")
					for _, option := range field.Options {
						buf.WriteString("        - value: " + strings.TrimSpace(option.Value) + "\n")
						if option.Label != "" {
							buf.WriteString("          label: " + strings.TrimSpace(option.Label) + "\n")
						}
						if option.LabelZhCN != "" {
							buf.WriteString("          label_zh_cn: " + strings.TrimSpace(option.LabelZhCN) + "\n")
						}
						if option.LabelEnUS != "" {
							buf.WriteString("          label_en_us: " + strings.TrimSpace(option.LabelEnUS) + "\n")
						}
					}
				}
			}
		}
	}
	if len(info.I18nLocales) > 0 {
		buf.WriteString("i18n_locales:\n")
		for _, locale := range info.I18nLocales {
			buf.WriteString("  - " + strings.TrimSpace(locale) + "\n")
		}
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
	if len(info.PermissionResources) > 0 {
		buf.WriteString("permissions:\n")
		for _, perm := range info.PermissionResources {
			buf.WriteString("  - key: " + strings.TrimSpace(perm.Key) + "\n")
			if perm.Type != "" {
				buf.WriteString("    type: " + strings.TrimSpace(perm.Type) + "\n")
			}
			if perm.Module != "" {
				buf.WriteString("    module: " + strings.TrimSpace(perm.Module) + "\n")
			}
			if perm.Name != "" {
				buf.WriteString("    name: " + strings.TrimSpace(perm.Name) + "\n")
			}
			if perm.Risk != "" {
				buf.WriteString("    risk: " + strings.TrimSpace(perm.Risk) + "\n")
			}
			if len(perm.Metadata) > 0 {
				keys := make([]string, 0, len(perm.Metadata))
				for key := range perm.Metadata {
					keys = append(keys, key)
				}
				sort.Strings(keys)
				for _, key := range keys {
					value := perm.Metadata[key]
					buf.WriteString("    metadata." + strings.TrimSpace(key) + ": " + strings.TrimSpace(value) + "\n")
				}
			}
		}
	} else if len(info.Permissions) > 0 {
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
