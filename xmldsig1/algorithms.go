package xmldsig1

import (
	"crypto"
	"crypto/dsa" // deprecated but standard; used verify-only for legacy XML-DSig interop (dsa-sha1). Signing is not offered.
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/hmac"
	"crypto/rand"
	"crypto/rsa"
	"encoding/asn1"
	"fmt"
	"math/big"
)

type algorithm struct {
	hash crypto.Hash // 0 for Ed25519 (no pre-hash)
	weak bool        // true for SHA-1-based algorithms (rejected by default)
}

var signatureAlgorithms = map[string]algorithm{
	AlgRSASHA1:     {hash: crypto.SHA1, weak: true},
	AlgRSASHA224:   {hash: crypto.SHA224},
	AlgRSASHA256:   {hash: crypto.SHA256},
	AlgRSASHA384:   {hash: crypto.SHA384},
	AlgRSASHA512:   {hash: crypto.SHA512},
	AlgECDSASHA1:   {hash: crypto.SHA1, weak: true},
	AlgECDSASHA224: {hash: crypto.SHA224},
	AlgECDSASHA256: {hash: crypto.SHA256},
	AlgECDSASHA384: {hash: crypto.SHA384},
	AlgECDSASHA512: {hash: crypto.SHA512},
	// DSA-SHA1 is verify-only legacy interop; signBytes rejects a signing
	// attempt with a clear DSA-unsupported error.
	AlgDSASHA1:    {hash: crypto.SHA1, weak: true},
	AlgHMACSHA1:   {hash: crypto.SHA1, weak: true},
	AlgHMACSHA224: {hash: crypto.SHA224},
	AlgHMACSHA256: {hash: crypto.SHA256},
	AlgHMACSHA384: {hash: crypto.SHA384},
	AlgHMACSHA512: {hash: crypto.SHA512},
	AlgEd25519:    {hash: 0},
}

var digestAlgorithms = map[string]algorithm{
	DigestSHA1:   {hash: crypto.SHA1, weak: true},
	DigestSHA224: {hash: crypto.SHA224},
	DigestSHA256: {hash: crypto.SHA256},
	DigestSHA384: {hash: crypto.SHA384},
	DigestSHA512: {hash: crypto.SHA512},
}

// lookupAlg resolves algURI in m, rejecting unknown algorithms with
// ErrUnsupportedAlgorithm and SHA-1-based ones with ErrWeakAlgorithm (unless
// allowSHA1). The unsupported check precedes the weak check.
func lookupAlg(m map[string]algorithm, algURI string, allowSHA1 bool) (crypto.Hash, error) {
	a, ok := m[algURI]
	if !ok {
		return 0, fmt.Errorf("%w: %s", ErrUnsupportedAlgorithm, algURI)
	}
	if a.weak && !allowSHA1 {
		return 0, fmt.Errorf("%w: %s", ErrWeakAlgorithm, algURI)
	}
	return a.hash, nil
}

// hashData returns hash(data).
func hashData(hash crypto.Hash, data []byte) []byte {
	h := hash.New()
	h.Write(data)
	return h.Sum(nil)
}

// computeDigest hashes data with the algorithm identified by algURI. SHA-1 is
// rejected with ErrWeakAlgorithm unless allowSHA1 is true.
func computeDigest(algURI string, data []byte, allowSHA1 bool) ([]byte, error) {
	hash, err := lookupAlg(digestAlgorithms, algURI, allowSHA1)
	if err != nil {
		return nil, err
	}
	return hashData(hash, data), nil
}

// signBytes signs data with the algorithm identified by algURI. SHA-1-based
// signature algorithms are rejected with ErrWeakAlgorithm unless allowSHA1 is
// true.
func signBytes(algURI string, key any, data []byte, allowSHA1 bool) ([]byte, error) {
	// DSA is verify-only legacy interop. It is present in the signature table so
	// the verify path recognizes it, so a signing attempt would otherwise fall
	// through to signRSA and fail with a confusing key-type mismatch. Reject it
	// here with a clear, DSA-specific message instead.
	if algURI == AlgDSASHA1 {
		return nil, fmt.Errorf("%w: DSA signing is not supported (dsa-sha1 is verify-only for legacy interop)", ErrUnsupportedAlgorithm)
	}

	hash, err := lookupAlg(signatureAlgorithms, algURI, allowSHA1)
	if err != nil {
		return nil, err
	}

	switch algURI {
	case AlgEd25519:
		return signEd25519(key, data)
	case AlgHMACSHA1, AlgHMACSHA224, AlgHMACSHA256, AlgHMACSHA384, AlgHMACSHA512:
		return signHMAC(hash, key, data)
	case AlgECDSASHA1, AlgECDSASHA224, AlgECDSASHA256, AlgECDSASHA384, AlgECDSASHA512:
		return signECDSA(hash, key, data)
	default:
		return signRSA(hash, key, data)
	}
}

