package xmlenc1

import (
	"context"
	"crypto/ecdh"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/domutil"
	"github.com/lestrrat-go/helium/internal/xmlbase64"
)

// parseState bundles the per-parse state parseEncryptedData and its callees
// thread through the document walk: cfg supplies the effective EncryptedKey
// candidate limit, keys carries the cumulative EncryptedKey ciphertext
// allowance across the whole element, payload caps the EncryptedData
// CipherValue, refs is the same-document scope a ds:RetrievalMethod resolves
// in, and visited holds the candidates already taken. Both budgets are charged
// at the earliest point that sees their values; the candidate limit is checked
// in retainEncryptedKey, which is where a candidate is retained.
//
// Bundling these into one struct keeps parseEncryptedData's signature from
// growing a parameter per addition; a later addition to this struct is
// expected and should not require touching every call site's parameter list.
//
// It is per-parse state and never shared between calls, which is part of what
// keeps a Decryptor safe to fire from several goroutines.
type parseState struct {
	cfg     *decryptConfig
	keys    *encryptedKeyBudget
	payload *payloadCipherValueBudget
	refs    *refScope

	// visited records the xenc:EncryptedKey elements already taken as
	// candidates of this EncryptedData. A document may offer one element
	// both inline and by ds:RetrievalMethod, and xmlenc-core1 §3.5.3 permits
	// several of those methods, so one element can be offered many ways; it
	// describes one key and must be charged and trial-decrypted once. This
	// is DEDUPLICATION, not a loop guard — see parseRetrievalMethod for why
	// no cycle is expressible. retainEncryptedKey is its only reader and its
	// only writer.
	visited map[*helium.Element]struct{}
}

// markVisited records elem as taken and reports whether it already was, so an
// element offered a second time adds no second candidate.
//
// The map is built on first use, so an EncryptedData that offers no candidate
// never allocates one and every parseState is usable with the field left zero,
// whichever caller assembled it.
func (ps *parseState) markVisited(elem *helium.Element) bool {
	if _, seen := ps.visited[elem]; seen {
		return true
	}
	if ps.visited == nil {
		ps.visited = make(map[*helium.Element]struct{})
	}
	ps.visited[elem] = struct{}{}
	return false
}

