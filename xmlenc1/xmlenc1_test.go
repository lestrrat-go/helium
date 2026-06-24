package xmlenc1_test

import (
	"crypto/rand"
	"crypto/rsa"
	"strings"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xmlenc1"
	"github.com/stretchr/testify/require"
)

func mustParseXML(t *testing.T, xml string) *helium.Document {
	t.Helper()
	doc, err := helium.NewParser().Parse(t.Context(), []byte(xml))
	require.NoError(t, err)
	return doc
}

func generateRSAKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	return key
}

func randKey(t *testing.T, n int) []byte {
	t.Helper()
	k := make([]byte, n)
	_, err := rand.Read(k)
	require.NoError(t, err)
	return k
}

func findEncryptedData(t *testing.T, n helium.Node) *helium.Element {
	t.Helper()
	elem, ok := n.(*helium.Element)
	if ok && elem.LocalName() == elemEncryptedData {
		return elem
	}
	for child := n.FirstChild(); child != nil; child = child.NextSibling() {
		if found := findEncryptedData(t, child); found != nil {
			return found
		}
	}
	return nil
}

// childNames returns the local names of the direct element children of n,
// in document order.
func childNames(n helium.Node) []string {
	var names []string
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		e, ok := c.(*helium.Element)
		if !ok {
			continue
		}
		names = append(names, e.LocalName())
	}
	return names
}

// spaceSeparateBase64 inserts an XML space every `col` characters into
// the base64 text inside every <xenc:tag>...</xenc:tag> element. XSD
// base64Binary permits interspersed whitespace (space/tab/CR/LF), but
// Go's encoding/base64 only tolerates CR/LF and rejects space/tab, so
// this exercises whitespace that the standard decoder rejects.
func spaceSeparateBase64(xml, tag string, col int) string {
	open := "<xenc:" + tag + ">"
	closeTag := "</xenc:" + tag + ">"
	var b strings.Builder
	rest := xml
	for {
		i := strings.Index(rest, open)
		if i < 0 {
			b.WriteString(rest)
			break
		}
		b.WriteString(rest[:i+len(open)])
		rest = rest[i+len(open):]
		j := strings.Index(rest, closeTag)
		if j < 0 {
			b.WriteString(rest)
			break
		}
		content := rest[:j]
		var wrapped strings.Builder
		for k := 0; k < len(content); k += col {
			if k > 0 {
				wrapped.WriteByte(' ')
			}
			end := min(k+col, len(content))
			wrapped.WriteString(content[k:end])
		}
		b.WriteString(wrapped.String())
		rest = rest[j:]
	}
	return b.String()
}

const elemEncryptedData = "EncryptedData"

const samlAssertion = `<saml:Assertion xmlns:saml="urn:oasis:names:tc:SAML:2.0:assertion" ID="_abc123" IssueInstant="2024-01-01T00:00:00Z" Version="2.0"><saml:Issuer>https://idp.example.com</saml:Issuer><saml:Subject><saml:NameID>user@example.com</saml:NameID></saml:Subject></saml:Assertion>`

