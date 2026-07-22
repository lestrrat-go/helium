package xmldsig1

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"math/big"
	"testing"
	"testing/fstest"
	"time"

	"github.com/stretchr/testify/require"
)

// selfSignedCert returns a throwaway self-signed certificate and its DER bytes.
func selfSignedCert(t *testing.T) (*x509.Certificate, []byte) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(12345),
		Subject:      pkix.Name{CommonName: "retrieval-test"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	require.NoError(t, err)
	cert, err := x509.ParseCertificate(der)
	require.NoError(t, err)
	return cert, der
}

func TestParseKeyName(t *testing.T) {
	t.Run("value trimmed", func(t *testing.T) {
		doc := mustParse(t, `<ds:KeyInfo xmlns:ds="`+NamespaceDSig+`"><ds:KeyName>  alpha-key
	</ds:KeyName><ds:KeyName>beta</ds:KeyName></ds:KeyInfo>`)
		data, err := parseKeyInfo(doc.DocumentElement())
		require.NoError(t, err)
		require.Equal(t, []string{"alpha-key", "beta"}, data.KeyNames)
	})

	t.Run("foreign namespace ignored", func(t *testing.T) {
		doc := mustParse(t, `<ds:KeyInfo xmlns:ds="`+NamespaceDSig+`" xmlns:evil="urn:evil"><evil:KeyName>attacker</evil:KeyName></ds:KeyInfo>`)
		data, err := parseKeyInfo(doc.DocumentElement())
		require.NoError(t, err)
		require.Empty(t, data.KeyNames)
	})
}

func TestParseX509SKI(t *testing.T) {
	t.Run("decodes raw bytes", func(t *testing.T) {
		raw := []byte{0xDE, 0xAD, 0xBE, 0xEF}
		enc := base64.StdEncoding.EncodeToString(raw)
		doc := mustParse(t, `<ds:KeyInfo xmlns:ds="`+NamespaceDSig+`"><ds:X509Data><ds:X509SKI>`+enc+`</ds:X509SKI></ds:X509Data></ds:KeyInfo>`)
		data, err := parseKeyInfo(doc.DocumentElement())
		require.NoError(t, err)
		require.Len(t, data.X509SKIs, 1)
		require.Equal(t, raw, data.X509SKIs[0])
	})

	t.Run("bad base64 fails closed", func(t *testing.T) {
		doc := mustParse(t, `<ds:KeyInfo xmlns:ds="`+NamespaceDSig+`"><ds:X509Data><ds:X509SKI>not!base64!</ds:X509SKI></ds:X509Data></ds:KeyInfo>`)
		_, err := parseKeyInfo(doc.DocumentElement())
		require.ErrorIs(t, err, ErrInvalidKeyInfo)
	})

	t.Run("foreign namespace ignored", func(t *testing.T) {
		doc := mustParse(t, `<ds:KeyInfo xmlns:ds="`+NamespaceDSig+`" xmlns:evil="urn:evil"><ds:X509Data><evil:X509SKI>3q2+7w==</evil:X509SKI></ds:X509Data></ds:KeyInfo>`)
		data, err := parseKeyInfo(doc.DocumentElement())
		require.NoError(t, err)
		require.Empty(t, data.X509SKIs)
	})
}

func TestResolveRetrievalMethodExternalRawX509(t *testing.T) {
	cert, der := selfSignedCert(t)
	fsys := fstest.MapFS{"certs/signer.der": {Data: der}}

	doc := mustParse(t, `<ds:KeyInfo xmlns:ds="`+NamespaceDSig+`"><ds:RetrievalMethod URI="certs/signer.der" Type="`+TypeRawX509Certificate+`"/></ds:KeyInfo>`)
	cfg := &verifierConfig{referenceResolver: FSReferenceResolver(fsys)}
	data := &KeyInfoData{}

	err := resolveRetrievalMethods(t.Context(), cfg, doc, doc.DocumentElement(), data)
	require.NoError(t, err)
	require.Len(t, data.X509Certificates, 1)
	require.Equal(t, cert.Raw, data.X509Certificates[0].Raw)
}

func TestResolveRetrievalMethodExternalX509Data(t *testing.T) {
	cert, der := selfSignedCert(t)
	x509Data := `<ds:X509Data xmlns:ds="` + NamespaceDSig + `"><ds:X509Certificate>` +
		base64.StdEncoding.EncodeToString(der) + `</ds:X509Certificate></ds:X509Data>`
	fsys := fstest.MapFS{"keyinfo/data.xml": {Data: []byte(x509Data)}}

	doc := mustParse(t, `<ds:KeyInfo xmlns:ds="`+NamespaceDSig+`"><ds:RetrievalMethod URI="keyinfo/data.xml" Type="`+TypeX509Data+`"/></ds:KeyInfo>`)
	cfg := &verifierConfig{referenceResolver: FSReferenceResolver(fsys)}
	data := &KeyInfoData{}

	err := resolveRetrievalMethods(t.Context(), cfg, doc, doc.DocumentElement(), data)
	require.NoError(t, err)
	require.Len(t, data.X509Certificates, 1)
	require.Equal(t, cert.Raw, data.X509Certificates[0].Raw)
}

