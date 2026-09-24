//go:build !windows

package discovery

import (
	"errors"
	"syscall"
)

// ProcessAlive reports whether pid names a running process. Signal 0 probes without delivering
// anything; EPERM means the process exists but belongs to someone else.
func ProcessAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	err := syscall.Kill(pid, 0)
	return err == nil || errors.Is(err, syscall.EPERM)
}
