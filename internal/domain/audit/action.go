package audit

import (
	"fmt"
	"regexp"
	"strings"
)

type AuditAction string

var auditActionPattern = regexp.MustCompile(`^[a-z][a-z0-9_-]{0,63}\.[a-z][a-z0-9_-]{0,63}\.[a-z][a-z0-9_-]{0,63}$`)

func ParseAuditAction(value string) (AuditAction, error) {
	action := AuditAction(strings.TrimSpace(strings.ToLower(value)))
	if err := action.Validate(); err != nil {
		return "", err
	}
	return action, nil
}

func (a AuditAction) Validate() error {
	if !auditActionPattern.MatchString(string(a)) {
		return fmt.Errorf("audit action must match module.resource.action")
	}
	return nil
}

func (a AuditAction) Module() string {
	parts := strings.Split(string(a), ".")
	if len(parts) != 3 {
		return ""
	}
	return parts[0]
}

func (a AuditAction) Resource() string {
	parts := strings.Split(string(a), ".")
	if len(parts) != 3 {
		return ""
	}
	return parts[1]
}

func (a AuditAction) Operation() string {
	parts := strings.Split(string(a), ".")
	if len(parts) != 3 {
		return ""
	}
	return parts[2]
}

func (a AuditAction) String() string {
	return string(a)
}
