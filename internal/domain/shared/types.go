package shared

import "time"

// ID is the canonical identifier type used across domain entities.
type ID string

func (id ID) String() string {
	return string(id)
}

func (id ID) IsZero() bool {
	return id == ""
}

// TimeRange represents an inclusive time window.
type TimeRange struct {
	From time.Time
	To   time.Time
}

func (r TimeRange) IsValid() bool {
	if r.From.IsZero() || r.To.IsZero() {
		return false
	}
	return !r.To.Before(r.From)
}

// AuditMeta stores common creation/update timestamps for domain entities.
type AuditMeta struct {
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (m *AuditMeta) Touch(now time.Time) {
	if m.CreatedAt.IsZero() {
		m.CreatedAt = now
	}
	m.UpdatedAt = now
}
