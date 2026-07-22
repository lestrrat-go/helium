package xmldsig1

import (
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

// base64TransformStub is a Transform whose URI is the base64 decode transform.
// There is no exported constructor for it (the transform is verify-only), so a
// test that drives the sign-side preflight builds one directly to prove the
// preflight rejects it fail-closed.
type base64TransformStub struct{}

func (base64TransformStub) URI() string { return TransformBase64 }

// objectFragment is the same-document Reference URI used by the base64 transform
// tests to point at the ds:Object element carrying Id="object".
const objectFragment = "#object"

// TestBase64PipelineResolution locks how resolveTransformPipeline treats the
// base64 decode transform: alone it ends the pipeline as an octet-producing step
// (base64 set, no default c14n), and it participates in the same fail-closed
// ordering guard as canonicalization — a base64 after a c14n, a transform after
// a base64, and a second base64 are all rejected.
func TestBase64PipelineResolution(t *testing.T) {
	t.Run("base64 alone ends the pipeline without a default c14n", func(t *testing.T) {
		pipe, err := resolveTransformPipeline([]transformStep{{algorithm: TransformBase64}})
		require.NoError(t, err)
		require.True(t, pipe.base64, "base64 must be marked as the octet-producing step")
		require.Empty(t, pipe.c14nMethod, "no canonicalization runs after base64")
	})

	t.Run("enveloped then base64 is allowed at resolution", func(t *testing.T) {
		// A node-set transform may precede the octet-producing base64 step at the
		// pipeline level; canonicalizeReference is where the unsupported
		// combination is fail-closed (see TestBase64ReferenceFailClosed).
		pipe, err := resolveTransformPipeline([]transformStep{
			{algorithm: TransformEnvelopedSignature},
			{algorithm: TransformBase64},
		})
		require.NoError(t, err)
		require.True(t, pipe.base64)
		require.True(t, pipe.hasEnveloped)
	})

	t.Run("base64 after c14n is rejected", func(t *testing.T) {
		_, err := resolveTransformPipeline([]transformStep{
			{algorithm: C14N10},
			{algorithm: TransformBase64},
		})
		require.ErrorIs(t, err, ErrUnsupportedTransform)
	})

	t.Run("transform after base64 is rejected", func(t *testing.T) {
		_, err := resolveTransformPipeline([]transformStep{
			{algorithm: TransformBase64},
			{algorithm: C14N10},
		})
		require.ErrorIs(t, err, ErrUnsupportedTransform)
	})

	t.Run("second base64 is rejected", func(t *testing.T) {
		_, err := resolveTransformPipeline([]transformStep{
			{algorithm: TransformBase64},
			{algorithm: TransformBase64},
		})
		require.ErrorIs(t, err, ErrUnsupportedTransform)
	})
}

// TestBase64ReferenceDecode drives a same-document Reference carrying the base64
// transform through canonicalizeReference: the resolved element's concatenated
// text is base64-decoded to the digested octets, and whitespace within the
// base64 text is ignored by the decoder.
func TestBase64ReferenceDecode(t *testing.T) {
	// "c29tZSB0ZXh0" decodes to "some text"; the second document splits the
	// base64 across a newline and indentation to exercise whitespace tolerance.
	cases := map[string]string{
		"contiguous":         `<root><Object Id="object">c29tZSB0ZXh0</Object></root>`,
		"whitespace wrapped": "<root><Object Id=\"object\">\n      c29tZSB0\n      ZXh0\n    </Object></root>",
	}
	for name, xml := range cases {
		t.Run(name, func(t *testing.T) {
			doc, err := helium.NewParser().Parse(t.Context(), []byte(xml))
			require.NoError(t, err)
			ref := parsedReference{
				uri:             objectFragment,
				digestAlgorithm: DigestSHA1,
				transforms:      []parsedTransform{{algorithm: TransformBase64}},
			}
			target, octets, _, err := canonicalizeReference(t.Context(), &verifierConfig{}, doc, nil, ref)
			require.NoError(t, err)
			require.Equal(t, "Object", target.LocalName())
			require.Equal(t, "some text", string(octets),
				"base64 transform output is the decoded octets, digested directly")
		})
	}
}

// TestBase64ReferenceInvalidInput confirms that base64 content the decoder
// rejects fails closed as ErrInvalidSignature rather than digesting partial or
// unintended bytes.
func TestBase64ReferenceInvalidInput(t *testing.T) {
	doc, err := helium.NewParser().Parse(t.Context(), []byte(`<root><Object Id="object">not!!!valid!!!</Object></root>`))
	require.NoError(t, err)
	ref := parsedReference{
		uri:             objectFragment,
		digestAlgorithm: DigestSHA1,
		transforms:      []parsedTransform{{algorithm: TransformBase64}},
	}
	_, canon, _, err := canonicalizeReference(t.Context(), &verifierConfig{}, doc, nil, ref)
	require.ErrorIs(t, err, ErrInvalidSignature)
	require.Nil(t, canon)
}

// TestBase64ReferenceFailClosed locks the unsupported combinations of the base64
// transform with a preceding node-set transform. The base64 transform digests a
// node-set's string-value directly, so pairing it with an enveloped-signature or
// XPath filter transform is rejected fail-closed rather than digesting an
// unintended value.
func TestBase64ReferenceFailClosed(t *testing.T) {
	doc, err := helium.NewParser().Parse(t.Context(), []byte(`<root><Object Id="object">c29tZSB0ZXh0</Object></root>`))
	require.NoError(t, err)
	sig := findSig(doc.DocumentElement()) // nil: the doc carries no Signature

	t.Run("enveloped before base64", func(t *testing.T) {
		ref := parsedReference{
			uri:             objectFragment,
			digestAlgorithm: DigestSHA1,
			transforms: []parsedTransform{
				{algorithm: TransformEnvelopedSignature},
				{algorithm: TransformBase64},
			},
		}
		_, canon, _, err := canonicalizeReference(t.Context(), &verifierConfig{}, doc, sig, ref)
		require.ErrorIs(t, err, ErrUnsupportedTransform)
		require.Nil(t, canon)
	})

	t.Run("xpath filter before base64", func(t *testing.T) {
		ref := parsedReference{
			uri:             objectFragment,
			digestAlgorithm: DigestSHA1,
			transforms: []parsedTransform{
				{algorithm: TransformXPath, xpathExpr: "true()"},
				{algorithm: TransformBase64},
			},
		}
		_, canon, _, err := canonicalizeReference(t.Context(), &verifierConfig{}, doc, sig, ref)
		require.ErrorIs(t, err, ErrUnsupportedTransform)
		require.Nil(t, canon)
	})
}

// TestBase64SignPreflightRejected proves the sign-side preflight rejects the
// base64 transform. The signing digest path canonicalizes its reference node-set
// and has no base64 branch, so accepting a base64 transform would silently digest
// the canonicalized subtree instead of the decoded octets — a fail-open
// signature. The preflight fails closed before any DOM mutation.
func TestBase64SignPreflightRejected(t *testing.T) {
	cfg := &signerConfig{
		references: []ReferenceConfig{{
			URI:             objectFragment,
			DigestAlgorithm: DigestSHA1,
			Transforms:      []Transform{base64TransformStub{}},
		}},
	}
	err := preflightSignerTransforms(cfg)
	require.ErrorIs(t, err, ErrUnsupportedTransform)
	var refErr *ReferenceError
	require.ErrorAs(t, err, &refErr, "the failure must name the offending Reference")
	require.Equal(t, 0, refErr.Reference)
}
