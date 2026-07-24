package xmldsig1_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"fmt"
	"regexp"
	"sync/atomic"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xmldsig1"
	"github.com/stretchr/testify/require"
)

// digestValueRE extracts the base64 text of every ds:DigestValue in a signed
// document, in document order.
var digestValueRE = regexp.MustCompile(`<ds:DigestValue[^>]*>([^<]*)</ds:DigestValue>`)

type manifestCountingResolver struct {
	calls atomic.Int32
}

func (r *manifestCountingResolver) ResolveReference(_ context.Context, _ string) ([]byte, error) {
	r.calls.Add(1)
	return []byte("external payload"), nil
}

type manifestCountingTransformer struct {
	calls atomic.Int32
}

func (t *manifestCountingTransformer) TransformXSLT(_ context.Context, _, input []byte) ([]byte, error) {
	t.calls.Add(1)
	return input, nil
}

// innerDigests signs a data-only document with one ExcC14N/SHA-256 reference per
// target id and returns the resulting DigestValue strings, in order. Exclusive
// C14N of a leaf subtree is independent of its siblings and namespace context,
// so a digest computed here is byte-identical to the one recomputed when the same
// element appears inside the final Manifest document.
func innerDigests(t *testing.T, key *rsa.PrivateKey, ids ...string) []string {
	t.Helper()

	const dataSrc = `<root Id="r"><d1 Id="d1">one</d1><d2 Id="d2">two</d2></root>`
	doc, err := helium.NewParser().Parse(t.Context(), []byte(dataSrc))
	require.NoError(t, err)

	signer := xmldsig1.NewSigner().SignatureAlgorithm(xmldsig1.AlgRSASHA256)
	for _, id := range ids {
		signer = signer.Reference(xmldsig1.ReferenceConfig{
			URI:             "#" + id,
			DigestAlgorithm: xmldsig1.DigestSHA256,
			Transforms:      []xmldsig1.Transform{xmldsig1.ExcC14NTransform()},
		})
	}
	require.NoError(t, signer.SignEnveloped(t.Context(), doc, doc.DocumentElement(), key))

	out, err := helium.WriteString(doc)
	require.NoError(t, err)

	matches := digestValueRE.FindAllStringSubmatch(out, -1)
	require.Len(t, matches, len(ids), "expected one DigestValue per id")
	digests := make([]string, len(matches))
	for i, m := range matches {
		digests[i] = m[1]
	}
	return digests
}

// buildManifestSignedDoc constructs a document holding a ds:Manifest (inside a
// ds:Object) whose inner references are built from innerReferencesXML, then
// signs it with a single top-level Reference of Type=Manifest covering the
// Manifest subtree. The returned document verifies at the core level regardless
// of whether the inner references match: the signature commits only to the
// Manifest's own bytes.
func buildManifestSignedDoc(t *testing.T, key *rsa.PrivateKey, innerReferencesXML string) *helium.Document {
	t.Helper()

	src := fmt.Sprintf(`<root Id="r"><d1 Id="d1">one</d1><d2 Id="d2">two</d2>`+
		`<ds:Object xmlns:ds="http://www.w3.org/2000/09/xmldsig#">`+
		`<ds:Manifest Id="man1">%s</ds:Manifest></ds:Object></root>`, innerReferencesXML)

	doc, err := helium.NewParser().Parse(t.Context(), []byte(src))
	require.NoError(t, err)

	// The top-level reference covers the Manifest subtree (URI="#man1"); it does
	// not cover the Signature, so no enveloped-signature transform is needed.
	signer := xmldsig1.NewSigner().
		SignatureAlgorithm(xmldsig1.AlgRSASHA256).
		Reference(xmldsig1.ReferenceConfig{
			URI:             "#man1",
			Type:            xmldsig1.TypeManifest,
			DigestAlgorithm: xmldsig1.DigestSHA256,
			Transforms:      []xmldsig1.Transform{xmldsig1.ExcC14NTransform()},
		})
	require.NoError(t, signer.SignEnveloped(t.Context(), doc, doc.DocumentElement(), key))
	return doc
}

