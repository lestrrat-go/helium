package xmldsig1

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestECDSARawToDERTypedError confirms a wrong-length raw ECDSA signature is
// classified with a package sentinel (ErrVerificationFailed) rather than an
// untyped error, so a caller's errors.Is check on the verify path succeeds.
func TestECDSARawToDERTypedError(t *testing.T) {
	_, err := ecdsaRawToDER([]byte{0x01, 0x02, 0x03}, 32)
	require.Error(t, err)
	require.True(t, errors.Is(err, ErrVerificationFailed))

	// A correctly sized raw signature round-trips to DER without error.
	raw := make([]byte, 64)
	raw[31] = 1
	raw[63] = 1
	der, err := ecdsaRawToDER(raw, 32)
	require.NoError(t, err)
	require.NotEmpty(t, der)
}

// TestExcC14NTransformCopiesPrefixes confirms ExcC14NTransform copies the
// prefix varargs at construction and Prefixes returns a copy, so a caller
// cannot mutate the transform's internal prefix list through either.
func TestExcC14NTransformCopiesPrefixes(t *testing.T) {
	prefixes := []string{"a", "b"}
	tr := ExcC14NTransform(prefixes...).(excC14NTransform)

	// Mutating the caller's slice must not change the transform.
	prefixes[0] = "z"
	require.Equal(t, []string{"a", "b"}, tr.Prefixes())

	// Mutating the returned slice must not change the transform either.
	got := tr.Prefixes()
	got[0] = "z"
	require.Equal(t, []string{"a", "b"}, tr.Prefixes())
}

// TestSignerCloneDeepCopiesTransforms confirms a Reference's Transforms slice is
// copied into the Signer, so a later mutation of the caller's slice (its
// elements, its backing prefix slices, or its length) cannot alter an
// already-configured Signer.
func TestSignerCloneDeepCopiesTransforms(t *testing.T) {
	transforms := []Transform{Enveloped(), ExcC14NTransform("p1")}
	ref := ReferenceConfig{URI: "#x", DigestAlgorithm: DigestSHA256, Transforms: transforms}
	signer := NewSigner().Reference(ref)

	// Mutate the caller's slice after configuring: overwrite an element and
	// append past the original length.
	transforms[0] = nil
	transforms = append(transforms, Enveloped())
	_ = transforms

	stored := signer.cfg.references[0].Transforms
	require.Len(t, stored, 2)
	require.NotNil(t, stored[0])
	require.Equal(t, TransformEnvelopedSignature, stored[0].URI())

	// The copied Exclusive C14N transform keeps its original prefix.
	exc, ok := stored[1].(excC14NTransform)
	require.True(t, ok)
	require.Equal(t, []string{"p1"}, exc.Prefixes())
}
