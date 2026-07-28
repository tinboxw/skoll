//go:build windows

package plugin

import (
	"fmt"
	"os"
	"unsafe"

	"github.com/tinboxw/skoll/internal/plugin/quota"
	"golang.org/x/sys/windows"
)

type managedProcessLimitHandle interface {
	Close() error
}

type windowsJobLimit struct {
	handle windows.Handle
}

func (h *windowsJobLimit) Close() error {
	if h == nil || h.handle == 0 {
		return nil
	}
	err := windows.CloseHandle(h.handle)
	h.handle = 0
	return err
}

func applyManagedProcessLimits(process *os.Process, policy quota.Policy) (managedProcessLimitHandle, error) {
	if process == nil || process.Pid <= 0 {
		return nil, fmt.Errorf("plugin process is required")
	}
	job, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		return nil, err
	}
	handle := &windowsJobLimit{handle: job}
	info := windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{}
	info.BasicLimitInformation.LimitFlags =
		windows.JOB_OBJECT_LIMIT_ACTIVE_PROCESS |
			windows.JOB_OBJECT_LIMIT_PROCESS_MEMORY |
			windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE
	info.BasicLimitInformation.ActiveProcessLimit = 1
	info.ProcessMemoryLimit = uintptr(policy.ProcessMemoryBytes)
	if _, err = windows.SetInformationJobObject(
		job,
		windows.JobObjectExtendedLimitInformation,
		uintptr(unsafe.Pointer(&info)),
		uint32(unsafe.Sizeof(info)),
	); err != nil {
		_ = handle.Close()
		return nil, err
	}
	processHandle, err := windows.OpenProcess(
		windows.PROCESS_SET_QUOTA|windows.PROCESS_TERMINATE|windows.PROCESS_QUERY_LIMITED_INFORMATION,
		false,
		uint32(process.Pid),
	)
	if err != nil {
		_ = handle.Close()
		return nil, err
	}
	defer windows.CloseHandle(processHandle)
	if err = windows.AssignProcessToJobObject(job, processHandle); err != nil {
		_ = handle.Close()
		return nil, err
	}
	return handle, nil
}
