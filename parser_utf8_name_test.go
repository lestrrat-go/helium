package helium_test

import (
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

func TestParseNameRejectsInvalidUTF8InContinuation(t *testing.T) {
	// Element name "ro\xffoot" contains invalid UTF-8 byte 0xff mid-name.
	xml := []byte("<ro\xffoot/>")
	p := helium.NewParser()
	_, err := p.Parse(t.Context(), xml)
	require.Error(t, err)
}

func TestParseNCNameRejectsInvalidUTF8InContinuation(t *testing.T) {
	// Attribute name "at\xffr" contains invalid UTF-8 byte 0xff mid-name.
	xml := []byte("<root at\xffr=\"v\"/>")
	p := helium.NewParser()
	_, err := p.Parse(t.Context(), xml)
	require.Error(t, err)
}

func TestParseNCNameReportsInvalidStartRune(t *testing.T) {
	// Attribute name starting with '1' is not a valid NCName start character.
	xml := []byte("<root 1a=\"v\"/>")
	p := helium.NewParser()
	_, err := p.Parse(t.Context(), xml)
	require.Error(t, err)
}
