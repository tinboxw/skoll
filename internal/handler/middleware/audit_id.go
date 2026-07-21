package middleware

import (
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/tinboxw/skoll/internal/domain/shared"
)

var auditIDSequence atomic.Uint64

func NewAuditID(prefix string, occurredAt time.Time) shared.ID {
	prefix = strings.TrimSpace(prefix)
	if prefix == "" {
		prefix = "audit-event"
	}
	if occurredAt.IsZero() {
		occurredAt = time.Now().UTC()
	}
	return shared.ID(prefix + "-" + strconv.FormatInt(occurredAt.UTC().UnixNano(), 10) + "-" + strconv.FormatUint(auditIDSequence.Add(1), 10))
}
