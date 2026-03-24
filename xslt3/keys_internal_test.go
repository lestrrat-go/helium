package xslt3

import (
	"testing"

	"github.com/lestrrat-go/helium/xpath3"
	"github.com/stretchr/testify/require"
)

func TestCanonicalKeyUsesQNameValueSpace(t *testing.T) {
	a := xpath3.AtomicValue{
		TypeName: xpath3.TypeQName,
		Value: xpath3.QNameValue{
			Prefix: "one",
			URI:    "urn:test",
			Local:  "mp3",
		},
	}
	b := xpath3.AtomicValue{
		TypeName: xpath3.TypeQName,
		Value: xpath3.QNameValue{
			Prefix: "two",
			URI:    "urn:test",
			Local:  "mp3",
		},
	}
	c := xpath3.AtomicValue{
		TypeName: "ex:nota",
		Value: xpath3.QNameValue{
			Prefix: "three",
			URI:    "urn:test",
			Local:  "mp3",
		},
	}

	require.Equal(t, canonicalKey(a), canonicalKey(b))
	require.Equal(t, canonicalKey(a), canonicalKey(c))
}
