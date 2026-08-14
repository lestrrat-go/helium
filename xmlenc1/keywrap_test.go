package xmlenc1_test

import (
	"encoding/hex"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xmlenc1"
	"github.com/stretchr/testify/require"
)

// RFC 3394 test vectors for AES Key Wrap.
func TestAESKeyWrapRFC3394(t *testing.T) {
	// Test vector from RFC 3394, Section 4.6
	// 256-bit KEK, 256-bit key data
	kek, err := hex.DecodeString("000102030405060708090A0B0C0D0E0F101112131415161718191A1B1C1D1E1F")
	require.NoError(t, err)
	keyData, err := hex.DecodeString("00112233445566778899AABBCCDDEEFF000102030405060708090A0B0C0D0E0F")
	require.NoError(t, err)
	expected, err := hex.DecodeString("28C9F404C4B810F4CBCCB35CFB87F8263F5786E2D80ED326CBC7F0E71A99F43BFB988B9B7A02DD21")
	require.NoError(t, err)

	// The wrapped key is exactly what a remote peer has to agree on, so the
	// wrap output is pinned against the vector itself. A round-trip cannot
	// stand in for that: a symmetric-but-wrong variant — say one running seven
	// rounds where RFC 3394 runs six — unwraps its own wrap perfectly while
	// putting different bytes on the wire.
	wrapped, err := xmlenc1.AESKeyWrapForTest(kek, keyData)
	require.NoError(t, err)
	require.Equal(t, expected, wrapped)

	// The public Encryptor/Decryptor pair then confirms the XML-level path
	// carries that same wrap, through SessionKey + KeyWrap.
	doc := mustParseXML(t, `<root>data</root>`)

	encryptor := xmlenc1.NewEncryptor().
		BlockAlgorithm(xmlenc1.AES256CBC).
		AllowLegacyCBC(true).
		KeyWrapAlgorithm(xmlenc1.AES256KeyWrap).
		KeyEncryptionKey(kek).
		SessionKey(keyData)

	edElem, err := encryptor.EncryptElement(t.Context(), doc.DocumentElement())
	require.NoError(t, err)

	decryptor := xmlenc1.NewDecryptor().KeyEncryptionKey(kek).AllowUnauthenticatedCBC(true)
	nodes, err := decryptor.Decrypt(t.Context(), edElem)
	require.NoError(t, err)
	require.Len(t, nodes, 1)
}

