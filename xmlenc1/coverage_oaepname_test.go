package xmlenc1_test

import (
	"testing"

	"github.com/lestrrat-go/helium/xmlenc1"
	"github.com/stretchr/testify/require"
)

// TestOAEPNameHelperBranches drives the digest/MGF-name formatting helpers
// (oaepDigestName / oaepMGFName) through their non-default and default
// branches. They are only reached when oaepHashFunc rejects a digest/MGF
// hash mismatch, so each case sets up an RSA-OAEP 1.1 configuration whose
// declared digest and MGF hashes disagree and asserts the encrypt path
// surfaces ErrEncryptionFailed.
func TestOAEPNameHelperBranches(t *testing.T) {
	key := generateRSAKey(t)

	t.Run("default digest vs explicit SHA256 MGF", func(t *testing.T) {
		// digest unset -> SHA1 (default); MGF SHA256 -> mismatch.
		// Exercises oaepDigestName("") default branch and the
		// oaepMGFName non-empty branch.
		doc := mustParseXML(t, samlAssertion)
		encryptor := xmlenc1.NewEncryptor().
			BlockAlgorithm(xmlenc1.AES256GCM).
			KeyTransportAlgorithm(xmlenc1.RSAOAEP11).
			OAEPMGF(xmlenc1.MGFSHA256).
			RecipientPublicKey(&key.PublicKey)

		_, err := encryptor.EncryptElement(t.Context(), doc.DocumentElement())
		require.ErrorIs(t, err, xmlenc1.ErrEncryptionFailed)
	})

	t.Run("explicit SHA256 digest vs default MGF", func(t *testing.T) {
		// digest SHA256; MGF unset -> SHA1 (default) -> mismatch.
		// Exercises the oaepMGFName default branch for RSAOAEP11.
		doc := mustParseXML(t, samlAssertion)
		encryptor := xmlenc1.NewEncryptor().
			BlockAlgorithm(xmlenc1.AES256GCM).
			KeyTransportAlgorithm(xmlenc1.RSAOAEP11).
			OAEPDigest(xmlenc1.DigestSHA256).
			RecipientPublicKey(&key.PublicKey)

		_, err := encryptor.EncryptElement(t.Context(), doc.DocumentElement())
		require.ErrorIs(t, err, xmlenc1.ErrEncryptionFailed)
	})

	t.Run("legacy mgf1p with SHA256 digest", func(t *testing.T) {
		// rsa-oaep-mgf1p fixes MGF to SHA1; a SHA256 digest mismatches.
		// Exercises oaepMGFName's "implied by rsa-oaep-mgf1p" branch.
		doc := mustParseXML(t, samlAssertion)
		encryptor := xmlenc1.NewEncryptor().
			BlockAlgorithm(xmlenc1.AES256GCM).
			KeyTransportAlgorithm(xmlenc1.RSAOAEP).
			OAEPDigest(xmlenc1.DigestSHA256).
			RecipientPublicKey(&key.PublicKey)

		_, err := encryptor.EncryptElement(t.Context(), doc.DocumentElement())
		require.ErrorIs(t, err, xmlenc1.ErrEncryptionFailed)
	})
}