func TestResolveRetrievalMethodNoResolverFailsClosed(t *testing.T) {
	doc := mustParse(t, `<ds:KeyInfo xmlns:ds="`+NamespaceDSig+`"><ds:RetrievalMethod URI="certs/signer.der" Type="`+TypeRawX509Certificate+`"/></ds:KeyInfo>`)
	cfg := &verifierConfig{}
	data := &KeyInfoData{}

	err := resolveRetrievalMethods(t.Context(), cfg, doc, doc.DocumentElement(), data)
	require.ErrorIs(t, err, ErrReferenceNotFound)
	require.Empty(t, data.X509Certificates)
}

func TestResolveRetrievalMethodLoopRejected(t *testing.T) {
	// A RetrievalMethod that references itself by id must fail closed rather than
	// dereferencing an unbounded chain.
	doc := mustParse(t, `<ds:KeyInfo xmlns:ds="`+NamespaceDSig+`"><ds:RetrievalMethod Id="rm1" URI="#rm1" Type="`+TypeX509Data+`"/></ds:KeyInfo>`)
	cfg := &verifierConfig{}
	data := &KeyInfoData{}

	err := resolveRetrievalMethods(t.Context(), cfg, doc, doc.DocumentElement(), data)
	require.ErrorIs(t, err, ErrRetrievalMethodLoop)
}

func TestResolveRetrievalMethodForeignNamespaceIgnored(t *testing.T) {
	// A foreign-namespace RetrievalMethod look-alike must not steer key retrieval,
	// even with no resolver configured (it must simply be skipped).
	doc := mustParse(t, `<ds:KeyInfo xmlns:ds="`+NamespaceDSig+`" xmlns:evil="urn:evil"><evil:RetrievalMethod URI="certs/signer.der" Type="`+TypeRawX509Certificate+`"/></ds:KeyInfo>`)
	cfg := &verifierConfig{}
	data := &KeyInfoData{}

	err := resolveRetrievalMethods(t.Context(), cfg, doc, doc.DocumentElement(), data)
	require.NoError(t, err)
	require.Empty(t, data.X509Certificates)
}

// oversizeResolver returns one byte more than the 64 MiB resolver cap, standing
// in for a caller-supplied ReferenceResolver that ignores the cap the shipped
// FSReferenceResolver enforces on itself.
type oversizeResolver struct{}

func (oversizeResolver) ResolveReference(_ context.Context, _ string) ([]byte, error) {
	return make([]byte, maxReferenceBytes+1), nil
}

// TestRetrievalMethodCapsCustomResolverResult proves the 64 MiB cap is enforced
// at the RetrievalMethod resolution site, not only inside FSReferenceResolver: a
// custom resolver returning an over-cap result fails closed with
// ErrReferenceTooLarge before the octets reach x509.ParseCertificate.
func TestRetrievalMethodCapsCustomResolverResult(t *testing.T) {
	doc := mustParse(t, `<ds:KeyInfo xmlns:ds="`+NamespaceDSig+`"><ds:RetrievalMethod URI="certs/big.der" Type="`+TypeRawX509Certificate+`"/></ds:KeyInfo>`)
	cfg := &verifierConfig{referenceResolver: oversizeResolver{}}
	data := &KeyInfoData{}

	err := resolveRetrievalMethods(t.Context(), cfg, doc, doc.DocumentElement(), data)
	require.ErrorIs(t, err, ErrReferenceTooLarge)
	require.Empty(t, data.X509Certificates)
}

// TestRetrievalMethodResolvesObjectInsideVerifiedSignature proves a same-document
// RetrievalMethod resolves an X509Data carried inside the Signature's own Object
// when the Signature is attached beneath the document (the verify-time layout).
// Passing the attached Signature as an extra resolution root would double-count
// the target — once through the document walk and again through the Signature
// subtree — and fail spuriously with ErrAmbiguousReference; resolving against the
// document only finds it exactly once.
func TestRetrievalMethodResolvesObjectInsideVerifiedSignature(t *testing.T) {
	cert, der := selfSignedCert(t)
	certB64 := base64.StdEncoding.EncodeToString(der)
	doc := mustParse(t, `<Root xmlns:ds="`+NamespaceDSig+`"><ds:Signature><ds:KeyInfo Id="ki">`+
		`<ds:RetrievalMethod URI="#cert-data" Type="`+TypeX509Data+`"/></ds:KeyInfo>`+
		`<ds:Object><ds:X509Data Id="cert-data"><ds:X509Certificate>`+certB64+
		`</ds:X509Certificate></ds:X509Data></ds:Object></ds:Signature></Root>`)
	kis := findElementsByIDUnder(doc.DocumentElement(), "ki")
	require.Len(t, kis, 1)
	cfg := &verifierConfig{}
	data := &KeyInfoData{}

	err := resolveRetrievalMethods(t.Context(), cfg, doc, kis[0], data)
	require.NoError(t, err)
	require.Len(t, data.X509Certificates, 1)
	require.Equal(t, cert.Raw, data.X509Certificates[0].Raw)
}

