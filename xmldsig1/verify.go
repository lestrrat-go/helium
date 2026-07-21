package xmldsig1

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/domutil"
	"github.com/lestrrat-go/helium/internal/xmlbase64"
)

// isNilKeySource reports whether a KeySource is effectively nil. A plain
// `cfg.keySource == nil` only catches an untyped-nil interface; a typed-nil
// pointer (e.g. `var ks *myKeySource; NewVerifier(ks)`) yields a non-nil
// interface whose underlying value is nil, so calling ResolveKey on it would
// panic on the nil-receiver dereference. Detect that case reflectively for any
// nil-capable underlying kind.
func isNilKeySource(ks KeySource) bool {
	if ks == nil {
		return true
	}
	v := reflect.ValueOf(ks)
	switch v.Kind() {
	case reflect.Pointer, reflect.Map, reflect.Chan, reflect.Func, reflect.Slice, reflect.Interface:
		return v.IsNil()
	default:
		return false
	}
}

// parsedSignature holds the parsed structure of a ds:Signature element.
type parsedSignature struct {
	signedInfoElem *helium.Element
	c14nMethod     string
	c14nPrefixes   []string // ec:InclusiveNamespaces PrefixList on CanonicalizationMethod
	signatureAlg   string
	references     []parsedReference
	signatureValue []byte
	keyInfoElem    *helium.Element
}

type parsedReference struct {
	uri             string
	digestAlgorithm string
	digestValue     []byte
	transforms      []parsedTransform
}

type parsedTransform struct {
	algorithm string
	prefixes  []string // for Exclusive C14N InclusiveNamespaces
}

func verifySignature(ctx context.Context, cfg *verifierConfig, doc *helium.Document, sigElem *helium.Element) (*VerifyResult, error) {
	// Honor an already-cancelled or already-expired context before any work. All
	// of the pre-loop steps below — signature-element parse, weak-algorithm
	// preflight, KeyInfo parse (x509.ParseCertificate per cert), KeySource
	// resolution, SignedInfo canonicalization, and one SignatureValue crypto
	// verify — are bounded but non-trivial, so a context the caller cancelled
	// before calling must short-circuit here rather than run them to completion.
	// The per-Reference loop below repeats this check each iteration.
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// A zero-value Verifier{} constructed directly (bypassing NewVerifier) has
	// a nil cfg, and a nil KeySource (e.g. NewVerifier(nil)) cannot resolve a
	// key. isNilKeySource also catches a typed-nil pointer KeySource, whose
	// non-nil interface would otherwise slip past a plain == nil check and panic
	// inside ResolveKey below. Reject all of these up front so config-controlled
	// cases return a typed error instead of panicking on a nil dereference.
	if cfg == nil || isNilKeySource(cfg.keySource) {
		return nil, ErrNoKeySource
	}

	parsed, err := parseSignatureElement(sigElem)
	if err != nil {
		return nil, err
	}

	// Reject weak (SHA-1) signature/digest algorithms before resolving KeyInfo
	// or invoking KeySource, so a rejected SHA-1 input returns ErrWeakAlgorithm
	// without triggering key resolution or surfacing unrelated key/signature
	// errors.
	if err := preflightParsedWeakAlgorithms(parsed, cfg.allowSHA1); err != nil {
		return nil, err
	}

	// Resolve key.
	var keyInfoData *KeyInfoData
	if parsed.keyInfoElem != nil {
		keyInfoData, err = parseKeyInfo(parsed.keyInfoElem)
		if err != nil {
			return nil, err
		}
	}

	key, err := cfg.keySource.ResolveKey(ctx, keyInfoData, parsed.signatureAlg)
	if err != nil {
		return nil, err
	}

	// Canonicalize SignedInfo, honoring any ec:InclusiveNamespaces PrefixList
	// declared on its CanonicalizationMethod (relevant for Exclusive C14N).
	canonical, err := canonicalizeSubtree(parsed.c14nMethod, parsed.signedInfoElem, parsed.c14nPrefixes)
	if err != nil {
		return nil, err
	}

	// Verify signature value. SHA-1-based signature algorithms are rejected
	// here unless the caller opted in via Verifier.AllowSHA1(true).
	if err := verifyBytes(parsed.signatureAlg, key, canonical, parsed.signatureValue, cfg.allowSHA1); err != nil {
		return nil, &VerificationError{Reference: -1, Err: err}
	}

	// Verify each reference and record the resolved element so callers can
	// confirm that the element they intend to consume is actually covered.
	result := &VerifyResult{Signature: sigElem}
	for i, ref := range parsed.references {
		// Honor context cancellation between references: a signature with very
		// many References must not be digested to completion once the caller's
		// context is cancelled or its deadline has passed.
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		target, err := verifyReference(doc, sigElem, ref, cfg.allowSHA1)
		if err != nil {
			return nil, &VerificationError{Reference: i, URI: ref.uri, Err: err}
		}
		result.References = append(result.References, VerifiedReference{
			URI:             ref.uri,
			Element:         target,
			DigestAlgorithm: ref.digestAlgorithm,
		})
	}

	return result, nil
}

