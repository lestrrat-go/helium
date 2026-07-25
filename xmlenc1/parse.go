package xmlenc1

import (
	"crypto/ecdh"
	"encoding/hex"
	"fmt"
	"slices"
	"strings"

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
		case isDSigElem(e, "KeyInfo"):
			agreement, err := parseAgreementMethodForKeyInfo(e)
			if err != nil {
				return nil, err
			}
			ek.AgreementMethod = agreement
		}
	}

	if ek.CipherValue == nil {
		return nil, fmt.Errorf("%w: EncryptedKey missing CipherData/CipherValue", ErrMalformedEncrypted)
	}

	return ek, nil
}

func parseAgreementMethodForKeyInfo(elem *helium.Element) (*AgreementMethod, error) {
	var agreement *AgreementMethod
	for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
		e, ok := helium.AsNode[*helium.Element](child)
		if !ok || !isXMLEncElem(e, "AgreementMethod") {
			continue
		}
		if agreement != nil {
			return nil, fmt.Errorf("%w: duplicate AgreementMethod", ErrMalformedEncrypted)
		}
		var err error
		agreement, err = parseAgreementMethod(e)
		if err != nil {
			return nil, err
		}
	}
	return agreement, nil
}

func parseAgreementMethod(elem *helium.Element) (*AgreementMethod, error) {
	algorithm, ok := elem.GetAttribute("Algorithm")
	if !ok || algorithm == "" {
		return nil, fmt.Errorf("%w: AgreementMethod missing/empty Algorithm", ErrMalformedEncrypted)
	}
	agreement := &AgreementMethod{Algorithm: algorithm}
	for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
		e, ok := helium.AsNode[*helium.Element](child)
		if !ok {
			continue
		}
		switch {
		case isXMLEncElem(e, "OriginatorKeyInfo"):
			if agreement.OriginatorKey != nil {
				return nil, fmt.Errorf("%w: duplicate OriginatorKeyInfo", ErrMalformedEncrypted)
			}
			key, err := parseOriginatorKeyInfo(e)
			if err != nil {
				return nil, err
			}
			agreement.OriginatorKey = key
		case isXMLEnc11Elem(e, "KeyDerivationMethod"):
			if agreement.KeyDerivationMethod != nil {
				return nil, fmt.Errorf("%w: duplicate KeyDerivationMethod", ErrMalformedEncrypted)
			}
			method, err := parseKeyDerivationMethod(e)
			if err != nil {
				return nil, err
			}
			agreement.KeyDerivationMethod = method
		}
	}
	return agreement, nil
}

func parseKeyDerivationMethod(elem *helium.Element) (*KeyDerivationMethod, error) {
	algorithm, ok := elem.GetAttribute("Algorithm")
	if !ok || algorithm == "" {
		return nil, fmt.Errorf("%w: KeyDerivationMethod missing/empty Algorithm", ErrMalformedEncrypted)
	}
	method := &KeyDerivationMethod{Algorithm: algorithm}
	for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
		e, ok := helium.AsNode[*helium.Element](child)
		if !ok || !isXMLEnc11Elem(e, "ConcatKDFParams") {
			continue
		}
		if method.ConcatKDF != nil {
			return nil, fmt.Errorf("%w: duplicate ConcatKDFParams", ErrMalformedEncrypted)
		}
		params, err := parseConcatKDFParams(e)
		if err != nil {
			return nil, err
		}
		method.ConcatKDF = params
	}
	if method.Algorithm == ConcatKDF && method.ConcatKDF == nil {
		return nil, fmt.Errorf("%w: ConcatKDF missing ConcatKDFParams", ErrMalformedEncrypted)
	}
	return method, nil
}

func parseConcatKDFParams(elem *helium.Element) (*ConcatKDFParams, error) {
	params := &ConcatKDFParams{}
	var err error
	for _, field := range []struct {
		name string
		dest *[]byte
	}{
		{name: "AlgorithmID", dest: &params.AlgorithmID},
		{name: "PartyUInfo", dest: &params.PartyUInfo},
		{name: "PartyVInfo", dest: &params.PartyVInfo},
		{name: "SuppPubInfo", dest: &params.SuppPubInfo},
		{name: "SuppPrivInfo", dest: &params.SuppPrivInfo},
	} {
		*field.dest, err = parseConcatKDFHexAttribute(elem, field.name)
		if err != nil {
			return nil, err
		}
	}
	for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
		e, ok := helium.AsNode[*helium.Element](child)
		if !ok || !isDSigElem(e, "DigestMethod") {
			continue
		}
		if params.DigestMethod != "" {
			return nil, fmt.Errorf("%w: duplicate ConcatKDF DigestMethod", ErrMalformedEncrypted)
		}
		params.DigestMethod, _ = e.GetAttribute("Algorithm")
		if params.DigestMethod == "" {
			return nil, fmt.Errorf("%w: ConcatKDF DigestMethod missing/empty Algorithm", ErrMalformedEncrypted)
		}
	}
	if params.DigestMethod == "" {
		return nil, fmt.Errorf("%w: ConcatKDF missing DigestMethod", ErrMalformedEncrypted)
	}
	return params, nil
}

