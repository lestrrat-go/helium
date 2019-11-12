package sax_test

import (
	"testing"

	"github.com/lestrrat-go/helium/sax"
)

func TestInterface(t *testing.T) {
	s := &sax.SAX2{}
	var sh sax.SAX2Handler = s
	_ = sh
}
