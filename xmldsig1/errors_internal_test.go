package xmldsig1

import (
	"errors"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

// TestVerificationErrorFormat covers both branches of VerificationError.Error
// (signature-value failure with Reference < 0, and a per-Reference failure).
func TestVerificationErrorFormat(t *testing.T) {
	cause := errors.New("boom")

	sigErr := &VerificationError{Reference: -1, Err: cause}
	require.Contains(t, sigErr.Error(), "signature value verification failed")
	require.ErrorIs(t, sigErr, cause)

	refErr := &VerificationError{Reference: 2, URI: "#x", Err: cause}
	msg := refErr.Error()
	require.Contains(t, msg, "reference 2")
	require.Contains(t, msg, "#x")
	require.ErrorIs(t, refErr, cause)
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
			{algorithm: "urn:bogus:transform"},
		},
	}

	_, refErr := verifyReference(t.Context(), doc, nil, ref, false)
	require.Error(t, refErr)

	// Wrap in the actual type callers receive and confirm errors.Is still
	// reaches ErrUnsupportedTransform through it.
	wrapped := &VerificationError{Reference: 0, URI: "", Err: refErr}
	require.ErrorIs(t, wrapped, ErrUnsupportedTransform)
	require.ErrorAs(t, error(wrapped), new(*VerificationError))

	// Negative control: an unrelated error must not match.
	require.False(t, errors.Is(errors.New("unrelated"), ErrUnsupportedTransform))
}