func verifyReference(doc *helium.Document, sigElem *helium.Element, ref parsedReference, allowSHA1 bool) (*helium.Element, error) {
	target, err := resolveReference(doc, ref.uri)
	if err != nil {
		return nil, err
	}

	// Interpret the transforms as an ordered pipeline. Fail closed: any
	// transform whose URI we cannot apply, or one ordered after an
	// octet-producing c14n transform, is rejected before digesting — otherwise a
	// Reference could declare an unsupported or mis-ordered transform and still
	// verify against the untransformed canonical bytes. When no c14n transform is
	// declared the default node-set->octet conversion is inclusive Canonical
	// XML 1.0.
	steps := make([]transformStep, len(ref.transforms))
	for i, t := range ref.transforms {
		steps[i] = transformStep(t)
	}
	c14nMethod, prefixes, hasEnveloped, err := resolveTransformPipeline(steps)
	if err != nil {
		return nil, err
	}

	// For enveloped signatures the Signature element and its descendants must
	// be omitted from the canonical input. canonicalizeEnveloped does this on a
	// deep copy of the document, never mutating the caller's live DOM (which
	// would race with concurrent readers and risk leaving the tree corrupted if
	// a restore failed).
	var canonical []byte
	switch {
	case hasEnveloped:
		canonical, err = canonicalizeEnveloped(c14nMethod, doc, target, sigElem, ref.uri == "", prefixes)
	case ref.uri == "":
		canonical, err = canonicalize(c14nMethod, doc, prefixes)
	default:
		canonical, err = canonicalizeSubtree(c14nMethod, target, prefixes)
	}
	if err != nil {
		return nil, err
	}

	// Compute and compare digest. A SHA-1 digest is rejected unless the
	// caller opted in via Verifier.AllowSHA1(true).
	computed, err := computeDigest(ref.digestAlgorithm, canonical, allowSHA1)
	if err != nil {
		return nil, err
	}

	if !digestEqual(computed, ref.digestValue) {
		return nil, ErrDigestMismatch
	}

	return target, nil
}

func digestEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	// Constant-time comparison to avoid timing attacks.
	var v byte
	for i := range a {
		v |= a[i] ^ b[i]
	}
	return v == 0
}

func parseSignatureElement(sigElem *helium.Element) (*parsedSignature, error) {
	parsed := &parsedSignature{}

	// The XML-Signature schema mandates exactly one SignedInfo and exactly one
	// SignatureValue per ds:Signature. This MUST be enforced rather than
	// last-one-wins: only a single SignedInfo is canonicalized and checked
	// against SignatureValue (see verifySignature), yet every SignedInfo's
	// References were being appended to the result. An attacker could prepend
	// a second, UNSIGNED SignedInfo whose References carry attacker-computed,
	// self-consistent digests; those References would then be reported as
	// verified even though they were never covered by the signature. Reject
	// duplicate SignedInfo / SignatureValue / KeyInfo outright.
	var signatureValueSeen bool
	for child := sigElem.FirstChild(); child != nil; child = child.NextSibling() {
		elem, ok := helium.AsNode[*helium.Element](child)
		if !ok {
			continue
		}
		// Only elements in the XML-Signature namespace count as core signature
		// children. Matching on local name alone lets a foreign-namespace
		// element masquerade as a core child (e.g. an <evil:Reference> passing
		// the at-least-one-Reference check), so namespace must be enforced. Core
		// children live ONLY in the core xmldsig# namespace; the 1.1 xmldsig11#
		// namespace is for new 1.1 elements and must not satisfy this check.
		if !isDSigCoreNS(elem) {
			continue
		}
		switch domutil.LocalName(elem) {
		case "SignedInfo":
			if parsed.signedInfoElem != nil {
				return nil, fmt.Errorf("%w: multiple SignedInfo elements", ErrInvalidSignature)
			}
			parsed.signedInfoElem = elem
			if err := parseSignedInfo(elem, parsed); err != nil {
				return nil, err
			}
		case "SignatureValue":
			if signatureValueSeen {
				return nil, fmt.Errorf("%w: multiple SignatureValue elements", ErrInvalidSignature)
			}
			signatureValueSeen = true
			decoded, err := xmlbase64.DecodeString(domutil.TextContent(elem))
			if err != nil {
				return nil, fmt.Errorf("%w: invalid SignatureValue base64: %v", ErrInvalidSignature, err)
			}
			parsed.signatureValue = decoded
		case "KeyInfo":
			if parsed.keyInfoElem != nil {
				return nil, fmt.Errorf("%w: multiple KeyInfo elements", ErrInvalidSignature)
			}
			parsed.keyInfoElem = elem
		}
	}

	if parsed.signedInfoElem == nil {
		return nil, fmt.Errorf("%w: missing SignedInfo", ErrInvalidSignature)
	}
	if !signatureValueSeen {
		return nil, fmt.Errorf("%w: missing SignatureValue", ErrInvalidSignature)
	}

	return parsed, nil
}

