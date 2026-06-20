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
	"time"

	helium "github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

// newKeyInfoCertElem builds a <KeyInfo><X509Data><X509Certificate>... subtree
// whose elements are placed in the namespace ns (declared under prefix px). The
// returned element is the KeyInfo root, ready to hand to parseKeyInfo.
func newKeyInfoCertElem(t *testing.T, doc *helium.Document, px, ns string, certDER []byte) *helium.Element {
	t.Helper()

	keyInfo := doc.CreateElement("KeyInfo")
	require.NoError(t, keyInfo.DeclareNamespace(px, ns))
	require.NoError(t, keyInfo.SetActiveNamespace(px, ns))

	x509Data := doc.CreateElement("X509Data")
	require.NoError(t, x509Data.SetActiveNamespace(px, ns))
	require.NoError(t, keyInfo.AddChild(x509Data))

	certElem := doc.CreateElement("X509Certificate")
	require.NoError(t, certElem.SetActiveNamespace(px, ns))
	require.NoError(t, certElem.AddChild(
		doc.CreateText([]byte(base64.StdEncoding.EncodeToString(certDER)))))
	require.NoError(t, x509Data.AddChild(certElem))

	return keyInfo
}

// selfSignedCertDER returns a throwaway self-signed certificate's DER bytes.
func selfSignedCertDER(t *testing.T) []byte {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "test"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	require.NoError(t, err)
	return der
}

// TestKeyInfoForeignNamespaceX509Ignored guards against namespace confusion in
// KeyInfo parsing. The parser fed key material (certificates, key values) into
// key resolution; matching those elements on local name alone would let a
// foreign-namespace <evil:X509Data>/<evil:X509Certificate> masquerade as the
// core ds:X509Data and supply an attacker-chosen verification certificate. A
// foreign-namespace KeyInfo subtree must therefore yield no parsed key material.
func TestKeyInfoForeignNamespaceX509Ignored(t *testing.T) {
	doc, err := helium.NewParser().Parse(context.Background(), []byte(`<root/>`))
	require.NoError(t, err)
	certDER := selfSignedCertDER(t)

	const evilNS = "urn:example:evil"
	keyInfo := newKeyInfoCertElem(t, doc, "evil", evilNS, certDER)
	require.Equal(t, evilNS, elementNamespaceURI(keyInfo))

	data, err := parseKeyInfo(keyInfo)
	require.NoError(t, err)
	require.Empty(t, data.X509Certificates,
		"a foreign-namespace X509Data look-alike must not supply a certificate")
}

// TestKeyInfoCoreNamespaceX509Parsed is the positive control: a correctly
// ds-namespaced KeyInfo/X509Data/X509Certificate subtree must still parse.
func TestKeyInfoCoreNamespaceX509Parsed(t *testing.T) {
	doc, err := helium.NewParser().Parse(context.Background(), []byte(`<root/>`))
	require.NoError(t, err)
	certDER := selfSignedCertDER(t)

	keyInfo := newKeyInfoCertElem(t, doc, nsPrefix, NamespaceDSig, certDER)
	require.Equal(t, NamespaceDSig, elementNamespaceURI(keyInfo))

	data, err := parseKeyInfo(keyInfo)
	require.NoError(t, err)
	require.Len(t, data.X509Certificates, 1,
		"a correctly ds-namespaced X509Certificate must still parse")
}

// TestKeyInfoForeignNamespaceRSAKeyValueIgnored ensures the same guard covers
// the RSAKeyValue path: a foreign-namespace KeyValue/RSAKeyValue look-alike
// must not supply key material.
func TestKeyInfoForeignNamespaceRSAKeyValueIgnored(t *testing.T) {
	doc, err := helium.NewParser().Parse(context.Background(), []byte(`<root/>`))
	require.NoError(t, err)

	const evilNS = "urn:example:evil"
	keyInfo := doc.CreateElement("KeyInfo")
	require.NoError(t, keyInfo.DeclareNamespace("evil", evilNS))
	require.NoError(t, keyInfo.SetActiveNamespace("evil", evilNS))

	keyValue := doc.CreateElement("KeyValue")
	require.NoError(t, keyValue.SetActiveNamespace("evil", evilNS))
	require.NoError(t, keyInfo.AddChild(keyValue))

	rsaKV := doc.CreateElement("RSAKeyValue")
	require.NoError(t, rsaKV.SetActiveNamespace("evil", evilNS))
	require.NoError(t, keyValue.AddChild(rsaKV))

	mod := doc.CreateElement("Modulus")
	require.NoError(t, mod.SetActiveNamespace("evil", evilNS))
	require.NoError(t, mod.AddChild(doc.CreateText([]byte(base64.StdEncoding.EncodeToString([]byte{1, 2, 3})))))
	require.NoError(t, rsaKV.AddChild(mod))

	data, err := parseKeyInfo(keyInfo)
	require.NoError(t, err)
	require.Nil(t, data.RSAKeyValue,
		"a foreign-namespace RSAKeyValue look-alike must not supply key material")
}
