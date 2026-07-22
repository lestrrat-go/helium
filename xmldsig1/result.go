package xmldsig1

import (
	helium "github.com/lestrrat-go/helium"
)

// VerifiedReference describes a single Reference that was successfully
// verified. Callers should use this to confirm the *Element they are about
// to consume from the document is actually covered by the signature —
// guarding against XML Signature Wrapping (XSW) attacks.
type VerifiedReference struct {
	// URI is the value of the Reference URI attribute as it appeared in the
	// signed document.
	URI string

	// Element is the element that the URI resolved to at verification time.
	// For the enveloped pattern (URI=""), this is the document element.
	// For fragment references (URI="#id"), this is the unique element with
	// that Id/ID attribute. If duplicate matches existed, verification fails
	// with ErrAmbiguousReference before this field is populated.
	//
	// Element is nil for an External reference: an external resource is a byte
	// stream outside the document, not an in-document element.
	Element *helium.Element

	// External reports whether this Reference was satisfied via a configured
	// ReferenceResolver (its URI is not a same-document form). An external
	// reference covers content outside the document, so Element is nil and
	// neither Covers nor SignedElement ever attributes in-document coverage to
	// it — a caller confirming a specific *Element was signed must rely on a
	// same-document reference, never an external one.
	External bool

	// DigestAlgorithm is the algorithm URI declared in the DigestMethod
	// element (e.g. DigestSHA256).
	DigestAlgorithm string

	// Type is the value of the Reference Type attribute as it appeared in the
	// signed document, or "" when the attribute was absent. A Reference whose
	// Type is TypeManifest points at a ds:Manifest element; when
	// Verifier.ValidateManifests(true) is set, that Manifest's inner references
	// are reported in VerifyResult.Manifests.
	Type string
}

// ManifestResult reports the outcome of validating the inner ds:Reference
// children of a ds:Manifest element. It is populated only when
// Verifier.ValidateManifests(true) is set and the top-level Manifest-typed
// Reference's own digest verified. Inner-reference results are ADVISORY: per
// XMLDSig core §5.1 the application decides Manifest policy, so a failed inner
// reference does not fail Verify — the top-level Manifest Reference's digest is
// what the signature commits to. Coverage attribution (VerifyResult.Covers /
// SignedElement) is never made through a Manifest.
type ManifestResult struct {
	// Reference is the top-level VerifiedReference whose Type is TypeManifest —
	// the reference the signature actually commits to. It points into
	// VerifyResult.References.
	Reference *VerifiedReference

	// Element is the ds:Manifest element that Reference resolved to.
	Element *helium.Element

	// References lists the result of digesting each inner ds:Reference child of
	// the Manifest, in document order. Only one level is walked: a Manifest
	// Reference nested inside this Manifest is digested but not recursively
	// expanded.
	References []ManifestReference
}

// ManifestReference reports the outcome of digesting a single inner
// ds:Reference child of a ds:Manifest. Valid reports whether the inner
// reference's recomputed digest matched its DigestValue; Err carries the reason
// it could not be validated (a resolution, transform, digest, or mismatch
// error). Both are advisory — see ManifestResult.
type ManifestReference struct {
	// URI is the value of the inner Reference URI attribute.
	URI string

	// DigestAlgorithm is the algorithm URI declared in the inner Reference's
	// DigestMethod element. It is "" when the inner Reference could not be
	// parsed.
	DigestAlgorithm string

	// Element is the element the inner Reference URI resolved to, or nil for an
	// external reference or when resolution failed.
	Element *helium.Element

	// Valid reports whether the inner Reference's recomputed digest matched its
	// declared DigestValue.
	Valid bool

	// Err is the reason the inner Reference could not be validated, or nil when
	// Valid is true.
	Err error
}

// VerifyResult is returned by Verifier.Verify and Verifier.VerifyElement on
// success. It exposes the set of elements that were actually covered by the
// signature so callers can correlate signed content with the element they
// intend to consume.
type VerifyResult struct {
	// Signature is the Signature element that was verified.
	Signature *helium.Element

	// References lists every Reference inside SignedInfo that was
	// successfully verified, in document order.
	References []VerifiedReference

	// Manifests lists, for each top-level Reference whose Type is TypeManifest,
	// the result of digesting that Manifest's inner references. It is populated
	// only when Verifier.ValidateManifests(true) is set; otherwise it is nil.
	// Inner-reference results are ADVISORY (see ManifestResult): they never
	// affect whether Verify succeeds, nor do they contribute to Covers or
	// SignedElement coverage attribution.
	Manifests []ManifestResult
}

// SignedElement returns the resolved element for the Reference with the
// given URI, or nil if no such Reference was verified. This is the
// preferred way to confirm an element was covered by the signature before
// consuming it. An External reference is skipped: it resolves to bytes outside
// the document, not an element, so it can never satisfy an in-document
// element lookup.
func (r *VerifyResult) SignedElement(uri string) *helium.Element {
	if r == nil {
		return nil
	}
	for _, ref := range r.References {
		if ref.External {
			continue
		}
		if ref.URI == uri {
			return ref.Element
		}
	}
	return nil
}

// Covers reports whether elem was covered by any verified Reference. An External
// reference never counts: it covers content outside the document, so it cannot
// attest that an in-document *Element was signed.
func (r *VerifyResult) Covers(elem *helium.Element) bool {
	if r == nil || elem == nil {
		return false
	}
	for _, ref := range r.References {
		if ref.External {
			continue
		}
		if ref.Element == elem {
			return true
		}
	}
	return false
}
