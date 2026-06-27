package xmlenc1

import (
	"fmt"
	"slices"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/domutil"
	"github.com/lestrrat-go/helium/internal/xmlbase64"
)

// parseEncryptedData parses an EncryptedData element.
func parseEncryptedData(elem *helium.Element) (*EncryptedData, error) {
	if elem == nil || !isXMLEncElem(elem, "EncryptedData") {
		return nil, fmt.Errorf("%w: expected xenc:EncryptedData", ErrMalformedEncrypted)
	}

	ed := &EncryptedData{}
	ed.ID, _ = elem.GetAttribute("Id")
	ed.Type, _ = elem.GetAttribute("Type")

	// Track CipherData separately: a decoded CipherValue can be a non-nil
	// empty slice, so a boolean is the reliable duplicate sentinel.
	var seenCipherData bool

	for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
		e, ok := helium.AsNode[*helium.Element](child)
		if !ok {
			continue
		}
		switch {
		case isXMLEncElem(e, "EncryptionMethod"):
			if ed.EncryptionMethod != nil {
				return nil, fmt.Errorf("%w: duplicate EncryptionMethod", ErrMalformedEncrypted)
			}
			em, err := parseEncryptionMethod(e)
			if err != nil {
				return nil, err
			}
			ed.EncryptionMethod = em
		case isDSigElem(e, "KeyInfo"):
			if err := parseKeyInfoForEncryption(e, ed); err != nil {
				return nil, err
			}
		case isXMLEncElem(e, "CipherData"):
			if seenCipherData {
				return nil, fmt.Errorf("%w: duplicate CipherData", ErrMalformedEncrypted)
			}
			seenCipherData = true
			cv, err := parseCipherData(e)
			if err != nil {
				return nil, err
			}
			ed.CipherValue = cv
		}
	}

	if ed.CipherValue == nil {
		return nil, fmt.Errorf("%w: missing CipherData/CipherValue", ErrMalformedEncrypted)
	}

	// Populate the deprecated single EncryptedKey field with the first
	// candidate so callers reading it keep working.
	if len(ed.EncryptedKeys) > 0 {
		ed.EncryptedKey = ed.EncryptedKeys[0]
	}

	return ed, nil
}

func parseKeyInfoForEncryption(elem *helium.Element, ed *EncryptedData) error {
	for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
		e, ok := helium.AsNode[*helium.Element](child)
		if !ok {
			continue
		}
		if isXMLEncElem(e, "EncryptedKey") {
			ek, err := parseEncryptedKey(e)
			if err != nil {
				return err
			}
			ed.EncryptedKeys = append(ed.EncryptedKeys, ek)
		}
	}
	return nil
}

// parseEncryptedKey parses an EncryptedKey element.
func parseEncryptedKey(elem *helium.Element) (*EncryptedKey, error) {
	if elem == nil || !isXMLEncElem(elem, "EncryptedKey") {
		return nil, fmt.Errorf("%w: expected xenc:EncryptedKey", ErrMalformedEncrypted)
	}

	ek := &EncryptedKey{}
	ek.ID, _ = elem.GetAttribute("Id")
	ek.Recipient, _ = elem.GetAttribute("Recipient")

	var seenCipherData bool

	for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
		e, ok := helium.AsNode[*helium.Element](child)
		if !ok {
			continue
		}
		switch {
		case isXMLEncElem(e, "EncryptionMethod"):
			if ek.EncryptionMethod != nil {
				return nil, fmt.Errorf("%w: duplicate EncryptionMethod", ErrMalformedEncrypted)
			}
			em, err := parseEncryptionMethod(e)
			if err != nil {
				return nil, err
			}
			ek.EncryptionMethod = em
		case isXMLEncElem(e, "CipherData"):
			if seenCipherData {
				return nil, fmt.Errorf("%w: duplicate CipherData", ErrMalformedEncrypted)
			}
			seenCipherData = true
			cv, err := parseCipherData(e)
			if err != nil {
				return nil, err
			}
			ek.CipherValue = cv
		case isXMLEncElem(e, "CarriedKeyName"):
			ek.CarriedKeyName = domutil.TextContent(e)
		}
	}

	if ek.CipherValue == nil {
		return nil, fmt.Errorf("%w: EncryptedKey missing CipherData/CipherValue", ErrMalformedEncrypted)
	}

	return ek, nil
}

