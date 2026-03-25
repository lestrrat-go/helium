package c14n_test

import (
	"bytes"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/c14n"
)

func FuzzCanonicalize(f *testing.F) {
	f.Add([]byte(`<?xml version="1.0"?><root/>`), uint8(0), false)
	f.Add([]byte(`<root xmlns="http://example.com" xmlns:ns="http://ns.example.com"><ns:child>text</ns:child></root>`), uint8(1), true)
	f.Add([]byte(`<doc><!-- comment --><a b="1" a="2"/></doc>`), uint8(2), false)
	f.Add([]byte(`<e xmlns:a="http://a" xmlns:b="http://b"><a:x/><b:y/></e>`), uint8(1), false)

	f.Fuzz(func(t *testing.T, data []byte, modeVal uint8, withComments bool) {
		if len(data) > 1<<20 {
			return
		}
		doc, err := helium.Parse(t.Context(), data)
		if err != nil {
			return
		}

		mode := c14n.C14N10
		switch modeVal % 3 {
		case 1:
			mode = c14n.ExclusiveC14N10
		case 2:
			mode = c14n.C14N11
		}

		can := c14n.NewCanonicalizer(mode)
		if withComments {
			can = can.Comments()
		}

		var buf bytes.Buffer
		_ = can.Canonicalize(doc, &buf)
	})
}