func TestKeyWrapSize(t *testing.T) {
	t.Run("AES-192 KEK round-trip", func(t *testing.T) {
		kek := randKey(t, 24)
		doc := mustParseXML(t, `<root><a>hi</a></root>`)
		enc := xmlenc1.NewEncryptor().
			BlockAlgorithm(xmlenc1.AES128GCM11).
			KeyWrapAlgorithm(xmlenc1.AES192KeyWrap).
			KeyEncryptionKey(kek)
		edElem, err := enc.EncryptElement(t.Context(), doc.DocumentElement())
		require.NoError(t, err)

		nodes, err := xmlenc1.NewDecryptor().KeyEncryptionKey(kek).Decrypt(t.Context(), edElem)
		require.NoError(t, err)
		require.Len(t, nodes, 1)
	})

	t.Run("correct size round-trip", func(t *testing.T) {
		kek := randKey(t, 32)
		doc := mustParseXML(t, samlAssertion)
		enc := xmlenc1.NewEncryptor().
			BlockAlgorithm(xmlenc1.AES256GCM).
			KeyWrapAlgorithm(xmlenc1.AES256KeyWrap).
			KeyEncryptionKey(kek)
		edElem, err := enc.EncryptElement(t.Context(), doc.DocumentElement())
		require.NoError(t, err)

		dec := xmlenc1.NewDecryptor().KeyEncryptionKey(kek)
		nodes, err := dec.Decrypt(t.Context(), edElem)
		require.NoError(t, err)
		require.Len(t, nodes, 1)
		s, err := helium.WriteString(nodes[0])
		require.NoError(t, err)
		require.Contains(t, s, "user@example.com")
	})

	// RFC 3394 wraps a K-byte key into exactly K+8 bytes, and the declared
	// block algorithm fixes K, so every other ciphertext length is rejected
	// before the unwrap rounds run.
	t.Run("wrapped key length must match the block algorithm", func(t *testing.T) {
		const dataAlgorithm = xmlenc1.AES256GCM // a 32-byte session key
		kek := randKey(t, 32)
		sessionKey := randKey(t, 32)

		newElem := func(t *testing.T, wrapped []byte) *helium.Element {
			t.Helper()
			cipher, err := xmlenc1.EncryptBytesForTest(dataAlgorithm, sessionKey, []byte("<x>secret</x>"))
			require.NoError(t, err)
			doc := mustParseXML(t, `<root/>`)
			elem, err := xmlenc1.MarshalEncryptedDataForTest(doc, &xmlenc1.EncryptedData{
				Type:             xmlenc1.TypeElement,
				EncryptionMethod: &xmlenc1.EncryptionMethod{Algorithm: dataAlgorithm},
				EncryptedKeys: []*xmlenc1.EncryptedKey{{
					EncryptionMethod: &xmlenc1.EncryptionMethod{Algorithm: xmlenc1.AES256KeyWrap},
					CipherValue:      wrapped,
				}},
				CipherValue: cipher,
			})
			require.NoError(t, err)
			return elem
		}

		t.Run("short wrap of a 16-byte key", func(t *testing.T) {
			// 24 bytes: a multiple of 8 and a real RFC 3394 wrap, but of a
			// key half the length the declared algorithm needs.
			wrapped, err := xmlenc1.AESKeyWrapForTest(kek, randKey(t, 16))
			require.NoError(t, err)
			require.Len(t, wrapped, 24)

			_, err = xmlenc1.NewDecryptor().KeyEncryptionKey(kek).Decrypt(t.Context(), newElem(t, wrapped))
			require.ErrorIs(t, err, xmlenc1.ErrKeyUnwrapFailed)
			require.ErrorIs(t, err, xmlenc1.ErrDecryptionFailed)
		})

		t.Run("oversized wrap", func(t *testing.T) {
			// 64 KiB, a multiple of 8: accepted by a length check that only
			// asks for "a multiple of 8, at least 24 bytes".
			_, err := xmlenc1.NewDecryptor().KeyEncryptionKey(kek).Decrypt(t.Context(), newElem(t, make([]byte, 64<<10)))
			require.ErrorIs(t, err, xmlenc1.ErrKeyUnwrapFailed)
			require.ErrorIs(t, err, xmlenc1.ErrDecryptionFailed)
		})

		t.Run("matching wrap round-trips", func(t *testing.T) {
			wrapped, err := xmlenc1.AESKeyWrapForTest(kek, sessionKey)
			require.NoError(t, err)
			require.Len(t, wrapped, 40)

			nodes, err := xmlenc1.NewDecryptor().KeyEncryptionKey(kek).Decrypt(t.Context(), newElem(t, wrapped))
			require.NoError(t, err)
			require.Len(t, nodes, 1)
			s, err := helium.WriteString(nodes[0])
			require.NoError(t, err)
			require.Contains(t, s, "secret")
		})
	})

	t.Run("encrypt KEK size mismatch", func(t *testing.T) {
		doc := mustParseXML(t, `<root><a>hi</a></root>`)
		enc := xmlenc1.NewEncryptor().
			BlockAlgorithm(xmlenc1.AES256GCM).
			KeyWrapAlgorithm(xmlenc1.AES256KeyWrap).
			KeyEncryptionKey(randKey(t, 16)) // AES-128 KEK declared as kw-aes256
		_, err := enc.EncryptElement(t.Context(), doc.DocumentElement())
		require.Error(t, err)
		var kse *xmlenc1.KeySizeError
		require.ErrorAs(t, err, &kse)
	})

	t.Run("encrypt URI must name a key wrap", func(t *testing.T) {
		// Wrapping is RFC 3394 AES Key Wrap whatever the URI says, so a URI
		// naming a block cipher would declare on the wire an algorithm that did
		// not produce the CipherValue — the document would say AES-GCM over a
		// 40-octet RFC 3394 wrap, and nothing, this package included, reads that
		// back. keySizeForAlgorithm answers a length for the block-cipher URIs
		// too, so the KEK-length binding cannot stand in for the check: every
		// case below carries the KEK length that binding asks for.
		for _, tc := range []struct {
			uri     string
			kekSize int
		}{
			{uri: xmlenc1.AES128CBC, kekSize: 16},
			{uri: xmlenc1.AES256CBC, kekSize: 32},
			{uri: xmlenc1.AES128GCM, kekSize: 16},
			{uri: xmlenc1.AES256GCM, kekSize: 32},
			{uri: xmlenc1.AES128GCM11, kekSize: 16},
			{uri: xmlenc1.AES192GCM11, kekSize: 24},
			{uri: xmlenc1.AES256GCM11, kekSize: 32},
		} {
			t.Run(tc.uri, func(t *testing.T) {
				doc := mustParseXML(t, samlAssertion)
				_, err := xmlenc1.NewEncryptor().
					BlockAlgorithm(xmlenc1.AES256GCM11).
					KeyWrapAlgorithm(tc.uri).
					KeyEncryptionKey(randKey(t, tc.kekSize)).
					EncryptElement(t.Context(), doc.DocumentElement())
				require.ErrorIs(t, err, xmlenc1.ErrEncryptionFailed)

				var uae *xmlenc1.UnsupportedAlgorithmError
				require.ErrorAs(t, err, &uae)
				require.Equal(t, tc.uri, uae.Algorithm)

				// Nothing was emitted and the caller's tree still holds the
				// plaintext, so no mis-declared document exists at all.
				xml, err := helium.WriteString(doc)
				require.NoError(t, err)
				require.Contains(t, xml, "user@example.com")
				require.NotContains(t, xml, "EncryptedData")
			})
		}
	})

	// Each genuine key wrap URI still round-trips, so the check above narrows
	// nothing a recipient can read.
	t.Run("every key wrap URI round-trips", func(t *testing.T) {
		for _, tc := range []struct {
			uri     string
			kekSize int
		}{
			{uri: xmlenc1.AES128KeyWrap, kekSize: 16},
			{uri: xmlenc1.AES192KeyWrap, kekSize: 24},
			{uri: xmlenc1.AES256KeyWrap, kekSize: 32},
		} {
			t.Run(tc.uri, func(t *testing.T) {
				kek := randKey(t, tc.kekSize)
				doc := mustParseXML(t, samlAssertion)
				edElem, err := xmlenc1.NewEncryptor().
					BlockAlgorithm(xmlenc1.AES256GCM11).
					KeyWrapAlgorithm(tc.uri).
					KeyEncryptionKey(kek).
					EncryptElement(t.Context(), doc.DocumentElement())
				require.NoError(t, err)

				nodes, err := xmlenc1.NewDecryptor().KeyEncryptionKey(kek).Decrypt(t.Context(), edElem)
				require.NoError(t, err)
				require.Len(t, nodes, 1)
			})
		}
	})

	// The URI is decided before any payload work, while the KEK length is bound
	// after the block encryption, so a caller who got both wrong hears about the
	// setting that can never work, and nothing about the key.
	t.Run("encrypt reports the URI ahead of the KEK length", func(t *testing.T) {
		doc := mustParseXML(t, samlAssertion)
		_, err := xmlenc1.NewEncryptor().
			BlockAlgorithm(xmlenc1.AES256GCM11).
			KeyWrapAlgorithm(xmlenc1.AES256GCM).
			KeyEncryptionKey(randKey(t, 16)).
			EncryptElement(t.Context(), doc.DocumentElement())
		require.ErrorIs(t, err, xmlenc1.ErrEncryptionFailed)

		var kse *xmlenc1.KeySizeError
		require.NotErrorAs(t, err, &kse)
	})

	t.Run("decrypt KEK size mismatch", func(t *testing.T) {
		kek := randKey(t, 32)
		doc := mustParseXML(t, samlAssertion)
		enc := xmlenc1.NewEncryptor().
			BlockAlgorithm(xmlenc1.AES256GCM).
			KeyWrapAlgorithm(xmlenc1.AES256KeyWrap).
			KeyEncryptionKey(kek)
		edElem, err := enc.EncryptElement(t.Context(), doc.DocumentElement())
		require.NoError(t, err)

		// kw-aes256 was declared on the wire; supply a 16-byte KEK.
		dec := xmlenc1.NewDecryptor().KeyEncryptionKey(randKey(t, 16))
		_, err = dec.Decrypt(t.Context(), edElem)
		require.Error(t, err)
		var kse *xmlenc1.KeySizeError
		require.ErrorAs(t, err, &kse)
	})
}