// innerRefXML renders one ds:Reference for a Manifest with an ExcC14N transform
// and a SHA-256 digest.
func innerRefXML(uri, digest string) string {
	return fmt.Sprintf(`<ds:Reference URI=%q>`+
		`<ds:Transforms><ds:Transform Algorithm="http://www.w3.org/2001/10/xml-exc-c14n#"/></ds:Transforms>`+
		`<ds:DigestMethod Algorithm="http://www.w3.org/2001/04/xmlenc#sha256"/>`+
		`<ds:DigestValue>%s</ds:DigestValue></ds:Reference>`, uri, digest)
}

func TestManifestValidation(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	t.Run("all inner references match", func(t *testing.T) {
		digests := innerDigests(t, key, "d1", "d2")
		inner := innerRefXML("#d1", digests[0]) + innerRefXML("#d2", digests[1])
		doc := buildManifestSignedDoc(t, key, inner)

		res, err := xmldsig1.NewVerifier(xmldsig1.StaticKey(&key.PublicKey)).
			ValidateManifests(true).
			Verify(t.Context(), doc)
		require.NoError(t, err, "core verification must succeed")

		require.Len(t, res.References, 1)
		require.Equal(t, xmldsig1.TypeManifest, res.References[0].Type, "top-level Type must be reported")

		require.Len(t, res.Manifests, 1)
		m := res.Manifests[0]
		require.Same(t, &res.References[0], m.Reference)
		require.NotNil(t, m.Element)
		require.Len(t, m.References, 2)
		for i, mr := range m.References {
			require.Truef(t, mr.Valid, "inner reference %d should be valid", i)
			require.NoError(t, mr.Err)
			require.NotNil(t, mr.Element)
		}
		require.Equal(t, "#d1", m.References[0].URI)
		require.Equal(t, "#d2", m.References[1].URI)
		require.Equal(t, xmldsig1.DigestSHA256, m.References[0].DigestAlgorithm)
	})

	t.Run("one inner reference mismatches", func(t *testing.T) {
		digests := innerDigests(t, key, "d1", "d2")
		// Corrupt the second inner digest so its recomputed digest will not match,
		// while the top-level Manifest digest (over the Manifest bytes as written)
		// still verifies.
		bad := "AAAA" + digests[1][4:]
		require.NotEqual(t, digests[1], bad)
		inner := innerRefXML("#d1", digests[0]) + innerRefXML("#d2", bad)
		doc := buildManifestSignedDoc(t, key, inner)

		res, err := xmldsig1.NewVerifier(xmldsig1.StaticKey(&key.PublicKey)).
			ValidateManifests(true).
			Verify(t.Context(), doc)
		require.NoError(t, err, "a failed inner reference must NOT fail core verification")

		require.Len(t, res.Manifests, 1)
		m := res.Manifests[0]
		require.Len(t, m.References, 2)
		require.True(t, m.References[0].Valid)
		require.False(t, m.References[1].Valid, "the corrupted inner reference must be invalid")
		require.ErrorIs(t, m.References[1].Err, xmldsig1.ErrDigestMismatch)
	})

	t.Run("inner reference with an unsupported transform", func(t *testing.T) {
		// An XSLT transform is an octet-consuming transform the package rejects
		// fail-closed. The DigestValue is irrelevant: the pipeline fails before
		// digesting.
		inner := fmt.Sprintf(`<ds:Reference URI="#d1">`+
			`<ds:Transforms><ds:Transform Algorithm="http://www.w3.org/TR/1999/REC-xslt-19991116"/></ds:Transforms>`+
			`<ds:DigestMethod Algorithm="http://www.w3.org/2001/04/xmlenc#sha256"/>`+
			`<ds:DigestValue>%s</ds:DigestValue></ds:Reference>`, "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=")
		doc := buildManifestSignedDoc(t, key, inner)

		res, err := xmldsig1.NewVerifier(xmldsig1.StaticKey(&key.PublicKey)).
			ValidateManifests(true).
			Verify(t.Context(), doc)
		require.NoError(t, err, "an unsupported inner transform must NOT fail core verification")

		require.Len(t, res.Manifests, 1)
		m := res.Manifests[0]
		require.Len(t, m.References, 1)
		require.False(t, m.References[0].Valid)
		require.ErrorIs(t, m.References[0].Err, xmldsig1.ErrUnsupportedTransform)
	})

	t.Run("toggle off leaves Manifests nil", func(t *testing.T) {
		digests := innerDigests(t, key, "d1", "d2")
		inner := innerRefXML("#d1", digests[0]) + innerRefXML("#d2", digests[1])
		doc := buildManifestSignedDoc(t, key, inner)

		res, err := xmldsig1.NewVerifier(xmldsig1.StaticKey(&key.PublicKey)).
			Verify(t.Context(), doc)
		require.NoError(t, err)
		require.Nil(t, res.Manifests, "default (toggle off) must not walk inner references")
		require.Equal(t, xmldsig1.TypeManifest, res.References[0].Type, "Type is reported even with the toggle off")
	})
}

