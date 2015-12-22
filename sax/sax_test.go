package sax_test

import (
	"testing"

	"github.com/lestrrat/helium/sax"
)

func TestInterface(t *testing.T) {
	s := &sax.SAX2{}
	var ch sax.ContentHandler = s
	_ = ch

	var dh sax.DeclHandler = s
	_ = dh

	var er sax.EntityResolver = s
	_ = er

	var lh sax.LexicalHandler = s
	_ = lh

	var dtdh sax.DTDHandler = s
	_ = dtdh
}