package xmldsig1_test

import (
	"context"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"strings"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xmldsig1"
	"github.com/stretchr/testify/require"
)

// TestKeyMismatch exercises the ErrKeyMismatch branches of the signers and
// verifiers by passing a key of the wrong type.
func TestKeyMismatch(t *testing.T) {
	// sign exercises the ErrKeyMismatch branches of each algorithm-specific
	// signer by passing a key of the wrong type.
	t.Run("sign", func(t *testing.T) {
		rsaKey := generateRSAKey(t)
		ecKey := generateECDSAKey(t, elliptic.P256())
		edKey := generateEd25519Key(t)

		tests := []struct {
			name string
			alg  string
			key  any
		}{
			{"rsa-wrong", xmldsig1.AlgRSASHA256, ecKey},
			{"ecdsa-wrong", xmldsig1.AlgECDSASHA256, rsaKey},
			{"hmac-wrong", xmldsig1.AlgHMACSHA256, rsaKey},
			{"ed25519-wrong", xmldsig1.AlgEd25519, rsaKey},
			// pass a public ed25519 key where a private is required
			{"ed25519-pub", xmldsig1.AlgEd25519, edKey.Public()},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				doc := mustParseXML(t, samlAssertion)
				signer := xmldsig1.NewSigner().
					SignatureAlgorithm(tt.alg).
					Reference(xmldsig1.NewEnvelopedReference())
				err := signer.SignEnveloped(t.Context(), doc, doc.DocumentElement(), tt.key)
				require.Error(t, err)
				require.ErrorIs(t, err, xmldsig1.ErrKeyMismatch)
			})
		}
	})

	// verify exercises the ErrKeyMismatch branches of the verifiers by resolving
	// a wrong-typed key at verification time.
	t.Run("verify", func(t *testing.T) {
		tests := []struct {
			name      string
			alg       string
			signKey   any
			verifyKey any
		}{
			{"rsa", xmldsig1.AlgRSASHA256, mustRSA(t), "not-a-key"},
			{"ecdsa", xmldsig1.AlgECDSASHA256, mustEC(t), "not-a-key"},
			{"hmac", xmldsig1.AlgHMACSHA256, mustHMAC(t), "not-a-key"},
			{"ed25519", xmldsig1.AlgEd25519, mustEd(t), "not-a-key"},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				doc := mustParseXML(t, samlAssertion)
				signer := xmldsig1.NewSigner().
					SignatureAlgorithm(tt.alg).
					Reference(xmldsig1.NewEnvelopedReference())
				require.NoError(t, signer.SignEnveloped(t.Context(), doc, doc.DocumentElement(), tt.signKey))

				verifier := xmldsig1.NewVerifier(xmldsig1.StaticKey(tt.verifyKey))
				_, err := verifier.Verify(t.Context(), doc)
				require.Error(t, err)
				require.ErrorIs(t, err, xmldsig1.ErrKeyMismatch)
			})
		}
	})
}

