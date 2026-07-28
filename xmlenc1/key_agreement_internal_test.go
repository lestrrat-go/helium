package xmlenc1

import (
	"crypto/ecdh"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
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

func TestConcatKDFSHA384AcceptsXMLDSigMoreURI(t *testing.T) {
	sharedSecret := []byte{0x01, 0x02, 0x03, 0x04}

	canonical, err := deriveConcatKDF(sharedSecret, &ConcatKDFParams{DigestMethod: DigestSHA384}, 48)
	require.NoError(t, err)
	alias, err := deriveConcatKDF(sharedSecret, &ConcatKDFParams{DigestMethod: DigestSHA384DSigMore}, 48)
	require.NoError(t, err)
	require.Equal(t, canonical, alias)
}

func TestDecryptECDHSessionKeyAllowsOnlyAESKeyWrap(t *testing.T) {
	tests := []struct {
		name      string
		algorithm string
		wantError bool
	}{
		{name: "AES-128 key wrap", algorithm: AES128KeyWrap},
		{name: "AES-192 key wrap", algorithm: AES192KeyWrap},
		{name: "AES-256 key wrap", algorithm: AES256KeyWrap},
		{name: "AES-128 GCM", algorithm: AES128GCM, wantError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			priv, encryptedKey, want := newECDHEncryptedKey(t, tt.algorithm)
			// newECDHEncryptedKey wraps a 16-byte session key.
			got, err := decryptECDHSessionKey(priv, encryptedKey, 16)
			if tt.wantError {
				require.ErrorIs(t, err, ErrDecryptionFailed)
				var unsupported *UnsupportedAlgorithmError
				require.ErrorAs(t, err, &unsupported)
				return
			}
			require.NoError(t, err)
			require.Equal(t, want, got)
		})
	}
}

func newECDHEncryptedKey(t *testing.T, algorithm string) (*ecdsa.PrivateKey, *EncryptedKey, []byte) {
	t.Helper()

	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	originatorKey, err := ecdh.P256().GenerateKey(rand.Reader)
	require.NoError(t, err)
	recipientECDH, err := privateKey.ECDH()
	require.NoError(t, err)
	sharedSecret, err := recipientECDH.ECDH(originatorKey.PublicKey())
	require.NoError(t, err)

	kekSize, err := keySizeForAlgorithm(paramKeyWrap, algorithm)
	require.NoError(t, err)
	params := &ConcatKDFParams{DigestMethod: DigestSHA256}
	kek, err := deriveConcatKDF(sharedSecret, params, kekSize)
	require.NoError(t, err)

	want := make([]byte, 16)
	_, err = rand.Read(want)
	require.NoError(t, err)
	cipherValue, err := aesKeyWrap(kek, want)
	require.NoError(t, err)

	return privateKey, &EncryptedKey{
		EncryptionMethod: &EncryptionMethod{Algorithm: algorithm},
		CipherValue:      cipherValue,
		AgreementMethod: &AgreementMethod{
			Algorithm: ECDHES,
			KeyDerivationMethod: &KeyDerivationMethod{
				Algorithm: ConcatKDF,
				ConcatKDF: params,
			},
			OriginatorKey: &ECKeyValue{
				Curve:     ecdh.P256(),
				PublicKey: originatorKey.PublicKey().Bytes(),
			},
		},
	}, want
}
