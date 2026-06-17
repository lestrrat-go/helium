package xmldsig1

import (
	"context"
	"errors"
	"fmt"
	"strings"

	helium "github.com/lestrrat-go/helium"
)

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
	parsed, err := parseSignatureElement(sigElem)
	if err != nil {
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

	// Verify signature value.
	if err := verifyBytes(parsed.signatureAlg, key, canonical, parsed.signatureValue); err != nil {
		return nil, &VerificationError{Reference: -1, Err: err}
	}

	// Verify each reference and record the resolved element so callers can
	// confirm that the element they intend to consume is actually covered.
	result := &VerifyResult{Signature: sigElem}
	for i, ref := range parsed.references {
		target, err := verifyReference(doc, sigElem, ref)
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

func verifyReference(doc *helium.Document, sigElem *helium.Element, ref parsedReference) (*helium.Element, error) {
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

	// Temporarily detach the Signature element for enveloped signatures,
	// remembering its exact position so we can restore it after
	// canonicalization. Naive reattach-via-AddChild would move the
	// Signature to the end of its parent and silently restructure the doc.
	var anchor sigAnchor
	if hasEnveloped {
		anchor = captureAnchor(sigElem)
		helium.UnlinkNode(sigElem)
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
			// Handled in the separate enveloped-signature pass above; no-op here.
		default:
			// Reattach the Signature element we detached above before
			// bailing out, so a rejected reference does not leave the
			// document structurally modified. The rejection is the primary
			// error, but if restore itself fails the document is left mutated,
			// so surface that failure too (joined) rather than dropping it
			// silently. errors.Is(ErrUnsupportedTransform) still holds.
			rejectErr := fmt.Errorf("%w: %s", ErrUnsupportedTransform, t.algorithm)
			if hasEnveloped {
				if rerr := anchor.restore(sigElem); rerr != nil {
					rejectErr = errors.Join(rejectErr, fmt.Errorf("failed to restore detached Signature element: %w", rerr))
				}
			}
			return nil, rejectErr
		}
	}

	var canonical []byte
	if ref.uri == "" {
		canonical, err = canonicalize(c14nMethod, doc, prefixes)
	} else {
		canonical, err = canonicalizeSubtree(c14nMethod, target, prefixes)
	}

	// Reattach the Signature element at its original sibling position.
	if hasEnveloped {
		if rerr := anchor.restore(sigElem); rerr != nil && err == nil {
			err = rerr
		}
	}

	if err != nil {
		return nil, err
	}

	// Compute and compare digest.
	computed, err := computeDigest(ref.digestAlgorithm, canonical)
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
		switch localName(elem) {
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
			decoded, err := decodeBase64(textContent(elem))
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
		switch localName(e) {
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
		switch localName(e) {
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
				if localName(te) != "Transform" {
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
					if localName(incElem) == "InclusiveNamespaces" {
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
			decoded, err := decodeBase64(textContent(e))
			if err != nil {
				return ref, fmt.Errorf("%w: invalid DigestValue base64: %v", ErrInvalidSignature, err)
			}
			ref.digestValue = decoded
		}
	}
	return ref, nil
}