func TestManifestValidationPreflightsAllReferencesBeforeResolver(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	inner := `<ds:Reference URI="external.xml"><ds:Transforms>` +
		`<ds:Transform Algorithm="` + xmldsig1.TransformXSLT + `">` +
		`<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="1.0"/>` +
		`</ds:Transform></ds:Transforms>` +
		`<ds:DigestMethod Algorithm="` + xmldsig1.DigestSHA256 + `"/>` +
		`<ds:DigestValue>AA==</ds:DigestValue></ds:Reference>` +
		`<ds:Reference URI="#d1"><ds:Transforms>` +
		`<ds:Transform Algorithm="` + xmldsig1.TransformXPath + `">` +
		`<ds:XPath>$missing</ds:XPath></ds:Transform></ds:Transforms>` +
		`<ds:DigestMethod Algorithm="` + xmldsig1.DigestSHA256 + `"/>` +
		`<ds:DigestValue>AA==</ds:DigestValue></ds:Reference>`
	doc := buildManifestSignedDoc(t, key, inner)
	resolver := &manifestCountingResolver{}
	transformer := &manifestCountingTransformer{}

	result, err := xmldsig1.NewVerifier(xmldsig1.StaticKey(&key.PublicKey)).
		ReferenceResolver(resolver).
		XSLTTransformer(transformer).
		ValidateManifests(true).
		Verify(t.Context(), doc)
	require.NoError(t, err, "Manifest reference failures remain advisory")
	require.Zero(t, resolver.calls.Load(), "resolver must not run before every Manifest reference passes static validation")
	require.Zero(t, transformer.calls.Load(), "transformer must not run before every Manifest reference passes static validation")
	require.Len(t, result.Manifests, 1)
	require.Len(t, result.Manifests[0].References, 2)
	require.False(t, result.Manifests[0].References[0].Valid)
	require.ErrorIs(t, result.Manifests[0].References[0].Err, xmldsig1.ErrUnsupportedTransform)
	require.Contains(t, result.Manifests[0].References[0].Err.Error(), "was not executed")
	require.ErrorIs(t, result.Manifests[0].References[1].Err, xmldsig1.ErrUnsupportedTransform)
}

// TestManifestValidationContextCancel confirms a cancelled context stops the
// inner-reference walk without panicking.
func TestManifestValidationContextCancel(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	digests := innerDigests(t, key, "d1", "d2")
	inner := innerRefXML("#d1", digests[0]) + innerRefXML("#d2", digests[1])
	doc := buildManifestSignedDoc(t, key, inner)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err = xmldsig1.NewVerifier(xmldsig1.StaticKey(&key.PublicKey)).
		ValidateManifests(true).
		Verify(ctx, doc)
	require.True(t, errors.Is(err, context.Canceled), "cancelled context must short-circuit")
}
