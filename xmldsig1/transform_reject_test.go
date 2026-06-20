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

	_, err = verifyReference(doc, nil, ref, false)
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

	_, err = verifyReference(doc, sigElem, ref, false)
	require.ErrorIs(t, err, ErrUnsupportedTransform)

	// The Signature element must have been reattached, not left detached.
	require.Same(t, sigElem, root.FirstChild(), "signature element must be restored after rejection")
}

// TestUnsupportedTransformErrorWrapping asserts that an unsupported-transform
// error stays matchable via errors.Is(ErrUnsupportedTransform) even after it is
// wrapped in the caller-facing VerificationError type, so callers can reliably
// distinguish this failure mode.
func TestUnsupportedTransformErrorWrapping(t *testing.T) {
	doc, err := helium.NewParser().Parse(t.Context(), []byte(`<root><data>hi</data></root>`))
	require.NoError(t, err)

	ref := parsedReference{
		uri:             "",
		digestAlgorithm: DigestSHA256,
		transforms: []parsedTransform{
			{algorithm: TransformXPath},
		},
	}

	_, refErr := verifyReference(doc, nil, ref, false)
	require.Error(t, refErr)

	// Wrap in the actual type callers receive and confirm errors.Is still
	// reaches ErrUnsupportedTransform through it.
	wrapped := &VerificationError{Reference: 0, URI: "", Err: refErr}
	require.ErrorIs(t, wrapped, ErrUnsupportedTransform)
	require.ErrorAs(t, error(wrapped), new(*VerificationError))

	// Negative control: an unrelated error must not match.
	require.False(t, errors.Is(errors.New("unrelated"), ErrUnsupportedTransform))
}
