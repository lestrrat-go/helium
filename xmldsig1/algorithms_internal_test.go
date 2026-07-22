package xmldsig1

import (
	"encoding/asn1"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestECDSADERToRawRejectsOversizedComponent proves ecdsaDERToRaw returns a
// typed error rather than panicking when a signer hands back an r or s value
// wider than the curve's byte width. A well-formed signer emits r and s in
// [1, n), each fitting keySize bytes; a buggy HSM/PKCS#11 plugin could emit an
// oversized component. Without the width guard, keySize-len(rBytes) goes negative
// and the copy into the fixed-width buffer panics with a slice-bounds error.
func TestECDSADERToRawRejectsOversizedComponent(t *testing.T) {
	const keySize = 32 // P-256 curve byte width

	// A 33-byte magnitude: one byte wider than the 32-byte P-256 width. Its
	// big-endian Bytes() length is 33, so keySize-33 is negative.
	oversized := new(big.Int).Lsh(big.NewInt(1), 8*33-1)
	require.Len(t, oversized.Bytes(), 33, "test fixture must be one byte too wide")

	cases := map[string]struct{ r, s *big.Int }{
		"oversized r": {r: oversized, s: big.NewInt(1)},
		"oversized s": {r: big.NewInt(1), s: oversized},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			der, err := asn1.Marshal(struct{ R, S *big.Int }{R: tc.r, S: tc.s})
			require.NoError(t, err)

			require.NotPanics(t, func() {
				raw, err := ecdsaDERToRaw(der, keySize)
				require.ErrorIs(t, err, ErrInvalidSignature)
				require.Nil(t, raw)
			})
		})
	}

	// Sanity: an in-width signature still converts to the fixed 2*keySize buffer.
	t.Run("in-width components convert", func(t *testing.T) {
		der, err := asn1.Marshal(struct{ R, S *big.Int }{R: big.NewInt(0x1234), S: big.NewInt(0x5678)})
		require.NoError(t, err)
		raw, err := ecdsaDERToRaw(der, keySize)
		require.NoError(t, err)
		require.Len(t, raw, keySize*2)
	})
}
