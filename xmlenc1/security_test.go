package xmlenc1_test

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"os"
	"strings"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xmlenc1"
	"github.com/stretchr/testify/require"
)

func TestDecryptCBC(t *testing.T) {
	// H2 Flaw 1: CBC is unauthenticated; require explicit opt-in to decrypt.
	//
	// Default Decryptor should refuse to decrypt AES-CBC ciphertext with
	// ErrCBCRequiresOptIn, since unauthenticated CBC is vulnerable to padding
	// oracle attacks (Jager/Somorovsky 2011).
	t.Run("default denied", func(t *testing.T) {
		sessionKey := make([]byte, 16)
		_, err := rand.Read(sessionKey)
		require.NoError(t, err)

		doc := mustParseXML(t, samlAssertion)

		encryptor := xmlenc1.NewEncryptor().
			BlockAlgorithm(xmlenc1.AES128CBC).
			AllowLegacyCBC(true).
			SessionKey(sessionKey)
		edElem, err := encryptor.EncryptElement(t.Context(), doc.DocumentElement())
		require.NoError(t, err)

		decryptor := xmlenc1.NewDecryptor().SessionKey(sessionKey)
		_, err = decryptor.Decrypt(t.Context(), edElem)
		require.Error(t, err)
		require.ErrorIs(t, err, xmlenc1.ErrCBCRequiresOptIn)
	})

	// H2 Flaw 1: explicit opt-in must allow CBC decryption.
	t.Run("opt-in allowed", func(t *testing.T) {
		sessionKey := make([]byte, 16)
		_, err := rand.Read(sessionKey)
		require.NoError(t, err)

		doc := mustParseXML(t, samlAssertion)

		encryptor := xmlenc1.NewEncryptor().
			BlockAlgorithm(xmlenc1.AES128CBC).
			AllowLegacyCBC(true).
			SessionKey(sessionKey)
		edElem, err := encryptor.EncryptElement(t.Context(), doc.DocumentElement())
		require.NoError(t, err)

		decryptor := xmlenc1.NewDecryptor().
			SessionKey(sessionKey).
			AllowUnauthenticatedCBC(true)
		nodes, err := decryptor.Decrypt(t.Context(), edElem)
		require.NoError(t, err)
		require.Len(t, nodes, 1)

		s, err := helium.WriteString(nodes[0])
		require.NoError(t, err)
		require.Contains(t, s, "user@example.com")
	})

	// H2 Flaw 4: padding oracle hardening, pinned over a fixed table instead
	// of two ad hoc bit flips. Every one of nine distinct causes — a
	// corrupted ciphertext, or one of eight different ways the recovered
	// plaintext fails to parse or fails to match its declared Type — must
	// report the exact same error string at the caller-visible boundary, so
	// none of them can be told apart. The table runs each cause under both
	// AES-CBC and AES-GCM, and under both Type=Element and Type=Content
	// wherever that combination can fail at all: a shape violation like
	// "two elements" only rejects Type=Element (Type=Content accepts
	// multiple nodes), so those rows are skipped for Type=Content rather
	// than asserted to fail.
	t.Run("failure modes report the identical error", func(t *testing.T) {
		sessionKey := randKey(t, 32) // AES-256: valid for both algorithms below

		deepNesting := strings.Repeat("<a>", 300) + "x" + strings.Repeat("</a>", 300)

		modes := []struct {
			name string
			// plaintext is encrypted normally to produce the CipherValue.
			// Ignored when cipherLevel is set.
			plaintext string
			// cipherLevel builds a CipherValue that fails inside the block
			// cipher itself, before any plaintext is parsed, instead of
			// encrypting plaintext.
			cipherLevel bool
			// elementOnly marks a mode that only Type=Element rejects; a
			// Type=Content decrypt of the same CipherValue succeeds and is
			// not exercised.
			elementOnly bool
		}{
			{name: "bad padding", cipherLevel: true},
			{name: "unparseable XML", plaintext: "<foo>"},
			{name: "two elements", plaintext: "<a/><b/>", elementOnly: true},
			{name: "zero nodes", plaintext: "", elementOnly: true},
			{name: "text-only", plaintext: "just text", elementOnly: true},
			{name: "comment-only", plaintext: "<!-- just a comment -->", elementOnly: true},
			{name: "depth limit", plaintext: deepNesting},
			{name: "undeclared prefix", plaintext: "<x:foo/>"},
			{name: "DOCTYPE", plaintext: "<!DOCTYPE foo><foo/>"},
		}

		algorithms := []struct {
			uri   string
			label string
		}{
			{xmlenc1.AES256CBC, "CBC"},
			{xmlenc1.AES256GCM, "GCM"},
		}
		types := []struct {
			uri   string
			label string
		}{
			{xmlenc1.TypeElement, "Element"},
			{xmlenc1.TypeContent, "Content"},
		}

		want := xmlenc1.ErrDecryptionFailed.Error()

		for _, mode := range modes {
			for _, alg := range algorithms {
				cipherValue := corruptedBlockCipherValue(t, alg.uri, sessionKey)
				if !mode.cipherLevel {
					var err error
					cipherValue, err = xmlenc1.EncryptBytesForTest(alg.uri, sessionKey, []byte(mode.plaintext))
					require.NoError(t, err)
				}

				for _, typ := range types {
					if mode.elementOnly && typ.uri == xmlenc1.TypeContent {
						continue
					}

					t.Run(mode.name+"/"+alg.label+"/"+typ.label, func(t *testing.T) {
						doc := mustParseXML(t, `<root/>`)
						ed := &xmlenc1.EncryptedData{
							Type:             typ.uri,
							EncryptionMethod: &xmlenc1.EncryptionMethod{Algorithm: alg.uri},
							CipherValue:      cipherValue,
						}
						edElem, err := xmlenc1.MarshalEncryptedDataForTest(doc, ed)
						require.NoError(t, err)

						decryptor := xmlenc1.NewDecryptor().
							SessionKey(sessionKey).
							AllowUnauthenticatedCBC(true)
						_, err = decryptor.Decrypt(t.Context(), edElem)
						require.Error(t, err)
						require.ErrorIs(t, err, xmlenc1.ErrDecryptionFailed)
						require.Equal(t, want, err.Error(),
							"mode %q under %s/%s must report the same error as every other mode", mode.name, alg.label, typ.label)
					})
				}
			}
		}
	})
}

