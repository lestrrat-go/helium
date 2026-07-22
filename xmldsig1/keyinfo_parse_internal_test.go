package xmldsig1

import (
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseKeyName(t *testing.T) {
	t.Run("value trimmed", func(t *testing.T) {
		doc := mustParse(t, `<ds:KeyInfo xmlns:ds="`+NamespaceDSig+`"><ds:KeyName>  alpha-key
	</ds:KeyName><ds:KeyName>beta</ds:KeyName></ds:KeyInfo>`)
		data, err := parseKeyInfo(t.Context(), newVerifyBudget(&verifierConfig{}), doc.DocumentElement())
		require.NoError(t, err)
		require.Equal(t, []string{"alpha-key", "beta"}, data.KeyNames)
	})

	t.Run("foreign namespace ignored", func(t *testing.T) {
		doc := mustParse(t, `<ds:KeyInfo xmlns:ds="`+NamespaceDSig+`" xmlns:evil="urn:evil"><evil:KeyName>attacker</evil:KeyName></ds:KeyInfo>`)
		data, err := parseKeyInfo(t.Context(), newVerifyBudget(&verifierConfig{}), doc.DocumentElement())
		require.NoError(t, err)
		require.Empty(t, data.KeyNames)
	})
}

func TestParseX509SKI(t *testing.T) {
	t.Run("decodes raw bytes", func(t *testing.T) {
		raw := []byte{0xDE, 0xAD, 0xBE, 0xEF}
		enc := base64.StdEncoding.EncodeToString(raw)
		doc := mustParse(t, `<ds:KeyInfo xmlns:ds="`+NamespaceDSig+`"><ds:X509Data><ds:X509SKI>`+enc+`</ds:X509SKI></ds:X509Data></ds:KeyInfo>`)
		data, err := parseKeyInfo(t.Context(), newVerifyBudget(&verifierConfig{}), doc.DocumentElement())
		require.NoError(t, err)
		require.Len(t, data.X509SKIs, 1)
		require.Equal(t, raw, data.X509SKIs[0])
	})

	t.Run("bad base64 fails closed", func(t *testing.T) {
		doc := mustParse(t, `<ds:KeyInfo xmlns:ds="`+NamespaceDSig+`"><ds:X509Data><ds:X509SKI>not!base64!</ds:X509SKI></ds:X509Data></ds:KeyInfo>`)
		_, err := parseKeyInfo(t.Context(), newVerifyBudget(&verifierConfig{}), doc.DocumentElement())
		require.ErrorIs(t, err, ErrInvalidKeyInfo)
	})

	t.Run("foreign namespace ignored", func(t *testing.T) {
		doc := mustParse(t, `<ds:KeyInfo xmlns:ds="`+NamespaceDSig+`" xmlns:evil="urn:evil"><ds:X509Data><evil:X509SKI>3q2+7w==</evil:X509SKI></ds:X509Data></ds:KeyInfo>`)
		data, err := parseKeyInfo(t.Context(), newVerifyBudget(&verifierConfig{}), doc.DocumentElement())
		require.NoError(t, err)
		require.Empty(t, data.X509SKIs)
	})
}
