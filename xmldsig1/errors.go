package xmldsig1

import (
	"errors"
	"fmt"
)

var (
	// ErrVerificationFailed is returned when signature verification fails.
	ErrVerificationFailed = errors.New("xmldsig1: verification failed")

	// ErrDigestMismatch is returned when a Reference digest does not match.
	ErrDigestMismatch = errors.New("xmldsig1: digest mismatch")

	// ErrSignatureNotFound is returned when no Signature element is found.
	ErrSignatureNotFound = errors.New("xmldsig1: signature element not found")

	// ErrUnsupportedAlgorithm is returned for unrecognized algorithm URIs.
	ErrUnsupportedAlgorithm = errors.New("xmldsig1: unsupported algorithm")

	// ErrUnsupportedTransform is returned for unrecognized transform URIs.
	ErrUnsupportedTransform = errors.New("xmldsig1: unsupported transform")

	// ErrKeyMismatch is returned when the key type does not match the algorithm.
	ErrKeyMismatch = errors.New("xmldsig1: key type does not match algorithm")

	// ErrNoReferences is returned when signing is attempted with no references.
	ErrNoReferences = errors.New("xmldsig1: no references configured")

	// ErrReferenceNotFound is returned when a Reference URI cannot be resolved.
	ErrReferenceNotFound = errors.New("xmldsig1: reference URI not resolved")

	// ErrInvalidKeyInfo is returned when KeyInfo content cannot be parsed.
	ErrInvalidKeyInfo = errors.New("xmldsig1: invalid KeyInfo")

	// ErrInvalidSignature is returned when the Signature element is malformed.
	ErrInvalidSignature = errors.New("xmldsig1: invalid signature structure")
)

// VerificationError provides details about which step of verification failed.
type VerificationError struct {
	// Reference is the 0-based index of the failing Reference, or -1 for
	// a SignatureValue failure.
	Reference int
	// URI is the Reference URI that failed (empty for SignatureValue).
	URI string
	// Err is the underlying cause.
	Err error
}

func (e *VerificationError) Error() string {
	if e.Reference < 0 {
		return fmt.Sprintf("xmldsig1: signature value verification failed: %v", e.Err)
	}
	return fmt.Sprintf("xmldsig1: reference %d (%q) verification failed: %v", e.Reference, e.URI, e.Err)
}

func (e *VerificationError) Unwrap() error {
	return e.Err
}