func parseSignedInfo(elem *helium.Element, parsed *parsedSignature) error {
	// The XML-Signature schema fixes SignedInfo's content model as
	// (CanonicalizationMethod, SignatureMethod, Reference+) with exactly one
	// CanonicalizationMethod and exactly one SignatureMethod. Enforce that
	// cardinality rather than accepting duplicates last-one-wins: a crafted
	// SignedInfo carrying two SignatureMethod (or CanonicalizationMethod)
	// children is schema-invalid and ambiguous about which algorithm the
	// signature actually commits to, so a conforming verifier must reject it.
	var c14nSeen, sigMethodSeen bool
	for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
		e, ok := helium.AsNode[*helium.Element](child)
		if !ok {
			continue
		}
		// Require the XML-Signature namespace: a foreign-namespace
		// <evil:Reference> must not be counted toward the mandatory
		// at-least-one-Reference rule below, which would otherwise re-open the
		// no-content-signature bypass. Only the core xmldsig# namespace counts;
		// the 1.1 xmldsig11# namespace must not satisfy this check.
		if !isDSigCoreNS(e) {
			continue
		}
		switch domutil.LocalName(e) {
		case "CanonicalizationMethod":
			if c14nSeen {
				return fmt.Errorf("%w: multiple CanonicalizationMethod elements", ErrInvalidSignature)
			}
			c14nSeen = true
			alg, ok := e.GetAttribute("Algorithm")
			if !ok {
				return fmt.Errorf("%w: CanonicalizationMethod missing Algorithm", ErrInvalidSignature)
			}
			parsed.c14nMethod = alg
			prefixes, err := parseCanonicalizationParameters(e, alg)
			if err != nil {
				return err
			}
			parsed.c14nPrefixes = prefixes
		case "SignatureMethod":
			if sigMethodSeen {
				return fmt.Errorf("%w: multiple SignatureMethod elements", ErrInvalidSignature)
			}
			sigMethodSeen = true
			alg, ok := e.GetAttribute("Algorithm")
			if !ok {
				return fmt.Errorf("%w: SignatureMethod missing Algorithm", ErrInvalidSignature)
			}
			parsed.signatureAlg = alg
			if err := rejectSignatureMethodParameters(e); err != nil {
				return err
			}
		case "Reference":
			ref, err := parseReferenceElement(e)
			if err != nil {
				return err
			}
			parsed.references = append(parsed.references, ref)
		}
	}

	// SignedInfo's content model fixes CanonicalizationMethod and
	// SignatureMethod at exactly one each, not merely at most one. Enforcing
	// only "at most one" lets a SignedInfo missing either element parse OK and
	// fail much later — as an unsupported-algorithm error, sometimes only after
	// key resolution — instead of as a clean ErrInvalidSignature. Reject the
	// absence here so a structurally invalid SignedInfo never reaches
	// canonicalization or key resolution.
	if !c14nSeen {
		return fmt.Errorf("%w: missing CanonicalizationMethod", ErrInvalidSignature)
	}
	if !sigMethodSeen {
		return fmt.Errorf("%w: missing SignatureMethod", ErrInvalidSignature)
	}

	// XML-Signature requires at least one Reference. A SignatureValue computed
	// over a reference-free SignedInfo verifies cryptographically yet covers no
	// document content, so accepting it would attest to nothing. Reject it.
	if len(parsed.references) == 0 {
		return fmt.Errorf("%w: SignedInfo has no Reference", ErrInvalidSignature)
	}
	return nil
}

