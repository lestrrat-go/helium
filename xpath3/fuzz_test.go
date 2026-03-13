package xpath3_test

import (
	"testing"

	"github.com/lestrrat-go/helium/xpath3"
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
	// XPath 3.1 additions
	f.Add(`for $x in (1, 2, 3) return $x * 2`)
	f.Add(`let $x := 42 return $x`)
	f.Add(`some $x in (1, 2, 3) satisfies $x > 2`)
	f.Add(`every $x in (1, 2, 3) satisfies $x > 0`)
	f.Add(`if (true()) then "yes" else "no"`)
	f.Add(`1 instance of xs:integer`)
	f.Add(`"42" cast as xs:integer`)
	f.Add(`"42" castable as xs:integer`)
	f.Add(`map { "a": 1, "b": 2 }`)
	f.Add(`[1, 2, 3]`)
	f.Add(`function($x) { $x + 1 }`)
	f.Add(`try { error() } catch * { "caught" }`)
	f.Add(`(1, 2, 3) ! (. * 2)`)
	f.Add(`upper-case("hello") => contains("HELL")`)
	f.Add(`string-join(("a", "b"), "-")`)
	f.Add(`1 to 10`)

	f.Fuzz(func(_ *testing.T, expr string) {
		if len(expr) > 4096 {
			return
		}
		_, _ = xpath3.Compile(expr)
	})
}
