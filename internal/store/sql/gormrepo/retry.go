package gormrepo

import (
	"database/sql"
	"database/sql/driver"
	"errors"
	"strings"
	"time"
)

func withDBRetry(op func() error) error {
	const attempts = 5
	const retryDelay = 40 * time.Millisecond

	var lastErr error
	for i := 0; i < attempts; i++ {
		err := op()
		if err == nil {
			return nil
		}
		lastErr = err
		if !isRetryableDBError(err) || i == attempts-1 {
			break
		}
		time.Sleep(time.Duration(i+1) * retryDelay)
	}
	return lastErr
}

func isRetryableDBError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, sql.ErrConnDone) || errors.Is(err, driver.ErrBadConn) {
		return true
	}

	message := strings.ToLower(err.Error())
	return strings.Contains(message, "invalid connection") ||
		strings.Contains(message, "bad connection") ||
		strings.Contains(message, "connection reset") ||
		strings.Contains(message, "connection refused") ||
		strings.Contains(message, "wsarecv") ||
		strings.Contains(message, "broken pipe") ||
		strings.Contains(message, "database is locked") ||
		strings.Contains(message, "database table is locked") ||
		strings.Contains(message, "deadlock found") ||
		strings.Contains(message, "lock wait timeout") ||
		strings.Contains(message, "could not serialize access") ||
		strings.Contains(message, "serialization failure") ||
		strings.Contains(message, "sqlstate 40001")
}
