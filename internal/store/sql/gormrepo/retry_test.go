package gormrepo

import (
	"errors"
	"testing"
)

func TestRetryableDBErrorIncludesConcurrencyFailures(t *testing.T) {
	for _, message := range []string{
		"database is locked",
		"database table is locked",
		"Deadlock found when trying to get lock",
		"Lock wait timeout exceeded",
		"could not serialize access due to concurrent update",
		"serialization failure (SQLSTATE 40001)",
	} {
		if !isRetryableDBError(errors.New(message)) {
			t.Fatalf("expected retryable database error: %s", message)
		}
	}
	if isRetryableDBError(errors.New("duplicate key value violates unique constraint")) {
		t.Fatal("non-transient constraint violations must not be retried")
	}
}
