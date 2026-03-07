package xpath_test

import (
	"testing"

	"github.com/lestrrat-go/helium/xpath"
)

func FuzzCompile(f *testing.F) {
	f.Add(`/root/child`)
	f.Add(`//node[@attr='value']`)
	f.Add(`count(//item)`)
	f.Add(`/root/ns:child`)
	f.Add(`position() > 1 and position() < last()`)
	f.Add(`concat(substring-before(., ':'), substring-after(., ':'))`)
	f.Add(`ancestor-or-self::node()`)
	f.Add(``)
	f.Add(`][invalid`)

	f.Fuzz(func(_ *testing.T, expr string) {
		_, _ = xpath.Compile(expr)
	})
}
