package xmlenc1

import (
	"fmt"
	"slices"

	helium "github.com/lestrrat-go/helium"
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

	for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
		e, ok := helium.AsNode[*helium.Element](child)
		if !ok {
			continue
		}
		switch {
		case isXMLEncElem(e, "EncryptionMethod"):
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
			ed.EncryptedKey = ek
			return nil
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

	for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
		e, ok := helium.AsNode[*helium.Element](child)
		if !ok {
			continue
		}
		switch {
		case isXMLEncElem(e, "EncryptionMethod"):
			em, err := parseEncryptionMethod(e)
			if err != nil {
				return nil, err
			}
			ek.EncryptionMethod = em
		case isXMLEncElem(e, "CipherData"):
			cv, err := parseCipherData(e)
			if err != nil {
				return nil, err
			}
			ek.CipherValue = cv
		case isXMLEncElem(e, "CarriedKeyName"):
			ek.CarriedKeyName = textContent(e)
		}
	}

	return ek, nil
}

func parseEncryptionMethod(elem *helium.Element) (*EncryptionMethod, error) {
	em := &EncryptionMethod{}
	alg, ok := elem.GetAttribute("Algorithm")
	if !ok {
		return nil, fmt.Errorf("%w: EncryptionMethod missing Algorithm", ErrMalformedEncrypted)
	}
	em.Algorithm = alg

	for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
		e, ok := helium.AsNode[*helium.Element](child)
		if !ok {
			continue
		}
		switch {
		case isDSigElem(e, "DigestMethod"):
			em.DigestMethod, _ = e.GetAttribute("Algorithm")
		case isMGFElem(e):
			em.MGFAlgorithm, _ = e.GetAttribute("Algorithm")
		case isXMLEncElem(e, "OAEPparams"):
			decoded, err := xmlbase64.DecodeString(textContent(e))
			if err != nil {
				return nil, fmt.Errorf("%w: invalid OAEPparams: %v", ErrMalformedEncrypted, err)
			}
			em.OAEPParams = decoded
		}
	}

	return em, nil
}

func parseCipherData(elem *helium.Element) ([]byte, error) {
	for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
		e, ok := helium.AsNode[*helium.Element](child)
		if !ok {
			continue
		}
		if isXMLEncElem(e, "CipherValue") {
			decoded, err := xmlbase64.DecodeString(textContent(e))
			if err != nil {
				return nil, fmt.Errorf("%w: invalid CipherValue: %v", ErrMalformedEncrypted, err)
			}
			return decoded, nil
		}
	}
	return nil, fmt.Errorf("%w: missing CipherValue", ErrMalformedEncrypted)
}

// localName returns the local name of an element, stripping any prefix.
func localName(e *helium.Element) string {
	name := e.Name()
	for i := range len(name) {
		if name[i] == ':' {
			return name[i+1:]
		}
	}
	return name
}

// isElemNS reports whether e has the given local name and one of the
// supplied namespace URIs. XML Encryption/Signature elements are
// namespace-qualified, so matching by local name alone would wrongly
// treat a foreign-namespaced element (e.g. someone else's
// "CipherValue") as an XMLEnc element. Every element match in this
// package must therefore require the correct namespace URI.
func isElemNS(e *helium.Element, local string, nsURIs ...string) bool {
	if localName(e) != local {
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

// textContent returns the concatenated text content of an element.
func textContent(e *helium.Element) string {
	var sb []byte
	for child := e.FirstChild(); child != nil; child = child.NextSibling() {
		sb = append(sb, child.Content()...)
	}
	return string(sb)
}