// newKeyWrapElem builds an EncryptedData element carrying wrapped as the
// sole EncryptedKey's CipherValue, wrapping sessionKey-encrypted plaintext
// under dataAlgorithm. It follows the same construction TestKeyWrapSize's
// "wrapped key length must match the block algorithm" subtest uses, pulled
// out to a named helper so TestKeyWrapIntegrityCheck's subtests can share it.
func newKeyWrapElem(t *testing.T, dataAlgorithm string, sessionKey, wrapped []byte) *helium.Element {
	t.Helper()
	cipher, err := xmlenc1.EncryptBytesForTest(dataAlgorithm, sessionKey, []byte("<x>secret</x>"))
	require.NoError(t, err)
	doc := mustParseXML(t, `<root/>`)
	elem, err := xmlenc1.MarshalEncryptedDataForTest(doc, &xmlenc1.EncryptedData{
		Type:             xmlenc1.TypeElement,
		EncryptionMethod: &xmlenc1.EncryptionMethod{Algorithm: dataAlgorithm},
		EncryptedKeys: []*xmlenc1.EncryptedKey{{
			EncryptionMethod: &xmlenc1.EncryptionMethod{Algorithm: xmlenc1.AES256KeyWrap},
			CipherValue:      wrapped,
		}},
		CipherValue: cipher,
	})
	require.NoError(t, err)
	return elem
}

