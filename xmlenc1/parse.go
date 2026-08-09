package xmlenc1

import (
	"context"
	"crypto/ecdh"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"slices"
	"strings"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/domutil"
	"github.com/lestrrat-go/helium/internal/xmlbase64"
)

// parseEncryptedData parses an EncryptedData element. encryptedKeyBudget
// carries the cumulative EncryptedKey ciphertext allowance across the whole
// element, payloadBudget caps the EncryptedData CipherValue, and cfg supplies
// the effective EncryptedKey candidate limit. Both budgets are charged at the
// earliest point that sees their values; the candidate limit is checked before
// an excess candidate is parsed or retained.
//
// ctx is observed throughout the parse, which is why every function below takes
// it. The document decides how many children each element it describes carries,
// and children the parse skips — a comment inside a CipherValue is the cheapest
// of them — are charged against no budget at all, so the trip count of these
// walks is bounded by nothing but the document's size. Every one of them
// therefore runs through eachSibling, and every error return runs through abort,
// so a caller that cancels is answered while the parse is still reading rather
// than after it finishes. A live context leaves both untouched.
func parseEncryptedData(ctx context.Context, elem *helium.Element, encryptedKeyBudget *encryptedKeyBudget, payloadBudget *payloadCipherValueBudget, cfg *decryptConfig) (*EncryptedData, error) {
	if elem == nil || !isXMLEncElem(elem, "EncryptedData") {
		return nil, abort(ctx, fmt.Errorf("%w: expected xenc:EncryptedData", ErrMalformedEncrypted))
	}

	ed := &EncryptedData{}
	ed.ID, _ = elem.GetAttribute("Id")
	ed.Type, _ = elem.GetAttribute("Type")

	// Track CipherData separately: a decoded CipherValue can be a non-nil
	// empty slice, so a boolean is the reliable duplicate sentinel.
	var seenCipherData bool

	if err := eachChildElement(ctx, elem, func(e *helium.Element) error {
		switch {
		case isXMLEncElem(e, "EncryptionMethod"):
			if ed.EncryptionMethod != nil {
				return fmt.Errorf("%w: duplicate EncryptionMethod", ErrMalformedEncrypted)
			}
			em, err := parseEncryptionMethod(ctx, e)
			if err != nil {
				return err
			}
			ed.EncryptionMethod = em
		case isDSigElem(e, "KeyInfo"):
			return parseKeyInfoForEncryption(ctx, e, ed, encryptedKeyBudget, cfg)
		case isXMLEncElem(e, "CipherData"):
			if seenCipherData {
				return fmt.Errorf("%w: duplicate CipherData", ErrMalformedEncrypted)
			}
			seenCipherData = true
			cv, err := parseCipherData(ctx, e, payloadBudget)
			if err != nil {
				return err
			}
			ed.CipherValue = cv
		}
		return nil
	}); err != nil {
		return nil, err
	}

	if ed.CipherValue == nil {
		return nil, abort(ctx, fmt.Errorf("%w: missing CipherData/CipherValue", ErrMalformedEncrypted))
	}

	// Populate the deprecated single EncryptedKey field with the first
	// candidate so callers reading it keep working.
	if len(ed.EncryptedKeys) > 0 {
		ed.EncryptedKey = ed.EncryptedKeys[0]
	}

	return ed, nil
}

func parseKeyInfoForEncryption(ctx context.Context, elem *helium.Element, ed *EncryptedData, budget *encryptedKeyBudget, cfg *decryptConfig) error {
	return eachChildElement(ctx, elem, func(e *helium.Element) error {
		if !isXMLEncElem(e, "EncryptedKey") {
			return nil
		}
		ek, err := parseEncryptedKey(ctx, e, budget)
		if err != nil {
			return err
		}
		if err := checkEncryptedKeyCap(cfg, len(ed.EncryptedKeys)+1); err != nil {
			return abort(ctx, err)
		}
		ed.EncryptedKeys = append(ed.EncryptedKeys, ek)
		return nil
	})
}

