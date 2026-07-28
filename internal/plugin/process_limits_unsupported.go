//go:build !windows && !linux

package plugin

import (
	"errors"
	"os"

	"github.com/tinboxw/skoll/internal/plugin/quota"
)

type managedProcessLimitHandle interface {
	Close() error
}

func applyManagedProcessLimits(*os.Process, quota.Policy) (managedProcessLimitHandle, error) {
	return nil, errors.New("managed plugin process limits are unsupported on this platform")
}
