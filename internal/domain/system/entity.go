package system

import "time"

type ConfigItem struct {
	Key       string
	Value     string
	UpdatedAt time.Time
}