func parseReferenceElement(elem *helium.Element) (parsedReference, error) {
	ref := parsedReference{}
	ref.uri, _ = elem.GetAttribute("URI")

	// The XML-Signature schema fixes Reference's content model as
	// (Transforms?, DigestMethod, DigestValue) with at most one Transforms and
	// exactly one DigestMethod and one DigestValue. Enforce that cardinality
	// rather than accepting duplicates last-one-wins: a crafted Reference with
	// two DigestValue children (the second crafted to match the recomputed
	// digest) is schema-invalid and ambiguous about which digest the signature
	// commits to, so a conforming verifier must reject it.
	var transformsSeen, digestMethodSeen, digestValueSeen bool
	for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
		e, ok := helium.AsNode[*helium.Element](child)
		if !ok {
			continue
		}
		// Core Reference children (Transforms/DigestMethod/DigestValue) must be
		// in the core XML-Signature namespace; do not honor foreign-namespace
		// look-alikes, and the 1.1 xmldsig11# namespace must not satisfy this
		// check.
		if !isDSigCoreNS(e) {
			continue
		}
		switch domutil.LocalName(e) {
		case "Transforms":
			if transformsSeen {
				return ref, fmt.Errorf("%w: multiple Transforms elements", ErrInvalidSignature)
			}
			transformsSeen = true
			for tc := e.FirstChild(); tc != nil; tc = tc.NextSibling() {
				te, ok := helium.AsNode[*helium.Element](tc)
				if !ok {
					continue
				}
				// A Transform element must be in the core XML-Signature
				// namespace; do not honor foreign-namespace look-alikes (e.g.
				// <evil:Transform Algorithm="...">), and the 1.1 xmldsig11#
				// namespace must not satisfy this check. Its InclusiveNamespaces
				// child lives in the xml-exc-c14n namespace and is handled
				// separately below.
				if !isDSigCoreNS(te) {
					continue
				}
				if domutil.LocalName(te) != "Transform" {
					continue
				}
				alg, _ := te.GetAttribute("Algorithm")
				// Validate the Transform's child elements by algorithm. For a
				// supported transform those children are algorithm parameters;
				// accepting an unknown one while digesting as if it were absent
				// would be fail-open. The only parameter helium honors is
				// ec:InclusiveNamespaces under the exclusive c14n transforms;
				// every other child — and ec:InclusiveNamespaces under a
				// non-exclusive algorithm (enveloped-signature/C14N10/C14N11/...)
				// — is rejected fail-closed, mirroring the SignedInfo
				// CanonicalizationMethod handling.
				prefixes, err := parseInclusiveNamespaceParameters(te, alg, "Transform")
				if err != nil {
					return ref, err
				}
				ref.transforms = append(ref.transforms, parsedTransform{algorithm: alg, prefixes: prefixes})
			}
		case "DigestMethod":
			if digestMethodSeen {
				return ref, fmt.Errorf("%w: multiple DigestMethod elements", ErrInvalidSignature)
			}
			digestMethodSeen = true
			alg, ok := e.GetAttribute("Algorithm")
			if !ok {
				return ref, fmt.Errorf("%w: DigestMethod missing Algorithm", ErrInvalidSignature)
			}
			ref.digestAlgorithm = alg
		case "DigestValue":
			if digestValueSeen {
				return ref, fmt.Errorf("%w: multiple DigestValue elements", ErrInvalidSignature)
			}
			digestValueSeen = true
			decoded, err := xmlbase64.DecodeString(domutil.TextContent(e))
			if err != nil {
				return ref, fmt.Errorf("%w: invalid DigestValue base64: %v", ErrInvalidSignature, err)
			}
			ref.digestValue = decoded
		}
	}

	// Reference's content model fixes DigestMethod and DigestValue at exactly
	// one each, not merely at most one. Enforcing only "at most one" lets a
	// Reference missing either element parse OK and fail much later — a missing
	// DigestMethod surfaces as an unsupported-digest error and a missing
	// DigestValue as a digest mismatch (the empty digest never matches) —
	// instead of as a clean ErrInvalidSignature. Reject the absence here.
	if !digestMethodSeen {
		return ref, fmt.Errorf("%w: missing DigestMethod", ErrInvalidSignature)
	}
	if !digestValueSeen {
		return ref, fmt.Errorf("%w: missing DigestValue", ErrInvalidSignature)
	}
	return ref, nil
}