// TestVerifyAlgorithm covers algorithm-specific verification branches.
func TestVerifyAlgorithm(t *testing.T) {
	// ed25519 with private key verifies the verifyEd25519 branch that accepts an
	// ed25519.PrivateKey and derives its public half.
	t.Run("ed25519 with private key", func(t *testing.T) {
		key := generateEd25519Key(t)
		doc := mustParseXML(t, samlAssertion)
		signer := xmldsig1.NewSigner().
			SignatureAlgorithm(xmldsig1.AlgEd25519).
			Reference(xmldsig1.NewEnvelopedReference())
		require.NoError(t, signer.SignEnveloped(t.Context(), doc, doc.DocumentElement(), key))

		// Resolve the private key (not the public) so the private-key branch runs.
		verifier := xmldsig1.NewVerifier(xmldsig1.StaticKey(key))
		_, err := verifier.Verify(t.Context(), doc)
		require.NoError(t, err)
	})

	// ed25519 bad signature exercises the ed25519 verification-failure branch by
	// tampering with a signed document.
	t.Run("ed25519 bad signature", func(t *testing.T) {
		key := generateEd25519Key(t)
		doc := mustParseXML(t, samlAssertion)
		signer := xmldsig1.NewSigner().
			SignatureAlgorithm(xmldsig1.AlgEd25519).
			Reference(xmldsig1.NewEnvelopedReference())
		require.NoError(t, signer.SignEnveloped(t.Context(), doc, doc.DocumentElement(), key))

		xml, err := helium.WriteString(doc)
		require.NoError(t, err)
		// flip a base64 char in the SignatureValue region by mutating issuer text,
		// which changes the canonicalized SignedInfo digest -> ed25519 verify fails.
		tampered := strings.Replace(xml, "https://idp.example.com", "https://idp.evil.com", 1)
		doc2 := mustParseXML(t, tampered)

		verifier := xmldsig1.NewVerifier(xmldsig1.StaticKey(key.Public()))
		_, err = verifier.Verify(t.Context(), doc2)
		require.Error(t, err)
	})

	// ecdsa wrong length signature exercises ecdsaRawToDER's invalid-length error
	// path. A P-384 signature presented to a P-256 verifier has the wrong length.
	t.Run("ecdsa wrong length signature", func(t *testing.T) {
		// Sign with P-384 then verify against a P-256 public key: the raw signature
		// length will not match the P-256 key size, hitting ecdsaRawToDER's error.
		key384 := generateECDSAKey(t, elliptic.P384())
		key256 := generateECDSAKey(t, elliptic.P256())
		doc := mustParseXML(t, samlAssertion)

		signer := xmldsig1.NewSigner().
			SignatureAlgorithm(xmldsig1.AlgECDSASHA256).
			Reference(xmldsig1.NewEnvelopedReference())
		// SignatureMethod ecdsa-sha256 with a P-384 key produces a 96-byte raw sig.
		require.NoError(t, signer.SignEnveloped(t.Context(), doc, doc.DocumentElement(), key384))

		verifier := xmldsig1.NewVerifier(xmldsig1.StaticKey(&key256.PublicKey))
		_, err := verifier.Verify(t.Context(), doc)
		require.Error(t, err)
	})

	// ecdsa tampered exercises verifyECDSA's VerifyASN1-failure path.
	t.Run("ecdsa tampered", func(t *testing.T) {
		key := generateECDSAKey(t, elliptic.P256())
		doc := mustParseXML(t, samlAssertion)
		signer := xmldsig1.NewSigner().
			SignatureAlgorithm(xmldsig1.AlgECDSASHA256).
			Reference(xmldsig1.NewEnvelopedReference())
		require.NoError(t, signer.SignEnveloped(t.Context(), doc, doc.DocumentElement(), key))

		xml, err := helium.WriteString(doc)
		require.NoError(t, err)
		tampered := strings.Replace(xml, "user@example.com", "attacker@evil.com", 1)
		doc2 := mustParseXML(t, tampered)

		verifier := xmldsig1.NewVerifier(xmldsig1.StaticKey(&key.PublicKey))
		_, err = verifier.Verify(t.Context(), doc2)
		require.Error(t, err)
	})
}

// TestUnsupportedAlgorithm covers the unknown-algorithm branches during signing.
func TestUnsupportedAlgorithm(t *testing.T) {
	// signature exercises lookupAlg's unknown-algorithm branch during signing.
	t.Run("signature", func(t *testing.T) {
		key := generateRSAKey(t)
		doc := mustParseXML(t, samlAssertion)
		signer := xmldsig1.NewSigner().
			SignatureAlgorithm("urn:made-up:alg").
			Reference(xmldsig1.NewEnvelopedReference())
		err := signer.SignEnveloped(t.Context(), doc, doc.DocumentElement(), key)
		require.ErrorIs(t, err, xmldsig1.ErrUnsupportedAlgorithm)
	})

	// digest exercises computeDigest's unknown-algorithm branch during signing.
	t.Run("digest", func(t *testing.T) {
		key := generateRSAKey(t)
		doc := mustParseXML(t, samlAssertion)
		signer := xmldsig1.NewSigner().
			SignatureAlgorithm(xmldsig1.AlgRSASHA256).
			Reference(xmldsig1.ReferenceConfig{
				URI:             "",
				DigestAlgorithm: "urn:made-up:digest",
				Transforms:      []xmldsig1.Transform{xmldsig1.Enveloped(), xmldsig1.ExcC14NTransform()},
			})
		err := signer.SignEnveloped(t.Context(), doc, doc.DocumentElement(), key)
		require.ErrorIs(t, err, xmldsig1.ErrUnsupportedAlgorithm)
	})
}