func TestEncryptDecryptRoundTrip(t *testing.T) {
	t.Run("element RSAOAEP AES128CBC", func(t *testing.T) {
		key := generateRSAKey(t)
		doc := mustParseXML(t, samlAssertion)
		elem := doc.DocumentElement()

		encryptor := xmlenc1.NewEncryptor().
			BlockAlgorithm(xmlenc1.AES128CBC).
			AllowLegacyCBC(true).
			KeyTransportAlgorithm(xmlenc1.RSAOAEP).
			RecipientPublicKey(&key.PublicKey)

		edElem, err := encryptor.EncryptElement(t.Context(), elem)
		require.NoError(t, err)
		require.NotNil(t, edElem)

		// Verify the document now has EncryptedData.
		xml, err := helium.WriteString(doc)
		require.NoError(t, err)
		require.Contains(t, xml, elemEncryptedData)
		require.NotContains(t, xml, "user@example.com")

		// Decrypt. CBC requires explicit opt-in (see ErrCBCRequiresOptIn).
		decryptor := xmlenc1.NewDecryptor().PrivateKey(key).AllowUnauthenticatedCBC(true)
		nodes, err := decryptor.Decrypt(t.Context(), edElem)
		require.NoError(t, err)
		require.Len(t, nodes, 1)

		s, err := helium.WriteString(nodes[0])
		require.NoError(t, err)
		require.Contains(t, s, "user@example.com")
	})

	t.Run("element RSAOAEP AES256GCM", func(t *testing.T) {
		key := generateRSAKey(t)
		doc := mustParseXML(t, samlAssertion)
		elem := doc.DocumentElement()

		encryptor := xmlenc1.NewEncryptor().
			BlockAlgorithm(xmlenc1.AES256GCM).
			KeyTransportAlgorithm(xmlenc1.RSAOAEP).
			RecipientPublicKey(&key.PublicKey)

		edElem, err := encryptor.EncryptElement(t.Context(), elem)
		require.NoError(t, err)

		decryptor := xmlenc1.NewDecryptor().PrivateKey(key)
		nodes, err := decryptor.Decrypt(t.Context(), edElem)
		require.NoError(t, err)
		require.Len(t, nodes, 1)

		s, err := helium.WriteString(nodes[0])
		require.NoError(t, err)
		require.Contains(t, s, "user@example.com")
	})

	t.Run("content RSAOAEP", func(t *testing.T) {
		xml := `<root><data>Secret content</data><more>Also secret</more></root>`
		key := generateRSAKey(t)
		doc := mustParseXML(t, xml)
		elem := doc.DocumentElement()

		encryptor := xmlenc1.NewEncryptor().
			BlockAlgorithm(xmlenc1.AES128GCM).
			KeyTransportAlgorithm(xmlenc1.RSAOAEP).
			RecipientPublicKey(&key.PublicKey)

		edElem, err := encryptor.EncryptContent(t.Context(), elem)
		require.NoError(t, err)
		require.NotNil(t, edElem)

		out, err := helium.WriteString(doc)
		require.NoError(t, err)
		require.Contains(t, out, elemEncryptedData)
		require.NotContains(t, out, "Secret content")

		// Decrypt.
		decryptor := xmlenc1.NewDecryptor().PrivateKey(key)
		nodes, err := decryptor.Decrypt(t.Context(), edElem)
		require.NoError(t, err)
		require.GreaterOrEqual(t, len(nodes), 1)
	})

	t.Run("AES key wrap", func(t *testing.T) {
		kek := make([]byte, 32) // AES-256 key encryption key
		_, err := rand.Read(kek)
		require.NoError(t, err)

		doc := mustParseXML(t, samlAssertion)
		elem := doc.DocumentElement()

		encryptor := xmlenc1.NewEncryptor().
			BlockAlgorithm(xmlenc1.AES256CBC).
			AllowLegacyCBC(true).
			KeyWrapAlgorithm(xmlenc1.AES256KeyWrap).
			KeyEncryptionKey(kek)

		edElem, err := encryptor.EncryptElement(t.Context(), elem)
		require.NoError(t, err)

		decryptor := xmlenc1.NewDecryptor().KeyEncryptionKey(kek).AllowUnauthenticatedCBC(true)
		nodes, err := decryptor.Decrypt(t.Context(), edElem)
		require.NoError(t, err)
		require.Len(t, nodes, 1)

		s, err := helium.WriteString(nodes[0])
		require.NoError(t, err)
		require.Contains(t, s, "user@example.com")
	})

	t.Run("session key", func(t *testing.T) {
		sessionKey := make([]byte, 16)
		_, err := rand.Read(sessionKey)
		require.NoError(t, err)

		doc := mustParseXML(t, samlAssertion)
		elem := doc.DocumentElement()

		encryptor := xmlenc1.NewEncryptor().
			BlockAlgorithm(xmlenc1.AES128CBC).
			AllowLegacyCBC(true).
			SessionKey(sessionKey)

		edElem, err := encryptor.EncryptElement(t.Context(), elem)
		require.NoError(t, err)

		decryptor := xmlenc1.NewDecryptor().SessionKey(sessionKey).AllowUnauthenticatedCBC(true)
		nodes, err := decryptor.Decrypt(t.Context(), edElem)
		require.NoError(t, err)
		require.Len(t, nodes, 1)

		s, err := helium.WriteString(nodes[0])
		require.NoError(t, err)
		require.Contains(t, s, "user@example.com")
	})
}