// verifyBytes verifies sig over data with the algorithm identified by algURI.
// SHA-1-based signature algorithms are rejected with ErrWeakAlgorithm unless
// allowSHA1 is true.
func verifyBytes(algURI string, key any, data, sig []byte, allowSHA1 bool) error {
	hash, err := lookupAlg(signatureAlgorithms, algURI, allowSHA1)
	if err != nil {
		return err
	}

	switch algURI {
	case AlgEd25519:
		return verifyEd25519(key, data, sig)
	case AlgHMACSHA1, AlgHMACSHA224, AlgHMACSHA256, AlgHMACSHA384, AlgHMACSHA512:
		return verifyHMAC(hash, key, data, sig)
	case AlgECDSASHA1, AlgECDSASHA224, AlgECDSASHA256, AlgECDSASHA384, AlgECDSASHA512:
		return verifyECDSA(hash, key, data, sig)
	case AlgDSASHA1:
		return verifyDSA(hash, key, data, sig)
	default:
		return verifyRSA(hash, key, data, sig)
	}
}

// RSA

func signRSA(hash crypto.Hash, key any, data []byte) ([]byte, error) {
	if priv, ok := key.(*rsa.PrivateKey); ok {
		return rsa.SignPKCS1v15(nil, priv, hash, hashData(hash, data))
	}
	// Fall back to an opaque crypto.Signer (HSM/KMS/PKCS#11). For an
	// *rsa.PrivateKey backing, Sign with a crypto.Hash opts yields the same
	// deterministic PKCS1v15 signature as rsa.SignPKCS1v15 above; rand is
	// ignored for PKCS1v15.
	signer, ok := key.(crypto.Signer)
	if !ok {
		return nil, fmt.Errorf("%w: expected *rsa.PrivateKey, got %T", ErrKeyMismatch, key)
	}
	if _, ok := signer.Public().(*rsa.PublicKey); !ok {
		return nil, fmt.Errorf("%w: crypto.Signer public key is %T, not *rsa.PublicKey", ErrKeyMismatch, signer.Public())
	}
	return signer.Sign(rand.Reader, hashData(hash, data), hash)
}

func verifyRSA(hash crypto.Hash, key any, data, sig []byte) error {
	pub, ok := key.(*rsa.PublicKey)
	if !ok {
		return fmt.Errorf("%w: expected *rsa.PublicKey, got %T", ErrKeyMismatch, key)
	}
	if err := rsa.VerifyPKCS1v15(pub, hash, hashData(hash, data), sig); err != nil {
		return fmt.Errorf("%w: %w", ErrVerificationFailed, err)
	}
	return nil
}

// ECDSA

func signECDSA(hash crypto.Hash, key any, data []byte) ([]byte, error) {
	if priv, ok := key.(*ecdsa.PrivateKey); ok {
		derSig, err := ecdsa.SignASN1(nil, priv, hashData(hash, data))
		if err != nil {
			return nil, err
		}
		return ecdsaDERToRaw(derSig, curveKeySize(priv.Curve))
	}
	// Fall back to an opaque crypto.Signer (HSM/KMS/PKCS#11). Sign returns an
	// ASN.1 DER signature; convert it to the XML-DSig r||s format with the key
	// size taken from the signer's public key.
	signer, ok := key.(crypto.Signer)
	if !ok {
		return nil, fmt.Errorf("%w: expected *ecdsa.PrivateKey, got %T", ErrKeyMismatch, key)
	}
	pub, ok := signer.Public().(*ecdsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("%w: crypto.Signer public key is %T, not *ecdsa.PublicKey", ErrKeyMismatch, signer.Public())
	}
	derSig, err := signer.Sign(rand.Reader, hashData(hash, data), hash)
	if err != nil {
		return nil, err
	}
	return ecdsaDERToRaw(derSig, curveKeySize(pub.Curve))
}

func verifyECDSA(hash crypto.Hash, key any, data, sig []byte) error {
	pub, ok := key.(*ecdsa.PublicKey)
	if !ok {
		return fmt.Errorf("%w: expected *ecdsa.PublicKey, got %T", ErrKeyMismatch, key)
	}
	derSig, err := ecdsaRawToDER(sig, curveKeySize(pub.Curve))
	if err != nil {
		return err
	}
	if !ecdsa.VerifyASN1(pub, hashData(hash, data), derSig) {
		return ErrVerificationFailed
	}
	return nil
}

func curveKeySize(c elliptic.Curve) int {
	return (c.Params().BitSize + 7) / 8
}

