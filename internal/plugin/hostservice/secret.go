package hostservice

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"

	domainsystem "github.com/tinboxw/skoll/internal/domain/system"
	systemsvc "github.com/tinboxw/skoll/internal/service/system"
	"github.com/tinboxw/skoll/pkg/pluginsdk"
	"github.com/tinboxw/skoll/pkg/security"
)

var secretNamePattern = regexp.MustCompile(`^[a-z][a-z0-9_.-]{1,63}$`)

type secretBackend interface {
	Upsert(ctx context.Context, in systemsvc.UpsertInput) (*domainsystem.Setting, error)
	GetByKey(ctx context.Context, key string) (*domainsystem.Setting, error)
}

type secretService struct {
	pluginID string
	settings secretBackend
	audit    pluginsdk.AuditService
	key      []byte
}

func NewSecretService(pluginID string, settings secretBackend, audit pluginsdk.AuditService, masterSecret string) (pluginsdk.SecretService, error) {
	pluginID = strings.ToLower(strings.TrimSpace(pluginID))
	if pluginID == "" {
		return nil, fmt.Errorf("plugin host identity is required")
	}
	if settings == nil || audit == nil {
		return nil, fmt.Errorf("plugin host secret dependencies are required")
	}
	masterSecret = strings.TrimSpace(masterSecret)
	if len(masterSecret) < 16 {
		return nil, fmt.Errorf("plugin host master secret must contain at least 16 characters")
	}
	key := sha256.Sum256([]byte("skoll/plugin-secrets/v1\x00" + masterSecret))
	return &secretService{pluginID: pluginID, settings: settings, audit: audit, key: key[:]}, nil
}

func (s *secretService) Get(ctx context.Context, key string) (string, error) {
	key, err := normalizeSecretName(key)
	if err != nil {
		return "", err
	}
	setting, err := s.settings.GetByKey(ctx, s.storageKey(key))
	if err != nil {
		return "", err
	}
	if setting == nil {
		return "", fmt.Errorf("plugin secret not found")
	}
	if !setting.Encrypted {
		return "", fmt.Errorf("plugin secret storage is not encrypted")
	}
	plain, err := security.Decrypt(setting.Value, s.key)
	if err != nil {
		return "", fmt.Errorf("decrypt plugin secret: %w", err)
	}
	return string(plain), nil
}

func (s *secretService) Set(ctx context.Context, key, value string) error {
	key, err := normalizeSecretName(key)
	if err != nil {
		return err
	}
	if value == "" {
		return fmt.Errorf("plugin secret value is required")
	}
	ciphertext, err := security.Encrypt([]byte(value), s.key)
	if err != nil {
		return err
	}
	if _, err := s.settings.Upsert(ctx, systemsvc.UpsertInput{Key: s.storageKey(key), Value: ciphertext, Encrypted: true}); err != nil {
		return err
	}
	_, err = s.audit.Record(ctx, pluginsdk.AuditEntry{
		Action: "secret.set", Resource: "secret", ResourceID: key,
		Risk: pluginsdk.AuditRiskHigh, Detail: map[string]any{"secretKey": key},
	})
	return err
}

func (s *secretService) storageKey(key string) string {
	hash := sha256.Sum256([]byte(s.pluginID))
	return "plugin.secret." + hex.EncodeToString(hash[:12]) + "." + key
}

func normalizeSecretName(value string) (string, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	if !secretNamePattern.MatchString(value) {
		return "", fmt.Errorf("plugin secret key is invalid")
	}
	return value, nil
}

var _ pluginsdk.SecretService = (*secretService)(nil)
