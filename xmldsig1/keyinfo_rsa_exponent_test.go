package xmldsig1

import (
	"encoding/base64"
	"fmt"
	"math/big"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

// rsaKeyInfoXML builds a ds:KeyInfo document carrying an RSAKeyValue with the
// given base64-encoded modulus and exponent.
func rsaKeyInfoXML(modulus, exponent string) string {
	return fmt.Sprintf(`<ds:KeyInfo xmlns:ds="%s">`+
		`<ds:KeyValue><ds:RSAKeyValue>`+
		`<ds:Modulus>%s</ds:Modulus>`+
		`<ds:Exponent>%s</ds:Exponent>`+
		`</ds:RSAKeyValue></ds:KeyValue></ds:KeyInfo>`, NamespaceDSig, modulus, exponent)
}

func TestParseKeyInfo_RSAExponentRange(t *testing.T) {
	modulus := base64.StdEncoding.EncodeToString(big.NewInt(3233).Bytes())

	encode := func(n *big.Int) string {
		return base64.StdEncoding.EncodeToString(n.Bytes())
	}

	t.Run("valid 65537", func(t *testing.T) {
		xml := rsaKeyInfoXML(modulus, "AQAB") // 65537
		doc, err := helium.NewParser().Parse(t.Context(), []byte(xml))
		require.NoError(t, err)

		data, err := parseKeyInfo(doc.DocumentElement())
		require.NoError(t, err)
		require.NotNil(t, data.RSAKeyValue)
		require.Equal(t, 65537, data.RSAKeyValue.Exponent)
	})

	oversized := map[string]*big.Int{
		"2^63": new(big.Int).Lsh(big.NewInt(1), 63),
	}
	// 2^64 + 65537
	bigVal := new(big.Int).Lsh(big.NewInt(1), 64)
	bigVal.Add(bigVal, big.NewInt(65537))
	oversized["2^64+65537"] = bigVal

	for name, val := range oversized {
		t.Run("oversized "+name, func(t *testing.T) {
			xml := rsaKeyInfoXML(modulus, encode(val))
			doc, err := helium.NewParser().Parse(t.Context(), []byte(xml))
			require.NoError(t, err)

			_, err = parseKeyInfo(doc.DocumentElement())
			require.Error(t, err)
			require.ErrorIs(t, err, ErrInvalidKeyInfo)
		})
	}

	t.Run("zero exponent rejected", func(t *testing.T) {
		xml := rsaKeyInfoXML(modulus, encode(big.NewInt(0)))
		doc, err := helium.NewParser().Parse(t.Context(), []byte(xml))
		require.NoError(t, err)

		_, err = parseKeyInfo(doc.DocumentElement())
		require.Error(t, err)
		require.ErrorIs(t, err, ErrInvalidKeyInfo)
	})
}