// TestKeySourceError exercises the path where ResolveKey returns an error.
func TestKeySourceError(t *testing.T) {
	key := generateRSAKey(t)
	doc := mustParseXML(t, samlAssertion)
	signer := xmldsig1.NewSigner().
		SignatureAlgorithm(xmldsig1.AlgRSASHA256).
		Reference(xmldsig1.NewEnvelopedReference())
	require.NoError(t, signer.SignEnveloped(t.Context(), doc, doc.DocumentElement(), key))

	sentinel := errors.New("resolve boom")
	ks := xmldsig1.KeySourceFunc(func(_ context.Context, _ *xmldsig1.KeyInfoData, _ string) (any, error) {
		return nil, sentinel
	})
	verifier := xmldsig1.NewVerifier(ks)
	_, err := verifier.Verify(t.Context(), doc)
	require.ErrorIs(t, err, sentinel)
}

// TestVerificationFailedSentinel confirms a failed cryptographic signature
// verification is classifiable via errors.Is(err, ErrVerificationFailed) for
// every signature algorithm, both on the raw algorithm error and through the
// *VerificationError (Reference == -1) returned by the public Verify. Each case
// signs with one key and verifies with a different, well-typed key so the
// SignatureValue check (which runs before any per-Reference digest check) is the
// failing step, without tripping ErrKeyMismatch. The RSA case additionally
// keeps rsa.ErrVerification observable so callers checking for it still can.
func TestVerificationFailedSentinel(t *testing.T) {
	ecB := generateECDSAKey(t, elliptic.P256())
	edB := generateEd25519Key(t)

	tests := []struct {
		name      string
		alg       string
		signKey   any
		verifyKey any
		cause     error
	}{
		{"rsa", xmldsig1.AlgRSASHA256, generateRSAKey(t), &generateRSAKey(t).PublicKey, rsa.ErrVerification},
		{"ecdsa", xmldsig1.AlgECDSASHA256, generateECDSAKey(t, elliptic.P256()), &ecB.PublicKey, nil},
		{"hmac", xmldsig1.AlgHMACSHA256, mustHMAC(t), mustHMAC(t), nil},
		{"ed25519", xmldsig1.AlgEd25519, generateEd25519Key(t), edB.Public(), nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := mustParseXML(t, samlAssertion)
			signer := xmldsig1.NewSigner().
				SignatureAlgorithm(tt.alg).
				Reference(xmldsig1.NewEnvelopedReference())
			require.NoError(t, signer.SignEnveloped(t.Context(), doc, doc.DocumentElement(), tt.signKey))

			verifier := xmldsig1.NewVerifier(xmldsig1.StaticKey(tt.verifyKey))
			_, err := verifier.Verify(t.Context(), doc)
			require.Error(t, err)
			require.ErrorIs(t, err, xmldsig1.ErrVerificationFailed)

			var verErr *xmldsig1.VerificationError
			require.ErrorAs(t, err, &verErr)
			require.Equal(t, -1, verErr.Reference)

			if tt.cause != nil {
				require.ErrorIs(t, err, tt.cause, "underlying cause must remain observable")
			}
		})
	}
}

func mustRSA(t *testing.T) *rsa.PrivateKey  { t.Helper(); return generateRSAKey(t) }
func mustEC(t *testing.T) *ecdsa.PrivateKey { t.Helper(); return generateECDSAKey(t, elliptic.P256()) }
func mustEd(t *testing.T) ed25519.PrivateKey {
	t.Helper()
	return generateEd25519Key(t)
}
func mustHMAC(t *testing.T) []byte {
	t.Helper()
	secret := make([]byte, 32)
	_, err := rand.Read(secret)
	require.NoError(t, err)
	return secret
}
