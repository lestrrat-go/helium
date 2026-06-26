package xmlenc1_test

import (
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xmlenc1"
	"github.com/stretchr/testify/require"
)

// TestMultiRecipientDecrypt covers XENC-002: an EncryptedData may carry
// several EncryptedKey candidates (one per recipient), and decryption must
// try each rather than committing to the first. This also makes a bogus
// EncryptedKey prepended to a legitimate one a non-issue instead of a DoS.
func TestMultiRecipientDecrypt(t *testing.T) {
	const algorithm = xmlenc1.AES256GCM

	newEncryptedData := func(t *testing.T, keys []*xmlenc1.EncryptedKey, cipher []byte) *helium.Element {
		t.Helper()
		doc := mustParseXML(t, `<root/>`)
		ed := &xmlenc1.EncryptedData{
			Type:             xmlenc1.TypeElement,
			EncryptionMethod: &xmlenc1.EncryptionMethod{Algorithm: algorithm},
			EncryptedKeys:    keys,
			CipherValue:      cipher,
		}
		elem, err := xmlenc1.MarshalEncryptedDataForTest(doc, ed)
		require.NoError(t, err)
		return elem
	}

	wrap := func(t *testing.T, kek, sessionKey []byte) []byte {
		t.Helper()
		wrapped, err := xmlenc1.AESKeyWrapForTest(kek, sessionKey)
		require.NoError(t, err)
		return wrapped
	}

	t.Run("second recipient matches", func(t *testing.T) {
		sessionKey := randKey(t, 32)
		cipher, err := xmlenc1.EncryptBytesForTest(algorithm, sessionKey, []byte("<x>secret</x>"))
		require.NoError(t, err)

		kekOther := randKey(t, 32)
		kekMine := randKey(t, 32)

		// Two legitimate recipients; only the second one's KEK is ours.
		elem := newEncryptedData(t, []*xmlenc1.EncryptedKey{
			{
				EncryptionMethod: &xmlenc1.EncryptionMethod{Algorithm: xmlenc1.AES256KeyWrap},
				CipherValue:      wrap(t, kekOther, sessionKey),
			},
			{
				EncryptionMethod: &xmlenc1.EncryptionMethod{Algorithm: xmlenc1.AES256KeyWrap},
				CipherValue:      wrap(t, kekMine, sessionKey),
			},
		}, cipher)

		nodes, err := xmlenc1.NewDecryptor().KeyEncryptionKey(kekMine).Decrypt(t.Context(), elem)
		require.NoError(t, err)
		require.Len(t, nodes, 1)
		s, err := helium.WriteString(nodes[0])
		require.NoError(t, err)
		require.Contains(t, s, "secret")
	})

	t.Run("bogus first key tolerated", func(t *testing.T) {
		sessionKey := randKey(t, 32)
		cipher, err := xmlenc1.EncryptBytesForTest(algorithm, sessionKey, []byte("<x>secret</x>"))
		require.NoError(t, err)

		kekMine := randKey(t, 32)

		// A junk EncryptedKey is prepended ahead of the legitimate one.
		// Under the old "first key only" behavior this denied service to
		// the real recipient.
		elem := newEncryptedData(t, []*xmlenc1.EncryptedKey{
			{
				EncryptionMethod: &xmlenc1.EncryptionMethod{Algorithm: xmlenc1.AES256KeyWrap},
				CipherValue:      randKey(t, 40), // not a valid AES-wrap of any key
			},
			{
				EncryptionMethod: &xmlenc1.EncryptionMethod{Algorithm: xmlenc1.AES256KeyWrap},
				CipherValue:      wrap(t, kekMine, sessionKey),
			},
		}, cipher)

		nodes, err := xmlenc1.NewDecryptor().KeyEncryptionKey(kekMine).Decrypt(t.Context(), elem)
		require.NoError(t, err)
		require.Len(t, nodes, 1)
		s, err := helium.WriteString(nodes[0])
		require.NoError(t, err)
		require.Contains(t, s, "secret")
	})
}

