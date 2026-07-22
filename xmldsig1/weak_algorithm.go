package xmldsig1

import "fmt"

// rejectWeakSignatureAlgorithm returns ErrWeakAlgorithm if algURI names a
// SHA-1-based signature algorithm and allowSHA1 is false. Unknown algorithms
// pass through here (they are rejected later by signBytes/verifyBytes with
// ErrUnsupportedAlgorithm); this gate is concerned solely with the weak-algorithm
// policy.
func rejectWeakSignatureAlgorithm(algURI string, allowSHA1 bool) error {
	sa, ok := signatureAlgorithms[algURI]
	if !ok {
		return nil
	}
	if sa.weak && !allowSHA1 {
		return fmt.Errorf("%w: %s", ErrWeakAlgorithm, algURI)
	}
	return nil
}

// rejectWeakDigestAlgorithm returns ErrWeakAlgorithm if algURI names a
// SHA-1-based digest algorithm and allowSHA1 is false. Unknown algorithms pass
// through here (they are rejected later by computeDigest).
func rejectWeakDigestAlgorithm(algURI string, allowSHA1 bool) error {
	da, ok := digestAlgorithms[algURI]
	if !ok {
		return nil
	}
	if da.weak && !allowSHA1 {
		return fmt.Errorf("%w: %s", ErrWeakAlgorithm, algURI)
	}
	return nil
}

// preflightSignerWeakAlgorithms validates the signer's weak-algorithm policy
// BEFORE any DOM mutation or node moves. Every sign entry point calls this
// first so that a rejected default-SHA-1 request returns ErrWeakAlgorithm
// without moving caller content into an <Object>, adding a Signature element,
// or otherwise mutating the input tree.
func preflightSignerWeakAlgorithms(cfg *signerConfig) error {
	if err := rejectWeakSignatureAlgorithm(cfg.signatureAlgorithm, cfg.allowSHA1); err != nil {
		return err
	}
	for i, ref := range cfg.references {
		if err := rejectWeakDigestAlgorithm(ref.DigestAlgorithm, cfg.allowSHA1); err != nil {
			// Carry the failing reference's index and URI so a caller signing
			// over a multi-reference configuration can pinpoint the offending
			// Reference, symmetric with the per-reference digest and transform
			// loops. The ErrWeakAlgorithm sentinel stays matchable via errors.Is
			// through ReferenceError.Unwrap.
			return &ReferenceError{Op: opSign, Reference: i, URI: ref.URI, Err: err}
		}
	}
	return nil
}

// preflightParsedWeakAlgorithms validates a parsed signature's weak-algorithm
// policy BEFORE KeyInfo/key resolution. The verify path calls this right after
// parsing so that a rejected SHA-1 input returns ErrWeakAlgorithm without
// invoking KeySource or surfacing unrelated key/signature errors.
func preflightParsedWeakAlgorithms(parsed *parsedSignature, allowSHA1 bool) error {
	if err := rejectWeakSignatureAlgorithm(parsed.signatureAlg, allowSHA1); err != nil {
		return err
	}
	for i, ref := range parsed.references {
		if err := rejectWeakDigestAlgorithm(ref.digestAlgorithm, allowSHA1); err != nil {
			// Carry the failing reference's index and URI so a caller verifying a
			// multi-reference signature can pinpoint the offending Reference,
			// symmetric with the per-reference digest loop. The ErrWeakAlgorithm
			// sentinel stays matchable via errors.Is through VerificationError.Unwrap.
			return &VerificationError{Reference: i, URI: ref.uri, Err: err}
		}
	}
	return nil
}