func TestEncryptErrors(t *testing.T) {
	t.Run("no config", func(t *testing.T) {
		doc := mustParseXML(t, samlAssertion)
		encryptor := xmlenc1.NewEncryptor()
		_, err := encryptor.EncryptElement(t.Context(), doc.DocumentElement())
		require.Error(t, err)
	})

	t.Run("no key", func(t *testing.T) {
		doc := mustParseXML(t, samlAssertion)
		encryptor := xmlenc1.NewEncryptor().BlockAlgorithm(xmlenc1.AES256GCM)
		_, err := encryptor.EncryptElement(t.Context(), doc.DocumentElement())
		require.Error(t, err)
		require.ErrorIs(t, err, xmlenc1.ErrMissingConfig)
	})

	t.Run("decrypt wrong key", func(t *testing.T) {
		key1 := generateRSAKey(t)
		key2 := generateRSAKey(t)
		doc := mustParseXML(t, samlAssertion)

		encryptor := xmlenc1.NewEncryptor().
			BlockAlgorithm(xmlenc1.AES128CBC).
			AllowLegacyCBC(true).
			KeyTransportAlgorithm(xmlenc1.RSAOAEP).
			RecipientPublicKey(&key1.PublicKey)

		edElem, err := encryptor.EncryptElement(t.Context(), doc.DocumentElement())
		require.NoError(t, err)

		// Decrypt with wrong key. Opt in to CBC so the failure exercises
		// the wrong-key (RSA) path rather than the CBC opt-in gate.
		decryptor := xmlenc1.NewDecryptor().PrivateKey(key2).AllowUnauthenticatedCBC(true)
		_, err = decryptor.Decrypt(t.Context(), edElem)
		require.Error(t, err)
	})
}

func TestEncryptElementPlacement(t *testing.T) {
	t.Run("preserves sibling order", func(t *testing.T) {
		for _, tc := range []struct {
			name   string
			xml    string
			target string // local name of element to encrypt
			want   []string
		}{
			{
				name:   "middle",
				xml:    `<root><a/><secret/><b/></root>`,
				target: "secret",
				want:   []string{"a", elemEncryptedData, "b"},
			},
			{
				name:   "first",
				xml:    `<root><secret/><a/><b/></root>`,
				target: "secret",
				want:   []string{elemEncryptedData, "a", "b"},
			},
			{
				name:   "last",
				xml:    `<root><a/><b/><secret/></root>`,
				target: "secret",
				want:   []string{"a", "b", elemEncryptedData},
			},
			{
				name:   "only",
				xml:    `<root><secret/></root>`,
				target: "secret",
				want:   []string{elemEncryptedData},
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				key := generateRSAKey(t)
				doc := mustParseXML(t, tc.xml)

				var target *helium.Element
				for c := doc.DocumentElement().FirstChild(); c != nil; c = c.NextSibling() {
					if e, ok := c.(*helium.Element); ok && e.LocalName() == tc.target {
						target = e
						break
					}
				}
				require.NotNil(t, target, "target element %q not found", tc.target)

				encryptor := xmlenc1.NewEncryptor().
					BlockAlgorithm(xmlenc1.AES128CBC).
					AllowLegacyCBC(true).
					KeyTransportAlgorithm(xmlenc1.RSAOAEP).
					RecipientPublicKey(&key.PublicKey)

				_, err := encryptor.EncryptElement(t.Context(), target)
				require.NoError(t, err)

				require.Equal(t, tc.want, childNames(doc.DocumentElement()))
			})
		}
	})

	t.Run("root in place", func(t *testing.T) {
		// Encrypting the document root element should replace it in place,
		// leaving EncryptedData as the new document element.
		key := generateRSAKey(t)
		doc := mustParseXML(t, `<secret>hidden</secret>`)

		encryptor := xmlenc1.NewEncryptor().
			BlockAlgorithm(xmlenc1.AES128CBC).
			AllowLegacyCBC(true).
			KeyTransportAlgorithm(xmlenc1.RSAOAEP).
			RecipientPublicKey(&key.PublicKey)

		edElem, err := encryptor.EncryptElement(t.Context(), doc.DocumentElement())
		require.NoError(t, err)
		require.NotNil(t, edElem)

		require.Equal(t, elemEncryptedData, doc.DocumentElement().LocalName())

		xml, err := helium.WriteString(doc)
		require.NoError(t, err)
		require.NotContains(t, xml, "hidden")
	})
}

