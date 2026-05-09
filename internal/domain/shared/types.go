package shared

import "time"

// ID identifies aggregate roots and entities across domains.
type ID string

type AuditInfo struct {
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Pager struct {
	Page     int
	PageSize int
}

func (p Pager) Normalize(defaultSize int) Pager {
	if p.Page < 1 {
		p.Page = 1
	}
	if p.PageSize < 1 {
		p.PageSize = defaultSize
	}
	return p
}