// corruptedBlockCipherValue returns a CipherValue that fails inside the
// block cipher itself, before any plaintext is ever parsed, for the given
// algorithm and key. Under AES-CBC it decrypts (with an all-zero key-derived
// keystream block XORed against an all-zero IV) to a block whose last byte
// is 0 — a PKCS#7 padding length of 0 is always invalid, so this reliably
// fails padding without depending on crypto/rand. AES-GCM has no padding to
// corrupt, so its analogous pre-parse cipher-level failure is a CipherValue
// too short to hold a nonce and authentication tag.
func corruptedBlockCipherValue(t *testing.T, algorithm string, key []byte) []byte {
	t.Helper()
	switch algorithm {
	case xmlenc1.AES256CBC:
		block, err := aes.NewCipher(key)
		require.NoError(t, err)
		raw := make([]byte, aes.BlockSize) // all-zero plaintext block
		iv := make([]byte, aes.BlockSize)  // all-zero IV: deterministic, test-only
		out := make([]byte, aes.BlockSize+len(raw))
		cipher.NewCBCEncrypter(block, iv).CryptBlocks(out[aes.BlockSize:], raw)
		return out
	case xmlenc1.AES256GCM:
		return nil // shorter than any nonce + authentication tag
	default:
		t.Fatalf("corruptedBlockCipherValue: unsupported algorithm %q", algorithm)
		return nil
	}
}

