//+build !debug

package debug

const Enabled = false

type guard struct{}
func (g guard) IRelease(f string, args ...interface{}) {}

// IPrintf is no op unless you comple with the `debug` tag
func IPrintf(f string, args ...interface{}) guard { return nil }

// Printf is no op unless you compile with the `debug` tag
func Printf(f string, args ...interface{}) {}

// Dump dumps the objects using go-spew
func Dump(v ...interface{}) {}
