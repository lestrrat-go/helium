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

	// ErrWeakAlgorithm is returned when a SHA-1-based signature or digest
	// algorithm is encountered while SHA-1 is not allowed. SHA-1 is rejected
	// by default; opt in with Verifier.AllowSHA1(true) (for verification) or
	// Signer.AllowSHA1(true) (for signing) to accept it for legacy interop.
	ErrWeakAlgorithm = errors.New("xmldsig1: weak algorithm SHA-1 not allowed")

	// ErrUnsupportedTransform is returned for unrecognized transform URIs.
	ErrUnsupportedTransform = errors.New("xmldsig1: unsupported transform")

	// ErrKeyMismatch is returned when the key type does not match the algorithm.
	ErrKeyMismatch = errors.New("xmldsig1: key type does not match algorithm")

	// ErrNoReferences is returned when signing is attempted with no references.
	ErrNoReferences = errors.New("xmldsig1: no references configured")

	// ErrReferenceNotFound is returned when a Reference URI cannot be resolved.
	ErrReferenceNotFound = errors.New("xmldsig1: reference URI not resolved")

	// ErrAmbiguousReference is returned when a Reference URI resolves to more
	// than one element. This is the primary defense against XML Signature
	// Wrapping (XSW) attacks where an attacker injects a duplicate-ID element
	// containing malicious content alongside the legitimately signed element.
	ErrAmbiguousReference = errors.New("xmldsig1: reference URI matches multiple elements")

	// ErrAmbiguousSignature is returned when the document contains more than
	// one Signature element and Verify cannot decide which one to verify.
	// Callers must use VerifyElement to disambiguate.
	ErrAmbiguousSignature = errors.New("xmldsig1: document contains multiple Signature elements")

	// ErrInvalidKeyInfo is returned when KeyInfo content cannot be parsed.
	ErrInvalidKeyInfo = errors.New("xmldsig1: invalid KeyInfo")

	// ErrInvalidSignature is returned when the Signature element is malformed.
	ErrInvalidSignature = errors.New("xmldsig1: invalid signature structure")

	// ErrNoKeySource is returned when a Verifier was created with a nil
	// KeySource and verification is attempted. Without a KeySource there is no
	// way to resolve a verification key, so this is rejected before any key
	// resolution rather than panicking on a nil dereference.
	ErrNoKeySource = errors.New("xmldsig1: no key source configured")
)

// opSign is the ReferenceError.Op value for a signing-side per-reference failure.
const opSign = "sign"

// ReferenceError identifies which Reference failed during a signing operation.
// A per-reference failure carries the reference's 0-based index and URI so a
// caller signing over a multi-reference configuration can pinpoint the offending
// Reference, symmetric with how VerificationError reports a verification-side
// per-reference failure. The underlying cause stays reachable via errors.Is and
// errors.As (Unwrap), so a bare sentinel such as ErrReferenceNotFound or
// ErrUnsupportedTransform remains matchable through the wrapper.
type ReferenceError struct {
	// Op is the operation during which the failure occurred ("sign").
	Op string
	// Reference is the 0-based index of the failing Reference.
	Reference int
	// URI is the Reference URI that failed.
	URI string
	// Err is the underlying cause.
	Err error
}

func (e *ReferenceError) Error() string {
	return fmt.Sprintf("xmldsig1: %s reference %d (%q) failed: %v", e.Op, e.Reference, e.URI, e.Err)
}

func (e *ReferenceError) Unwrap() error {
	return e.Err
}

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
