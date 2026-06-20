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
