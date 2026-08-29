package xmlenc1_test

import (
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"runtime"
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

func TestDecryptCopiedDeclaredEntityCipherValue(t *testing.T) {
	t.Parallel()

	sessionKey := []byte("0123456789abcdef0123456789abcdef")
	doc := mustParseXML(t, `<secret>copy-safe</secret>`)
	_, err := xmlenc1.NewEncryptor().
		BlockAlgorithm(xmlenc1.AES256GCM).
		SessionKey(sessionKey).
		EncryptElement(t.Context(), doc.DocumentElement())
	require.NoError(t, err)

	encryptedXML, err := helium.WriteString(doc)
	require.NoError(t, err)
	const cipherOpen = `<xenc:CipherValue>`
	cipherStart := strings.Index(encryptedXML, cipherOpen)
	require.NotEqual(t, -1, cipherStart)
	cipherStart += len(cipherOpen)
	require.GreaterOrEqual(t, len(encryptedXML)-cipherStart, 4)
	quantum := encryptedXML[cipherStart : cipherStart+4]
	encryptedXML = encryptedXML[:cipherStart] + `&quantum;` + encryptedXML[cipherStart+4:]
	rootStart := strings.Index(encryptedXML, `<xenc:EncryptedData`)
	require.NotEqual(t, -1, rootStart)
	encryptedXML = encryptedXML[:rootStart] +
		`<!DOCTYPE xenc:EncryptedData [<!ENTITY quantum "` + quantum + `">]>` +
		encryptedXML[rootStart:]

	parsed := mustParseXML(t, encryptedXML)
	copied, err := helium.CopyDoc(parsed)
	require.NoError(t, err)

	for name, candidate := range map[string]*helium.Document{"original": parsed, "copy": copied} {
		t.Run(name, func(t *testing.T) {
			encryptedData := findEncryptedData(t, candidate)
			require.NotNil(t, encryptedData)
			nodes, decryptErr := xmlenc1.NewDecryptor().
				SessionKey(sessionKey).
				Decrypt(t.Context(), encryptedData)
			require.NoError(t, decryptErr)
			require.Len(t, nodes, 1)
			plaintext, writeErr := helium.WriteString(nodes[0])
			require.NoError(t, writeErr)
			require.Equal(t, `<secret>copy-safe</secret>`, plaintext)
		})
	}
}

// The default block algorithm is what a caller who configures nothing puts on
// the wire, so it must be an identifier a standards-conforming peer
// recognizes. W3C xmlenc-core1 §5.2 defines AES-GCM only in the XML
// Encryption 1.1 namespace, and the identifiers in the 2001 namespace this
// package also decrypts are defined by no specification.
func TestDefaultBlockAlgorithm(t *testing.T) {
	t.Run("emits the XML Encryption 1.1 GCM identifier", func(t *testing.T) {
		require.Equal(t, xmlenc1.AES256GCM11, xmlenc1.DefaultBlockAlgorithm)

		sessionKey := randKey(t, 32)
		doc := mustParseXML(t, samlAssertion)

		// No BlockAlgorithm set.
		edElem, err := xmlenc1.NewEncryptor().
			SessionKey(sessionKey).
			EncryptElement(t.Context(), doc.DocumentElement())
		require.NoError(t, err)

		ed, err := xmlenc1.ParseEncryptedDataForTest(edElem)
		require.NoError(t, err)
		require.NotNil(t, ed.EncryptionMethod)
		require.Equal(t, xmlenc1.AES256GCM11, ed.EncryptionMethod.Algorithm)
	})

	// A caller may still select a 2001-namespace GCM identifier explicitly,
	// and this package decrypts one whatever wrote it, so every document it
	// has ever emitted keeps decrypting.
	t.Run("explicitly selected AES256GCM round-trips", func(t *testing.T) {
		sessionKey := randKey(t, 32)
		doc := mustParseXML(t, samlAssertion)

		edElem, err := xmlenc1.NewEncryptor().
			BlockAlgorithm(xmlenc1.AES256GCM).
			SessionKey(sessionKey).
			EncryptElement(t.Context(), doc.DocumentElement())
		require.NoError(t, err)

		ed, err := xmlenc1.ParseEncryptedDataForTest(edElem)
		require.NoError(t, err)
		require.NotNil(t, ed.EncryptionMethod)
		require.Equal(t, xmlenc1.AES256GCM, ed.EncryptionMethod.Algorithm)

		nodes, err := xmlenc1.NewDecryptor().
			SessionKey(sessionKey).
			Decrypt(t.Context(), edElem)
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
		// the wrong-key (RSA) path, well past the CBC opt-in gate.
		decryptor := xmlenc1.NewDecryptor().PrivateKey(key2).AllowUnauthenticatedCBC(true)
		_, err = decryptor.Decrypt(t.Context(), edElem)
		require.Error(t, err)
	})
}

func TestEncryptNilElement(t *testing.T) {
	encryptor := xmlenc1.NewEncryptor().
		BlockAlgorithm(xmlenc1.AES256GCM).
		SessionKey(randKey(t, 32))

	for _, encrypt := range []struct {
		name string
		fn   func() (*helium.Element, error)
	}{
		{
			name: "EncryptElement",
			fn: func() (*helium.Element, error) {
				return encryptor.EncryptElement(t.Context(), nil)
			},
		},
		{
			name: "EncryptContent",
			fn: func() (*helium.Element, error) {
				return encryptor.EncryptContent(t.Context(), nil)
			},
		},
	} {
		t.Run(encrypt.name, func(t *testing.T) {
			_, err := encrypt.fn()
			require.EqualError(t, err, "xmlenc1: encryption failed: nil node")
			require.ErrorIs(t, err, xmlenc1.ErrEncryptionFailed)
		})
	}
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

	// Every encrypt entry point reports a nil operand the same way: the
	// operation sentinel first, so a caller matching ErrEncryptionFailed
	// catches it whichever terminal it called, and helium.ErrNilNode under it
	// so a caller that wants to know WHICH operand still can.
	t.Run("reports a missing document", func(t *testing.T) {
		_, err := xmlenc1.NewEncryptor().
			SessionKey(randKey(t, 32)).
			EncryptBytes(t.Context(), nil, payload)
		require.ErrorIs(t, err, xmlenc1.ErrEncryptionFailed)
		require.ErrorIs(t, err, helium.ErrNilNode)
		require.NotErrorIs(t, err, xmlenc1.ErrMissingConfig)
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
		// The message must name the mechanism and the key-transport algorithm
		// the document declared, not only the setter that supplies the key.
		require.Contains(t, err.Error(), "RSA key transport")
		require.Contains(t, err.Error(), xmlenc1.RSAOAEP)
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
		// The message must name the mechanism and the key-wrap algorithm the
		// document declared, not only the setter that supplies the key.
		require.Contains(t, err.Error(), "AES key wrap")
		require.Contains(t, err.Error(), xmlenc1.AES256KeyWrap)
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
		require.Contains(t, err.Error(), "key agreement")
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

// The configuration errors decided before serialization — including algorithm
// validation, key-protection conflicts, missing key source, and CBC opt-in —
// must reach the caller even when the payload cannot be serialized. No payload
// can make the configuration usable, so returning a serialization error here
// would point the caller away from the real mistake.
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

	t.Run("unsupported block algorithm", func(t *testing.T) {
		_, err := xmlenc1.NewEncryptor().
			BlockAlgorithm("urn:example:unsupported-block").
			SessionKey(randKey(t, 32)).
			EncryptElement(t.Context(), unserializableElement(t))
		require.Error(t, err)
		var unsupported *xmlenc1.UnsupportedAlgorithmError
		require.ErrorAs(t, err, &unsupported)
		require.Equal(t, "block algorithm", unsupported.Parameter)
		require.Equal(t, "urn:example:unsupported-block", unsupported.Algorithm)
		require.ErrorIs(t, err, xmlenc1.ErrEncryptionFailed)
		require.NotContains(t, err.Error(), "invalid element name")
	})

	t.Run("unsupported key transport algorithm", func(t *testing.T) {
		_, err := xmlenc1.NewEncryptor().
			KeyTransportAlgorithm("urn:example:unsupported-key-transport").
			RecipientPublicKey(&key.PublicKey).
			EncryptElement(t.Context(), unserializableElement(t))
		require.ErrorIs(t, err, xmlenc1.ErrEncryptionFailed)
		var unsupported *xmlenc1.UnsupportedAlgorithmError
		require.ErrorAs(t, err, &unsupported)
		require.Equal(t, "key transport algorithm", unsupported.Parameter)
		require.Equal(t, "urn:example:unsupported-key-transport", unsupported.Algorithm)
		require.NotContains(t, err.Error(), "invalid element name")
	})
}

// encryptAllocPayload is the plaintext the allocation cases below hand to
// EncryptBytes, and encryptAllocSlack is what a rejected configuration may
// allocate on top of nothing.
//
// The slack is a CONSTANT, deliberately: a bound stated as a fraction of the
// payload can never fail on a cost proportional to that payload, which is the
// only cost being pinned. It covers resolving the configuration, building the
// error, and the runtime's own bookkeeping, all of which are fixed.
const (
	encryptAllocPayload = 4 << 20
	encryptAllocSlack   = 1 << 20
)

// encryptBytesAlloc reports the TotalAlloc delta across one EncryptBytes of
// plaintext, together with its verdict. The owning document and the plaintext
// are built outside the measurement, so the delta is the call alone.
func encryptBytesAlloc(t *testing.T, enc xmlenc1.Encryptor, plaintext []byte) (uint64, error) {
	t.Helper()
	doc := helium.NewDefaultDocument()

	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	_, err := enc.EncryptBytes(t.Context(), doc, plaintext)
	runtime.ReadMemStats(&after)

	return after.TotalAlloc - before.TotalAlloc, err
}

// TestEncryptConfigRejectionAllocatesNoPayload pins what a configuration
// resolveEncryptConfig refuses may ALLOCATE, which the error assertions
// elsewhere cannot see. A setting that can never produce a readable document is
// worth nothing proportional to the plaintext, and each accepted case measured
// beside it shows the payload work the rejected ones must not reach: the
// accepted bound is the floor a single copy of the plaintext already exceeds.
//
// Each case reads the process-wide TotalAlloc delta across EncryptBytes, so
// these subtests must NOT run in parallel: a concurrent test's allocations
// would pollute the delta.
func TestEncryptConfigRejectionAllocatesNoPayload(t *testing.T) {
	// no t.Parallel(): isolated so each delta reflects only its own call.
	ecKey := generateECKey(t, elliptic.P256())
	kek := randKey(t, 32)
	plaintext := make([]byte, encryptAllocPayload)

	for _, tc := range []struct {
		name string
		enc  xmlenc1.Encryptor
		// accepted marks the baselines: they encrypt the payload, so they cost
		// at least one copy of it.
		accepted bool
	}{
		{
			name: "key agreement with a non-key-wrap algorithm",
			enc: xmlenc1.NewEncryptor().
				KeyWrapAlgorithm(xmlenc1.AES256GCM).
				RecipientECPublicKey(&ecKey.PublicKey),
		},
		{
			name: "key agreement with oversized OtherInfo",
			enc: xmlenc1.NewEncryptor().
				KeyWrapAlgorithm(xmlenc1.AES256KeyWrap).
				RecipientECPublicKey(&ecKey.PublicKey).
				KeyDerivationParams(&xmlenc1.ConcatKDFParams{
					DigestMethod: xmlenc1.DigestSHA256,
					AlgorithmID:  make([]byte, xmlenc1.MaxConcatKDFOtherInfoBytesForTest+1),
				}),
		},
		{
			name: "key agreement with an unsupported KDF digest",
			enc: xmlenc1.NewEncryptor().
				KeyWrapAlgorithm(xmlenc1.AES256KeyWrap).
				RecipientECPublicKey(&ecKey.PublicKey).
				KeyDerivationParams(&xmlenc1.ConcatKDFParams{DigestMethod: "urn:example:nope"}),
		},
		{
			name: "key wrap with a non-key-wrap algorithm",
			enc: xmlenc1.NewEncryptor().
				KeyWrapAlgorithm(xmlenc1.AES256GCM).
				KeyEncryptionKey(kek),
		},
		{
			name: "key agreement",
			enc: xmlenc1.NewEncryptor().
				KeyWrapAlgorithm(xmlenc1.AES256KeyWrap).
				RecipientECPublicKey(&ecKey.PublicKey),
			accepted: true,
		},
		{
			name: "key wrap",
			enc: xmlenc1.NewEncryptor().
				KeyWrapAlgorithm(xmlenc1.AES256KeyWrap).
				KeyEncryptionKey(kek),
			accepted: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			allocated, err := encryptBytesAlloc(t, tc.enc, plaintext)

			if tc.accepted {
				require.NoError(t, err)
				require.Greater(t, allocated, uint64(encryptAllocPayload),
					"encrypting %d bytes allocated only %d bytes", len(plaintext), allocated)
				return
			}

			require.ErrorIs(t, err, xmlenc1.ErrEncryptionFailed)
			require.Less(t, allocated, uint64(encryptAllocSlack),
				"a rejected configuration allocated %d bytes for a %d byte payload", allocated, len(plaintext))
		})
	}
}
