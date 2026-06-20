package xmlenc1_test

import (
	"crypto/rand"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xmlenc1"
	"github.com/stretchr/testify/require"
)

// D-ENC-001: decrypting TypeContent plaintext that uses a namespace prefix
// declared on an ANCESTOR of EncryptedData must succeed. The temporary parse
// wrapper has to carry the in-scope namespace declarations so prefixes in the
// decrypted fragment resolve.
func TestDecryptContent_InScopeAncestorNamespace(t *testing.T) {
	sessionKey := make([]byte, 32)
	_, err := rand.Read(sessionKey)
	require.NoError(t, err)

	algorithm := xmlenc1.AES256GCM

	// The decrypted content uses the saml: prefix, which is NOT declared in
	// the fragment itself — only on an ancestor of EncryptedData.
	plaintext := []byte(`<saml:NameID>user@example.com</saml:NameID>`)
	cipher, err := xmlenc1.EncryptBytesForTest(algorithm, sessionKey, plaintext)
	require.NoError(t, err)

	// Build a document where saml: is declared on the grandparent of
	// EncryptedData, then splice the EncryptedData element in as content.
	doc := mustParseXML(t,
		`<saml:Subject xmlns:saml="urn:oasis:names:tc:SAML:2.0:assertion"><wrapper/></saml:Subject>`)

	ed := &xmlenc1.EncryptedData{
		Type:             xmlenc1.TypeContent,
		EncryptionMethod: &xmlenc1.EncryptionMethod{Algorithm: algorithm},
		CipherValue:      cipher,
	}
	edElem, err := xmlenc1.MarshalEncryptedDataForTest(doc, ed)
	require.NoError(t, err)

	// Attach EncryptedData under <wrapper>, so saml: is in-scope via the
	// ancestor chain (wrapper -> saml:Subject).
	wrapper := doc.DocumentElement().FirstChild().(*helium.Element)
	require.NoError(t, wrapper.AddChild(edElem))

	decryptor := xmlenc1.NewDecryptor().SessionKey(sessionKey)
	nodes, err := decryptor.Decrypt(t.Context(), edElem)
	require.NoError(t, err)
	require.Len(t, nodes, 1)

	elem, ok := nodes[0].(*helium.Element)
	require.True(t, ok, "decrypted node should be an element")
	require.Equal(t, "NameID", elem.LocalName())
	require.Equal(t, "urn:oasis:names:tc:SAML:2.0:assertion", elem.URI(),
		"saml: prefix must resolve to the ancestor-declared namespace URI")
}

// D-ENC-002: decrypting TypeContent plaintext whose unprefixed elements must
// resolve in the PARENT's default namespace — NOT in the XML-Encryption
// namespace that EncryptedData itself declares. The decrypted content replaces
// EncryptedData, so it must be parsed in the in-scope-namespace context of
// EncryptedData's parent, not EncryptedData's own declarations.
func TestDecryptContent_ParentDefaultNamespace(t *testing.T) {
	sessionKey := make([]byte, 32)
	_, err := rand.Read(sessionKey)
	require.NoError(t, err)

	algorithm := xmlenc1.AES256GCM

	// The decrypted content is an unprefixed element. At its replacement
	// position (a child of <wrapper> in the app namespace) it must inherit
	// the parent's default namespace.
	plaintext := []byte(`<child/>`)
	cipher, err := xmlenc1.EncryptBytesForTest(algorithm, sessionKey, plaintext)
	require.NoError(t, err)

	// The parent declares a default application namespace that differs from
	// the XML-Encryption namespace EncryptedData declares for itself.
	const appNS = "http://example.com/app"
	doc := mustParseXML(t,
		`<root xmlns="`+appNS+`"><wrapper/></root>`)

	ed := &xmlenc1.EncryptedData{
		Type:             xmlenc1.TypeContent,
		EncryptionMethod: &xmlenc1.EncryptionMethod{Algorithm: algorithm},
		CipherValue:      cipher,
	}
	edElem, err := xmlenc1.MarshalEncryptedDataForTest(doc, ed)
	require.NoError(t, err)

	// EncryptedData declares its OWN default namespace (the XML-Encryption
	// namespace). Parsing the decrypted content with EncryptedData's own
	// declarations would wrongly resolve <child/> into XMLENC; it must use
	// the parent's in-scope default namespace instead.
	require.NoError(t, edElem.DeclareNamespace("", xmlenc1.NamespaceXMLEnc))

	wrapper := doc.DocumentElement().FirstChild().(*helium.Element)
	require.NoError(t, wrapper.AddChild(edElem))

	decryptor := xmlenc1.NewDecryptor().SessionKey(sessionKey)
	nodes, err := decryptor.Decrypt(t.Context(), edElem)
	require.NoError(t, err)
	require.Len(t, nodes, 1)

	elem, ok := nodes[0].(*helium.Element)
	require.True(t, ok, "decrypted node should be an element")
	require.Equal(t, "child", elem.LocalName())
	require.Equal(t, appNS, elem.URI(),
		"unprefixed element must resolve in the parent's default namespace, not XMLENC")
}

// D-ENC-003: a payload prefix declared on EncryptedData's parent ancestor
// still resolves when the content is parsed in the parent's context.
func TestDecryptContent_PrefixOnParentAncestor(t *testing.T) {
	sessionKey := make([]byte, 32)
	_, err := rand.Read(sessionKey)
	require.NoError(t, err)

	algorithm := xmlenc1.AES256GCM

	plaintext := []byte(`<app:item>v</app:item>`)
	cipher, err := xmlenc1.EncryptBytesForTest(algorithm, sessionKey, plaintext)
	require.NoError(t, err)

	const appNS = "http://example.com/app"
	doc := mustParseXML(t,
		`<root xmlns:app="`+appNS+`"><branch><wrapper/></branch></root>`)

	ed := &xmlenc1.EncryptedData{
		Type:             xmlenc1.TypeContent,
		EncryptionMethod: &xmlenc1.EncryptionMethod{Algorithm: algorithm},
		CipherValue:      cipher,
	}
	edElem, err := xmlenc1.MarshalEncryptedDataForTest(doc, ed)
	require.NoError(t, err)

	wrapper := doc.DocumentElement().FirstChild().FirstChild().(*helium.Element)
	require.NoError(t, wrapper.AddChild(edElem))

	decryptor := xmlenc1.NewDecryptor().SessionKey(sessionKey)
	nodes, err := decryptor.Decrypt(t.Context(), edElem)
	require.NoError(t, err)
	require.Len(t, nodes, 1)

	elem, ok := nodes[0].(*helium.Element)
	require.True(t, ok, "decrypted node should be an element")
	require.Equal(t, "item", elem.LocalName())
	require.Equal(t, appNS, elem.URI(),
		"app: prefix declared on parent ancestor must resolve")
}
