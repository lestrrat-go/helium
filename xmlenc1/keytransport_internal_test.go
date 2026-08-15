package xmlenc1

import (
	"crypto"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOAEPHashesSHA224(t *testing.T) {
	digest, mgf, err := oaepHashes(RSAOAEP11, DigestSHA224, MGFSHA224)
	require.NoError(t, err)
	require.Equal(t, crypto.SHA224, digest)
	require.Equal(t, crypto.SHA224, mgf)
}

func TestOAEPHashesLegacyRejectsExplicitSHA224MGF(t *testing.T) {
	_, _, err := oaepHashes(RSAOAEP, DigestSHA224, MGFSHA224)
	require.Error(t, err)
	var unsupported *UnsupportedAlgorithmError
	require.ErrorAs(t, err, &unsupported)
	require.Equal(t, paramMGF, unsupported.Parameter)
}
