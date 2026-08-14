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
//
// The vector decrypts end to end, so it is evidence for two things at once:
// the reference resolves and supplies the candidate, and the AES-CBC path reads
// padding this package did not write. The vector pads the way xmlenc-core1
// §5.2.1 allows — N-1 octets of arbitrary value and a final octet N — which is
// exactly what Decryptor.StrictPKCS7Padding would refuse, so the subtest below
// pins that refusal against the same document.
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

	nodes, err := xmlenc1.NewDecryptor().
		KeyEncryptionKey([]byte(merlinKeyJed)).
		AllowUnauthenticatedCBC(true).
		Decrypt(t.Context(), elem)
	require.NoError(t, err)
	require.Len(t, nodes, 1)
	require.Equal(t, "PaymentInfo", nodes[0].Name())

	out, err := helium.WriteString(nodes[0])
	require.NoError(t, err)
	require.Contains(t, out, "1234 567890 12345")
	require.Contains(t, out, "Foo B Baz")

	// The same document under the PKCS#7 opt-in. This is what that option
	// costs: a conforming third-party vector stops decrypting.
	t.Run("refused under StrictPKCS7Padding", func(t *testing.T) {
		_, err := xmlenc1.NewDecryptor().
			KeyEncryptionKey([]byte(merlinKeyJed)).
			AllowUnauthenticatedCBC(true).
			StrictPKCS7Padding(true).
			Decrypt(t.Context(), elem)
		require.Error(t, err)
		require.ErrorIs(t, err, xmlenc1.ErrDecryptionFailed)

		// The session key was still reached and unwrapped: only the block
		// decryption refused, which is what makes this a padding refusal
		// rather than a key one.
		require.NotErrorIs(t, err, xmlenc1.ErrKeyUnwrapFailed)
		require.NotErrorIs(t, err, xmlenc1.ErrMissingKey)
	})
}
