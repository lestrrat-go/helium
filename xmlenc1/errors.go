package xmlenc1

import (
	"errors"
	"fmt"
)

var (
	// ErrDecryptionFailed is returned when decryption fails.
	ErrDecryptionFailed = errors.New("xmlenc1: decryption failed")

	// ErrEncryptionFailed is returned when encryption fails.
	ErrEncryptionFailed = errors.New("xmlenc1: encryption failed")

	// ErrMissingKey is returned when no decryption key is available.
	ErrMissingKey = errors.New("xmlenc1: no decryption key available")

	// ErrInvalidPadding is returned when PKCS#7 padding is invalid.
	ErrInvalidPadding = errors.New("xmlenc1: invalid PKCS#7 padding")

	// ErrKeyUnwrapFailed is returned when AES key unwrap integrity check fails.
	ErrKeyUnwrapFailed = errors.New("xmlenc1: AES key unwrap integrity check failed")

	// ErrMalformedEncrypted is returned when an EncryptedData element is malformed.
	ErrMalformedEncrypted = errors.New("xmlenc1: malformed EncryptedData element")

	// ErrMissingConfig is returned when required encryption config is missing.
	ErrMissingConfig = errors.New("xmlenc1: missing required configuration")

	// ErrCBCRequiresOptIn is returned when a Decryptor is asked to
	// decrypt an AES-CBC ciphertext but the caller has not opted in
	// to unauthenticated CBC via Decryptor.AllowUnauthenticatedCBC(true).
	//
	// AES-CBC under XML Encryption 1.0 is unauthenticated and is
	// vulnerable to padding-oracle attacks (Jager/Somorovsky 2011).
	// XML Encryption 1.1 deprecated CBC in favor of AES-GCM. Callers
	// that must interoperate with legacy CBC ciphertexts can opt in
	// after evaluating the attack surface (e.g. ensuring decryption
	// errors are not exposed to remote attackers).
	ErrCBCRequiresOptIn = errors.New("xmlenc1: AES-CBC decryption requires AllowUnauthenticatedCBC(true)")

	// ErrCBCEncryptionRequiresOptIn is returned when an Encryptor is
	// configured to emit a new AES-CBC ciphertext (via a CBC
	// BlockAlgorithm) but the caller has not opted in to legacy CBC
	// encryption via Encryptor.AllowLegacyCBC(true).
	//
	// The Encryptor defaults to AES-256-GCM (authenticated). AES-CBC
	// under XML Encryption 1.0 is unauthenticated and vulnerable to
	// padding-oracle attacks (Jager/Somorovsky 2011); XML Encryption
	// 1.1 deprecated it in favor of AES-GCM. Emitting new CBC
	// ciphertext therefore requires an explicit acknowledgement.
	ErrCBCEncryptionRequiresOptIn = errors.New("xmlenc1: AES-CBC encryption requires AllowLegacyCBC(true)")
)

// UnsupportedAlgorithmError is returned for unrecognized algorithm URIs.
type UnsupportedAlgorithmError struct {
	Algorithm string
}

func (e *UnsupportedAlgorithmError) Error() string {
	return fmt.Sprintf("xmlenc1: unsupported algorithm %q", e.Algorithm)
}

// KeySizeError is returned when a key (session key or key-encryption key)
// does not match the exact length required by its declared algorithm URI.
// It guards against algorithm/key-size confusion, e.g. declaring AES-256
// on the wire while supplying a 16-byte key that crypto/aes would silently
// treat as AES-128.
type KeySizeError struct {
	Algorithm string
	Want      int
	Got       int
}

func (e *KeySizeError) Error() string {
	return fmt.Sprintf("xmlenc1: algorithm %q requires a %d-byte key, got %d bytes", e.Algorithm, e.Want, e.Got)
}
