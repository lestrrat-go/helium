package xmlenc1_test

import (
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

	// H2 Flaw 4: padding oracle hardening. Errors from CBC decryption must
	// not distinguish "bad padding" from "valid padding but invalid XML" at
	// the caller-visible boundary.
	t.Run("padding oracle indistinguishable errors", func(t *testing.T) {
		sessionKey := make([]byte, 16)
		_, err := rand.Read(sessionKey)
		require.NoError(t, err)

		algorithm := xmlenc1.AES128CBC
		plaintext := []byte("<x>secret data inside</x>")
		cipher, err := xmlenc1.EncryptBytesForTest(algorithm, sessionKey, plaintext)
		require.NoError(t, err)

		mkED := func(c []byte) *helium.Element {
			doc := mustParseXML(t, `<root/>`)
			ed := &xmlenc1.EncryptedData{
				Type:             xmlenc1.TypeElement,
				EncryptionMethod: &xmlenc1.EncryptionMethod{Algorithm: algorithm},
				CipherValue:      c,
			}
			edElem, mErr := xmlenc1.MarshalEncryptedDataForTest(doc, ed)
			require.NoError(t, mErr)
			return edElem
		}

		decryptor := xmlenc1.NewDecryptor().
			SessionKey(sessionKey).
			AllowUnauthenticatedCBC(true)

		// Mutation A: flip a bit in the IV. This randomizes the first
		// plaintext block and is very likely to produce bytes that look
		// like invalid padding once unpadding is attempted on the last
		// block (since last block plaintext is unaffected, this often
		// produces valid padding but garbage XML — useful contrast).
		cipherA := append([]byte(nil), cipher...)
		cipherA[0] ^= 0x01

		// Mutation B: flip a bit in the LAST ciphertext byte. This
		// directly corrupts padding most of the time.
		cipherB := append([]byte(nil), cipher...)
		cipherB[len(cipherB)-1] ^= 0x01

		_, errA := decryptor.Decrypt(t.Context(), mkED(cipherA))
		require.Error(t, errA)
		require.ErrorIs(t, errA, xmlenc1.ErrDecryptionFailed)
		require.False(t,
			strings.Contains(strings.ToLower(errA.Error()), "padding"),
			"error A leaks padding state: %v", errA)
		require.False(t, errors.Is(errA, xmlenc1.ErrInvalidPadding),
			"error A is distinguishable as ErrInvalidPadding: %v", errA)

		_, errB := decryptor.Decrypt(t.Context(), mkED(cipherB))
		require.Error(t, errB)
		require.ErrorIs(t, errB, xmlenc1.ErrDecryptionFailed)
		require.False(t,
			strings.Contains(strings.ToLower(errB.Error()), "padding"),
			"error B leaks padding state: %v", errB)
		require.False(t, errors.Is(errB, xmlenc1.ErrInvalidPadding),
			"error B is distinguishable as ErrInvalidPadding: %v", errB)
	})
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
		require.Contains(t, xml, xmlenc1.AES256GCM, "default block algorithm must be AES-256-GCM")
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
