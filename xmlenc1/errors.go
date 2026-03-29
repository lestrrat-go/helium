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
)

// UnsupportedAlgorithmError is returned for unrecognized algorithm URIs.
type UnsupportedAlgorithmError struct {
	Algorithm string
}

func (e *UnsupportedAlgorithmError) Error() string {
	return fmt.Sprintf("xmlenc1: unsupported algorithm %q", e.Algorithm)
}
