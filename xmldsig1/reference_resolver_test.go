package xmldsig1_test

import (
	"context"
	"crypto/dsa"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xmldsig1"
	"github.com/stretchr/testify/require"
)

// mapReferenceResolver resolves external Reference URIs from an in-memory map. It
// doubles as a demonstration of the ReferenceResolver interface for a caller that
// dereferences absolute URLs out of band (e.g. from a trusted mirror).
type mapReferenceResolver map[string][]byte

func (m mapReferenceResolver) ResolveReference(_ context.Context, uri string) ([]byte, error) {
	data, ok := m[uri]
	if !ok {
		return nil, errors.New("no such reference: " + uri)
	}
	return data, nil
}

// dsaKeySource builds a *dsa.PublicKey from an inline ds:DSAKeyValue, so the
// merlin external-dsa vector can be verified from its own KeyInfo.
func dsaKeySource() xmldsig1.KeySource {
	return xmldsig1.KeySourceFunc(func(_ context.Context, ki *xmldsig1.KeyInfoData, _ string) (any, error) {
		if ki == nil || ki.DSAKeyValue == nil {
			return nil, errors.New("no DSAKeyValue in KeyInfo")
		}
		v := ki.DSAKeyValue
		return &dsa.PublicKey{
			Parameters: dsa.Parameters{P: v.P, Q: v.Q, G: v.G},
			Y:          v.Y,
		}, nil
	})
}

func readInterop(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "interop", name))
	require.NoError(t, err)
	return data
}

// TestVerifyExternalDSA verifies the W3C merlin signature-external-dsa vector
// end-to-end: its single Reference points at the ABSOLUTE URL
// http://www.w3.org/TR/xml-stylesheet with no transforms, so the resolved octets
// are digested directly (SHA-1). A map resolver supplies the target document, the
// DSA key comes from the inline DSAKeyValue, and AllowSHA1 opts into the legacy
// SHA-1 signature and digest.
func TestVerifyExternalDSA(t *testing.T) {
	sig := readInterop(t, "signature-external-dsa.xml")
	target := readInterop(t, "xml-stylesheet")

	doc, err := helium.NewParser().Parse(t.Context(), sig)
	require.NoError(t, err)

	resolver := mapReferenceResolver{
		"http://www.w3.org/TR/xml-stylesheet": target,
	}

	result, err := xmldsig1.NewVerifier(dsaKeySource()).
		AllowSHA1(true).
		ReferenceResolver(resolver).
		Verify(t.Context(), doc)
	require.NoError(t, err, "external DSA signature must verify")

	require.Len(t, result.References, 1)
	ref := result.References[0]
	require.Equal(t, "http://www.w3.org/TR/xml-stylesheet", ref.URI)
	require.True(t, ref.External, "reference must be marked External")
	require.Nil(t, ref.Element, "external reference resolves to bytes, not an element")

	// An external reference must never be reported as covering document content.
	require.False(t, result.Covers(doc.DocumentElement()))
	require.Nil(t, result.SignedElement("http://www.w3.org/TR/xml-stylesheet"))
}

// TestVerifyExternalDSANilResolver locks the fail-closed default: without a
// resolver the same external vector stays ErrReferenceNotFound, byte-identical to
// the pre-resolver behavior.
func TestVerifyExternalDSANilResolver(t *testing.T) {
	doc, err := helium.NewParser().Parse(t.Context(), readInterop(t, "signature-external-dsa.xml"))
	require.NoError(t, err)

	_, err = xmldsig1.NewVerifier(dsaKeySource()).
		AllowSHA1(true).
		Verify(t.Context(), doc)
	require.ErrorIs(t, err, xmldsig1.ErrReferenceNotFound)
}

// TestVerifyDefCanExternalFS verifies the W3C xmldsig2ed defCan-1 vector
// end-to-end. Its Reference URI is the RELATIVE path c14n11/xml-base-input.xml
// with an XPath filter transform followed by Canonical XML 1.1, so the resolved
// octets must be parsed into XML (via the default locked-down ReferenceParser),
// the XPath filter applied, and the node-set canonicalized. The HMAC key is the
// ASCII "secret" and the signature/digest are SHA-1 (AllowSHA1).
func TestVerifyDefCanExternalFS(t *testing.T) {
	doc, err := helium.NewParser().Parse(t.Context(), readInterop(t, "defCan-1-signature.xml"))
	require.NoError(t, err)

	// The joined URI is the relative path itself (no document base), served from
	// the interop testdata directory where c14n11/xml-base-input.xml lives.
	resolver := xmldsig1.FSReferenceResolver(os.DirFS(filepath.Join("testdata", "interop")))

	result, err := xmldsig1.NewVerifier(xmldsig1.StaticKey([]byte("secret"))).
		AllowSHA1(true).
		ReferenceResolver(resolver).
		Verify(t.Context(), doc)
	require.NoError(t, err, "defCan-1 external reference must verify")

	require.Len(t, result.References, 1)
	require.True(t, result.References[0].External)
	require.Equal(t, "c14n11/xml-base-input.xml", result.References[0].URI)
	require.False(t, result.Covers(doc.DocumentElement()))
}