func TestEncryptCBC(t *testing.T) {
	// D-ENC-003: selecting a CBC BlockAlgorithm without AllowLegacyCBC must
	// be refused with ErrCBCEncryptionRequiresOptIn and emit no ciphertext.
	t.Run("default denied", func(t *testing.T) {
		for _, alg := range []string{xmlenc1.AES128CBC, xmlenc1.AES256CBC} {
			sessionKey := make([]byte, 16)
			if alg == xmlenc1.AES256CBC {
				sessionKey = make([]byte, 32)
			}
			_, err := rand.Read(sessionKey)
			require.NoError(t, err)

			doc := mustParseXML(t, samlAssertion)
			encryptor := xmlenc1.NewEncryptor().
				BlockAlgorithm(alg).
				SessionKey(sessionKey)
			_, err = encryptor.EncryptElement(t.Context(), doc.DocumentElement())
			require.Error(t, err)
			require.ErrorIs(t, err, xmlenc1.ErrCBCEncryptionRequiresOptIn)

			// Nothing should have been serialized into the tree.
			xml, werr := helium.WriteString(doc)
			require.NoError(t, werr)
			require.NotContains(t, xml, elemEncryptedData)
			require.Contains(t, xml, "user@example.com")
		}
	})

	// D-ENC-003: with AllowLegacyCBC(true), CBC encryption works (legacy
	// interop) and round-trips against a CBC-opted-in Decryptor.
	t.Run("opt-in allowed", func(t *testing.T) {
		sessionKey := make([]byte, 16)
		_, err := rand.Read(sessionKey)
		require.NoError(t, err)

		doc := mustParseXML(t, samlAssertion)
		encryptor := xmlenc1.NewEncryptor().
			BlockAlgorithm(xmlenc1.AES128CBC).
			AllowLegacyCBC(true).
			SessionKey(sessionKey)
		edElem, err := encryptor.EncryptElement(t.Context(), doc.DocumentElement())
		require.NoError(t, err)

		xml, err := helium.WriteString(doc)
		require.NoError(t, err)
		require.Contains(t, xml, xmlenc1.AES128CBC)

		decryptor := xmlenc1.NewDecryptor().
			SessionKey(sessionKey).
			AllowUnauthenticatedCBC(true)
		nodes, err := decryptor.Decrypt(t.Context(), edElem)
		require.NoError(t, err)
		require.Len(t, nodes, 1)
		s, err := helium.WriteString(nodes[0])
		require.NoError(t, err)
		require.Contains(t, s, "user@example.com")
	})
}

func TestGCM(t *testing.T) {
	// H2 Flaw 3: GCM round-trip with algorithm URI bound as AAD must succeed.
	t.Run("round-trip with AAD", func(t *testing.T) {
		sessionKey := make([]byte, 32)
		_, err := rand.Read(sessionKey)
		require.NoError(t, err)

		doc := mustParseXML(t, samlAssertion)
		encryptor := xmlenc1.NewEncryptor().
			BlockAlgorithm(xmlenc1.AES256GCM).
			SessionKey(sessionKey)
		edElem, err := encryptor.EncryptElement(t.Context(), doc.DocumentElement())
		require.NoError(t, err)

		decryptor := xmlenc1.NewDecryptor().SessionKey(sessionKey)
		nodes, err := decryptor.Decrypt(t.Context(), edElem)
		require.NoError(t, err)
		require.Len(t, nodes, 1)
	})

	// H2 Flaw 3: swapping the EncryptionMethod/@Algorithm URI between encrypt
	// and decrypt must cause AAD verification to fail.
	t.Run("algorithm swap fails", func(t *testing.T) {
		// Same key length on both sides (128 bits) so AES-128-GCM works
		// at the cipher level; the AAD binding must still reject the swap.
		sessionKey := make([]byte, 16)
		_, err := rand.Read(sessionKey)
		require.NoError(t, err)

		// Encrypt the raw plaintext under AES-128-GCM with a known AAD
		// (the algorithm URI). Then assemble an EncryptedData whose
		// EncryptionMethod/@Algorithm is a *different* GCM URI of the
		// same key size... wait, there is no other 128-bit GCM URI in
		// xmlenc. Instead, encrypt under AES-128-GCM and then mutate the
		// EncryptedData to claim AES-256-GCM; the decryptor must refuse
		// (either at key-size validation or AAD verification — both are
		// correct failure modes).
		algorithm := xmlenc1.AES128GCM
		plaintext := []byte("<x>secret</x>")
		cipher, err := xmlenc1.EncryptBytesForTest(algorithm, sessionKey, plaintext)
		require.NoError(t, err)

		doc := mustParseXML(t, `<root/>`)
		ed := &xmlenc1.EncryptedData{
			Type:             xmlenc1.TypeElement,
			EncryptionMethod: &xmlenc1.EncryptionMethod{Algorithm: algorithm},
			CipherValue:      cipher,
		}
		edElem, err := xmlenc1.MarshalEncryptedDataForTest(doc, ed)
		require.NoError(t, err)

		// Swap the on-the-wire Algorithm attribute to a different URI.
		swapEncryptionMethodAlgorithm(t, edElem, xmlenc1.AES256GCM)

		decryptor := xmlenc1.NewDecryptor().SessionKey(sessionKey)
		_, err = decryptor.Decrypt(t.Context(), edElem)
		require.Error(t, err)
	})

	// D-ENC-003: the Encryptor must default to authenticated AES-GCM. A
	// caller that sets no BlockAlgorithm gets AES-256-GCM, and the emitted
	// EncryptedData advertises a GCM URI (never an unauthenticated CBC URI).
	t.Run("defaults to GCM", func(t *testing.T) {
		sessionKey := make([]byte, 32) // AES-256 session key
		_, err := rand.Read(sessionKey)
		require.NoError(t, err)

		doc := mustParseXML(t, samlAssertion)

		// No BlockAlgorithm set.
		encryptor := xmlenc1.NewEncryptor().SessionKey(sessionKey)
		edElem, err := encryptor.EncryptElement(t.Context(), doc.DocumentElement())
		require.NoError(t, err)

		xml, err := helium.WriteString(doc)
		require.NoError(t, err)
		require.Contains(t, xml, xmlenc1.AES256GCM11, "default block algorithm must be AES-256-GCM")
		// The serialized XML embeds a random base64 CipherValue whose alphabet
		// (A-Za-z0-9+/=) can by chance spell "cbc"; assert against the actual CBC
		// algorithm URIs instead, which contain a hyphen base64 never produces.
		require.NotContains(t, xml, "-cbc", "default must never emit a CBC algorithm URI")
		require.NotContains(t, xml, xmlenc1.AES128CBC, "default must never emit a CBC algorithm URI")
		require.NotContains(t, xml, xmlenc1.AES256CBC, "default must never emit a CBC algorithm URI")

		// A default Decryptor (no CBC opt-in) must round-trip GCM ciphertext.
		decryptor := xmlenc1.NewDecryptor().SessionKey(sessionKey)
		nodes, err := decryptor.Decrypt(t.Context(), edElem)
		require.NoError(t, err)
		require.Len(t, nodes, 1)
		s, err := helium.WriteString(nodes[0])
		require.NoError(t, err)
		require.Contains(t, s, "user@example.com")
	})
}

