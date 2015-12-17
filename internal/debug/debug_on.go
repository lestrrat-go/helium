//+build debug

package debug

import (
	"log"
	"os"

	"github.com/davecgh/go-spew/spew"
)

const Enabled = true

var logger = log.New(os.Stdout, "|DEBUG| ", 0)

// Printf prints debug messages. Only available if compiled with "debug" tag
func Printf(f string, args ...interface{}) {
	logger.Printf(f, args...)
}

func Dump(v ...interface{}) {
	spew.Dump(v...)
}
