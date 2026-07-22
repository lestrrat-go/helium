package xmldsig1

import (
	"context"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

// staticResolver returns fixed octets for any URI.
type staticResolver []byte

func (s staticResolver) ResolveReference(_ context.Context, _ string) ([]byte, error) {
	return []byte(s), nil
}

// TestExternalOctetSymmetry proves the binding invariant that the sign side
// digests exactly what the verify side digests for the same external input.
// Both the sign path (signReferenceOctets) and the verify path
// (resolveExternalReference) funnel through externalReferenceDigestInput, so for
// identical octets and transform lists they must yield byte-identical digest
// input.
func TestExternalOctetSymmetry(t *testing.T) {
	doc, err := helium.NewParser().Parse(t.Context(), []byte(`<root/>`))
	require.NoError(t, err)

	octets := []byte(`<ext xmlns:a="urn:a"><a:child>text</a:child></ext>`)
	res := staticResolver(octets)

	cases := []struct {
		name       string
		signTrans  []Transform
		parsedTran []parsedTransform
	}{
		{
			name:       "empty chain digests octets directly",
			signTrans:  nil,
			parsedTran: nil,
		},
		{
			name:       "c14n chain parses and canonicalizes",
			signTrans:  []Transform{C14NTransform(C14N10)},
			parsedTran: []parsedTransform{{algorithm: C14N10}},
		},
		{
			name:       "c14n11 chain",
			signTrans:  []Transform{C14NTransform(C14N11URI)},
			parsedTran: []parsedTransform{{algorithm: C14N11URI}},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			signCfg := &signerConfig{referenceResolver: res}
			signOut, err := signReferenceOctets(t.Context(), signCfg, doc, nil,
				ReferenceConfig{URI: "http://example.com/x", Transforms: tc.signTrans}, nil)
			require.NoError(t, err)

			verCfg := &verifierConfig{referenceResolver: res}
			verOut, err := resolveExternalReference(t.Context(), verCfg, doc,
				parsedReference{uri: "http://example.com/x", transforms: tc.parsedTran})
			require.NoError(t, err)

			require.NotEmpty(t, signOut)
			require.Equal(t, signOut, verOut, "sign and verify must digest identical bytes")
		})
	}
}

// TestExternalReferenceDigestInputEnveloped locks the fail-closed rejection of an
// enveloped-signature transform on an external reference.
func TestExternalReferenceDigestInputEnveloped(t *testing.T) {
	pipe := transformPipeline{hasEnveloped: true, c14nMethod: C14N10}
	_, err := externalReferenceDigestInput(t.Context(), []byte(`<x/>`), pipe, false, helium.NewParser())
	require.ErrorIs(t, err, ErrUnsupportedTransform)
}

// TestExternalReferenceDigestInputEmptyIsRaw confirms an empty chain digests the
// resolved octets verbatim (no canonicalization of an external octet stream).
func TestExternalReferenceDigestInputEmptyIsRaw(t *testing.T) {
	octets := []byte("not even xml \x00 bytes")
	out, err := externalReferenceDigestInput(t.Context(), octets, transformPipeline{}, false, helium.NewParser())
	require.NoError(t, err)
	require.Equal(t, octets, out)
}

func TestURIHasScheme(t *testing.T) {
	cases := map[string]bool{
		"http://host/p":             true,
		"https://host/p":            true,
		"file:///etc/passwd":        true,
		"urn:isbn:0":                true,
		`C:\windows`:                true,
		"x:opaque":                  true,
		"c14n11/xml-base-input.xml": false,
		"data.xml":                  false,
		"sub/data.xml":              false,
		"../escape":                 false,
		"a?q:notscheme":             false,
		"a#f:notscheme":             false,
	}
	for uri, want := range cases {
		t.Run(uri, func(t *testing.T) {
			require.Equal(t, want, uriHasScheme(uri))
		})
	}
}

func TestStepsHaveC14N(t *testing.T) {
	require.False(t, stepsHaveC14N(nil))
	require.False(t, stepsHaveC14N([]transformStep{{algorithm: TransformEnvelopedSignature}}))
	require.False(t, stepsHaveC14N([]transformStep{{algorithm: TransformBase64}}))
	require.True(t, stepsHaveC14N([]transformStep{{algorithm: C14N10}}))
	require.True(t, stepsHaveC14N([]transformStep{{algorithm: TransformXPath}, {algorithm: C14N11URI}}))
}