func TestXXE(t *testing.T) {
	// H2 Flaw 2: a hardened inner parser must not resolve external entities
	// declared in the decrypted plaintext.
	//
	// We point the entity at a sentinel file we control, then check that
	// the file's contents do NOT appear in the parser output. (The DOCTYPE
	// SYSTEM identifier itself may be echoed in the serialization — that
	// is harmless, we care that the referenced file was not fetched.)
	t.Run("hardened inner parser blocks XXE", func(t *testing.T) {
		sentinel := t.TempDir() + "/secret.txt"
		require.NoError(t, os.WriteFile(sentinel, []byte("XXE_LEAKED_SECRET"), 0o600))

		xxePlain := `<!DOCTYPE foo [<!ENTITY x SYSTEM "file://` + sentinel + `">]><foo>&x;</foo>`
		innerDoc, err := xmlenc1.HardenedParserForTest().Parse(t.Context(), []byte(xxePlain))
		if err == nil {
			out, werr := helium.WriteString(innerDoc)
			require.NoError(t, werr)
			require.NotContains(t, out, "XXE_LEAKED_SECRET",
				"external entity was resolved: %s", out)
		}
	})

	// H2 Flaw 2: end-to-end test that a decrypted XXE payload is parsed by
	// the hardened inner parser and does not load the external entity.
	t.Run("decrypt XXE not resolved", func(t *testing.T) {
		sentinel := t.TempDir() + "/secret.txt"
		require.NoError(t, os.WriteFile(sentinel, []byte("XXE_LEAKED_SECRET"), 0o600))

		sessionKey := make([]byte, 32)
		_, err := rand.Read(sessionKey)
		require.NoError(t, err)

		algorithm := xmlenc1.AES256GCM
		xxePlain := []byte(`<!DOCTYPE foo [<!ENTITY x SYSTEM "file://` + sentinel + `">]><foo>&x;</foo>`)
		cipher, err := xmlenc1.EncryptBytesForTest(algorithm, sessionKey, xxePlain)
		require.NoError(t, err)

		doc := mustParseXML(t, `<root/>`)
		ed := &xmlenc1.EncryptedData{
			Type:             xmlenc1.TypeElement,
			EncryptionMethod: &xmlenc1.EncryptionMethod{Algorithm: algorithm},
			CipherValue:      cipher,
		}
		edElem, err := xmlenc1.MarshalEncryptedDataForTest(doc, ed)
		require.NoError(t, err)

		decryptor := xmlenc1.NewDecryptor().SessionKey(sessionKey)
		nodes, err := decryptor.Decrypt(t.Context(), edElem)
		// Parsing may succeed (with &x; unresolved) or fail; either is
		// acceptable as long as the external entity is not loaded.
		if err == nil {
			require.NotEmpty(t, nodes)
			for _, n := range nodes {
				s, werr := helium.WriteString(n)
				require.NoError(t, werr)
				require.NotContains(t, s, "XXE_LEAKED_SECRET",
					"external entity was resolved: %s", s)
			}
		}
	})
}

