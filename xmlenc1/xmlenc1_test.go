package xmlenc1_test

import (
	"crypto/elliptic"
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

// EncryptBytes is the counterpart of DecryptBytes: the package could consume
// arbitrary-octet EncryptedData but had no way to produce it, so the only
// bytes-in path was a test-only export.
func TestEncryptBytes(t *testing.T) {
	payload := []byte{0x00, 0x01, 0xff, 0xfe, 'n', 'o', 't', ' ', 'x', 'm', 'l'}

	t.Run("round trip under RSA key transport", func(t *testing.T) {
		key := generateRSAKey(t)
		doc := mustParseXML(t, `<root/>`)

		edElem, err := xmlenc1.NewEncryptor().
			KeyTransportAlgorithm(xmlenc1.RSAOAEP).
			RecipientPublicKey(&key.PublicKey).
			EncryptBytes(t.Context(), doc, payload)
		require.NoError(t, err)

		got, err := xmlenc1.NewDecryptor().PrivateKey(key).DecryptBytes(t.Context(), edElem)
		require.NoError(t, err)
		require.Equal(t, payload, got)
	})

	t.Run("round trip under AES key wrap", func(t *testing.T) {
		kek := randKey(t, 32)
		doc := mustParseXML(t, `<root/>`)

		edElem, err := xmlenc1.NewEncryptor().
			KeyWrapAlgorithm(xmlenc1.AES256KeyWrap).
			KeyEncryptionKey(kek).
			EncryptBytes(t.Context(), doc, payload)
		require.NoError(t, err)

		got, err := xmlenc1.NewDecryptor().KeyEncryptionKey(kek).DecryptBytes(t.Context(), edElem)
		require.NoError(t, err)
		require.Equal(t, payload, got)
	})

	t.Run("round trip under a pre-shared session key", func(t *testing.T) {
		sessionKey := randKey(t, 32)
		doc := mustParseXML(t, `<root/>`)

		edElem, err := xmlenc1.NewEncryptor().
			BlockAlgorithm(xmlenc1.AES256GCM11).
			SessionKey(sessionKey).
			EncryptBytes(t.Context(), doc, payload)
		require.NoError(t, err)

		got, err := xmlenc1.NewDecryptor().SessionKey(sessionKey).DecryptBytes(t.Context(), edElem)
		require.NoError(t, err)
		require.Equal(t, payload, got)
	})

	// W3C xmlenc-core1 §3.1: @Type is absent when the plaintext is neither an
	// element nor element content.
	t.Run("emits no Type attribute", func(t *testing.T) {
		doc := mustParseXML(t, `<root/>`)
		edElem, err := xmlenc1.NewEncryptor().
			SessionKey(randKey(t, 32)).
			EncryptBytes(t.Context(), doc, payload)
		require.NoError(t, err)

		s, err := helium.WriteString(edElem)
		require.NoError(t, err)
		require.NotContains(t, s, "Type=")
	})

	// The returned element is detached: unlike EncryptElement, nothing is
	// spliced into the document.
	t.Run("leaves the document untouched", func(t *testing.T) {
		doc := mustParseXML(t, `<root><keep/></root>`)
		before, err := helium.WriteString(doc)
		require.NoError(t, err)

		_, err = xmlenc1.NewEncryptor().
			SessionKey(randKey(t, 32)).
			EncryptBytes(t.Context(), doc, payload)
		require.NoError(t, err)

		after, err := helium.WriteString(doc)
		require.NoError(t, err)
		require.Equal(t, before, after)
	})

	t.Run("reports a missing document", func(t *testing.T) {
		_, err := xmlenc1.NewEncryptor().
			SessionKey(randKey(t, 32)).
			EncryptBytes(t.Context(), nil, payload)
		require.ErrorIs(t, err, xmlenc1.ErrMissingConfig)
	})

	t.Run("applies the same key-source checks", func(t *testing.T) {
		doc := mustParseXML(t, `<root/>`)
		_, err := xmlenc1.NewEncryptor().EncryptBytes(t.Context(), doc, payload)
		require.ErrorIs(t, err, xmlenc1.ErrMissingConfig)
	})

	t.Run("applies the CBC opt-in gate", func(t *testing.T) {
		doc := mustParseXML(t, `<root/>`)
		_, err := xmlenc1.NewEncryptor().
			BlockAlgorithm(xmlenc1.AES256CBC).
			SessionKey(randKey(t, 32)).
			EncryptBytes(t.Context(), doc, payload)
		require.ErrorIs(t, err, xmlenc1.ErrCBCEncryptionRequiresOptIn)
	})
}

// A zero-value Encryptor/Decryptor is a usable value, not a panic: it behaves
// as a default one built by NewEncryptor/NewDecryptor with nothing configured,
// matching xmldsig1's zero-value Signer. Callers reach the zero value through
// an embedded struct field or a var declaration, and a nil-map-style panic is
// a poor diagnostic for "you forgot to configure a key".
func TestZeroValueBuilders(t *testing.T) {
	t.Run("Encryptor reports missing config", func(t *testing.T) {
		var e xmlenc1.Encryptor

		doc := mustParseXML(t, samlAssertion)
		_, err := e.EncryptElement(t.Context(), doc.DocumentElement())
		require.ErrorIs(t, err, xmlenc1.ErrMissingConfig)

		doc = mustParseXML(t, samlAssertion)
		_, err = e.EncryptContent(t.Context(), doc.DocumentElement())
		require.ErrorIs(t, err, xmlenc1.ErrMissingConfig)

		doc = mustParseXML(t, samlAssertion)
		_, err = e.EncryptBytes(t.Context(), doc, []byte("payload"))
		require.ErrorIs(t, err, xmlenc1.ErrMissingConfig)
	})

	t.Run("Encryptor builds on from the zero value", func(t *testing.T) {
		var e xmlenc1.Encryptor
		key := generateRSAKey(t)

		doc := mustParseXML(t, samlAssertion)
		edElem, err := e.
			KeyTransportAlgorithm(xmlenc1.RSAOAEP).
			RecipientPublicKey(&key.PublicKey).
			EncryptElement(t.Context(), doc.DocumentElement())
		require.NoError(t, err)

		nodes, err := xmlenc1.NewDecryptor().PrivateKey(key).Decrypt(t.Context(), edElem)
		require.NoError(t, err)
		require.Len(t, nodes, 1)
	})

	t.Run("Decryptor reports missing key", func(t *testing.T) {
		key := generateRSAKey(t)
		doc := mustParseXML(t, samlAssertion)
		edElem, err := xmlenc1.NewEncryptor().
			KeyTransportAlgorithm(xmlenc1.RSAOAEP).
			RecipientPublicKey(&key.PublicKey).
			EncryptElement(t.Context(), doc.DocumentElement())
		require.NoError(t, err)

		var d xmlenc1.Decryptor
		_, err = d.Decrypt(t.Context(), edElem)
		require.ErrorIs(t, err, xmlenc1.ErrMissingKey)

		_, err = d.DecryptBytes(t.Context(), edElem)
		require.ErrorIs(t, err, xmlenc1.ErrMissingKey)
	})

	t.Run("Decryptor builds on from the zero value", func(t *testing.T) {
		key := generateRSAKey(t)
		doc := mustParseXML(t, samlAssertion)
		edElem, err := xmlenc1.NewEncryptor().
			KeyTransportAlgorithm(xmlenc1.RSAOAEP).
			RecipientPublicKey(&key.PublicKey).
			EncryptElement(t.Context(), doc.DocumentElement())
		require.NoError(t, err)

		var d xmlenc1.Decryptor
		nodes, err := d.PrivateKey(key).Decrypt(t.Context(), edElem)
		require.NoError(t, err)
		require.Len(t, nodes, 1)
	})
}

// Decryption errors must say which key is missing and which setter supplies
// it. Every key kind previously failed with the same bare "no decryption key
// available", which does not tell a caller whether to set PrivateKey,
// KeyEncryptionKey, or ECPrivateKey.
func TestMissingKeyNamesTheSetter(t *testing.T) {
	t.Run("RSA key transport", func(t *testing.T) {
		key := generateRSAKey(t)
		doc := mustParseXML(t, samlAssertion)
		edElem, err := xmlenc1.NewEncryptor().
			KeyTransportAlgorithm(xmlenc1.RSAOAEP).
			RecipientPublicKey(&key.PublicKey).
			EncryptElement(t.Context(), doc.DocumentElement())
		require.NoError(t, err)

		_, err = xmlenc1.NewDecryptor().Decrypt(t.Context(), edElem)
		require.ErrorIs(t, err, xmlenc1.ErrMissingKey)
		require.Contains(t, err.Error(), "Decryptor.PrivateKey")
	})

	t.Run("AES key wrap", func(t *testing.T) {
		kek := randKey(t, 32)
		doc := mustParseXML(t, samlAssertion)
		edElem, err := xmlenc1.NewEncryptor().
			KeyWrapAlgorithm(xmlenc1.AES256KeyWrap).
			KeyEncryptionKey(kek).
			EncryptElement(t.Context(), doc.DocumentElement())
		require.NoError(t, err)

		_, err = xmlenc1.NewDecryptor().Decrypt(t.Context(), edElem)
		require.ErrorIs(t, err, xmlenc1.ErrMissingKey)
		require.Contains(t, err.Error(), "Decryptor.KeyEncryptionKey")
	})

	t.Run("ECDH-ES key agreement", func(t *testing.T) {
		key := generateECKey(t, elliptic.P256())
		doc := mustParseXML(t, samlAssertion)
		edElem, err := xmlenc1.NewEncryptor().
			KeyWrapAlgorithm(xmlenc1.AES256KeyWrap).
			RecipientECPublicKey(&key.PublicKey).
			EncryptElement(t.Context(), doc.DocumentElement())
		require.NoError(t, err)

		_, err = xmlenc1.NewDecryptor().Decrypt(t.Context(), edElem)
		require.ErrorIs(t, err, xmlenc1.ErrMissingKey)
		// The message must name the agreement algorithm the document
		// declared, exactly as the RSA and AES branches name theirs.
		require.Contains(t, err.Error(), xmlenc1.ECDHES)
		require.Contains(t, err.Error(), "Decryptor.ECPrivateKey")
	})

	t.Run("no EncryptedKey at all", func(t *testing.T) {
		doc := mustParseXML(t, `<root/>`)
		ed := &xmlenc1.EncryptedData{
			Type:             xmlenc1.TypeElement,
			EncryptionMethod: &xmlenc1.EncryptionMethod{Algorithm: xmlenc1.AES256GCM},
			CipherValue:      make([]byte, 64),
		}
		edElem, err := xmlenc1.MarshalEncryptedDataForTest(doc, ed)
		require.NoError(t, err)

		_, err = xmlenc1.NewDecryptor().Decrypt(t.Context(), edElem)
		require.ErrorIs(t, err, xmlenc1.ErrMissingKey)
		require.Contains(t, err.Error(), "Decryptor.SessionKey")
	})
}

// A failed AES key unwrap is a decryption failure and must satisfy
// errors.Is(err, ErrDecryptionFailed), exactly as a failed RSA key transport
// does. Only ErrKeyUnwrapFailed matched before, so a caller testing the
// general sentinel silently missed the key-wrap path.
func TestKeyUnwrapFailureIsDecryptionFailure(t *testing.T) {
	doc := mustParseXML(t, samlAssertion)
	edElem, err := xmlenc1.NewEncryptor().
		KeyWrapAlgorithm(xmlenc1.AES256KeyWrap).
		KeyEncryptionKey(randKey(t, 32)).
		EncryptElement(t.Context(), doc.DocumentElement())
	require.NoError(t, err)

	_, err = xmlenc1.NewDecryptor().KeyEncryptionKey(randKey(t, 32)).Decrypt(t.Context(), edElem)
	require.ErrorIs(t, err, xmlenc1.ErrDecryptionFailed)
	require.ErrorIs(t, err, xmlenc1.ErrKeyUnwrapFailed)
}

// ErrTooManyEncryptedKeys must report the counts involved and the setter that
// controls the cap, not just that some unnamed limit was hit.
func TestTooManyEncryptedKeysReportsCounts(t *testing.T) {
	doc := mustParseXML(t, `<root/>`)
	keys := make([]*xmlenc1.EncryptedKey, 3)
	for i := range keys {
		keys[i] = &xmlenc1.EncryptedKey{
			EncryptionMethod: &xmlenc1.EncryptionMethod{Algorithm: xmlenc1.RSAOAEP},
			CipherValue:      make([]byte, 256),
		}
	}
	ed := &xmlenc1.EncryptedData{
		Type:             xmlenc1.TypeElement,
		EncryptionMethod: &xmlenc1.EncryptionMethod{Algorithm: xmlenc1.AES256GCM},
		EncryptedKeys:    keys,
		CipherValue:      make([]byte, 64),
	}
	edElem, err := xmlenc1.MarshalEncryptedDataForTest(doc, ed)
	require.NoError(t, err)

	_, err = xmlenc1.NewDecryptor().MaxEncryptedKeys(2).Decrypt(t.Context(), edElem)
	require.ErrorIs(t, err, xmlenc1.ErrTooManyEncryptedKeys)
	require.Contains(t, err.Error(), "3 candidates exceed the limit of 2")
	require.Contains(t, err.Error(), "Decryptor.MaxEncryptedKeys")
}

// UnsupportedAlgorithmError must name the configuration slot that rejected the
// URI. A digest URI mistakenly passed to OAEPMGF otherwise reports only
// "unsupported algorithm", which does not say which setter is wrong.
func TestUnsupportedAlgorithmNamesTheParameter(t *testing.T) {
	key := generateRSAKey(t)
	doc := mustParseXML(t, samlAssertion)

	_, err := xmlenc1.NewEncryptor().
		KeyTransportAlgorithm(xmlenc1.RSAOAEP11).
		RecipientPublicKey(&key.PublicKey).
		OAEPMGF(xmlenc1.DigestSHA256). // a digest URI in the MGF slot
		EncryptElement(t.Context(), doc.DocumentElement())
	require.Error(t, err)

	var uae *xmlenc1.UnsupportedAlgorithmError
	require.ErrorAs(t, err, &uae)
	require.Equal(t, "MGF algorithm", uae.Parameter)
	require.Contains(t, uae.Error(), "unsupported MGF algorithm")
}

// Key transport and key wrapping both protect the same session key, and only
// one EncryptedKey is emitted. Configuring both used to silently prefer key
// transport and drop the KEK, so a recipient holding only the KEK failed to
// decrypt with an error that pointed nowhere near the real mistake.
func TestConflictingKeyProtection(t *testing.T) {
	key := generateRSAKey(t)
	kek := randKey(t, 32)

	newEncryptor := func() xmlenc1.Encryptor {
		return xmlenc1.NewEncryptor().
			KeyTransportAlgorithm(xmlenc1.RSAOAEP).
			RecipientPublicKey(&key.PublicKey).
			KeyWrapAlgorithm(xmlenc1.AES256KeyWrap).
			KeyEncryptionKey(kek)
	}

	t.Run("EncryptElement", func(t *testing.T) {
		doc := mustParseXML(t, samlAssertion)
		_, err := newEncryptor().EncryptElement(t.Context(), doc.DocumentElement())
		require.ErrorIs(t, err, xmlenc1.ErrConflictingKeyConfig)
		require.Contains(t, err.Error(), xmlenc1.RSAOAEP)
		require.Contains(t, err.Error(), xmlenc1.AES256KeyWrap)
	})

	t.Run("EncryptContent", func(t *testing.T) {
		doc := mustParseXML(t, samlAssertion)
		_, err := newEncryptor().EncryptContent(t.Context(), doc.DocumentElement())
		require.ErrorIs(t, err, xmlenc1.ErrConflictingKeyConfig)
	})

	// A session key alongside one protection mechanism is not a conflict: it
	// supplies the key that the mechanism then protects.
	t.Run("session key with key transport is allowed", func(t *testing.T) {
		doc := mustParseXML(t, samlAssertion)
		edElem, err := xmlenc1.NewEncryptor().
			BlockAlgorithm(xmlenc1.AES256GCM).
			SessionKey(randKey(t, 32)).
			KeyTransportAlgorithm(xmlenc1.RSAOAEP).
			RecipientPublicKey(&key.PublicKey).
			EncryptElement(t.Context(), doc.DocumentElement())
		require.NoError(t, err)

		nodes, err := xmlenc1.NewDecryptor().PrivateKey(key).Decrypt(t.Context(), edElem)
		require.NoError(t, err)
		require.Len(t, nodes, 1)
	})

	t.Run("session key with key wrap is allowed", func(t *testing.T) {
		doc := mustParseXML(t, samlAssertion)
		edElem, err := xmlenc1.NewEncryptor().
			BlockAlgorithm(xmlenc1.AES256GCM).
			SessionKey(randKey(t, 32)).
			KeyWrapAlgorithm(xmlenc1.AES256KeyWrap).
			KeyEncryptionKey(kek).
			EncryptElement(t.Context(), doc.DocumentElement())
		require.NoError(t, err)

		nodes, err := xmlenc1.NewDecryptor().KeyEncryptionKey(kek).Decrypt(t.Context(), edElem)
		require.NoError(t, err)
		require.Len(t, nodes, 1)
	})
}

// unserializableElement returns an element that cannot be serialized:
// Document.CreateElement only rejects a colon in the name, while the writer
// rejects the whole name at emit time.
func unserializableElement(t *testing.T) *helium.Element {
	t.Helper()
	doc := helium.NewDefaultDocument()
	elem, err := doc.CreateElement(`root injected="1"`)
	require.NoError(t, err)
	require.NoError(t, doc.SetDocumentElement(elem))
	return elem
}

// unserializableContent returns a well-named element whose only child cannot
// be serialized, so encrypting its content fails where encrypting the element
// name alone would not.
func unserializableContent(t *testing.T) *helium.Element {
	t.Helper()
	doc := helium.NewDefaultDocument()
	root, err := doc.CreateElement("root")
	require.NoError(t, err)
	require.NoError(t, doc.SetDocumentElement(root))
	child, err := doc.CreateElement(`child injected="1"`)
	require.NoError(t, err)
	require.NoError(t, root.AddChild(child))
	return root
}

// A misconfigured Encryptor must report the configuration error even when the
// payload cannot be serialized. No payload can make the configuration usable,
// so returning ErrEncryptionFailed here would point the caller away from the
// real mistake.
func TestConfigErrorPrecedesSerializationFailure(t *testing.T) {
	key := generateRSAKey(t)
	kek := randKey(t, 32)

	for _, tc := range []struct {
		name string
		enc  xmlenc1.Encryptor
		want error
	}{
		{
			name: "conflicting key protection",
			enc: xmlenc1.NewEncryptor().
				KeyTransportAlgorithm(xmlenc1.RSAOAEP).
				RecipientPublicKey(&key.PublicKey).
				KeyWrapAlgorithm(xmlenc1.AES256KeyWrap).
				KeyEncryptionKey(kek),
			want: xmlenc1.ErrConflictingKeyConfig,
		},
		{
			name: "no key source",
			enc:  xmlenc1.NewEncryptor(),
			want: xmlenc1.ErrMissingConfig,
		},
		{
			name: "CBC without opt-in",
			enc: xmlenc1.NewEncryptor().
				BlockAlgorithm(xmlenc1.AES256CBC).
				KeyWrapAlgorithm(xmlenc1.AES256KeyWrap).
				KeyEncryptionKey(kek),
			want: xmlenc1.ErrCBCEncryptionRequiresOptIn,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Run("EncryptElement", func(t *testing.T) {
				_, err := tc.enc.EncryptElement(t.Context(), unserializableElement(t))
				require.ErrorIs(t, err, tc.want)
				require.NotErrorIs(t, err, xmlenc1.ErrEncryptionFailed)
			})

			t.Run("EncryptContent", func(t *testing.T) {
				_, err := tc.enc.EncryptContent(t.Context(), unserializableContent(t))
				require.ErrorIs(t, err, tc.want)
				require.NotErrorIs(t, err, xmlenc1.ErrEncryptionFailed)
			})

			t.Run("EncryptBytes", func(t *testing.T) {
				_, err := tc.enc.EncryptBytes(t.Context(), helium.NewDefaultDocument(), []byte("payload"))
				require.ErrorIs(t, err, tc.want)
			})
		})
	}
}
