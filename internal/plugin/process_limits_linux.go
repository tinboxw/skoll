//go:build linux

package plugin

import (
	"fmt"
	"os"

	"github.com/tinboxw/skoll/internal/plugin/quota"
	"golang.org/x/sys/unix"
)

type managedProcessLimitHandle interface {
	Close() error
}

type linuxProcessLimit struct{}

func (*linuxProcessLimit) Close() error { return nil }

func applyManagedProcessLimits(process *os.Process, policy quota.Policy) (managedProcessLimitHandle, error) {
	if process == nil || process.Pid <= 0 {
		return nil, fmt.Errorf("plugin process is required")
	}
	limit := &unix.Rlimit{Cur: uint64(policy.ProcessMemoryBytes), Max: uint64(policy.ProcessMemoryBytes)}
	if err := unix.Prlimit(process.Pid, unix.RLIMIT_AS, limit, nil); err != nil {
		return nil, err
	}
	return &linuxProcessLimit{}, nil
}