// excInclusiveNamespacePrefixes reports whether elem is an ec:InclusiveNamespaces
// element. InclusiveNamespaces is an Exclusive XML Canonicalization element and
// lives ONLY in the exc-c14n namespace (http://www.w3.org/2001/10/xml-exc-c14n#),
// not the core XML-Signature namespace. Matching on local name alone would let a
// foreign-namespace look-alike inject a PrefixList and alter which namespaces are
// canonicalized, so the exact exc-c14n namespace is required. When it matches, the
// PrefixList attribute is split into its individual prefixes.
func excInclusiveNamespacePrefixes(elem *helium.Element) ([]string, bool) {
	if !isExcC14NNS(elem) || domutil.LocalName(elem) != "InclusiveNamespaces" {
		return nil, false
	}
	pl, _ := elem.GetAttribute("PrefixList")
	if pl == "" {
		return nil, true
	}
	return strings.Fields(pl), true
}

// parseCanonicalizationParameters extracts the ec:InclusiveNamespaces PrefixList
// from a CanonicalizationMethod element and fails closed on any other child
// element, which would be a canonicalization parameter we cannot honor. Silently
// ignoring an unknown parameter would canonicalize SignedInfo differently from
// what the signer intended, so it is rejected.
func parseCanonicalizationParameters(elem *helium.Element, alg string) ([]string, error) {
	return parseInclusiveNamespaceParameters(elem, alg, "CanonicalizationMethod")
}

// parseInclusiveNamespaceParameters validates the child elements of a
// CanonicalizationMethod (SignedInfo) or per-Reference Transform element and
// returns the ec:InclusiveNamespaces PrefixList when one is present. It is the
// single fail-closed gate for both call sites so they behave identically.
//
// The only honored child is ec:InclusiveNamespaces, and only under the
// exclusive c14n algorithms (ExcC14N10 / ExcC14N10Comments); canonicalize()
// honors the PrefixList for exclusive modes alone. Under any other algorithm
// (enveloped-signature/C14N10/C14N11/...) the prefixes are silently dropped
// during canonicalization, so an ec:InclusiveNamespaces there — even with an
// empty PrefixList — is rejected. Any other child element is an unknown
// parameter we cannot honor; accepting it while digesting as if absent would be
// fail-open, so it too is rejected. A second ec:InclusiveNamespaces is rejected
// rather than silently letting the last one win. The context label
// ("CanonicalizationMethod" / "Transform") only shapes the error message.
func parseInclusiveNamespaceParameters(elem *helium.Element, alg, context string) ([]string, error) {
	var prefixes []string
	var seen bool
	for c := elem.FirstChild(); c != nil; c = c.NextSibling() {
		ce, ok := helium.AsNode[*helium.Element](c)
		if !ok {
			continue
		}
		px, matched := excInclusiveNamespacePrefixes(ce)
		if !matched {
			return nil, fmt.Errorf("%w: unsupported %s parameter %s", ErrUnsupportedTransform, context, domutil.LocalName(ce))
		}
		if err := gateInclusiveNamespaces(alg); err != nil {
			return nil, err
		}
		if seen {
			return nil, fmt.Errorf("%w: multiple ec:InclusiveNamespaces under %s", ErrUnsupportedTransform, context)
		}
		seen = true
		prefixes = px
	}
	return prefixes, nil
}

// gateInclusiveNamespaces rejects an ec:InclusiveNamespaces PrefixList declared
// under a non-exclusive c14n algorithm. The PrefixList is only honored for the
// exclusive c14n modes (ExcC14N10 / ExcC14N10Comments); on any other algorithm
// (C14N10/C14N11/enveloped-signature/...) the prefixes are silently dropped
// during canonicalization, so accepting one would canonicalize differently from
// what the signer declared. Shared by the SignedInfo CanonicalizationMethod and
// per-Reference Transform gating so both fail closed identically.
func gateInclusiveNamespaces(alg string) error {
	if alg != ExcC14N10 && alg != ExcC14N10Comments {
		return fmt.Errorf("%w: ec:InclusiveNamespaces is only valid for exclusive c14n, not %s", ErrUnsupportedTransform, alg)
	}
	return nil
}

// rejectSignatureMethodParameters fails closed on any child element of
// SignatureMethod. The only standard child is ds:HMACOutputLength, which requests
// a truncated HMAC; helium always computes and compares the full-length MAC, so a
// truncation request is unsupported. Silently ignoring such a parameter would
// verify against bytes that differ from what the signer intended, so any
// SignatureMethod parameter is rejected.
func rejectSignatureMethodParameters(elem *helium.Element) error {
	for c := elem.FirstChild(); c != nil; c = c.NextSibling() {
		ce, ok := helium.AsNode[*helium.Element](c)
		if !ok {
			continue
		}
		return fmt.Errorf("%w: unsupported SignatureMethod parameter %s", ErrUnsupportedAlgorithm, domutil.LocalName(ce))
	}
	return nil
}
