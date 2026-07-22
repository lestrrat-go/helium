package xmldsig1

import (
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
// skid, when non-empty, is set as the SubjectKeyIdentifier so the SKI-match path
// can be exercised.
func selfSignedCert(t *testing.T, skid []byte) (*x509.Certificate, []byte) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(12345),
		Subject:      pkix.Name{CommonName: "retrieval-test"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		SubjectKeyId: skid,
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
	cert, der := selfSignedCert(t, nil)
	fsys := fstest.MapFS{"certs/signer.der": {Data: der}}

	doc := mustParse(t, `<ds:KeyInfo xmlns:ds="`+NamespaceDSig+`"><ds:RetrievalMethod URI="certs/signer.der" Type="`+TypeRawX509Certificate+`"/></ds:KeyInfo>`)
	cfg := &verifierConfig{referenceResolver: FSReferenceResolver(fsys)}
	data := &KeyInfoData{}

	err := resolveRetrievalMethods(t.Context(), cfg, doc, doc.DocumentElement(), nil, data)
	require.NoError(t, err)
	require.Len(t, data.X509Certificates, 1)
	require.Equal(t, cert.Raw, data.X509Certificates[0].Raw)
}

func TestResolveRetrievalMethodExternalX509Data(t *testing.T) {
	cert, der := selfSignedCert(t, nil)
	x509Data := `<ds:X509Data xmlns:ds="` + NamespaceDSig + `"><ds:X509Certificate>` +
		base64.StdEncoding.EncodeToString(der) + `</ds:X509Certificate></ds:X509Data>`
	fsys := fstest.MapFS{"keyinfo/data.xml": {Data: []byte(x509Data)}}

	doc := mustParse(t, `<ds:KeyInfo xmlns:ds="`+NamespaceDSig+`"><ds:RetrievalMethod URI="keyinfo/data.xml" Type="`+TypeX509Data+`"/></ds:KeyInfo>`)
	cfg := &verifierConfig{referenceResolver: FSReferenceResolver(fsys)}
	data := &KeyInfoData{}

	err := resolveRetrievalMethods(t.Context(), cfg, doc, doc.DocumentElement(), nil, data)
	require.NoError(t, err)
	require.Len(t, data.X509Certificates, 1)
	require.Equal(t, cert.Raw, data.X509Certificates[0].Raw)
}

func TestResolveRetrievalMethodNoResolverFailsClosed(t *testing.T) {
	doc := mustParse(t, `<ds:KeyInfo xmlns:ds="`+NamespaceDSig+`"><ds:RetrievalMethod URI="certs/signer.der" Type="`+TypeRawX509Certificate+`"/></ds:KeyInfo>`)
	cfg := &verifierConfig{}
	data := &KeyInfoData{}

	err := resolveRetrievalMethods(t.Context(), cfg, doc, doc.DocumentElement(), nil, data)
	require.ErrorIs(t, err, ErrReferenceNotFound)
	require.Empty(t, data.X509Certificates)
}

func TestResolveRetrievalMethodLoopRejected(t *testing.T) {
	// A RetrievalMethod that references itself by id must fail closed rather than
	// dereferencing an unbounded chain.
	doc := mustParse(t, `<ds:KeyInfo xmlns:ds="`+NamespaceDSig+`"><ds:RetrievalMethod Id="rm1" URI="#rm1" Type="`+TypeX509Data+`"/></ds:KeyInfo>`)
	cfg := &verifierConfig{}
	data := &KeyInfoData{}

	err := resolveRetrievalMethods(t.Context(), cfg, doc, doc.DocumentElement(), nil, data)
	require.ErrorIs(t, err, ErrRetrievalMethodLoop)
}

func TestResolveRetrievalMethodForeignNamespaceIgnored(t *testing.T) {
	// A foreign-namespace RetrievalMethod look-alike must not steer key retrieval,
	// even with no resolver configured (it must simply be skipped).
	doc := mustParse(t, `<ds:KeyInfo xmlns:ds="`+NamespaceDSig+`" xmlns:evil="urn:evil"><evil:RetrievalMethod URI="certs/signer.der" Type="`+TypeRawX509Certificate+`"/></ds:KeyInfo>`)
	cfg := &verifierConfig{}
	data := &KeyInfoData{}

	err := resolveRetrievalMethods(t.Context(), cfg, doc, doc.DocumentElement(), nil, data)
	require.NoError(t, err)
	require.Empty(t, data.X509Certificates)
}
