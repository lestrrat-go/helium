package xmldsig1

import (
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

// base64TransformStub is a Transform whose URI is the base64 decode transform.
// There is no exported constructor, so sign-side tests supply the URI through
// the public Transform interface.
type base64TransformStub struct{}

func (base64TransformStub) URI() string { return TransformBase64 }

// objectFragment is the same-document Reference URI used by the base64 transform
// tests to point at the ds:Object element carrying Id="object".
const objectFragment = "#object"

// TestBase64PipelineTransitions covers Base64 on both value kinds and proves it
// no longer closes the transform list.
func TestBase64PipelineTransitions(t *testing.T) {
	runtime := transformRuntime{parser: helium.NewParser(), external: true}

	t.Run("base64 then c14n reparses decoded XML", func(t *testing.T) {
		out, err := externalReferenceDigestInput(t.Context(), []byte("PHgvPg=="), []transformStep{
			{algorithm: TransformBase64},
			{algorithm: C14N10},
		}, runtime)
		require.NoError(t, err)
		require.Equal(t, `<x></x>`, string(out))
	})

	t.Run("second base64 decodes the first result", func(t *testing.T) {
		out, err := externalReferenceDigestInput(t.Context(), []byte("YzI5dFpTQjBaWGgw"), []transformStep{
			{algorithm: TransformBase64},
			{algorithm: TransformBase64},
		}, runtime)
		require.NoError(t, err)
		require.Equal(t, "some text", string(out))
	})

	t.Run("c14n then base64 runs and rejects non-base64 canonical XML", func(t *testing.T) {
		_, err := externalReferenceDigestInput(t.Context(), []byte(`<x/>`), []transformStep{
			{algorithm: C14N10},
			{algorithm: TransformBase64},
		}, runtime)
		require.ErrorIs(t, err, ErrInvalidSignature)
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

// TestBase64ReferenceNodeSetTransforms proves Base64 consumes the text nodes
// remaining after earlier node-set transforms.
func TestBase64ReferenceNodeSetTransforms(t *testing.T) {
	doc, err := helium.NewParser().Parse(t.Context(), []byte(`<root><Object Id="object"><keep>c29tZSB0ZXh0</keep><drop>not-base64!</drop></Object></root>`))
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
		require.ErrorIs(t, err, ErrInvalidSignature)
		require.Nil(t, canon, "the unremoved non-base64 text remains part of the node-set")
	})

	t.Run("xpath filter before base64", func(t *testing.T) {
		ref := parsedReference{
			uri:             objectFragment,
			digestAlgorithm: DigestSHA1,
			transforms: []parsedTransform{
				{algorithm: TransformXPath, xpathExpr: "not(ancestor-or-self::drop)"},
				{algorithm: TransformBase64},
			},
		}
		_, canon, _, err := canonicalizeReference(t.Context(), &verifierConfig{}, doc, sig, ref)
		require.NoError(t, err)
		require.Equal(t, "some text", string(canon))
	})
}

// TestBase64SignPreflightAccepted proves signing and execution share Base64
// support for a caller-supplied Transform implementation.
func TestBase64SignPreflightAccepted(t *testing.T) {
	cfg := &signerConfig{
		references: []ReferenceConfig{{
			URI:             objectFragment,
			DigestAlgorithm: DigestSHA1,
			Transforms:      []Transform{base64TransformStub{}},
		}},
	}
	err := preflightSignerTransforms(cfg)
	require.NoError(t, err)
}

func TestBase64SignVerifyRoundTrip(t *testing.T) {
	key := []byte("base64-transform-test-key")
	doc, err := helium.NewParser().Parse(t.Context(), []byte(`<root><Object Id="object">c29tZSB0ZXh0</Object></root>`))
	require.NoError(t, err)

	sig, err := NewSigner().
		SignatureAlgorithm(AlgHMACSHA256).
		Reference(ReferenceConfig{
			URI:             objectFragment,
			DigestAlgorithm: DigestSHA256,
			Transforms:      []Transform{base64TransformStub{}},
		}).
		SignDetached(t.Context(), doc, key)
	require.NoError(t, err)
	require.NoError(t, doc.DocumentElement().AddChild(sig))

	result, err := NewVerifier(StaticKey(key)).Verify(t.Context(), doc)
	require.NoError(t, err)
	require.Len(t, result.References, 1)
}
