package xmldsig1

import (
	"context"
	"encoding/base64"
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

func verifySignature(ctx context.Context, cfg *verifierConfig, doc *helium.Document, sigElem *helium.Element) error {
	parsed, err := parseSignatureElement(sigElem)
	if err != nil {
		return err
	}

	// Resolve key.
	var keyInfoData *KeyInfoData
	if parsed.keyInfoElem != nil {
		keyInfoData, err = parseKeyInfo(parsed.keyInfoElem)
		if err != nil {
			return err
		}
	}

	key, err := cfg.keySource.ResolveKey(ctx, keyInfoData, parsed.signatureAlg)
	if err != nil {
		return err
	}

	// Canonicalize SignedInfo.
	canonical, err := canonicalizeSubtree(parsed.c14nMethod, parsed.signedInfoElem, nil)
	if err != nil {
		return err
	}

	// Verify signature value.
	if err := verifyBytes(parsed.signatureAlg, key, canonical, parsed.signatureValue); err != nil {
		return &VerificationError{Reference: -1, Err: err}
	}

	// Verify each reference.
	for i, ref := range parsed.references {
		if err := verifyReference(doc, sigElem, ref); err != nil {
			return &VerificationError{Reference: i, URI: ref.uri, Err: err}
		}
	}

	return nil
}

func verifyReference(doc *helium.Document, sigElem *helium.Element, ref parsedReference) error {
	target, err := resolveReference(doc, ref.uri)
	if err != nil {
		return err
	}

	// Check for enveloped-signature transform.
	hasEnveloped := false
	for _, t := range ref.transforms {
		if t.algorithm == TransformEnvelopedSignature {
			hasEnveloped = true
			break
		}
	}

	// Temporarily detach the Signature element for enveloped signatures.
	var sigParent *helium.Element
	if hasEnveloped {
		if p, ok := helium.AsNode[*helium.Element](sigElem.Parent()); ok {
			sigParent = p
		}
		helium.UnlinkNode(sigElem)
	}

	// Find the c14n method.
	c14nMethod := ExcC14N10
	var prefixes []string
	for _, t := range ref.transforms {
		switch t.algorithm {
		case C14N10, C14N10Comments, ExcC14N10, ExcC14N10Comments, C14N11URI, C14N11Comments:
			c14nMethod = t.algorithm
			prefixes = t.prefixes
		}
	}

	var canonical []byte
	if ref.uri == "" {
		canonical, err = canonicalize(c14nMethod, doc, prefixes)
	} else {
		canonical, err = canonicalizeSubtree(c14nMethod, target, prefixes)
	}

	// Reattach the Signature element.
	if hasEnveloped && sigParent != nil {
		_ = sigParent.AddChild(sigElem)
	}

	if err != nil {
		return err
	}

	// Compute and compare digest.
	computed, err := computeDigest(ref.digestAlgorithm, canonical)
	if err != nil {
		return err
	}

	if !digestEqual(computed, ref.digestValue) {
		return ErrDigestMismatch
	}

	return nil
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

	for child := sigElem.FirstChild(); child != nil; child = child.NextSibling() {
		elem, ok := helium.AsNode[*helium.Element](child)
		if !ok {
			continue
		}
		switch localName(elem) {
		case "SignedInfo":
			parsed.signedInfoElem = elem
			if err := parseSignedInfo(elem, parsed); err != nil {
				return nil, err
			}
		case "SignatureValue":
			text := strings.TrimSpace(textContent(elem))
			decoded, err := base64.StdEncoding.DecodeString(text)
			if err != nil {
				return nil, fmt.Errorf("%w: invalid SignatureValue base64: %v", ErrInvalidSignature, err)
			}
			parsed.signatureValue = decoded
		case "KeyInfo":
			parsed.keyInfoElem = elem
		}
	}

	if parsed.signedInfoElem == nil {
		return nil, fmt.Errorf("%w: missing SignedInfo", ErrInvalidSignature)
	}
	if parsed.signatureValue == nil {
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
		switch localName(e) {
		case "Transforms":
			for tc := e.FirstChild(); tc != nil; tc = tc.NextSibling() {
				te, ok := helium.AsNode[*helium.Element](tc)
				if !ok {
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
			text := strings.TrimSpace(textContent(e))
			decoded, err := base64.StdEncoding.DecodeString(text)
			if err != nil {
				return ref, fmt.Errorf("%w: invalid DigestValue base64: %v", ErrInvalidSignature, err)
			}
			ref.digestValue = decoded
		}
	}
	return ref, nil
}