func parseEncryptionMethod(elem *helium.Element) (*EncryptionMethod, error) {
	em := &EncryptionMethod{}
	alg, ok := elem.GetAttribute("Algorithm")
	if !ok || alg == "" {
		return nil, fmt.Errorf("%w: EncryptionMethod missing/empty Algorithm", ErrMalformedEncrypted)
	}
	em.Algorithm = alg

	// Enforce at-most-one cardinality on the optional sub-elements,
	// mirroring the duplicate-EncryptionMethod/CipherData guards in the
	// parent parsers. Boolean sentinels are used because an empty
	// attribute/text value is otherwise ambiguous.
	var seenDigestMethod, seenMGF, seenOAEPParams, seenKeySize bool

	for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
		e, ok := helium.AsNode[*helium.Element](child)
		if !ok {
			continue
		}
		switch {
		case isDSigElem(e, "DigestMethod"):
			if seenDigestMethod {
				return nil, fmt.Errorf("%w: duplicate DigestMethod", ErrMalformedEncrypted)
			}
			seenDigestMethod = true
			alg, ok := e.GetAttribute("Algorithm")
			if !ok || alg == "" {
				return nil, fmt.Errorf("%w: DigestMethod missing/empty Algorithm", ErrMalformedEncrypted)
			}
			em.DigestMethod = alg
		case isMGFElem(e):
			if seenMGF {
				return nil, fmt.Errorf("%w: duplicate MGF", ErrMalformedEncrypted)
			}
			seenMGF = true
			alg, ok := e.GetAttribute("Algorithm")
			if !ok || alg == "" {
				return nil, fmt.Errorf("%w: MGF missing/empty Algorithm", ErrMalformedEncrypted)
			}
			em.MGFAlgorithm = alg
		case isXMLEncElem(e, "KeySize"):
			// KeySize is an optional singleton in the schema. The package
			// derives key sizes from the algorithm URI and does not consume
			// KeySize, so enforce at-most-one cardinality to stay consistent
			// with the other sub-element guards.
			if seenKeySize {
				return nil, fmt.Errorf("%w: duplicate KeySize", ErrMalformedEncrypted)
			}
			seenKeySize = true
		case isXMLEncElem(e, "OAEPparams"):
			if seenOAEPParams {
				return nil, fmt.Errorf("%w: duplicate OAEPparams", ErrMalformedEncrypted)
			}
			seenOAEPParams = true
			decoded, err := xmlbase64.DecodeString(domutil.TextContent(e))
			if err != nil {
				return nil, fmt.Errorf("%w: invalid OAEPparams: %v", ErrMalformedEncrypted, err)
			}
			em.OAEPParams = decoded
		}
	}

	return em, nil
}

// parseCipherData parses a CipherData element. Per the XML-Enc schema,
// CipherData is a choice of EXACTLY ONE CipherValue or one CipherReference.
// A second choice member of either kind (CipherValue+CipherValue,
// CipherValue+CipherReference, CipherReference+CipherValue, or two
// CipherReferences) is schema-invalid and rejected at parse rather than
// silently using the first. CipherReference (indirect cipher text fetched
// via a URI plus transforms) is not supported by helium and is rejected
// explicitly; ignoring it would both lose data and defeat the
// exactly-one-choice rule.
func parseCipherData(elem *helium.Element) ([]byte, error) {
	var decoded []byte
	var seenChoice bool
	for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
		e, ok := helium.AsNode[*helium.Element](child)
		if !ok {
			continue
		}
		switch {
		case isXMLEncElem(e, "CipherValue"):
			if seenChoice {
				return nil, fmt.Errorf("%w: CipherData allows exactly one of CipherValue or CipherReference", ErrMalformedEncrypted)
			}
			seenChoice = true
			d, err := xmlbase64.DecodeString(domutil.TextContent(e))
			if err != nil {
				return nil, fmt.Errorf("%w: invalid CipherValue: %v", ErrMalformedEncrypted, err)
			}
			decoded = d
		case isXMLEncElem(e, "CipherReference"):
			if seenChoice {
				return nil, fmt.Errorf("%w: CipherData allows exactly one of CipherValue or CipherReference", ErrMalformedEncrypted)
			}
			return nil, fmt.Errorf("%w: CipherReference is not supported", ErrMalformedEncrypted)
		}
	}
	if !seenChoice {
		return nil, fmt.Errorf("%w: missing CipherValue", ErrMalformedEncrypted)
	}
	return decoded, nil
}

// isElemNS reports whether e has the given local name and one of the
// supplied namespace URIs. XML Encryption/Signature elements are
// namespace-qualified, so matching by local name alone would wrongly
// treat a foreign-namespaced element (e.g. someone else's
// "CipherValue") as an XMLEnc element. Every element match in this
// package must therefore require the correct namespace URI.
func isElemNS(e *helium.Element, local string, nsURIs ...string) bool {
	if domutil.LocalName(e) != local {
		return false
	}
	return slices.Contains(nsURIs, e.URI())
}

// isXMLEncElem reports whether e is an XML Encryption element
// (namespace http://www.w3.org/2001/04/xmlenc#) with the given local name.
func isXMLEncElem(e *helium.Element, local string) bool {
	return isElemNS(e, local, NamespaceXMLEnc)
}

// isDSigElem reports whether e is an XML Digital Signature element
// (namespace http://www.w3.org/2000/09/xmldsig#) with the given local name.
// KeyInfo and DigestMethod are defined in the dsig namespace.
func isDSigElem(e *helium.Element, local string) bool {
	return isElemNS(e, local, NamespaceDSig)
}

// isMGFElem reports whether e is an MGF element. The element is defined
// in the XML Encryption 1.1 namespace, but accept the base xmlenc
// namespace too for robustness against producers that misqualify it.
func isMGFElem(e *helium.Element) bool {
	return isElemNS(e, "MGF", NamespaceXMLEnc11, NamespaceXMLEnc)
}
