package xmlenc1_test

import (
	"os"
	"path/filepath"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xmlenc1"
	"github.com/stretchr/testify/require"
)

// merlinKeyJed is the key the merlin-xmlenc-five Readme names "jed": the
// literal ASCII bytes "abcdefghijklmnopqrstuvwxyz012345", used there as the
// AES-256 key-encryption key. It is published test key material, not a secret.
const merlinKeyJed = "abcdefghijklmnopqrstuvwxyz012345"

// TestInteropRetrievalMethod runs the one third-party vector that exercises a
// same-document ds:RetrievalMethod:
// merlin-xmlenc-five/encrypt-element-aes256-cbc-retrieved-kw-aes256.xml, from
// the Baltimore/Merlin interop examples carried in the Apache Santuario test
// corpus. Its EncryptedData carries no EncryptedKey of its own: its ds:KeyInfo
// holds only a <ds:RetrievalMethod Type=".../EncryptedKey" URI="#encrypt-key-0"/>
// naming an EncryptedKey that sits in a ds:KeyInfo elsewhere in the document.
// Before same-document references resolved, this document had no key at all.
//
// The vector is decrypted as far as this package decrypts it. Its session key
// is reached and unwrapped — proved below by which error the decrypt returns
// rather than by plaintext — and the block decryption then stops on the
// CBC padding, which is a property of the vector and not of the reference:
// xmlenc-core1 §5.2.1 pads with N-1 ARBITRARY octets and a final octet N,
// while this package's unpadding requires PKCS#7, where every padding octet
// equals N. That is a separate limitation of the CBC path, so the assertions
// below pin exactly the part the reference is responsible for.
func TestInteropRetrievalMethod(t *testing.T) {
	src, err := os.ReadFile(filepath.Join("testdata", "interop", "encrypt-element-aes256-cbc-retrieved-kw-aes256.xml"))
	require.NoError(t, err)

	doc, err := helium.NewParser().Parse(t.Context(), src)
	require.NoError(t, err)
	elem := findEncryptedData(t, doc.DocumentElement())
	require.NotNil(t, elem)

	// The candidate the parse retained came from the RetrievalMethod: the
	// EncryptedData's own ds:KeyInfo holds no EncryptedKey, so without
	// resolution this list is empty.
	ed, err := xmlenc1.ParseEncryptedDataForTest(elem)
	require.NoError(t, err)
	require.Len(t, ed.EncryptedKeys, 1)
	require.NotNil(t, ed.EncryptedKeys[0].EncryptionMethod)
	require.Equal(t, xmlenc1.AES256KeyWrap, ed.EncryptedKeys[0].EncryptionMethod.Algorithm)

	_, err = xmlenc1.NewDecryptor().
		KeyEncryptionKey([]byte(merlinKeyJed)).
		AllowUnauthenticatedCBC(true).
		Decrypt(t.Context(), elem)
	require.Error(t, err)

	// ErrMissingKey would mean the reference produced no candidate, and
	// ErrReferenceNotFound that it resolved to nothing. Neither is what this
	// document does.
	require.NotErrorIs(t, err, xmlenc1.ErrMissingKey)
	require.NotErrorIs(t, err, xmlenc1.ErrReferenceNotFound)
	require.NotErrorIs(t, err, xmlenc1.ErrAmbiguousReference)

	// The remaining failure is the block decryption. ErrKeyUnwrapFailed would
	// mean the referenced EncryptedKey was reached but did not unwrap under
	// the "jed" key-encryption key; its absence is what says the session key
	// this vector transports was recovered.
	require.ErrorIs(t, err, xmlenc1.ErrDecryptionFailed)
	require.NotErrorIs(t, err, xmlenc1.ErrKeyUnwrapFailed)
}