func parseConcatKDFHexAttribute(elem *helium.Element, name string) ([]byte, error) {
	value, ok := elem.GetAttribute(name)
	if !ok || strings.TrimSpace(value) == "" {
		return nil, nil
	}
	value = strings.TrimSpace(value)
	decoded, err := hex.DecodeString(value)
	if err != nil || len(decoded) == 0 {
		return nil, fmt.Errorf("%w: invalid ConcatKDF %s", ErrMalformedEncrypted, name)
	}
	if decoded[0] != 0 {
		return nil, fmt.Errorf("%w: non-byte-aligned ConcatKDF %s is not supported", ErrMalformedEncrypted, name)
	}
	return decoded[1:], nil
}

func parseOriginatorKeyInfo(elem *helium.Element) (*ECKeyValue, error) {
	for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
		keyValue, ok := helium.AsNode[*helium.Element](child)
		if !ok || !isDSigElem(keyValue, "KeyValue") {
			continue
		}
		for nested := keyValue.FirstChild(); nested != nil; nested = nested.NextSibling() {
			ecValue, ok := helium.AsNode[*helium.Element](nested)
			if !ok || !isDSig11Elem(ecValue, "ECKeyValue") {
				continue
			}
			return parseECKeyValue(ecValue)
		}
	}
	return nil, fmt.Errorf("%w: OriginatorKeyInfo missing dsig11:ECKeyValue", ErrMalformedEncrypted)
}

func parseECKeyValue(elem *helium.Element) (*ECKeyValue, error) {
	var curve ecdh.Curve
	var publicKey []byte
	for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
		e, ok := helium.AsNode[*helium.Element](child)
		if !ok || e.URI() != NamespaceDSig11 {
			continue
		}
		switch domutil.LocalName(e) {
		case "NamedCurve":
			uri, _ := e.GetAttribute("URI")
			var err error
			curve, err = ecdhCurveForURI(uri)
			if err != nil {
				return nil, err
			}
		case "PublicKey":
			decoded, err := xmlbase64.DecodeString(domutil.TextContent(e))
			if err != nil {
				return nil, fmt.Errorf("%w: invalid ECKeyValue base64: %v", ErrMalformedEncrypted, err)
			}
			publicKey = decoded
		}
	}
	if curve == nil || len(publicKey) == 0 {
		return nil, fmt.Errorf("%w: ECKeyValue requires NamedCurve and PublicKey", ErrMalformedEncrypted)
	}
	if _, err := curve.NewPublicKey(publicKey); err != nil {
		return nil, fmt.Errorf("%w: invalid EC public key: %v", ErrMalformedEncrypted, err)
	}
	return &ECKeyValue{Curve: curve, PublicKey: publicKey}, nil
}

func ecdhCurveForURI(uri string) (ecdh.Curve, error) {
	switch uri {
	case "urn:oid:1.2.840.10045.3.1.7":
		return ecdh.P256(), nil
	case "urn:oid:1.3.132.0.34":
		return ecdh.P384(), nil
	case "urn:oid:1.3.132.0.35":
		return ecdh.P521(), nil
	default:
		return nil, fmt.Errorf("%w: unsupported EC curve %q", ErrMalformedEncrypted, uri)
	}
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

func isXMLEnc11Elem(e *helium.Element, local string) bool {
	return isElemNS(e, local, NamespaceXMLEnc11)
}

// isDSigElem reports whether e is an XML Digital Signature element
// (namespace http://www.w3.org/2000/09/xmldsig#) with the given local name.
// KeyInfo and DigestMethod are defined in the dsig namespace.
func isDSigElem(e *helium.Element, local string) bool {
	return isElemNS(e, local, NamespaceDSig)
}

func isDSig11Elem(e *helium.Element, local string) bool {
	return isElemNS(e, local, NamespaceDSig11)
}

// isMGFElem reports whether e is an MGF element. The element is defined
// in the XML Encryption 1.1 namespace, but accept the base xmlenc
// namespace too for robustness against producers that misqualify it.
func isMGFElem(e *helium.Element) bool {
	return isElemNS(e, "MGF", NamespaceXMLEnc11, NamespaceXMLEnc)
}