func TestDecrypt(t *testing.T) {
	// TestDecryptElementYieldsNonElement covers the decryptElement branch that
	// rejects a Type=Element payload whose decrypted plaintext is not a single
	// element (here: plain text content).
	t.Run("element yields non-element", func(t *testing.T) {
		sessionKey := make([]byte, 32)
		algorithm := xmlenc1.AES256GCM

		// Plaintext that is bare text, not an element. For Type=Element this
		// must be rejected as ErrDecryptionFailed.
		cipher, err := xmlenc1.EncryptBytesForTest(algorithm, sessionKey, []byte("just text, no element"))
		require.NoError(t, err)

		doc := mustParseXML(t, `<root/>`)
		ed := &xmlenc1.EncryptedData{
			Type:             xmlenc1.TypeElement,
			EncryptionMethod: &xmlenc1.EncryptionMethod{Algorithm: algorithm},
			CipherValue:      cipher,
		}
		elem, err := xmlenc1.MarshalEncryptedDataForTest(doc, ed)
		require.NoError(t, err)

		_, err = xmlenc1.NewDecryptor().SessionKey(sessionKey).Decrypt(t.Context(), elem)
		require.ErrorIs(t, err, xmlenc1.ErrDecryptionFailed)
	})

	// TestDecryptContentMultipleNodes covers the Type=Content branch of
	// decryptElement that collects multiple decrypted child nodes.
	t.Run("content multiple nodes", func(t *testing.T) {
		key := generateRSAKey(t)
		doc := mustParseXML(t, `<root><a>1</a><b>2</b>text</root>`)

		encryptor := xmlenc1.NewEncryptor().
			BlockAlgorithm(xmlenc1.AES256GCM).
			KeyTransportAlgorithm(xmlenc1.RSAOAEP11).
			RecipientPublicKey(&key.PublicKey)

		elem, err := encryptor.EncryptContent(t.Context(), doc.DocumentElement())
		require.NoError(t, err)

		decryptor := xmlenc1.NewDecryptor().PrivateKey(key)
		nodes, err := decryptor.Decrypt(t.Context(), elem)
		require.NoError(t, err)
		require.GreaterOrEqual(t, len(nodes), 2)

		// Sanity: the recovered content includes the original element children.
		var names []string
		for _, n := range nodes {
			if e, ok := helium.AsNode[*helium.Element](n); ok {
				names = append(names, e.Name())
			}
		}
		require.Contains(t, names, "a")
		require.Contains(t, names, "b")
	})

	// TestDecryptLineWrappedCipherValue verifies that base64 CipherValue text
	// containing interspersed XML whitespace (here, spaces) decodes
	// correctly. base64Binary in XSD permits interspersed whitespace, and
	// real producers routinely wrap base64 at fixed columns.
	t.Run("line-wrapped CipherValue", func(t *testing.T) {
		key := generateRSAKey(t)
		doc := mustParseXML(t, samlAssertion)
		elem := doc.DocumentElement()

		encryptor := xmlenc1.NewEncryptor().
			BlockAlgorithm(xmlenc1.AES256GCM).
			KeyTransportAlgorithm(xmlenc1.RSAOAEP).
			RecipientPublicKey(&key.PublicKey)

		_, err := encryptor.EncryptElement(t.Context(), elem)
		require.NoError(t, err)

		xml, err := helium.WriteString(doc)
		require.NoError(t, err)

		// Insert spaces into the base64 inside every CipherValue (both the
		// RSA-encrypted session key and the AES-encrypted payload).
		wrapped := spaceSeparateBase64(xml, "CipherValue", 16)
		require.NotEqual(t, xml, wrapped, "expected spaces inserted into base64")

		reDoc := mustParseXML(t, wrapped)
		edElem := reDoc.DocumentElement()

		decryptor := xmlenc1.NewDecryptor().PrivateKey(key)
		nodes, err := decryptor.Decrypt(t.Context(), edElem)
		require.NoError(t, err)
		require.Len(t, nodes, 1)

		s, err := helium.WriteString(nodes[0])
		require.NoError(t, err)
		require.Contains(t, s, "user@example.com")
	})
}

func TestEncryptorImmutability(t *testing.T) {
	e1 := xmlenc1.NewEncryptor().BlockAlgorithm(xmlenc1.AES128CBC).AllowLegacyCBC(true)
	e2 := e1.BlockAlgorithm(xmlenc1.AES256GCM)

	// e1 should still be AES128CBC (not modified by e2).
	key := generateRSAKey(t)
	doc1 := mustParseXML(t, samlAssertion)
	doc2 := mustParseXML(t, samlAssertion)

	e1 = e1.KeyTransportAlgorithm(xmlenc1.RSAOAEP).RecipientPublicKey(&key.PublicKey)
	e2 = e2.KeyTransportAlgorithm(xmlenc1.RSAOAEP).RecipientPublicKey(&key.PublicKey)

	_, err := e1.EncryptElement(t.Context(), doc1.DocumentElement())
	require.NoError(t, err)

	_, err = e2.EncryptElement(t.Context(), doc2.DocumentElement())
	require.NoError(t, err)
}
