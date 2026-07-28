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

	// ErrTooManyEncryptedKeys is returned when an EncryptedData carries more
	// EncryptedKey candidates than the Decryptor's effective limit, which
	// guards against CPU amplification (DoS). Decryptor.MaxEncryptedKeys owns
	// the cap: the per-candidate cost it bounds and the effective-limit
	// rules. See also DefaultMaxEncryptedKeys.
	ErrTooManyEncryptedKeys = errors.New("xmlenc1: too many EncryptedKey candidates")

	// ErrEncryptedKeyBytesExceeded is returned when the EncryptedKey
	// candidates of one EncryptedData carry more ciphertext together than
	// the Decryptor's effective byte budget, which guards against memory
	// amplification (DoS). Decryptor.MaxEncryptedKeyBytes owns the budget:
	// what it covers, when it is charged, and the effective-limit rules.
	// See also DefaultMaxEncryptedKeyBytes.
	//
	// It is distinct from ErrTooManyEncryptedKeys because the two bound
	// different things and are raised or lifted by different setters: too
	// many candidates is a count, too much ciphertext is a size, and a
	// caller handling one must be able to tell which limit to raise.
	ErrEncryptedKeyBytesExceeded = errors.New("xmlenc1: EncryptedKey ciphertext exceeds the byte budget")

	// ErrInvalidPadding names invalid PKCS#7 padding. Decryption never
	// returns it: distinguishing a padding failure from any other CBC
	// failure is exactly what a padding oracle needs, so decryptCBC
	// collapses every cause to ErrDecryptionFailed. It remains exported so
	// that decryptCipherValue can defensively squash it should any future
	// code path start returning it, and so tests can assert it never
	// escapes.
	//
	// Deprecated: matching on this sentinel never succeeds. Test for
	// ErrDecryptionFailed instead.
	ErrInvalidPadding = errors.New("xmlenc1: invalid PKCS#7 padding")

	// ErrKeyUnwrapFailed is returned when AES key unwrap integrity check
	// fails. It is always wrapped in ErrDecryptionFailed, so a caller that
	// tests only for ErrDecryptionFailed catches a failed key unwrap the
	// same way it catches a failed RSA key transport.
	ErrKeyUnwrapFailed = errors.New("xmlenc1: AES key unwrap integrity check failed")

	// ErrMalformedEncrypted is returned when an EncryptedData element is malformed.
	ErrMalformedEncrypted = errors.New("xmlenc1: malformed EncryptedData element")

	// ErrMissingConfig is returned when required encryption config is missing.
	ErrMissingConfig = errors.New("xmlenc1: missing required configuration")

	// ErrConflictingKeyConfig is returned when an Encryptor configures two of
	// the three ways to protect the session key: RSA key transport
	// (KeyTransportAlgorithm + RecipientPublicKey), ECDH-ES key agreement
	// (KeyWrapAlgorithm + RecipientECPublicKey), and AES key wrapping
	// (KeyWrapAlgorithm + KeyEncryptionKey). Any pair of them fails with this
	// error, naming both of the configured things so the caller knows which
	// two to choose between. A SessionKey alongside a single mechanism is
	// not a pair: it supplies the key that mechanism protects.
	//
	// An EncryptedData carries a single EncryptedKey here, so honoring one
	// mechanism means silently discarding the other — and a recipient
	// holding only the discarded key then fails to decrypt with an error
	// that points nowhere near the real mistake. The caller must pick one.
	ErrConflictingKeyConfig = errors.New("xmlenc1: conflicting key protection configured")

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

// Parameter roles reported by UnsupportedAlgorithmError and KeySizeError.
// They name the configuration knob (or the wire attribute it maps to) that
// the rejected value came from, so an error identifies which setter to fix
// rather than only which URI was refused. They are diagnostic text: match
// on the error type and its Algorithm field, not on these strings.
const (
	paramBlockAlgorithm = "block algorithm"
	paramKeyTransport   = "key transport algorithm"
	paramKeyWrap        = "key wrap algorithm"
	paramOAEPDigest     = "OAEP digest algorithm"
	paramMGF            = "MGF algorithm"
	paramKeyAgreement   = "key agreement algorithm"
	paramKeyDerivation  = "key derivation algorithm"
	paramConcatKDF      = "ConcatKDF digest algorithm"
	paramSessionKey     = "session key"
	paramKEK            = "key-encryption key"
)

// UnsupportedAlgorithmError is returned for unrecognized algorithm URIs.
//
// Construct it with keyed fields only, as in
// &UnsupportedAlgorithmError{Algorithm: uri}. The field set grows as the
// diagnostics improve, and every such addition stops an unkeyed composite
// literal from compiling. That break is accepted under this package's
// EXPERIMENTAL, pre-1.0 API posture; keyed literals are unaffected and are
// the supported form.
type UnsupportedAlgorithmError struct {
	// Parameter names the algorithm slot that rejected the URI, e.g.
	// "block algorithm" or "MGF algorithm". It is diagnostic text and may
	// be empty when the slot is not known at the point of failure.
	Parameter string
	Algorithm string
}

func (e *UnsupportedAlgorithmError) Error() string {
	if e.Parameter == "" {
		return fmt.Sprintf("xmlenc1: unsupported algorithm %q", e.Algorithm)
	}
	return fmt.Sprintf("xmlenc1: unsupported %s %q", e.Parameter, e.Algorithm)
}

// KeySizeError is returned when a key (session key or key-encryption key)
// does not match the exact length required by its declared algorithm URI.
// It guards against algorithm/key-size confusion, e.g. declaring AES-256
// on the wire while supplying a 16-byte key that crypto/aes would silently
// treat as AES-128.
//
// Construct it with keyed fields only, as in
// &KeySizeError{Algorithm: uri, Want: want, Got: got}; see
// UnsupportedAlgorithmError for why unkeyed literals are unsupported here.
type KeySizeError struct {
	// Key names the key that was the wrong length, e.g. "session key" or
	// "key-encryption key". It is diagnostic text and may be empty when
	// the role is not known at the point of failure.
	Key       string
	Algorithm string
	Want      int
	Got       int
}

func (e *KeySizeError) Error() string {
	key := e.Key
	if key == "" {
		key = "key"
	}
	return fmt.Sprintf("xmlenc1: algorithm %q requires a %d-byte %s, got %d bytes", e.Algorithm, e.Want, key, e.Got)
}
