package utils

import "time"

// NowUTC returns current timestamp in UTC.
func NowUTC() time.Time {
	return time.Now().UTC()
}

// FormatRFC3339 formats timestamp using RFC3339 in UTC.
func FormatRFC3339(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}

// ParseRFC3339 parses RFC3339 and normalizes to UTC.
func ParseRFC3339(v string) (time.Time, error) {
	t, err := time.Parse(time.RFC3339, v)
	if err != nil {
		return time.Time{}, err
	}
	return t.UTC(), nil
}

// UnixMilli returns Unix milliseconds.
func UnixMilli(t time.Time) int64 {
	return t.UTC().UnixMilli()
}
