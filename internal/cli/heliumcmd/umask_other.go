//go:build windows || plan9

package heliumcmd

// currentUmask returns 0 on platforms without a POSIX umask. File permission
// bits are largely advisory there, so applying no mask matches os.Create's
// effective behavior.
func currentUmask() int { return 0 }
