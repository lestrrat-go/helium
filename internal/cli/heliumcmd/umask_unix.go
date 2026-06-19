//go:build !windows && !plan9

package heliumcmd

import (
	"sync"
	"syscall"
)

// umaskMu serializes the read-modify-write of the process umask. syscall.Umask
// has no read-only form: it sets a new value and returns the previous one, so
// we set it back immediately and hold the lock across the pair.
var umaskMu sync.Mutex

// currentUmask returns the current process umask without changing it.
func currentUmask() int {
	umaskMu.Lock()
	defer umaskMu.Unlock()
	old := syscall.Umask(0)
	syscall.Umask(old)
	return old
}
