package xmlenc1

import (
	"crypto/sha256"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

func TestConcatKDFNonByteAlignedOtherInfo(t *testing.T) {
	doc, err := helium.NewParser().Parse(t.Context(), []byte(
		`<xenc11:ConcatKDFParams xmlns:xenc11="`+NamespaceXMLEnc11+`" xmlns:ds="`+NamespaceDSig+`" PartyUInfo="03D8" PartyVInfo="0780"><ds:DigestMethod Algorithm="`+DigestSHA256+`"/></xenc11:ConcatKDFParams>`,
	))
	require.NoError(t, err)

	params, err := parseConcatKDFParams(doc.DocumentElement())
	require.NoError(t, err)
	require.Equal(t, []byte{0xd8}, params.PartyUInfo)
	require.Equal(t, uint8(3), params.partyUInfoUnusedBits)
	require.Equal(t, []byte{0x80}, params.PartyVInfo)
	require.Equal(t, uint8(7), params.partyVInfoUnusedBits)

	sharedSecret := []byte{0x01, 0x02, 0x03, 0x04}
	got, err := deriveConcatKDF(sharedSecret, params, sha256.Size)
	require.NoError(t, err)

	input := []byte{0, 0, 0, 1}
	input = append(input, sharedSecret...)
	input = append(input, 0xdc) // 11011 || 1, repacked with two trailing zero bits.
	want := sha256.Sum256(input)
	require.Equal(t, want[:], got)
}