// TestDecryptBoundaryRegression runs a table of attacker-shaped documents
// against one fixed key configuration (one RSA key pair and one KEK,
// configured once and reused for every row) and requires that every one of
// them fails through a sentinel Decrypt already documents: ErrDecryptionFailed,
// ErrMalformedEncrypted, ErrMissingKey, or one of the budget sentinels. A
// caller written to treat anything outside that set as an internal fault
// worth logging verbatim must never see one from a document it did not write
// itself.
//
// Two rows regress the fix in this package and are confirmed to fail before
// it: a too-short session key wrapped under the recipient's own RSA public
// key (attacker-controlled, since the public key is public), and a document
// declaring kw-aes128 while the recipient's configured KeyEncryptionKey is 32
// bytes (recipient-configured, so a distinguishable error would disclose its
// length). Both used to surface a bare *KeySizeError, which satisfies none of
// the four sentinels above.
func TestDecryptBoundaryRegression(t *testing.T) {
	rsaKey := generateRSAKey(t)
	kek := randKey(t, 32)

	decryptor := xmlenc1.NewDecryptor().
		PrivateKey(rsaKey).
		KeyEncryptionKey(kek).
		AllowUnauthenticatedCBC(true)

	for _, tc := range []struct {
		name  string
		build func(t *testing.T) *helium.Element
	}{
		{
			// Regression: RSA key transport delivers whatever length the
			// wrapped ciphertext decrypts to, and RSA-OAEP places no
			// constraint on that length. The declared data algorithm
			// (AES-256-GCM) needs 32 bytes; this transports 16.
			name: "wrong-length session key under RSA key transport",
			build: func(t *testing.T) *helium.Element {
				shortSessionKey := randKey(t, 16)
				transported := rsaTransportedKey(t, rsaKey, shortSessionKey)
				cipher, err := xmlenc1.EncryptBytesForTest(xmlenc1.AES128GCM, shortSessionKey, []byte("<x>secret</x>"))
				require.NoError(t, err)

				doc := mustParseXML(t, `<root/>`)
				ed := &xmlenc1.EncryptedData{
					Type:             xmlenc1.TypeElement,
					EncryptionMethod: &xmlenc1.EncryptionMethod{Algorithm: xmlenc1.AES256GCM},
					EncryptedKeys: []*xmlenc1.EncryptedKey{{
						EncryptionMethod: &xmlenc1.EncryptionMethod{Algorithm: xmlenc1.RSAOAEP},
						CipherValue:      transported,
					}},
					CipherValue: cipher,
				}
				elem, err := xmlenc1.MarshalEncryptedDataForTest(doc, ed)
				require.NoError(t, err)
				return elem
			},
		},
		{
			// Regression: the document picks the key-wrap algorithm; the
			// KEK length that algorithm is checked against belongs to the
			// recipient, not the document.
			name: "kw-aes128 declared against a 32-byte KEK",
			build: func(t *testing.T) *helium.Element {
				doc := mustParseXML(t, `<root/>`)
				ed := &xmlenc1.EncryptedData{
					Type:             xmlenc1.TypeElement,
					EncryptionMethod: &xmlenc1.EncryptionMethod{Algorithm: xmlenc1.AES256GCM},
					EncryptedKeys: []*xmlenc1.EncryptedKey{{
						EncryptionMethod: &xmlenc1.EncryptionMethod{Algorithm: xmlenc1.AES128KeyWrap},
						CipherValue:      randKey(t, 24), // never reached: rejected before unwrap
					}},
					CipherValue: make([]byte, 64),
				}
				elem, err := xmlenc1.MarshalEncryptedDataForTest(doc, ed)
				require.NoError(t, err)
				return elem
			},
		},
		{
			name: "no EncryptedKey at all",
			build: func(t *testing.T) *helium.Element {
				doc := mustParseXML(t, `<root/>`)
				ed := &xmlenc1.EncryptedData{
					Type:             xmlenc1.TypeElement,
					EncryptionMethod: &xmlenc1.EncryptionMethod{Algorithm: xmlenc1.AES256GCM},
					CipherValue:      make([]byte, 64),
				}
				elem, err := xmlenc1.MarshalEncryptedDataForTest(doc, ed)
				require.NoError(t, err)
				return elem
			},
		},
		{
			name: "unsupported EncryptedData Type",
			build: func(t *testing.T) *helium.Element {
				doc := mustParseXML(t, `<root/>`)
				ed := &xmlenc1.EncryptedData{
					Type:             "urn:example:bogus-type",
					EncryptionMethod: &xmlenc1.EncryptionMethod{Algorithm: xmlenc1.AES256GCM},
					EncryptedKeys: []*xmlenc1.EncryptedKey{{
						EncryptionMethod: &xmlenc1.EncryptionMethod{Algorithm: xmlenc1.RSAOAEP},
						CipherValue:      make([]byte, 256),
					}},
					CipherValue: make([]byte, 64),
				}
				elem, err := xmlenc1.MarshalEncryptedDataForTest(doc, ed)
				require.NoError(t, err)
				return elem
			},
		},
		{
			name: "unsupported block algorithm",
			build: func(t *testing.T) *helium.Element {
				doc := mustParseXML(t, `<root/>`)
				ed := &xmlenc1.EncryptedData{
					Type:             xmlenc1.TypeElement,
					EncryptionMethod: &xmlenc1.EncryptionMethod{Algorithm: "urn:example:unsupported-block"},
					EncryptedKeys: []*xmlenc1.EncryptedKey{{
						EncryptionMethod: &xmlenc1.EncryptionMethod{Algorithm: xmlenc1.RSAOAEP},
						CipherValue:      make([]byte, 256),
					}},
					CipherValue: make([]byte, 64),
				}
				elem, err := xmlenc1.MarshalEncryptedDataForTest(doc, ed)
				require.NoError(t, err)
				return elem
			},
		},
		{
			// Budget sentinel: more EncryptedKey candidates than the
			// Decryptor's default cap allows.
			name: "too many EncryptedKey candidates",
			build: func(t *testing.T) *helium.Element {
				doc := mustParseXML(t, `<root/>`)
				keys := make([]*xmlenc1.EncryptedKey, xmlenc1.DefaultMaxEncryptedKeys+1)
				for i := range keys {
					keys[i] = &xmlenc1.EncryptedKey{
						EncryptionMethod: &xmlenc1.EncryptionMethod{Algorithm: xmlenc1.RSAOAEP},
						CipherValue:      make([]byte, 8),
					}
				}
				ed := &xmlenc1.EncryptedData{
					Type:             xmlenc1.TypeElement,
					EncryptionMethod: &xmlenc1.EncryptionMethod{Algorithm: xmlenc1.AES256GCM},
					EncryptedKeys:    keys,
					CipherValue:      make([]byte, 64),
				}
				elem, err := xmlenc1.MarshalEncryptedDataForTest(doc, ed)
				require.NoError(t, err)
				return elem
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := decryptor.Decrypt(t.Context(), tc.build(t))
			require.Error(t, err)
			require.True(t, isDocumentedDecryptFailure(err),
				"error matches none of the documented decrypt-failure sentinels: %v", err)
		})
	}
}