// parseEncryptedKey parses an EncryptedKey element, charging its CipherValue
// to budget.
//
// A xenc:CarriedKeyName child is stepped over rather than read;
// [EncryptedKey.CarriedKeyName] owns why the parse leaves that field unset.
func parseEncryptedKey(ctx context.Context, elem *helium.Element, budget *encryptedKeyBudget) (*EncryptedKey, error) {
	if elem == nil || !isXMLEncElem(elem, "EncryptedKey") {
		return nil, abort(ctx, fmt.Errorf("%w: expected xenc:EncryptedKey", ErrMalformedEncrypted))
	}

	ek := &EncryptedKey{}
	ek.ID, _ = elem.GetAttribute("Id")
	ek.Recipient, _ = elem.GetAttribute("Recipient")

	var seenCipherData bool

	if err := eachChildElement(ctx, elem, func(e *helium.Element) error {
		switch {
		case isXMLEncElem(e, "EncryptionMethod"):
			if ek.EncryptionMethod != nil {
				return fmt.Errorf("%w: duplicate EncryptionMethod", ErrMalformedEncrypted)
			}
			em, err := parseEncryptionMethod(ctx, e)
			if err != nil {
				return err
			}
			ek.EncryptionMethod = em
		case isXMLEncElem(e, "CipherData"):
			if seenCipherData {
				return fmt.Errorf("%w: duplicate CipherData", ErrMalformedEncrypted)
			}
			seenCipherData = true
			cv, err := parseCipherData(ctx, e, budget)
			if err != nil {
				return err
			}
			ek.CipherValue = cv
		case isDSigElem(e, "KeyInfo"):
			agreement, err := parseAgreementMethodForKeyInfo(ctx, e)
			if err != nil {
				return err
			}
			ek.AgreementMethod = agreement
		}
		return nil
	}); err != nil {
		return nil, err
	}

	if ek.CipherValue == nil {
		return nil, abort(ctx, fmt.Errorf("%w: EncryptedKey missing CipherData/CipherValue", ErrMalformedEncrypted))
	}

	return ek, nil
}

func parseAgreementMethodForKeyInfo(ctx context.Context, elem *helium.Element) (*AgreementMethod, error) {
	var agreement *AgreementMethod
	if err := eachChildElement(ctx, elem, func(e *helium.Element) error {
		if !isXMLEncElem(e, "AgreementMethod") {
			return nil
		}
		if agreement != nil {
			return fmt.Errorf("%w: duplicate AgreementMethod", ErrMalformedEncrypted)
		}
		parsed, err := parseAgreementMethod(ctx, e)
		if err != nil {
			return err
		}
		agreement = parsed
		return nil
	}); err != nil {
		return nil, err
	}
	return agreement, nil
}