// TestParseRejectsDuplicateCardinality covers XENC-003: XML Encryption
// allows at most one EncryptionMethod and one CipherData per EncryptedData
// (and per EncryptedKey). Duplicates were previously accepted last-one-wins;
// they must now be rejected during parse.
func TestParseRejectsDuplicateCardinality(t *testing.T) {
	const xenc = `xmlns:xenc="http://www.w3.org/2001/04/xmlenc#"`
	const ds = `xmlns:ds="http://www.w3.org/2000/09/xmldsig#"`

	for _, tc := range []struct {
		name string
		xml  string
	}{
		{
			name: "duplicate EncryptionMethod in EncryptedData",
			xml: `<xenc:EncryptedData ` + xenc + `>` +
				`<xenc:EncryptionMethod Algorithm="` + xmlenc1.AES256GCM + `"/>` +
				`<xenc:EncryptionMethod Algorithm="` + xmlenc1.AES128GCM + `"/>` +
				`<xenc:CipherData><xenc:CipherValue>AAAA</xenc:CipherValue></xenc:CipherData>` +
				`</xenc:EncryptedData>`,
		},
		{
			name: "duplicate CipherData in EncryptedData",
			xml: `<xenc:EncryptedData ` + xenc + `>` +
				`<xenc:EncryptionMethod Algorithm="` + xmlenc1.AES256GCM + `"/>` +
				`<xenc:CipherData><xenc:CipherValue>AAAA</xenc:CipherValue></xenc:CipherData>` +
				`<xenc:CipherData><xenc:CipherValue>BBBB</xenc:CipherValue></xenc:CipherData>` +
				`</xenc:EncryptedData>`,
		},
		{
			name: "duplicate EncryptionMethod in EncryptedKey",
			xml: `<xenc:EncryptedData ` + xenc + ` ` + ds + `>` +
				`<xenc:EncryptionMethod Algorithm="` + xmlenc1.AES256GCM + `"/>` +
				`<ds:KeyInfo><xenc:EncryptedKey>` +
				`<xenc:EncryptionMethod Algorithm="` + xmlenc1.RSAOAEP + `"/>` +
				`<xenc:EncryptionMethod Algorithm="` + xmlenc1.RSAOAEP11 + `"/>` +
				`<xenc:CipherData><xenc:CipherValue>AAAA</xenc:CipherValue></xenc:CipherData>` +
				`</xenc:EncryptedKey></ds:KeyInfo>` +
				`<xenc:CipherData><xenc:CipherValue>AAAA</xenc:CipherValue></xenc:CipherData>` +
				`</xenc:EncryptedData>`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			doc := mustParseXML(t, tc.xml)
			_, err := xmlenc1.ParseEncryptedDataForTest(doc.DocumentElement())
			require.ErrorIs(t, err, xmlenc1.ErrMalformedEncrypted)
		})
	}
}

// TestDecryptType covers XENC-004: a non-empty Type other than Element or
// Content (including unknown URIs) must be rejected rather than silently
// treated as Element. An omitted Type keeps the historical Element default.
func TestDecryptType(t *testing.T) {
	const algorithm = xmlenc1.AES256GCM

	build := func(t *testing.T, typeURI string, cipher []byte) *helium.Element {
		t.Helper()
		doc := mustParseXML(t, `<root/>`)
		ed := &xmlenc1.EncryptedData{
			Type:             typeURI,
			EncryptionMethod: &xmlenc1.EncryptionMethod{Algorithm: algorithm},
			CipherValue:      cipher,
		}
		elem, err := xmlenc1.MarshalEncryptedDataForTest(doc, ed)
		require.NoError(t, err)
		return elem
	}

	t.Run("unknown Type rejected", func(t *testing.T) {
		sessionKey := randKey(t, 32)
		cipher, err := xmlenc1.EncryptBytesForTest(algorithm, sessionKey, []byte("<x>secret</x>"))
		require.NoError(t, err)

		elem := build(t, "urn:example:bogus-type", cipher)
		_, err = xmlenc1.NewDecryptor().SessionKey(sessionKey).Decrypt(t.Context(), elem)
		require.ErrorIs(t, err, xmlenc1.ErrMalformedEncrypted)
	})

	t.Run("omitted Type defaults to Element", func(t *testing.T) {
		sessionKey := randKey(t, 32)
		cipher, err := xmlenc1.EncryptBytesForTest(algorithm, sessionKey, []byte("<x>secret</x>"))
		require.NoError(t, err)

		elem := build(t, "", cipher)
		nodes, err := xmlenc1.NewDecryptor().SessionKey(sessionKey).Decrypt(t.Context(), elem)
		require.NoError(t, err)
		require.Len(t, nodes, 1)
		require.Equal(t, helium.ElementNode, nodes[0].Type())
	})
}
