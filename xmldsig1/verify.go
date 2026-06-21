package xmldsig1

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/domutil"
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

	// Canonicalize SignedInfo.
	canonical, err := canonicalizeSubtree(parsed.c14nMethod, parsed.signedInfoElem, nil)
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

	// Check for enveloped-signature transform.
	hasEnveloped := false
	for _, t := range ref.transforms {
		if t.algorithm == TransformEnvelopedSignature {
			hasEnveloped = true
			break
		}
	}

	// Find the c14n method. Fail closed: any transform whose URI we cannot
	// apply must be rejected before digesting, otherwise a Reference could
	// declare an unsupported transform (XPath, Base64, custom URI) and still
	// verify against the untransformed canonical bytes.
	c14nMethod := ExcC14N10
	var prefixes []string
	for _, t := range ref.transforms {
		switch t.algorithm {
		case C14N10, C14N10Comments, ExcC14N10, ExcC14N10Comments, C14N11URI, C14N11Comments:
			c14nMethod = t.algorithm
			prefixes = t.prefixes
		case TransformEnvelopedSignature:
			// Handled by canonicalizeEnveloped below; no-op here.
		default:
			return nil, fmt.Errorf("%w: %s", ErrUnsupportedTransform, t.algorithm)
		}
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
			decoded, err := decodeBase64(domutil.TextContent(elem))
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
		case "SignatureMethod":
			alg, ok := e.GetAttribute("Algorithm")
			if !ok {
				return fmt.Errorf("%w: SignatureMethod missing Algorithm", ErrInvalidSignature)
			}
			parsed.signatureAlg = alg
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
				// Parse InclusiveNamespaces for Exclusive C14N.
				for inc := te.FirstChild(); inc != nil; inc = inc.NextSibling() {
					incElem, ok := helium.AsNode[*helium.Element](inc)
					if !ok {
						continue
					}
					// InclusiveNamespaces is an Exclusive XML Canonicalization
					// element and lives ONLY in the exc-c14n namespace
					// (http://www.w3.org/2001/10/xml-exc-c14n#), not the core
					// XML-Signature namespace. Matching on local name alone would
					// let a foreign-namespace look-alike inject a PrefixList and
					// alter which namespaces are canonicalized, so require the
					// exact exc-c14n namespace.
					if !isExcC14NNS(incElem) {
						continue
					}
					if domutil.LocalName(incElem) == "InclusiveNamespaces" {
						pl, _ := incElem.GetAttribute("PrefixList")
						if pl != "" {
							t.prefixes = strings.Fields(pl)
						}
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
			decoded, err := decodeBase64(domutil.TextContent(e))
			if err != nil {
				return ref, fmt.Errorf("%w: invalid DigestValue base64: %v", ErrInvalidSignature, err)
			}
			ref.digestValue = decoded
		}
	}
	return ref, nil
}