func parseAgreementMethod(ctx context.Context, elem *helium.Element) (*AgreementMethod, error) {
	algorithm, ok := elem.GetAttribute("Algorithm")
	if !ok || algorithm == "" {
		return nil, abort(ctx, fmt.Errorf("%w: AgreementMethod missing/empty Algorithm", ErrMalformedEncrypted))
	}
	agreement := &AgreementMethod{Algorithm: algorithm}
	if err := eachChildElement(ctx, elem, func(e *helium.Element) error {
		switch {
		case isXMLEncElem(e, "OriginatorKeyInfo"):
			if agreement.OriginatorKey != nil {
				return fmt.Errorf("%w: duplicate OriginatorKeyInfo", ErrMalformedEncrypted)
			}
			key, err := parseOriginatorKeyInfo(ctx, e)
			if err != nil {
				return err
			}
			agreement.OriginatorKey = key
		case isXMLEnc11Elem(e, "KeyDerivationMethod"):
			if agreement.KeyDerivationMethod != nil {
				return fmt.Errorf("%w: duplicate KeyDerivationMethod", ErrMalformedEncrypted)
			}
			method, err := parseKeyDerivationMethod(ctx, e)
			if err != nil {
				return err
			}
			agreement.KeyDerivationMethod = method
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return agreement, nil
}

func parseKeyDerivationMethod(ctx context.Context, elem *helium.Element) (*KeyDerivationMethod, error) {
	algorithm, ok := elem.GetAttribute("Algorithm")
	if !ok || algorithm == "" {
		return nil, abort(ctx, fmt.Errorf("%w: KeyDerivationMethod missing/empty Algorithm", ErrMalformedEncrypted))
	}
	method := &KeyDerivationMethod{Algorithm: algorithm}
	if err := eachChildElement(ctx, elem, func(e *helium.Element) error {
		if !isXMLEnc11Elem(e, "ConcatKDFParams") {
			return nil
		}
		if method.ConcatKDF != nil {
			return fmt.Errorf("%w: duplicate ConcatKDFParams", ErrMalformedEncrypted)
		}
		params, err := parseConcatKDFParams(ctx, e)
		if err != nil {
			return err
		}
		method.ConcatKDF = params
		return nil
	}); err != nil {
		return nil, err
	}
	if method.Algorithm == ConcatKDF && method.ConcatKDF == nil {
		return nil, abort(ctx, fmt.Errorf("%w: ConcatKDF missing ConcatKDFParams", ErrMalformedEncrypted))
	}
	return method, nil
}

func parseConcatKDFParams(ctx context.Context, elem *helium.Element) (*ConcatKDFParams, error) {
	params := &ConcatKDFParams{}
	var err error
	// This loop is over a fixed five-element table, not over anything the
	// document sizes, so it is not an eachSibling site: its trip count is five
	// whatever the input says. Its error returns still run through abort.
	for _, field := range []struct {
		name       string
		dest       *[]byte
		unusedBits *uint8
	}{
		{name: "AlgorithmID", dest: &params.AlgorithmID, unusedBits: &params.algorithmIDUnusedBits},
		{name: "PartyUInfo", dest: &params.PartyUInfo, unusedBits: &params.partyUInfoUnusedBits},
		{name: "PartyVInfo", dest: &params.PartyVInfo, unusedBits: &params.partyVInfoUnusedBits},
		{name: "SuppPubInfo", dest: &params.SuppPubInfo, unusedBits: &params.suppPubInfoUnusedBits},
		{name: "SuppPrivInfo", dest: &params.SuppPrivInfo, unusedBits: &params.suppPrivInfoUnusedBits},
	} {
		*field.dest, *field.unusedBits, err = parseConcatKDFHexAttribute(elem, field.name)
		if err != nil {
			return nil, abort(ctx, err)
		}
	}
	// The five fields are only bounded as a set, and this is the first point
	// that holds all of them. Rejecting here keeps an oversized document out
	// of the candidate list entirely, so no ECDH exchange, key unwrap, or
	// OtherInfo packing is ever attempted for it.
	if err := checkConcatKDFOtherInfoBudget(params); err != nil {
		return nil, abort(ctx, err)
	}
	if err := eachChildElement(ctx, elem, func(e *helium.Element) error {
		if !isDSigElem(e, "DigestMethod") {
			return nil
		}
		if params.DigestMethod != "" {
			return fmt.Errorf("%w: duplicate ConcatKDF DigestMethod", ErrMalformedEncrypted)
		}
		params.DigestMethod, _ = e.GetAttribute("Algorithm")
		if params.DigestMethod == "" {
			return fmt.Errorf("%w: ConcatKDF DigestMethod missing/empty Algorithm", ErrMalformedEncrypted)
		}
		return nil
	}); err != nil {
		return nil, err
	}
	if params.DigestMethod == "" {
		return nil, abort(ctx, fmt.Errorf("%w: ConcatKDF missing DigestMethod", ErrMalformedEncrypted))
	}
	return params, nil
}

func parseConcatKDFHexAttribute(elem *helium.Element, name string) ([]byte, uint8, error) {
	value, ok := elem.GetAttribute(name)
	if !ok || strings.TrimSpace(value) == "" {
		return nil, 0, nil
	}
	value = strings.TrimSpace(value)
	// A single field can never be larger than the whole set, so the same
	// budget rules this attribute out from its hex length alone — before
	// hex.DecodeString allocates half of it. The set-wide check in
	// parseConcatKDFParams still decides the cumulative case. The encoded
	// form is two characters per octet and carries a leading unused-bit
	// octet on top of the field's own bytes.
	if len(value) > 2*(maxConcatKDFOtherInfoBytes+1) {
		return nil, 0, fmt.Errorf("%w: ConcatKDF %s alone is over the %d byte OtherInfo limit", ErrMalformedEncrypted, name, maxConcatKDFOtherInfoBytes)
	}
	decoded, err := hex.DecodeString(value)
	if err != nil || len(decoded) == 0 {
		return nil, 0, fmt.Errorf("%w: invalid ConcatKDF %s", ErrMalformedEncrypted, name)
	}
	if decoded[0] > 7 || (len(decoded) == 1 && decoded[0] != 0) {
		return nil, 0, fmt.Errorf("%w: invalid ConcatKDF %s", ErrMalformedEncrypted, name)
	}
	return decoded[1:], decoded[0], nil
}

// parseOriginatorKeyInfo returns the FIRST dsig11:ECKeyValue under the first
// ds:KeyValue that carries one. Both walks run to the end of their sibling
// chain once it is found rather than returning from inside, so the whole child
// list is observed against ctx at one rate; every child past the first match is
// only stepped over, never parsed, so the key the document supplies is the same
// one either way.
func parseOriginatorKeyInfo(ctx context.Context, elem *helium.Element) (*ECKeyValue, error) {
	var found *ECKeyValue
	if err := eachChildElement(ctx, elem, func(keyValue *helium.Element) error {
		if found != nil || !isDSigElem(keyValue, "KeyValue") {
			return nil
		}
		return eachChildElement(ctx, keyValue, func(ecValue *helium.Element) error {
			if found != nil || !isDSig11Elem(ecValue, "ECKeyValue") {
				return nil
			}
			parsed, err := parseECKeyValue(ctx, ecValue)
			if err != nil {
				return err
			}
			found = parsed
			return nil
		})
	}); err != nil {
		return nil, err
	}
	if found == nil {
		return nil, abort(ctx, fmt.Errorf("%w: OriginatorKeyInfo missing dsig11:ECKeyValue", ErrMalformedEncrypted))
	}
	return found, nil
}

func parseECKeyValue(ctx context.Context, elem *helium.Element) (*ECKeyValue, error) {
	var curve ecdh.Curve
	var publicKey []byte
	if err := eachChildElement(ctx, elem, func(e *helium.Element) error {
		if e.URI() != NamespaceDSig11 {
			return nil
		}
		switch domutil.LocalName(e) {
		case "NamedCurve":
			uri, _ := e.GetAttribute("URI")
			named, err := ecdhCurveForURI(uri)
			if err != nil {
				return err
			}
			curve = named
		case "PublicKey":
			decoded, err := decodeBoundedBase64(ctx, e, "ECKeyValue PublicKey", maxECPublicKeyBytes, "invalid ECKeyValue base64")
			if err != nil {
				return err
			}
			publicKey = decoded
		}
		return nil
	}); err != nil {
		return nil, err
	}
	if curve == nil || len(publicKey) == 0 {
		return nil, abort(ctx, fmt.Errorf("%w: ECKeyValue requires NamedCurve and PublicKey", ErrMalformedEncrypted))
	}
	if _, err := curve.NewPublicKey(publicKey); err != nil {
		return nil, abort(ctx, fmt.Errorf("%w: invalid EC public key: %v", ErrMalformedEncrypted, err))
	}
	return &ECKeyValue{Curve: curve, PublicKey: publicKey}, nil
}

// maxECPublicKeyBytes bounds the decoded octet length of one dsig11:PublicKey
// element inside a dsig11:ECKeyValue. It is the single statement of that limit;
// decodeBoundedBase64 applies it.
//
// Unlike this package's other ceilings, it is not a policy choice: it is the
// largest public key the three supported curves can encode. dsig11:PublicKey
// carries a SEC1 uncompressed point, one 0x04 tag octet followed by the two
// field elements, so it is 65 octets on P-256, 97 on P-384 and 133 on P-521 —
// and [crypto/ecdh] accepts nothing else, refusing the compressed form and every
// over-length input on all three. A value over 133 octets is therefore rejected
// by the curve whatever it holds, and refusing it before it is built only moves
// the same verdict earlier.
//
// It is the maximum across ALL THREE curves rather than the selected curve's own
// size, because at the moment the value is weighed there may be no selected
// curve to weigh it by. dsig11:ECKeyValue puts no order on its children, so
// dsig11:NamedCurve may follow dsig11:PublicKey, and a document carrying no
// NamedCurve at all still has its PublicKey read before the missing-curve error
// is raised. A per-curve bound would hold only for the one child order.
const maxECPublicKeyBytes = 133

func ecdhCurveForURI(uri string) (ecdh.Curve, error) {
	switch uri {
	case curveURIP256:
		return ecdh.P256(), nil
	case curveURIP384:
		return ecdh.P384(), nil
	case curveURIP521:
		return ecdh.P521(), nil
	default:
		return nil, fmt.Errorf("%w: unsupported EC curve %q", ErrMalformedEncrypted, uri)
	}
}

// ecdhURIForCurve is the inverse of ecdhCurveForURI, used when serializing
// an originator key. A curve outside the supported three is rejected rather
// than emitted without a NamedCurve, which would produce an EncryptedKey no
// recipient can parse.
func ecdhURIForCurve(curve ecdh.Curve) (string, error) {
	switch curve {
	case ecdh.P256():
		return curveURIP256, nil
	case ecdh.P384():
		return curveURIP384, nil
	case ecdh.P521():
		return curveURIP521, nil
	default:
		return "", fmt.Errorf("%w: unsupported EC curve for ECDH-ES", ErrEncryptionFailed)
	}
}

// parseEncryptionMethod parses an EncryptionMethod element. Both call sites —
// the EncryptedData's own method and each EncryptedKey's — are reached while
// the document is read, so everything here runs before any key is resolved and
// before anything the document says has been authenticated. The only child it
// decodes is OAEPparams, and maxOAEPParamsBytes owns what bounds that.
func parseEncryptionMethod(ctx context.Context, elem *helium.Element) (*EncryptionMethod, error) {
	em := &EncryptionMethod{}
	alg, ok := elem.GetAttribute("Algorithm")
	if !ok || alg == "" {
		return nil, abort(ctx, fmt.Errorf("%w: EncryptionMethod missing/empty Algorithm", ErrMalformedEncrypted))
	}
	em.Algorithm = alg

	// Enforce at-most-one cardinality on the optional sub-elements,
	// mirroring the duplicate-EncryptionMethod/CipherData guards in the
	// parent parsers. Boolean sentinels are used because an empty
	// attribute/text value is otherwise ambiguous.
	var seenDigestMethod, seenMGF, seenOAEPParams, seenKeySize bool

	if err := eachChildElement(ctx, elem, func(e *helium.Element) error {
		switch {
		case isDSigElem(e, "DigestMethod"):
			if seenDigestMethod {
				return fmt.Errorf("%w: duplicate DigestMethod", ErrMalformedEncrypted)
			}
			seenDigestMethod = true
			alg, ok := e.GetAttribute("Algorithm")
			if !ok || alg == "" {
				return fmt.Errorf("%w: DigestMethod missing/empty Algorithm", ErrMalformedEncrypted)
			}
			em.DigestMethod = alg
		case isMGFElem(e):
			if seenMGF {
				return fmt.Errorf("%w: duplicate MGF", ErrMalformedEncrypted)
			}
			seenMGF = true
			alg, ok := e.GetAttribute("Algorithm")
			if !ok || alg == "" {
				return fmt.Errorf("%w: MGF missing/empty Algorithm", ErrMalformedEncrypted)
			}
			em.MGFAlgorithm = alg
		case isXMLEncElem(e, "KeySize"):
			// KeySize is an optional singleton in the schema. The package
			// derives key sizes from the algorithm URI and does not consume
			// KeySize, so enforce at-most-one cardinality to stay consistent
			// with the other sub-element guards.
			if seenKeySize {
				return fmt.Errorf("%w: duplicate KeySize", ErrMalformedEncrypted)
			}
			seenKeySize = true
		case isXMLEncElem(e, "OAEPparams"):
			if seenOAEPParams {
				return fmt.Errorf("%w: duplicate OAEPparams", ErrMalformedEncrypted)
			}
			seenOAEPParams = true
			decoded, err := decodeBoundedBase64(ctx, e, "OAEPparams", maxOAEPParamsBytes, "invalid OAEPparams")
			if err != nil {
				return err
			}
			em.OAEPParams = decoded
		}
		return nil
	}); err != nil {
		return nil, err
	}

	return em, nil
}

// maxOAEPParamsBytes bounds the decoded octet length of one xenc:OAEPparams
// element. It is the single statement of that limit; decodeBoundedBase64
// applies it.
//
// OAEPparams carries the RSA-OAEP label — the "L" of RFC 8017 — which sender
// and recipient agree on out of band. RFC 8017 hashes the label down to the
// digest length before it is used, so nothing downstream is sized by it and a
// real one is a handful of octets: an application tag or a short identifier,
// twelve in this package's own round-trip test. 1 KiB is two orders of
// magnitude above that, so a hostile document can no longer make the parser
// hold a label sized only by what the XML parser happened to accept. That
// parser's own per-node content cap does not stand in for this one: it bounds a
// single indivisible run of characters, and a value spread over CDATA sections
// is as many runs as the document likes.
//
// The limit is a deliberate POLICY ceiling, not a conformance boundary. W3C
// xmlenc-core1 types OAEPparams as xs:base64Binary with no length facet, and
// RFC 8017 bounds the label only at its hash function's input limit, so a
// larger label is conforming and this package refuses it in both directions:
// resolveEncryptConfig holds Encryptor.OAEPParams to this same limit, so what
// this package writes is what it reads back and no label of its own making
// yields ciphertext it cannot decrypt.
const maxOAEPParamsBytes = 1024

// decodeBoundedBase64 decodes elem's character data into the octets of one
// xs:base64Binary value held to a FIXED ceiling, refusing a value over maxBytes
// before anything is built from it. It reads the two values this package bounds
// that way — an xenc:OAEPparams label against maxOAEPParamsBytes and an ECDH-ES
// dsig11:PublicKey against maxECPublicKeyBytes — and both are read while the
// document is parsed, so each is reached before any key is resolved and before
// anything the document says has been authenticated.
//
// valueName names the value in every refusal, including the child-kind refusals
// base64CharacterData raises, and invalidPhrase is how the decoder's own refusal
// is reported. Those two strings are the only thing in an error that says which
// value a document got wrong — every refusal here wraps ErrMalformedEncrypted —
// so each caller passes its own, and TestBoundedBase64ErrorsNameTheirOwnValue
// pins that they stay distinct.
//
// xs:base64Binary permits XML whitespace between characters and the value may
// arrive as any number of text and CDATA children, so the lexical length an
// attacker controls has no relation to the octets the limit measures. Joining
// the children into one string first would allocate that lexical length before
// the limit could refuse it, and would keep allocating it for every value the
// limit accepts, which leaves the accepted case unbounded as well.
//
// So the walk reads each child exactly once: it folds the child into an
// xmlbase64.Counter, which carries the counting state across every child
// boundary — a base64 quantum, and padding itself, may be split between
// children — and appends only the base64 characters. It stops the moment those
// characters pass maxChars, and the exact decoded-length test runs on the
// counter afterwards.
//
// What the parse RETAINS is therefore bounded by the limit and nothing else:
// the kept characters at most maxChars, the decoded bytes at most maxBytes. What
// is not bounded, and cannot be bounded here, is the copy [helium.Text.Content]
// hands out per child: a DOM offers no read-only view of a node's bytes, so
// weighing a value costs one copy of its lexical text. That copy is the floor,
// this walk pays it exactly once, and it is the whole remaining cost — a value
// padded with a megabyte of whitespace makes the parse allocate that megabyte
// once, not once per pass and not per subtree. Keeping it to once is what
// base64CharacterData's rule on which children may be read, and how far into one
// of them the walk goes, protects.
//
// The walk is over EVERY child kind, not just elements, so it goes through
// eachSibling directly: the children a value is spread across are text, CDATA,
// entity references, comments and processing instructions, and the ones that
// carry no characters at all cost the least to write. [xmlbase64.DecodeElement]
// counts and builds the same way but takes no context.Context, so routing this
// walk through it would drop the per-child cancellation poll eachSibling makes.
func decodeBoundedBase64(ctx context.Context, elem *helium.Element, valueName string, maxBytes int, invalidPhrase string) ([]byte, error) {
	// maxChars is the most base64 characters a value within maxBytes can hold:
	// ceil(maxBytes/3) quanta of four characters each. It sizes the buffer the
	// walk assembles and tells the walk when to stop, so what is being built is
	// sized by the limit and never by the document.
	//
	// Stopping there is only ever early, never a different verdict. Past this
	// count a value the decoder would accept has at least one more whole
	// quantum, which is three more octets less at most two of padding, so it is
	// over the limit; and a value the decoder would reject is charged its full
	// quantum count, which is over it too. Either way the exact
	// [xmlbase64.Counter.DecodedLen] test that follows the walk would refuse the
	// same value.
	maxChars := (maxBytes + 2) / 3 * 4
	var counter xmlbase64.Counter
	chars := make([]byte, 0, maxChars)
	if err := eachSibling(ctx, elem.FirstChild(), func(child helium.Node) error {
		text, err := base64CharacterData(child, valueName)
		if err != nil {
			return err
		}
		counter.Add(text)
		if counter.Chars() > maxChars {
			return fmt.Errorf("%w: %s is over the %d byte limit", ErrMalformedEncrypted, valueName, maxBytes)
		}
		chars = xmlbase64.AppendStripped(chars, text)
		return nil
	}); err != nil {
		return nil, err
	}
	if counter.DecodedLen() > maxBytes {
		return nil, abort(ctx, fmt.Errorf("%w: %s is over the %d byte limit", ErrMalformedEncrypted, valueName, maxBytes))
	}
	// The characters are already stripped, so this is the decode
	// xmlbase64.DecodeString performs minus the copy that would convert them to
	// a string, sized at the counted decoded length rather than at
	// encoding/base64's own padding-blind DecodedLen — see
	// [xmlbase64.Counter.DecodedLen], which is exact for a value the decoder
	// accepts and never below what a rejected decode writes.
	decoded := make([]byte, counter.DecodedLen())
	n, err := base64.StdEncoding.Decode(decoded, chars)
	if err != nil {
		return nil, abort(ctx, fmt.Errorf("%w: %s: %v", ErrMalformedEncrypted, invalidPhrase, err))
	}
	return decoded[:n], nil
}

// base64CharacterData returns the character data child contributes to an
// xs:base64Binary value, which is none at all for a child that carries none.
//
// Text and CDATA children carry it directly. An entity reference carries it
// through exactly one hop: a parsed [helium.EntityRef]'s first child is the
// [helium.Entity] it resolves to, and [helium.Entity.Content] is a leaf
// accessor returning the DECLARED replacement literal without recursing, so
// reading it costs one copy of that literal — the same one-copy floor a Text
// child already costs. Nothing below the Entity is read: an entity whose
// replacement is itself a reference or markup contributes that literal text
// (&inner;, <x/>), which is not base64, so the Counter marks the value
// undecodable and the decode fails, which is the right verdict for it. The
// Entity's NextSibling is deliberately not followed either — it is the next
// declaration in the DTD, not part of this value.
//
// A CHILDLESS EntityRef is a normal parser output, not a caller-built shape:
// per XML 1.0's "Entity Declared" constraint, an undeclared general entity is
// only a fatal well-formedness error when the document is standalone="yes" or
// has neither an external subset nor a parameter-entity reference. Otherwise it
// is a validity error, so a non-validating parse keeps the reference as an
// EntityRef with no Entity under it. Such a reference carries no character data
// and contributes none, which is what [c14n] does with it too — it
// canonicalizes an EntityRef by walking its children, so a childless one
// contributes nothing there either. Refusing it here would reject documents the
// parser accepts and the canonicalizer reads as empty. An EntityRef with a
// NON-NIL first child that is not an Entity is reachable only in a caller-built
// tree and is refused, so the one-hop bound holds for every tree.
//
// [helium.Element] is refused: it answers Content by aggregating its whole
// subtree into one buffer, so asking such a child for its content spends the
// entire subtree's text before a limit measured in DECODED OCTETS can look at
// it — and since a subtree of whitespace decodes to nothing, the limit would
// not fire at all. Refusing rather than skipping is what the value says too: an
// element child makes the content invalid xs:base64Binary, and quietly dropping
// it would decode a value the document did not write.
//
// Comments and processing instructions may appear inside any element and are
// not character data, so they are skipped, exactly as XSD ignores them when it
// builds a simple type's value. Reading them instead would splice comment text
// straight into the base64 stream.
func base64CharacterData(child helium.Node, valueName string) ([]byte, error) {
	if t, ok := helium.AsNode[*helium.Text](child); ok {
		return t.Content(), nil
	}
	if c, ok := helium.AsNode[*helium.CDATASection](child); ok {
		return c.Content(), nil
	}
	if ref, ok := helium.AsNode[*helium.EntityRef](child); ok {
		first := ref.FirstChild()
		if first == nil {
			return nil, nil
		}
		entity, ok := helium.AsNode[*helium.Entity](first)
		if !ok {
			return nil, fmt.Errorf("%w: %s holds an entity reference whose first child is not an entity declaration", ErrMalformedEncrypted, valueName)
		}
		return entity.Content(), nil
	}
	if _, ok := helium.AsNode[*helium.Comment](child); ok {
		return nil, nil
	}
	if _, ok := helium.AsNode[*helium.ProcessingInstruction](child); ok {
		return nil, nil
	}
	return nil, fmt.Errorf("%w: %s holds a child of type %s, which is not character data", ErrMalformedEncrypted, valueName, child.Type())
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
//
// budget, when non-nil, is charged what decoding the CipherValue would cost —
// xmlbase64.Counter owns that count, including what it charges base64 the
// decoder will reject — before anything is built from it, so an over-budget
// value never reaches the decoder. decodeCipherValue owns how that charge is
// kept ahead of the work. A nil budget leaves the value unbounded, which is
// only used by parser-only test helpers.
func parseCipherData(ctx context.Context, elem *helium.Element, budget cipherValueBudget) ([]byte, error) {
	var decoded []byte
	var seenChoice bool
	if err := eachChildElement(ctx, elem, func(e *helium.Element) error {
		switch {
		case isXMLEncElem(e, "CipherValue"):
			if seenChoice {
				return fmt.Errorf("%w: CipherData allows exactly one of CipherValue or CipherReference", ErrMalformedEncrypted)
			}
			seenChoice = true
			d, err := decodeCipherValue(ctx, e, budget)
			if err != nil {
				return err
			}
			decoded = d
		case isXMLEncElem(e, "CipherReference"):
			if seenChoice {
				return fmt.Errorf("%w: CipherData allows exactly one of CipherValue or CipherReference", ErrMalformedEncrypted)
			}
			return fmt.Errorf("%w: CipherReference is not supported", ErrMalformedEncrypted)
		}
		return nil
	}); err != nil {
		return nil, err
	}
	if !seenChoice {
		return nil, abort(ctx, fmt.Errorf("%w: missing CipherValue", ErrMalformedEncrypted))
	}
	return decoded, nil
}

// decodeCipherValue charges budget for elem's base64 content and then decodes
// it. It never builds the value's lexical text.
//
// That distinction is the whole point. xs:base64Binary permits XML whitespace
// between characters, and a CipherValue's text may arrive as any number of
// text and CDATA children, so the lexical length an attacker controls has no
// relation to the decoded bytes the budget charges. Joining the children into
// one string first would allocate that lexical length before the budget could
// refuse it, and would keep allocating it for every value the budget accepts,
// which leaves the accepted case unbounded as well.
//
// So one pass counts the value with an xmlbase64.Counter, which carries the
// counting state across each child boundary — a base64 quantum, and padding
// itself, may be split between children — and only then is the charge made.
// The second pass builds just the characters the decoder will see.
//
// What is held is therefore bounded by the budget: the stripped characters at
// most four thirds of it, the decoded bytes at most it, and nothing at all
// scaling with the lexical length. The copy [helium.Text.Content] returns per
// child is the floor, so the peak is the largest single child rather than the
// whole value.
// Both passes walk EVERY child kind, so they go through eachSibling directly
// rather than eachChildElement. The children that contribute nothing to the
// value — a comment, a processing instruction — are charged against no budget,
// which makes their number the one thing here the document sets for free; the
// per-child poll is what bounds the time a cancelled caller waits for them.
func decodeCipherValue(ctx context.Context, elem *helium.Element, budget cipherValueBudget) ([]byte, error) {
	var counter xmlbase64.Counter
	if err := eachSibling(ctx, elem.FirstChild(), func(child helium.Node) error {
		content, err := base64CharacterData(child, "CipherValue")
		if err != nil {
			return err
		}
		counter.Add(content)
		return nil
	}); err != nil {
		return nil, err
	}
	if budget != nil {
		if err := budget.charge(counter.DecodedLen()); err != nil {
			return nil, abort(ctx, err)
		}
	}
	chars := make([]byte, 0, counter.Chars())
	if err := eachSibling(ctx, elem.FirstChild(), func(child helium.Node) error {
		content, err := base64CharacterData(child, "CipherValue")
		if err != nil {
			return err
		}
		chars = xmlbase64.AppendStripped(chars, content)
		return nil
	}); err != nil {
		return nil, err
	}
	// The characters are already stripped, so this is the decode
	// xmlbase64.DecodeString performs minus the copy that would convert them to
	// a string, sized at the count the budget charged rather than at
	// encoding/base64's own padding-blind DecodedLen — see
	// [xmlbase64.Counter.DecodedLen].
	decoded := make([]byte, counter.DecodedLen())
	n, err := base64.StdEncoding.Decode(decoded, chars)
	if err != nil {
		return nil, abort(ctx, fmt.Errorf("%w: invalid CipherValue: %v", ErrMalformedEncrypted, err))
	}
	return decoded[:n], nil
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
