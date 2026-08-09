package xmlenc1

import (
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

func TestParseKeyInfoRejectsBeforeAppendingBeyondCap(t *testing.T) {
	doc, err := helium.NewParser().Parse(t.Context(), []byte(
		`<xenc:EncryptedData xmlns:xenc="`+NamespaceXMLEnc+`" xmlns:ds="`+NamespaceDSig+`"><ds:KeyInfo>`+
			`<xenc:EncryptedKey Id="first"><xenc:CipherData><xenc:CipherValue>AAAA</xenc:CipherValue></xenc:CipherData></xenc:EncryptedKey>`+
			`<xenc:EncryptedKey Id="second"><xenc:CipherData><xenc:CipherValue>AAAA</xenc:CipherValue></xenc:CipherData></xenc:EncryptedKey>`+
			`</ds:KeyInfo></xenc:EncryptedData>`,
	))
	require.NoError(t, err)

	keyInfo, ok := helium.AsNode[*helium.Element](doc.DocumentElement().FirstChild())
	require.True(t, ok)
	ed := &EncryptedData{}
	cfg := &decryptConfig{maxEncryptedKeys: 1}
	err = parseKeyInfoForEncryption(t.Context(), keyInfo, ed, newEncryptedKeyBudget(cfg), cfg)
	require.ErrorIs(t, err, ErrTooManyEncryptedKeys)
	require.Contains(t, err.Error(), "2 candidates exceed the limit of 1")
	require.Len(t, ed.EncryptedKeys, 1)
	require.Equal(t, "first", ed.EncryptedKeys[0].ID)
}
