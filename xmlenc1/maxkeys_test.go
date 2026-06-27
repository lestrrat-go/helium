package xmlenc1_test

import (
	"context"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xmlenc1"
	"github.com/stretchr/testify/require"
)

// manyKeyEncryptedData builds an EncryptedData element carrying n junk RSA
// EncryptedKey candidates, used to exercise the trial-decrypt cap.
func manyKeyEncryptedData(t *testing.T, n int) *helium.Element {
	t.Helper()
	doc := mustParseXML(t, `<root/>`)
	keys := make([]*xmlenc1.EncryptedKey, 0, n)
	for range n {
		keys = append(keys, &xmlenc1.EncryptedKey{
			EncryptionMethod: &xmlenc1.EncryptionMethod{Algorithm: xmlenc1.RSAOAEP},
			CipherValue:      make([]byte, 256),
		})
	}
	ed := &xmlenc1.EncryptedData{
		Type:             xmlenc1.TypeElement,
		EncryptionMethod: &xmlenc1.EncryptionMethod{Algorithm: xmlenc1.AES256GCM},
		EncryptedKeys:    keys,
		CipherValue:      make([]byte, 48),
	}
	elem, err := xmlenc1.MarshalEncryptedDataForTest(doc, ed)
	require.NoError(t, err)
	return elem
}

func TestMaxEncryptedKeys(t *testing.T) {
	t.Run("over default cap fails fast", func(t *testing.T) {
		elem := manyKeyEncryptedData(t, xmlenc1.DefaultMaxEncryptedKeys+1)
		_, err := xmlenc1.NewDecryptor().PrivateKey(generateRSAKey(t)).Decrypt(t.Context(), elem)
		require.ErrorIs(t, err, xmlenc1.ErrTooManyEncryptedKeys)
	})

	t.Run("at default cap is not rejected by the cap", func(t *testing.T) {
		elem := manyKeyEncryptedData(t, xmlenc1.DefaultMaxEncryptedKeys)
		_, err := xmlenc1.NewDecryptor().PrivateKey(generateRSAKey(t)).Decrypt(t.Context(), elem)
		require.Error(t, err)
		require.NotErrorIs(t, err, xmlenc1.ErrTooManyEncryptedKeys)
	})

	t.Run("explicit cap rejects above it", func(t *testing.T) {
		elem := manyKeyEncryptedData(t, 3)
		_, err := xmlenc1.NewDecryptor().PrivateKey(generateRSAKey(t)).MaxEncryptedKeys(2).Decrypt(t.Context(), elem)
		require.ErrorIs(t, err, xmlenc1.ErrTooManyEncryptedKeys)
	})

	t.Run("negative cap removes the limit", func(t *testing.T) {
		elem := manyKeyEncryptedData(t, xmlenc1.DefaultMaxEncryptedKeys+5)
		_, err := xmlenc1.NewDecryptor().PrivateKey(generateRSAKey(t)).MaxEncryptedKeys(-1).Decrypt(t.Context(), elem)
		require.Error(t, err)
		require.NotErrorIs(t, err, xmlenc1.ErrTooManyEncryptedKeys)
	})

	t.Run("normal document still decrypts", func(t *testing.T) {
		key := generateRSAKey(t)
		doc := mustParseXML(t, samlAssertion)
		encryptor := xmlenc1.NewEncryptor().
			BlockAlgorithm(xmlenc1.AES256GCM).
			KeyTransportAlgorithm(xmlenc1.RSAOAEP11).
			OAEPDigest(xmlenc1.DigestSHA256).
			OAEPMGF(xmlenc1.MGFSHA256).
			RecipientPublicKey(&key.PublicKey)
		edElem, err := encryptor.EncryptElement(t.Context(), doc.DocumentElement())
		require.NoError(t, err)

		nodes, err := xmlenc1.NewDecryptor().PrivateKey(key).Decrypt(t.Context(), edElem)
		require.NoError(t, err)
		require.Len(t, nodes, 1)
	})

	t.Run("cancelled context aborts the candidate loop", func(t *testing.T) {
		key := generateRSAKey(t)
		doc := mustParseXML(t, samlAssertion)
		encryptor := xmlenc1.NewEncryptor().
			BlockAlgorithm(xmlenc1.AES256GCM).
			KeyTransportAlgorithm(xmlenc1.RSAOAEP11).
			OAEPDigest(xmlenc1.DigestSHA256).
			OAEPMGF(xmlenc1.MGFSHA256).
			RecipientPublicKey(&key.PublicKey)
		edElem, err := encryptor.EncryptElement(t.Context(), doc.DocumentElement())
		require.NoError(t, err)

		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		_, err = xmlenc1.NewDecryptor().PrivateKey(key).Decrypt(ctx, edElem)
		require.ErrorIs(t, err, context.Canceled)
	})
}
