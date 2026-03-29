package xmldsig1

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/hmac"
	"crypto/rsa"
	"encoding/asn1"
	"errors"
	"fmt"
	"math/big"
)

type signatureAlgorithm struct {
	hash crypto.Hash // 0 for Ed25519 (no pre-hash)
}

type digestAlgorithm struct {
	hash crypto.Hash
}

var signatureAlgorithms = map[string]signatureAlgorithm{
	AlgRSASHA1:     {hash: crypto.SHA1},
	AlgRSASHA256:   {hash: crypto.SHA256},
	AlgECDSASHA256: {hash: crypto.SHA256},
	AlgECDSASHA384: {hash: crypto.SHA384},
	AlgHMACSHA1:    {hash: crypto.SHA1},
	AlgHMACSHA256:  {hash: crypto.SHA256},
	AlgEd25519:     {hash: 0},
}

var digestAlgorithms = map[string]digestAlgorithm{
	DigestSHA1:   {hash: crypto.SHA1},
	DigestSHA256: {hash: crypto.SHA256},
	DigestSHA384: {hash: crypto.SHA384},
	DigestSHA512: {hash: crypto.SHA512},
}

func computeDigest(algURI string, data []byte) ([]byte, error) {
	da, ok := digestAlgorithms[algURI]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedAlgorithm, algURI)
	}
	h := da.hash.New()
	h.Write(data)
	return h.Sum(nil), nil
}

func signBytes(algURI string, key any, data []byte) ([]byte, error) {
	sa, ok := signatureAlgorithms[algURI]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedAlgorithm, algURI)
	}

	switch {
	case algURI == AlgEd25519:
		return signEd25519(key, data)
	case algURI == AlgHMACSHA1 || algURI == AlgHMACSHA256:
		return signHMAC(sa.hash, key, data)
	case algURI == AlgECDSASHA256 || algURI == AlgECDSASHA384:
		return signECDSA(sa.hash, key, data)
	default:
		return signRSA(sa.hash, key, data)
	}
}

func verifyBytes(algURI string, key any, data, sig []byte) error {
	sa, ok := signatureAlgorithms[algURI]
	if !ok {
		return fmt.Errorf("%w: %s", ErrUnsupportedAlgorithm, algURI)
	}

	switch {
	case algURI == AlgEd25519:
		return verifyEd25519(key, data, sig)
	case algURI == AlgHMACSHA1 || algURI == AlgHMACSHA256:
		return verifyHMAC(sa.hash, key, data, sig)
	case algURI == AlgECDSASHA256 || algURI == AlgECDSASHA384:
		return verifyECDSA(sa.hash, key, data, sig)
	default:
		return verifyRSA(sa.hash, key, data, sig)
	}
}

// RSA

func signRSA(hash crypto.Hash, key any, data []byte) ([]byte, error) {
	priv, ok := key.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("%w: expected *rsa.PrivateKey, got %T", ErrKeyMismatch, key)
	}
	h := hash.New()
	h.Write(data)
	return rsa.SignPKCS1v15(nil, priv, hash, h.Sum(nil))
}

func verifyRSA(hash crypto.Hash, key any, data, sig []byte) error {
	pub, ok := key.(*rsa.PublicKey)
	if !ok {
		return fmt.Errorf("%w: expected *rsa.PublicKey, got %T", ErrKeyMismatch, key)
	}
	h := hash.New()
	h.Write(data)
	return rsa.VerifyPKCS1v15(pub, hash, h.Sum(nil), sig)
}

// ECDSA

func signECDSA(hash crypto.Hash, key any, data []byte) ([]byte, error) {
	priv, ok := key.(*ecdsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("%w: expected *ecdsa.PrivateKey, got %T", ErrKeyMismatch, key)
	}
	h := hash.New()
	h.Write(data)
	derSig, err := ecdsa.SignASN1(nil, priv, h.Sum(nil))
	if err != nil {
		return nil, err
	}
	return ecdsaDERToRaw(derSig, curveKeySize(priv.Curve))
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
	h := hash.New()
	h.Write(data)
	if !ecdsa.VerifyASN1(pub, h.Sum(nil), derSig) {
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
	priv, ok := key.(ed25519.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("%w: expected ed25519.PrivateKey, got %T", ErrKeyMismatch, key)
	}
	return ed25519.Sign(priv, data), nil
}

func verifyEd25519(key any, data, sig []byte) error {
	pub, ok := key.(ed25519.PublicKey)
	if !ok {
		if priv, ok2 := key.(ed25519.PrivateKey); ok2 {
			pub = priv.Public().(ed25519.PublicKey)
		} else {
			return fmt.Errorf("%w: expected ed25519.PublicKey, got %T", ErrKeyMismatch, key)
		}
	}
	if !ed25519.Verify(pub, data, sig) {
		return errors.New("xmldsig1: ed25519 verification failed")
	}
	return nil
}
