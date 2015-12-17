//+build !debug

package debug

const Enabled = false

// Printf is no op unless you compile with the `debug` tag
func Printf(f string, args ...interface{}) {}

// Dump dumps the objects using go-spew
func Dump(v ...interface{}) {}