// isDocumentedDecryptFailure reports whether err matches one of the
// sentinels Decrypt/DecryptBytes document as possible failures: a
// decryption failure, a malformed EncryptedData, a payload whose @Type
// declares no XML content, a missing key, or one of the byte/count budget
// sentinels.
func isDocumentedDecryptFailure(err error) bool {
	return errors.Is(err, xmlenc1.ErrDecryptionFailed) ||
		errors.Is(err, xmlenc1.ErrMalformedEncrypted) ||
		errors.Is(err, xmlenc1.ErrOpaquePayload) ||
		errors.Is(err, xmlenc1.ErrMissingKey) ||
		errors.Is(err, xmlenc1.ErrTooManyEncryptedKeys) ||
		errors.Is(err, xmlenc1.ErrEncryptedKeyBytesExceeded) ||
		errors.Is(err, xmlenc1.ErrCipherValueBytesExceeded)
}

// swapEncryptionMethodAlgorithm finds the EncryptionMethod child of
// edElem and rewrites its Algorithm attribute to newAlg.
func swapEncryptionMethodAlgorithm(t *testing.T, edElem *helium.Element, newAlg string) {
	t.Helper()
	for child := edElem.FirstChild(); child != nil; child = child.NextSibling() {
		e, ok := helium.AsNode[*helium.Element](child)
		if !ok {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, "EncryptionMethod") {
			continue
		}
		require.NoError(t, e.SetAttribute("Algorithm", newAlg))
		return
	}
	t.Fatalf("EncryptionMethod child not found in EncryptedData")
}
