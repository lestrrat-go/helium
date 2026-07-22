package xmldsig1_test

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"testing"
	"time"

	"github.com/lestrrat-go/helium/xmldsig1"
	"github.com/stretchr/testify/require"
)

// poolCert returns a throwaway self-signed certificate whose SubjectKeyId is
// skid, for exercising the X509CertPoolKeySource selection paths.
func poolCert(t *testing.T, cn string, serial int64, skid []byte) *x509.Certificate {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(serial),
		Subject:      pkix.Name{CommonName: cn},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		SubjectKeyId: skid,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	require.NoError(t, err)
	cert, err := x509.ParseCertificate(der)
	require.NoError(t, err)
	return cert
}

func TestKeyByNameSource(t *testing.T) {
	key := &rsa.PublicKey{N: big.NewInt(65537), E: 3}
	ks := xmldsig1.KeyByNameSource(map[string]any{"signing-key": key})

	t.Run("hit returns mapped key", func(t *testing.T) {
		got, err := ks.ResolveKey(t.Context(), &xmldsig1.KeyInfoData{KeyNames: []string{"other", "signing-key"}}, "")
		require.NoError(t, err)
		require.Same(t, key, got)
	})

	t.Run("miss fails closed", func(t *testing.T) {
		_, err := ks.ResolveKey(t.Context(), &xmldsig1.KeyInfoData{KeyNames: []string{"unknown"}}, "")
		require.ErrorIs(t, err, xmldsig1.ErrNoKeySource)
	})

	t.Run("nil KeyInfo fails closed", func(t *testing.T) {
		_, err := ks.ResolveKey(t.Context(), nil, "")
		require.ErrorIs(t, err, xmldsig1.ErrNoKeySource)
	})
}

func TestX509CertPoolKeySource(t *testing.T) {
	skid := []byte{0x01, 0x02, 0x03, 0x04}
	cert := poolCert(t, "pool-signer", 777, skid)
	other := poolCert(t, "pool-other", 888, []byte{0x09})
	ks := xmldsig1.X509CertPoolKeySource(other, cert)

	t.Run("raw certificate match", func(t *testing.T) {
		got, err := ks.ResolveKey(t.Context(), &xmldsig1.KeyInfoData{X509Certificates: []*x509.Certificate{cert}}, "")
		require.NoError(t, err)
		require.Same(t, cert.PublicKey, got)
	})

	t.Run("SKI match", func(t *testing.T) {
		got, err := ks.ResolveKey(t.Context(), &xmldsig1.KeyInfoData{X509SKIs: [][]byte{skid}}, "")
		require.NoError(t, err)
		require.Same(t, cert.PublicKey, got)
	})

	t.Run("issuer serial match", func(t *testing.T) {
		got, err := ks.ResolveKey(t.Context(), &xmldsig1.KeyInfoData{
			X509IssuerSerials: []*xmldsig1.X509IssuerSerial{
				{IssuerName: cert.Issuer.String(), SerialNumber: cert.SerialNumber},
			},
		}, "")
		require.NoError(t, err)
		require.Same(t, cert.PublicKey, got)
	})

	t.Run("subject name match", func(t *testing.T) {
		got, err := ks.ResolveKey(t.Context(), &xmldsig1.KeyInfoData{X509SubjectNames: []string{cert.Subject.String()}}, "")
		require.NoError(t, err)
		require.Same(t, cert.PublicKey, got)
	})

	t.Run("no selector fails closed", func(t *testing.T) {
		_, err := ks.ResolveKey(t.Context(), &xmldsig1.KeyInfoData{X509SKIs: [][]byte{{0xFF}}}, "")
		require.ErrorIs(t, err, xmldsig1.ErrNoKeySource)
	})

	t.Run("nil KeyInfo fails closed", func(t *testing.T) {
		_, err := ks.ResolveKey(t.Context(), nil, "")
		require.ErrorIs(t, err, xmldsig1.ErrNoKeySource)
	})
}
