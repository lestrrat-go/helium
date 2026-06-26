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
				t := parsedTransform{algorithm: alg}
				// Parse InclusiveNamespaces for Exclusive C14N. A
				// foreign-namespace look-alike contributes no prefixes (it is
				// silently ignored here rather than rejected, because an unknown
				// child of a per-Reference Transform is not necessarily fatal).
				var hasInclusiveNS bool
				for inc := te.FirstChild(); inc != nil; inc = inc.NextSibling() {
					incElem, ok := helium.AsNode[*helium.Element](inc)
					if !ok {
						continue
					}
					if px, ok := excInclusiveNamespacePrefixes(incElem); ok {
						t.prefixes = px
						hasInclusiveNS = true
					}
				}
				// ec:InclusiveNamespaces is an Exclusive C14N parameter; it is
				// only honored for the exclusive c14n transforms. Under any other
				// transform algorithm (C14N10/C14N11/enveloped-signature/...) the
				// prefixes are silently dropped during canonicalization, so the
				// Reference would digest bytes that differ from what the signer
				// declared — a fail-open gap. Reject it fail-closed, including an
				// empty PrefixList, mirroring the SignedInfo gating.
				if hasInclusiveNS {
					if err := gateInclusiveNamespaces(alg); err != nil {
						return ref, err
					}
				}
				ref.transforms = append(ref.transforms, t)
			}
		case "DigestMethod":
			alg, ok := e.GetAttribute("Algorithm")
			if !ok {
				return ref, fmt.Errorf("%w: DigestMethod missing Algorithm", ErrInvalidSignature)
			}
			ref.digestAlgorithm = alg
		case "DigestValue":
			decoded, err := xmlbase64.DecodeString(domutil.TextContent(e))
			if err != nil {
				return ref, fmt.Errorf("%w: invalid DigestValue base64: %v", ErrInvalidSignature, err)
			}
			ref.digestValue = decoded
		}
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
//
// ec:InclusiveNamespaces is an Exclusive XML Canonicalization parameter and is
// only meaningful for the exclusive c14n algorithms (ExcC14N10 /
// ExcC14N10Comments); canonicalize() only honors the PrefixList for exclusive
// modes. If it appeared under a non-exclusive method (C14N10/C14N11), the
// verifier would silently ignore it, canonicalizing SignedInfo differently from
// what the signer declared. To keep parameter handling fail-closed, reject an
// ec:InclusiveNamespaces parameter on any non-exclusive c14n method.
func parseCanonicalizationParameters(elem *helium.Element, alg string) ([]string, error) {
	var prefixes []string
	for c := elem.FirstChild(); c != nil; c = c.NextSibling() {
		ce, ok := helium.AsNode[*helium.Element](c)
		if !ok {
			continue
		}
		px, matched := excInclusiveNamespacePrefixes(ce)
		if !matched {
			return nil, fmt.Errorf("%w: unsupported CanonicalizationMethod parameter %s", ErrUnsupportedTransform, domutil.LocalName(ce))
		}
		if err := gateInclusiveNamespaces(alg); err != nil {
			return nil, err
		}
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