// ecdsaDERToRaw converts an ASN.1 DER-encoded ECDSA signature to the
// XML DSig r||s concatenation format.
func ecdsaDERToRaw(der []byte, keySize int) ([]byte, error) {
	var sig struct {
		R, S *big.Int
	}
	if _, err := asn1.Unmarshal(der, &sig); err != nil {
		return nil, err
	}
	raw := make([]byte, keySize*2)
	rBytes := sig.R.Bytes()
	sBytes := sig.S.Bytes()
	copy(raw[keySize-len(rBytes):keySize], rBytes)
	copy(raw[2*keySize-len(sBytes):], sBytes)
	return raw, nil
}

// ecdsaRawToDER converts an XML DSig r||s concatenation to ASN.1 DER.
func ecdsaRawToDER(raw []byte, keySize int) ([]byte, error) {
	if len(raw) != keySize*2 {
		return nil, fmt.Errorf("xmldsig1: invalid ECDSA signature length: got %d, expected %d", len(raw), keySize*2)
	}
	var sig struct {
		R, S *big.Int
	}
	sig.R = new(big.Int).SetBytes(raw[:keySize])
	sig.S = new(big.Int).SetBytes(raw[keySize:])
	return asn1.Marshal(sig)
}

// DSA
//
// DSA is supported for VERIFICATION ONLY, for legacy XML-DSig interop
// (dsa-sha1). crypto/dsa is deprecated but remains the standard library type;
// an *dsa.PublicKey also arrives when x509.ParseCertificate parses a DSA
// certificate, so a cert-derived DSA key verifies through this same path.

func verifyDSA(hash crypto.Hash, key any, data, sig []byte) error {
	pub, ok := key.(*dsa.PublicKey)
	if !ok {
		return fmt.Errorf("%w: expected *dsa.PublicKey, got %T", ErrKeyMismatch, key)
	}
	// XML-DSig encodes a DSA signature as the fixed-width r||s concatenation,
	// each integer left-padded to the byte length of the subgroup order Q (20
	// bytes for a 160-bit Q). Split at the midpoint: both halves share Q's
	// width, so an odd total length cannot be a valid encoding.
	if len(sig) == 0 || len(sig)%2 != 0 {
		return fmt.Errorf("%w: DSA signature length %d is not a valid r||s encoding", ErrVerificationFailed, len(sig))
	}
	half := len(sig) / 2
	r := new(big.Int).SetBytes(sig[:half])
	s := new(big.Int).SetBytes(sig[half:])
	if !dsa.Verify(pub, hashData(hash, data), r, s) {
		return ErrVerificationFailed
	}
	return nil
}

// HMAC

func signHMAC(hash crypto.Hash, key any, data []byte) ([]byte, error) {
	secret, ok := key.([]byte)
	if !ok {
		return nil, fmt.Errorf("%w: expected []byte, got %T", ErrKeyMismatch, key)
	}
	mac := hmac.New(hash.New, secret)
	mac.Write(data)
	return mac.Sum(nil), nil
}

func verifyHMAC(hash crypto.Hash, key any, data, sig []byte) error {
	expected, err := signHMAC(hash, key, data)
	if err != nil {
		return err
	}
	if !hmac.Equal(expected, sig) {
		return ErrVerificationFailed
	}
	return nil
}

// Ed25519

func signEd25519(key any, data []byte) ([]byte, error) {
	if priv, ok := key.(ed25519.PrivateKey); ok {
		return ed25519.Sign(priv, data), nil
	}
	// Fall back to an opaque crypto.Signer (HSM/KMS/PKCS#11). A crypto.Hash(0)
	// opts selects pure Ed25519 over the raw message, matching ed25519.Sign;
	// the result is deterministic.
	signer, ok := key.(crypto.Signer)
	if !ok {
		return nil, fmt.Errorf("%w: expected ed25519.PrivateKey, got %T", ErrKeyMismatch, key)
	}
	if _, ok := signer.Public().(ed25519.PublicKey); !ok {
		return nil, fmt.Errorf("%w: crypto.Signer public key is %T, not ed25519.PublicKey", ErrKeyMismatch, signer.Public())
	}
	return signer.Sign(rand.Reader, data, crypto.Hash(0))
}

func verifyEd25519(key any, data, sig []byte) error {
	pub, ok := key.(ed25519.PublicKey)
	if !ok {
		if priv, ok2 := key.(ed25519.PrivateKey); ok2 {
			var ok3 bool
			pub, ok3 = priv.Public().(ed25519.PublicKey)
			if !ok3 {
				return fmt.Errorf("%w: expected ed25519.PublicKey, got %T", ErrKeyMismatch, key)
			}
		} else {
			return fmt.Errorf("%w: expected ed25519.PublicKey, got %T", ErrKeyMismatch, key)
		}
	}
	if !ed25519.Verify(pub, data, sig) {
		return fmt.Errorf("%w: ed25519 verification failed", ErrVerificationFailed)
	}
	return nil
}