// TestVerifyDefCanExternalBaseJoin exercises base-URI joining: the signed
// document is parsed with a BaseURI carrying a directory, so the relative
// Reference URI c14n11/xml-base-input.xml joins to d/c14n11/xml-base-input.xml,
// which the resolver serves from that joined path.
func TestVerifyDefCanExternalBaseJoin(t *testing.T) {
	doc, err := helium.NewParser().
		BaseURI("d/defCan-1-signature.xml").
		Parse(t.Context(), readInterop(t, "defCan-1-signature.xml"))
	require.NoError(t, err)

	resolver := xmldsig1.FSReferenceResolver(fstest.MapFS{
		"d/c14n11/xml-base-input.xml": &fstest.MapFile{Data: readInterop(t, "xml-base-input.xml")},
	})

	result, err := xmldsig1.NewVerifier(xmldsig1.StaticKey([]byte("secret"))).
		AllowSHA1(true).
		ReferenceResolver(resolver).
		Verify(t.Context(), doc)
	require.NoError(t, err, "defCan-1 must verify after base-URI join")
	require.True(t, result.References[0].External)
}

// oversizeFS serves one file whose Read never ends, so the resolver's size cap is
// the only thing that stops it.
type oversizeFS struct{}

func (oversizeFS) Open(string) (fs.File, error) { return &oversizeFile{}, nil }

type oversizeFile struct{}

func (*oversizeFile) Stat() (fs.FileInfo, error) { return nil, fs.ErrInvalid }
func (*oversizeFile) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = 'a'
	}
	return len(p), nil
}
func (*oversizeFile) Close() error { return nil }

// TestFSReferenceResolverRejections locks the FSReferenceResolver security
// posture: scheme URIs, path escapes, leftover fragments, and oversized files
// are all refused fail-closed.
func TestFSReferenceResolverRejections(t *testing.T) {
	base := fstest.MapFS{
		"data.xml":     &fstest.MapFile{Data: []byte("<root/>")},
		"sub/data.xml": &fstest.MapFile{Data: []byte("<root/>")},
	}
	r := xmldsig1.FSReferenceResolver(base)

	t.Run("plain path resolves", func(t *testing.T) {
		got, err := r.ResolveReference(t.Context(), "sub/data.xml")
		require.NoError(t, err)
		require.Equal(t, "<root/>", string(got))
	})

	schemeURIs := []string{
		"http://evil.example/x",
		"https://evil.example/x",
		"file:///etc/passwd",
		"urn:isbn:0",
		`C:\windows\system32`,
	}
	for _, uri := range schemeURIs {
		t.Run("scheme "+uri, func(t *testing.T) {
			_, err := r.ResolveReference(t.Context(), uri)
			require.ErrorIs(t, err, xmldsig1.ErrReferenceNotFound)
		})
	}

	escapes := []string{
		"../data.xml",
		"sub/../../data.xml",
		"/etc/passwd",
		"/data.xml",
	}
	for _, uri := range escapes {
		t.Run("escape "+uri, func(t *testing.T) {
			_, err := r.ResolveReference(t.Context(), uri)
			require.ErrorIs(t, err, xmldsig1.ErrReferenceNotFound)
		})
	}

	t.Run("leftover fragment", func(t *testing.T) {
		_, err := r.ResolveReference(t.Context(), "data.xml#frag")
		require.ErrorIs(t, err, xmldsig1.ErrReferenceNotFound)
	})

	t.Run("missing file", func(t *testing.T) {
		_, err := r.ResolveReference(t.Context(), "nope.xml")
		require.ErrorIs(t, err, xmldsig1.ErrReferenceNotFound)
	})

	t.Run("oversized file", func(t *testing.T) {
		_, err := xmldsig1.FSReferenceResolver(oversizeFS{}).ResolveReference(t.Context(), "big.bin")
		require.ErrorIs(t, err, xmldsig1.ErrReferenceTooLarge)
	})
}

// TestVerifyExternalEnvelopedRejected confirms the enveloped-signature transform
// is rejected fail-closed on an external reference: removing the Signature's own
// subtree is meaningless on a resource that does not contain the Signature.
// Signing surfaces the same rejection through the resolver-backed sign path, so
// the check is exercised without a hand-built valid SignatureValue.
func TestVerifyExternalEnvelopedRejected(t *testing.T) {
	doc, err := helium.NewParser().Parse(t.Context(), []byte(`<root Id="r1"><data>x</data></root>`))
	require.NoError(t, err)

	resolver := mapReferenceResolver{
		"http://example.com/data.xml": []byte(`<ext/>`),
	}

	_, err = xmldsig1.NewSigner().
		SignatureAlgorithm(xmldsig1.AlgHMACSHA256).
		ReferenceResolver(resolver).
		Reference(xmldsig1.ReferenceConfig{
			URI:             "http://example.com/data.xml",
			DigestAlgorithm: xmldsig1.DigestSHA256,
			Transforms:      []xmldsig1.Transform{xmldsig1.Enveloped(), xmldsig1.ExcC14NTransform()},
		}).
		SignDetached(t.Context(), doc, []byte("secret"))
	require.ErrorIs(t, err, xmldsig1.ErrUnsupportedTransform)
}
