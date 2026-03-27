package xpointer_test

import (
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpointer"
)

const fuzzXPointerDoc = `<?xml version="1.0"?>
<root xml:id="root-id" xmlns:ns="urn:test">
  <child xml:id="child-id">value</child>
  <ns:item xml:id="item-id"/>
</root>`

func FuzzParseFragmentID(f *testing.F) {
	f.Add("foo")
	f.Add("xpointer(/root/child)")
	f.Add("xpath1(//child)")
	f.Add("element(/1/1)")
	f.Add("xmlns(ns=urn:test) xpath1(/root/ns:item)")
	f.Add("")

	f.Fuzz(func(_ *testing.T, fragment string) {
		if len(fragment) > 4096 {
			return
		}
		_, _, _ = xpointer.ParseFragmentID(fragment)
	})
}

func FuzzEvaluate(f *testing.F) {
	f.Add("child-id")
	f.Add("element(/1/1)")
	f.Add("xpath1(/root/child)")
	f.Add("xmlns(ns=urn:test) xpointer(/root/ns:item)")
	f.Add("bogus(data)element(/1/1)")

	f.Fuzz(func(t *testing.T, expr string) {
		if len(expr) > 4096 {
			return
		}

		doc, err := helium.NewParser().Parse(t.Context(), []byte(fuzzXPointerDoc))
		if err != nil {
			t.Fatalf("parse seed doc: %v", err)
		}

		_, _ = xpointer.Evaluate(t.Context(), doc, expr)
	})
}