// TestRetrievalMethodRejectsUnsupportedTransform proves a RetrievalMethod's
// ds:Transforms are inspected and applied, not ignored: an unsupported transform
// fails closed with ErrUnsupportedTransform for both external and same-document
// targets, rather than silently accepting the resolved certificate.
func TestRetrievalMethodRejectsUnsupportedTransform(t *testing.T) {
	t.Run("external", func(t *testing.T) {
		_, der := selfSignedCert(t)
		fsys := fstest.MapFS{"certs/signer.der": {Data: der}}
		doc := mustParse(t, `<ds:KeyInfo xmlns:ds="`+NamespaceDSig+`"><ds:RetrievalMethod URI="certs/signer.der" Type="`+TypeRawX509Certificate+`">`+
			`<ds:Transforms><ds:Transform Algorithm="urn:unsupported"/></ds:Transforms></ds:RetrievalMethod></ds:KeyInfo>`)
		cfg := &verifierConfig{referenceResolver: FSReferenceResolver(fsys)}
		data := &KeyInfoData{}

		err := resolveRetrievalMethods(t.Context(), cfg, doc, doc.DocumentElement(), data)
		require.ErrorIs(t, err, ErrUnsupportedTransform)
		require.Empty(t, data.X509Certificates)
	})

	t.Run("same-document", func(t *testing.T) {
		_, der := selfSignedCert(t)
		certB64 := base64.StdEncoding.EncodeToString(der)
		doc := mustParse(t, `<ds:KeyInfo xmlns:ds="`+NamespaceDSig+`"><ds:RetrievalMethod URI="#cert" Type="`+TypeRawX509Certificate+`">`+
			`<ds:Transforms><ds:Transform Algorithm="urn:unsupported"/></ds:Transforms></ds:RetrievalMethod>`+
			`<ds:SPKIData Id="cert">`+certB64+`</ds:SPKIData></ds:KeyInfo>`)
		cfg := &verifierConfig{}
		data := &KeyInfoData{}

		err := resolveRetrievalMethods(t.Context(), cfg, doc, doc.DocumentElement(), data)
		require.ErrorIs(t, err, ErrUnsupportedTransform)
		require.Empty(t, data.X509Certificates)
	})
}

// TestRetrievalMethodAppliesSupportedTransform proves a supported transform
// pipeline is applied before Type interpretation: a same-document c14n transform
// canonicalizes the target X509Data subtree before it is reparsed, and an
// external base64 transform decodes the retrieved octets before the certificate
// is parsed.
func TestRetrievalMethodAppliesSupportedTransform(t *testing.T) {
	t.Run("same-document c14n", func(t *testing.T) {
		cert, der := selfSignedCert(t)
		certB64 := base64.StdEncoding.EncodeToString(der)
		doc := mustParse(t, `<ds:KeyInfo xmlns:ds="`+NamespaceDSig+`"><ds:RetrievalMethod URI="#x509" Type="`+TypeX509Data+`">`+
			`<ds:Transforms><ds:Transform Algorithm="`+C14N10+`"/></ds:Transforms></ds:RetrievalMethod>`+
			`<ds:X509Data Id="x509"><ds:X509Certificate>`+certB64+`</ds:X509Certificate></ds:X509Data></ds:KeyInfo>`)
		cfg := &verifierConfig{}
		data := &KeyInfoData{}

		err := resolveRetrievalMethods(t.Context(), cfg, doc, doc.DocumentElement(), data)
		require.NoError(t, err)
		require.Len(t, data.X509Certificates, 1)
		require.Equal(t, cert.Raw, data.X509Certificates[0].Raw)
	})

	t.Run("external base64", func(t *testing.T) {
		cert, der := selfSignedCert(t)
		b64File := base64.StdEncoding.EncodeToString(der)
		fsys := fstest.MapFS{"certs/signer.b64": {Data: []byte(b64File)}}
		doc := mustParse(t, `<ds:KeyInfo xmlns:ds="`+NamespaceDSig+`"><ds:RetrievalMethod URI="certs/signer.b64" Type="`+TypeRawX509Certificate+`">`+
			`<ds:Transforms><ds:Transform Algorithm="`+TransformBase64+`"/></ds:Transforms></ds:RetrievalMethod></ds:KeyInfo>`)
		cfg := &verifierConfig{referenceResolver: FSReferenceResolver(fsys)}
		data := &KeyInfoData{}

		err := resolveRetrievalMethods(t.Context(), cfg, doc, doc.DocumentElement(), data)
		require.NoError(t, err)
		require.Len(t, data.X509Certificates, 1)
		require.Equal(t, cert.Raw, data.X509Certificates[0].Raw)
	})
}