// parseEncryptedData parses an EncryptedData element.
//
// ctx is observed throughout the parse, which is why every function below takes
// it. The document decides how many children each element it describes carries,
// and children the parse skips — a comment inside a CipherValue is the cheapest
// of them — are charged against no budget at all, so the trip count of these
// walks is bounded by nothing but the document's size. Every one of them
// therefore runs through eachSibling, and every error return runs through abort,
// so a caller that cancels is answered while the parse is still reading rather
// than after it finishes. A live context leaves both untouched.
func parseEncryptedData(ctx context.Context, elem *helium.Element, ps *parseState) (*encryptedData, error) {
	if elem == nil || !isXMLEncElem(elem, "EncryptedData") {
		return nil, abort(ctx, fmt.Errorf("%w: expected xenc:EncryptedData", ErrMalformedEncrypted))
	}

	ed := &encryptedData{}
	ed.ID, _ = elem.GetAttribute("Id")
	ed.Type, _ = elem.GetAttribute("Type")

	// Track CipherData separately: a decoded CipherValue can be a non-nil
	// empty slice, so a boolean is the reliable duplicate sentinel.
	var seenCipherData bool

	// KeyInfo is a singleton in the xenc:EncryptedType sequence. Its branch
	// appends candidates rather than assigning a field, so a boolean is the
	// only reliable sentinel.
	var seenKeyInfo bool

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
			if seenKeyInfo {
				return fmt.Errorf("%w: duplicate KeyInfo", ErrMalformedEncrypted)
			}
			seenKeyInfo = true
			return parseKeyInfoForEncryption(ctx, e, ed, ps)
		case isXMLEncElem(e, "CipherData"):
			if seenCipherData {
				return fmt.Errorf("%w: duplicate CipherData", ErrMalformedEncrypted)
			}
			seenCipherData = true
			cv, err := parseCipherData(ctx, e, ps, ps.payload)
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

// parseKeyInfoForEncryption reads the ds:KeyInfo of an EncryptedData for the
// session-key candidates it offers: an xenc:EncryptedKey carried inline, and a
// ds:RetrievalMethod naming one elsewhere in the same document (xmlenc-core1
// §3.5.3).
//
// The two branches are taken in DOCUMENT ORDER rather than inline-first, so a
// reference-supplied candidate lands where the document put the reference. The
// candidate list is tried in order, so reordering it would change which key a
// document with several candidates is decrypted under.
func parseKeyInfoForEncryption(ctx context.Context, elem *helium.Element, ed *encryptedData, ps *parseState) error {
	return eachChildElement(ctx, elem, func(e *helium.Element) error {
		switch {
		case isXMLEncElem(e, "EncryptedKey"):
			return retainEncryptedKey(ctx, e, ed, ps)
		case isDSigElem(e, "RetrievalMethod"):
			return parseRetrievalMethod(ctx, e, ed, ps)
		}
		return nil
	})
}

// retainEncryptedKey takes elem as one more session-key candidate of ed, or
// takes nothing when elem is already one. It is the SINGLE retention point for
// both ways a ds:KeyInfo offers an xenc:EncryptedKey — carried inline, and
// named by a ds:RetrievalMethod (xmlenc-core1 §3.5.3) — because a document may
// offer the same element BOTH ways in either order, and one element describes
// one key however many times it is offered.
//
// The order of the three steps is the invariant:
//
//   - the visited-set dedup runs FIRST, so an element already taken adds no
//     second candidate whichever way it was taken the first time;
//   - the candidate slot is charged only for an element being retained, so the
//     peak charge equals the final candidate count exactly;
//   - parseEncryptedKey, which spends the cumulative EncryptedKey ciphertext
//     budget, runs exactly once per retained candidate and never for a repeat.
func retainEncryptedKey(ctx context.Context, elem *helium.Element, ed *encryptedData, ps *parseState) error {
	if ps.markVisited(elem) {
		return nil
	}
	if err := checkEncryptedKeyCap(ps.cfg, len(ed.EncryptedKeys)+1); err != nil {
		return abort(ctx, err)
	}
	ek, err := parseEncryptedKey(ctx, elem, ps)
	if err != nil {
		return err
	}
	ed.EncryptedKeys = append(ed.EncryptedKeys, ek)
	return nil
}

// parseRetrievalMethod reads one ds:RetrievalMethod child of a ds:KeyInfo and
// appends the EncryptedKey candidate it supplies to ed, appending nothing when
// the reference supplies none. xmlenc-core1 §3.5.3 makes support for the
// same-document form REQUIRED: it links to the EncryptedKey holding the key
// needed to decrypt the associated CipherData, is always a child of
// ds:KeyInfo, and may appear several times.
//
// It appends rather than returning the candidate because most of its outcomes
// are "no candidate, no error" — a Type this package does not read, and a
// target already taken by an earlier reference — and a returned (nil, nil)
// would make every one of those indistinguishable from a bug at the call site.
//
// The @Type decides how far the reference is read, in this order:
//
//   - a Type this package does not implement (typeDerivedKey, or any foreign
//     URI) is stepped over with no error and no cost — no candidate slot, no
//     URI lookup — so a document naming a construct this package never
//     honored behaves exactly as it did before references were resolved;
//   - TypeEncryptedKey must name an xenc:EncryptedKey. Naming anything else is
//     a contradiction inside the document, refused as ErrMalformedEncrypted;
//   - an absent Type says nothing about what the URI names, so the target is
//     used only if it IS an xenc:EncryptedKey and passed over otherwise.
//
// A retained target is handed to retainEncryptedKey, which owns what retaining
// one costs and in which order. A reference that retains nothing — one naming
// a target already taken, or a target that is not an xenc:EncryptedKey — costs
// one probe of the id index that resolveSameDocument builds once per decrypt,
// and nothing else.
//
// A URI that is not a same-document reference is ErrReferenceNotFound whatever
// it names, and no setting lifts that: §3.5 mandates only the same-document
// form, and an external key location decides which key material the recipient
// trial-decrypts.
//
// # Why no recursion is reachable
//
// Resolution is one step deep BY CONSTRUCTION, not by a depth limit.
// parseEncryptedKey inspects exactly three children — xenc:EncryptionMethod,
// xenc:CipherData, and ds:KeyInfo, and of that ds:KeyInfo only its
// xenc:AgreementMethod — and no path below it ever looks for a
// ds:RetrievalMethod. A resolved target therefore cannot itself resolve a
// reference, so §6.5's EncryptedKey A → B → A cycle is not expressible here
// and there is no depth constant to tune. The visited set is DEDUPLICATION,
// for the several references §3.5.3 permits; it is not a loop guard, and the
// termination argument does not rest on it.
func parseRetrievalMethod(ctx context.Context, elem *helium.Element, ed *encryptedData, ps *parseState) error {
	typ, _ := elem.GetAttribute("Type")
	switch typ {
	case "", TypeEncryptedKey:
	case typeDerivedKey:
		// Recognized, and deliberately not resolved: this package derives no
		// key from master key material. README.md's Conformance scope section
		// owns what such a document fails with.
		return nil
	default:
		// A Type from another specification entirely, e.g. a ds:X509Data.
		// Nothing here reads it, so nothing here refuses it either.
		return nil
	}

	uri, _ := elem.GetAttribute("URI")
	target, err := resolveSameDocument(ctx, ps.refs, uri)
	if err != nil {
		return abort(ctx, err)
	}
	if !isXMLEncElem(target, "EncryptedKey") {
		if typ == "" {
			return nil
		}
		return abort(ctx, fmt.Errorf("%w: ds:RetrievalMethod URI %q declares Type %q but names a %s", ErrMalformedEncrypted, uri, typ, target.Name()))
	}
	return retainEncryptedKey(ctx, target, ed, ps)
}

// parseEncryptedKey parses an EncryptedKey element, charging its CipherValue
// to ps.keys.
//
// A xenc:CarriedKeyName child is stepped over rather than read;
// [encryptedKey.CarriedKeyName] owns why the parse leaves that field unset.
func parseEncryptedKey(ctx context.Context, elem *helium.Element, ps *parseState) (*encryptedKey, error) {
	if elem == nil || !isXMLEncElem(elem, "EncryptedKey") {
		return nil, abort(ctx, fmt.Errorf("%w: expected xenc:EncryptedKey", ErrMalformedEncrypted))
	}

	ek := &encryptedKey{}
	ek.ID, _ = elem.GetAttribute("Id")
	ek.recipient, _ = elem.GetAttribute("Recipient")

	// KeyInfo is a singleton in the xenc:EncryptedType sequence. Its branch
	// appends candidates rather than assigning a field, so a boolean is the
	// only reliable sentinel.
	var seenCipherData, seenKeyInfo bool

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
			cv, err := parseCipherData(ctx, e, ps, ps.keys)
			if err != nil {
				return err
			}
			ek.CipherValue = cv
		case isDSigElem(e, "KeyInfo"):
			if seenKeyInfo {
				return fmt.Errorf("%w: duplicate KeyInfo", ErrMalformedEncrypted)
			}
			seenKeyInfo = true
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

func parseAgreementMethodForKeyInfo(ctx context.Context, elem *helium.Element) (*agreementMethod, error) {
	var agreement *agreementMethod
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

func parseAgreementMethod(ctx context.Context, elem *helium.Element) (*agreementMethod, error) {
	algorithm, ok := elem.GetAttribute("Algorithm")
	if !ok || algorithm == "" {
		return nil, abort(ctx, fmt.Errorf("%w: AgreementMethod missing/empty Algorithm", ErrMalformedEncrypted))
	}
	agreement := &agreementMethod{Algorithm: algorithm}
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

func parseKeyDerivationMethod(ctx context.Context, elem *helium.Element) (*keyDerivationMethod, error) {
	algorithm, ok := elem.GetAttribute("Algorithm")
	if !ok || algorithm == "" {
		return nil, abort(ctx, fmt.Errorf("%w: KeyDerivationMethod missing/empty Algorithm", ErrMalformedEncrypted))
	}
	method := &keyDerivationMethod{Algorithm: algorithm}
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
	// xs:hexBinary carries whiteSpace='collapse', whose whitespace is exactly
	// #x20, #x9, #xD and #xA. strings.TrimSpace would also strip U+00A0 and
	// the other Unicode spaces, which are not in the lexical space.
	if !ok || strings.Trim(value, " \t\r\n") == "" {
		return nil, 0, nil
	}
	value = strings.Trim(value, " \t\r\n")
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
func parseOriginatorKeyInfo(ctx context.Context, elem *helium.Element) (*ecKeyValue, error) {
	var found *ecKeyValue
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

func parseECKeyValue(ctx context.Context, elem *helium.Element) (*ecKeyValue, error) {
	var curve ecdh.Curve
	var publicKey []byte
	// NamedCurve and PublicKey are both singletons in the
	// dsig11:ECKeyValueType sequence. Boolean sentinels are used because a
	// decoded PublicKey can be a non-nil empty slice, the same reason the
	// CipherData guards use one.
	var seenNamedCurve, seenPublicKey bool
	if err := eachChildElement(ctx, elem, func(e *helium.Element) error {
		if e.URI() != NamespaceDSig11 {
			return nil
		}
		switch domutil.LocalName(e) {
		case "NamedCurve":
			if seenNamedCurve {
				return fmt.Errorf("%w: duplicate ECKeyValue NamedCurve", ErrMalformedEncrypted)
			}
			seenNamedCurve = true
			uri, _ := e.GetAttribute("URI")
			named, err := ecdhCurveForURI(uri)
			if err != nil {
				return err
			}
			curve = named
		case "PublicKey":
			if seenPublicKey {
				return fmt.Errorf("%w: duplicate ECKeyValue PublicKey", ErrMalformedEncrypted)
			}
			seenPublicKey = true
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
	return &ecKeyValue{curve: curve, PublicKey: publicKey}, nil
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
func parseEncryptionMethod(ctx context.Context, elem *helium.Element) (*encryptionMethod, error) {
	em := &encryptionMethod{}
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
			// derives key sizes from the algorithm URI and never uses KeySize
			// as a length SOURCE, but xmlenc-core1 §3.2 makes a KeySize that
			// CONTRADICTS the algorithm an error ("the presence of a KeySize
			// child inconsistent with the algorithm MUST be treated as an
			// error"; §5.3 states the same rule from the other side), so a
			// present value is still read and checked against it.
			if seenKeySize {
				return fmt.Errorf("%w: duplicate KeySize", ErrMalformedEncrypted)
			}
			seenKeySize = true
			bits, err := parseKeySizeBits(ctx, e)
			if err != nil {
				return err
			}
			if err := checkDeclaredKeySize(em.Algorithm, bits); err != nil {
				return err
			}
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

// maxKeySizeChars bounds the lexical length of one xenc:KeySize value.
// KeySize is xs:integer naming a key length in bits (§5.6.2.2), and every
// value that can agree with an algorithm this package implements is three
// digits (128, 192, 256). 64 leaves ample room for the whitespace, sign and
// leading zeros xs:integer's lexical space permits, while still keeping the
// parse from holding a number sized only by what the document says.
const maxKeySizeChars = 64

// parseKeySizeBits reads elem's character data as the lexical value of one
// xenc:KeySize element and returns it as a number of bits.
//
// It walks elem's children directly rather than calling [helium.Element.Content],
// for the same reason [decodeBoundedBase64] does: that accessor aggregates the
// whole descendant subtree into one buffer, and this value is read while the
// document is parsed, before anything it says is authenticated. It reuses
// base64CharacterData, which already encodes the child-kind rules this value
// needs too — despite the helper's name, those rules hold for any XML simple
// content, not only base64: text and CDATA carry the value, an entity
// reference contributes its declared replacement text through one hop, a
// comment or processing instruction is ignored, and an element child is
// refused. maxKeySizeChars is enforced as the characters are gathered, so a
// document cannot make this walk hold a value sized only by what it sends.
//
// The gathered text is trimmed of the four characters xs:integer's
// whiteSpace='collapse' facet permits and parsed with [strconv.Atoi], which
// accepts the rest of that lexical space (an optional sign and leading zeros).
// A value outside the lexical space is ErrMalformedEncrypted naming it. A value
// INSIDE it that no int can hold is not: xs:integer is unbounded, so such a
// value is returned clamped and left to the consistency check.
func parseKeySizeBits(ctx context.Context, elem *helium.Element) (int, error) {
	var chars []byte
	if err := eachSibling(ctx, elem.FirstChild(), func(child helium.Node) error {
		text, err := base64CharacterData(child, "KeySize")
		if err != nil {
			return err
		}
		if len(chars)+len(text) > maxKeySizeChars {
			return fmt.Errorf("%w: KeySize is over the %d character limit", ErrMalformedEncrypted, maxKeySizeChars)
		}
		chars = append(chars, text...)
		return nil
	}); err != nil {
		return 0, err
	}
	// xs:integer carries whiteSpace='collapse', whose whitespace is exactly
	// #x20, #x9, #xD and #xA. strings.TrimSpace would also strip U+00A0 and
	// the other Unicode spaces, which are not in the lexical space.
	value := strings.Trim(string(chars), " \t\r\n")
	bits, err := strconv.Atoi(value)
	if err != nil && !errors.Is(err, strconv.ErrRange) {
		return 0, abort(ctx, fmt.Errorf("%w: invalid KeySize %q", ErrMalformedEncrypted, value))
	}
	// A well-formed xs:integer too large for an int is a value the schema
	// permits, so it is not malformed. Atoi has already clamped it to
	// MaxInt/MinInt, which can never equal an algorithm's key size, so a URI
	// that implies a length still reports the contradiction and one that
	// implies none still ignores it.
	return bits, nil
}

// checkDeclaredKeySize enforces xmlenc-core1 §3.2's rule that a KeySize
// inconsistent with the algorithm is an error (§5.3 restates it from the
// EncryptionMethod side, and §5.6.2.2 fixes the unit: KeySize is "The size
// in bits of the key") against algorithm's own implied length.
//
// keySizeForAlgorithm answers with an error for a URI that implies no length
// at all — RSA key transport above all — and that is not a contradiction:
// nothing on the wire disagrees with anything, so bits is accepted and
// silently ignored, which is exactly what a URI silent on key length
// anticipates. Only a URI that DOES imply a length is checked against it.
func checkDeclaredKeySize(algorithm string, bits int) error {
	want, err := keySizeForAlgorithm(paramBlockAlgorithm, algorithm)
	if err == nil && bits != want*8 {
		return fmt.Errorf("%w: KeySize %d bits is inconsistent with algorithm %q, which requires %d bits", ErrMalformedEncrypted, bits, algorithm, want*8)
	}
	return nil
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
// The literal is returned in FULL rather than truncated to what the caller's
// limit still has room for, and that whole cost is one transient copy of one
// child: every caller weighs the returned text against its own limit and stops
// at the first child that exceeds it, so at most one such copy is ever taken.
// Its size is bounded twice over by the parser that built the tree —
// [helium.DefaultMaxNodeContentSize] caps the declared literal itself, and
// [helium.DefaultMaxEntityAmplification] caps the total expanded entity bytes
// a document may reach as a multiple of its own input size, which is what
// bounds how many references to that literal it can carry. Because the literal
// is UNEXPANDED, a chain of entities
// each referencing the one below costs the length of the reference text it is
// written with (&inner;&inner;) rather than the length that text would expand
// to, so nesting buys a document no size it did not already write. All of that
// is measurable: a 4 MiB entity replacement and a 4 MiB plain text child
// allocate the same 4.2 MB through [Decryptor.Decrypt].
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
// silently using the first.
//
// Both members yield the same thing — the cipher text octets — so the two
// branches differ only in where those octets come from: a CipherValue carries
// them inline as base64, while a CipherReference names them by URI and
// resolveCipherReference (reference.go) owns what that means and what it costs.
//
// The cardinality is settled in a first walk that reads no value at all, and
// only then is the single member it found turned into octets. That order is
// what keeps the schema-validity verdict independent of what the members hold:
// a document offering two of them is refused for offering two, whether or not
// the first one would have decoded or resolved. It also keeps a schema-invalid
// document from spending anything — a CipherReference that reached a caller's
// resolver first would turn an invalid document into a dereference.
//
// budget, when non-nil, is charged what the chosen member costs before anything
// is built from it, so an over-budget value never reaches the decoder or the
// canonicalizer. decodeCipherValue and resolveCipherReference each own how that
// charge is kept ahead of their work. A nil budget leaves the value unbounded,
// which is only used by parser-only test helpers.
//
// budget is passed explicitly rather than sourced from ps, because which of
// ps.payload or ps.keys applies is a property of the caller (an EncryptedData's
// own CipherData vs. an EncryptedKey's), not of parseState itself.
func parseCipherData(ctx context.Context, elem *helium.Element, ps *parseState, budget cipherValueBudget) ([]byte, error) {
	var choice *helium.Element
	if err := eachChildElement(ctx, elem, func(e *helium.Element) error {
		if !isXMLEncElem(e, "CipherValue") && !isXMLEncElem(e, "CipherReference") {
			return nil
		}
		if choice != nil {
			return fmt.Errorf("%w: CipherData allows exactly one of CipherValue or CipherReference", ErrMalformedEncrypted)
		}
		choice = e
		return nil
	}); err != nil {
		return nil, err
	}
	if choice == nil {
		return nil, abort(ctx, fmt.Errorf("%w: CipherData carries neither a CipherValue nor a CipherReference", ErrMalformedEncrypted))
	}
	if isXMLEncElem(choice, "CipherValue") {
		return decodeCipherValue(ctx, choice, budget)
	}
	return resolveCipherReference(ctx, choice, ps, budget)
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