// flipLowBit returns a copy of b with bit 0 of the byte at position i
// flipped, leaving every other byte untouched.
func flipLowBit(b []byte, i int) []byte {
	corrupted := make([]byte, len(b))
	copy(corrupted, b)
	corrupted[i] ^= 0x01
	return corrupted
}

// TestKeyWrapIntegrityCheck exercises aesKeyUnwrap's RFC 3394 integrity
// check — the comparison of the recovered A register against defaultIV —
// through the public Decryptor. It pins the current pass/fail/error-message
// behavior so a refactor of that comparison (e.g. to a constant-time
// primitive) cannot silently change which wraps are accepted or what a
// caller sees when one is rejected.
func TestKeyWrapIntegrityCheck(t *testing.T) {
	const dataAlgorithm = xmlenc1.AES256GCM // a 32-byte session key
	kek := randKey(t, 32)
	sessionKey := randKey(t, 32)

	wrapped, err := xmlenc1.AESKeyWrapForTest(kek, sessionKey)
	require.NoError(t, err)
	require.Len(t, wrapped, 40)

	t.Run("the intact wrap still unwraps", func(t *testing.T) {
		elem := newKeyWrapElem(t, dataAlgorithm, sessionKey, wrapped)
		nodes, err := xmlenc1.NewDecryptor().KeyEncryptionKey(kek).Decrypt(t.Context(), elem)
		require.NoError(t, err)
		require.Len(t, nodes, 1)
		s, err := helium.WriteString(nodes[0])
		require.NoError(t, err)
		require.Contains(t, s, "secret")
	})

	// Corrupting any one of the 40 wrapped-key wire bytes must be rejected,
	// and every rejection must look identical to a caller: same sentinel
	// errors, same error string. A position-dependent message would leak
	// which byte the unwrap disagreed on.
	t.Run("every corrupted wrap byte is rejected identically", func(t *testing.T) {
		var wantErr string
		for i := range wrapped {
			elem := newKeyWrapElem(t, dataAlgorithm, sessionKey, flipLowBit(wrapped, i))
			_, err := xmlenc1.NewDecryptor().KeyEncryptionKey(kek).Decrypt(t.Context(), elem)
			require.ErrorIs(t, err, xmlenc1.ErrKeyUnwrapFailed)
			require.ErrorIs(t, err, xmlenc1.ErrDecryptionFailed)
			if i == 0 {
				wantErr = err.Error()
				continue
			}
			require.Equal(t, wantErr, err.Error())
		}
	})

	// Wire bytes 0-7 carry the A register the integrity check actually
	// compares against defaultIV. Covering each of those 8 bytes
	// individually, including byte 7, is the direct exercise of that
	// comparison, and not of diffusion through the surrounding rounds.
	t.Run("a corrupted integrity register is rejected", func(t *testing.T) {
		var wantErr string
		for i := range 8 {
			elem := newKeyWrapElem(t, dataAlgorithm, sessionKey, flipLowBit(wrapped, i))
			_, err := xmlenc1.NewDecryptor().KeyEncryptionKey(kek).Decrypt(t.Context(), elem)
			require.ErrorIs(t, err, xmlenc1.ErrKeyUnwrapFailed)
			require.ErrorIs(t, err, xmlenc1.ErrDecryptionFailed)
			if i == 0 {
				wantErr = err.Error()
				continue
			}
			require.Equal(t, wantErr, err.Error())
		}
	})
}
