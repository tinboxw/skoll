package system

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/tinboxw/skoll/internal/domain/shared"
)

var settingKeyPattern = regexp.MustCompile(`^[a-z][a-z0-9_.-]{1,127}$`)

type Setting struct {
	ID        shared.ID
	Key       string
	Value     string
	Encrypted bool
	Meta      shared.AuditMeta
}

func NewSetting(id shared.ID, key, value string, encrypted bool, now time.Time) (*Setting, error) {
	if id.IsZero() {
		return nil, fmt.Errorf("id is required")
	}
	v := strings.TrimSpace(key)
	if !settingKeyPattern.MatchString(v) {
		return nil, fmt.Errorf("invalid setting key")
	}
	s := &Setting{
		ID:        id,
		Key:       strings.ToLower(v),
		Value:     value,
		Encrypted: encrypted,
	}
	s.Meta.Touch(now)
	return s, nil
}

func (s *Setting) UpdateValue(value string, now time.Time) {
	s.Value = value
	s.Meta.Touch(now)
}
