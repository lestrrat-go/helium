package xmlenc1_test

import (
	"testing"

	"github.com/lestrrat-go/helium/xmldsig1"
	"github.com/lestrrat-go/helium/xmlenc1"
	"github.com/stretchr/testify/require"
)

// SHA-384 has no identifier in the 2001 xmlenc namespace, unlike SHA-256 and
// SHA-512, so DigestSHA384 must use the xmldsig-more URI: the same one
// DigestSHA384DSigMore names and the same one xmldsig1.DigestSHA384 already
// uses.
func TestDigestSHA384URI(t *testing.T) {
	const want = "http://www.w3.org/2001/04/xmldsig-more#sha384"
	require.Equal(t, want, xmlenc1.DigestSHA384)
	require.Equal(t, xmlenc1.DigestSHA384DSigMore, xmlenc1.DigestSHA384)
	require.Equal(t, xmldsig1.DigestSHA384, xmlenc1.DigestSHA384)
}
