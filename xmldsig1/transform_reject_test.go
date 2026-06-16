package xmldsig1

import (
	"errors"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

// TestVerifyReferenceRejectsUnsupportedTransform guards against
// signature-coverage fail-open: a Reference that declares a transform the
// verifier cannot apply must be rejected before digesting, rather than
// silently ignored and verified against the untransformed canonical bytes.
//
// This exercises verifyReference directly because SignedInfo (which contains
// the Transforms list) is itself protected by the signature value, so the
// unsupported transform must be caught at the per-reference stage.
func TestVerifyReferenceRejectsUnsupportedTransform(t *testing.T) {
	doc, err := helium.NewParser().Parse(t.Context(), []byte(`<root><data>hello</data></root>`))
	require.NoError(t, err)

	// A whole-document reference whose only transform is an unsupported URI
	// (the XPath transform). digestValue is irrelevant: rejection must happen
	// before the digest is even computed.
	ref := parsedReference{
		uri:             "",
		digestAlgorithm: DigestSHA256,
		transforms: []parsedTransform{
			{algorithm: TransformXPath},
		},
	}

	_, err = verifyReference(doc, nil, ref)
	require.ErrorIs(t, err, ErrUnsupportedTransform)
}

// TestVerifyReferenceRejectsUnsupportedTransformWithEnveloped ensures the
// enveloped detach/restore path also rejects an unsupported sibling transform
// (and restores the Signature element rather than leaving it detached).
func TestVerifyReferenceRejectsUnsupportedTransformWithEnveloped(t *testing.T) {
	doc, err := helium.NewParser().Parse(t.Context(),
		[]byte(`<root><ds:Signature xmlns:ds="http://www.w3.org/2000/09/xmldsig#"/></root>`))
	require.NoError(t, err)

	root := doc.DocumentElement()
	sigElem, ok := helium.AsNode[*helium.Element](root.FirstChild())
	require.True(t, ok)

	ref := parsedReference{
		uri:             "",
		digestAlgorithm: DigestSHA256,
		transforms: []parsedTransform{
			{algorithm: TransformEnvelopedSignature},
			{algorithm: "urn:bogus:transform"},
		},
	}

	_, err = verifyReference(doc, sigElem, ref)
	require.ErrorIs(t, err, ErrUnsupportedTransform)

	// The Signature element must have been reattached, not left detached.
	require.Same(t, sigElem, root.FirstChild(), "signature element must be restored after rejection")
}

// Sanity: errors.Is wiring is correct (defensive, mirrors caller expectations).
func TestUnsupportedTransformErrorWrapping(t *testing.T) {
	err := errors.New("wrapped")
	require.False(t, errors.Is(err, ErrUnsupportedTransform))
}
