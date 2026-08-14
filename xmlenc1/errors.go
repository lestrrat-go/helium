package xmlenc1

import (
	"errors"
	"fmt"
)

var (
	// ErrDecryptionFailed is returned when decryption fails.
	ErrDecryptionFailed = errors.New("xmlenc1: decryption failed")

	// ErrEncryptionFailed is returned when encryption fails.
	ErrEncryptionFailed = errors.New("xmlenc1: encryption failed")

	// ErrMissingKey is returned when no decryption key is available.
	ErrMissingKey = errors.New("xmlenc1: no decryption key available")

	// ErrUnsupportedKeyDerivation is returned when an EncryptedData offers its
	// session key ONLY through an xenc11:DerivedKey (xmlenc-core1 §3.5.2),
	// which tells the recipient to derive the content key from master key
	// material it already holds. This package implements no key derivation from
	// a master key, so it cannot decrypt such a document.
	//
	// It is deliberately NOT ErrMissingKey. That sentinel says no decryption key
	// is available, which sends a caller to audit the keys it configured — and no
	// key it configures can help, because the document asked for a facility this
	// package does not have. A caller must be able to tell "I set this up wrong"
	// from "helium cannot read this document at all". Match with errors.Is.
	//
	// It covers both ways a ds:KeyInfo offers the construct: an xenc11:DerivedKey
	// carried inline, and a ds:RetrievalMethod whose Type names one. It is raised
	// only when derivation was the ONLY option: an EncryptedData that ALSO
	// carries a usable xenc:EncryptedKey decrypts under that key, since the
	// unimplemented construct costs a document nothing it could otherwise do.
	//
	// A pre-shared [Decryptor.SessionKey] never reaches it. That early return
	// precedes key resolution entirely, so a caller holding the session key
	// decrypts a document whose ds:KeyInfo this package cannot read.
	ErrUnsupportedKeyDerivation = errors.New("xmlenc1: key derivation from master key material is not supported")

	// ErrTooManyEncryptedKeys is returned when an EncryptedData carries more
	// EncryptedKey candidates than the Decryptor's effective limit, which
	// guards against CPU amplification (DoS). Decryptor.MaxEncryptedKeys owns
	// the cap: the per-candidate cost it bounds and the effective-limit
	// rules. See also DefaultMaxEncryptedKeys.
	ErrTooManyEncryptedKeys = errors.New("xmlenc1: too many EncryptedKey candidates")

	// ErrEncryptedKeyBytesExceeded is returned when the EncryptedKey
	// candidates of one EncryptedData carry more ciphertext together than
	// the Decryptor's effective byte budget, which guards against memory
	// amplification (DoS). Decryptor.MaxEncryptedKeyBytes owns the budget:
	// what it covers, when it is charged, and the effective-limit rules.
	// See also DefaultMaxEncryptedKeyBytes.
	//
	// It is distinct from ErrTooManyEncryptedKeys because the two bound
	// different things and are raised or lifted by different setters: too
	// many candidates is a count, too much ciphertext is a size, and a
	// caller handling one must be able to tell which limit to raise.
	ErrEncryptedKeyBytesExceeded = errors.New("xmlenc1: EncryptedKey ciphertext exceeds the byte budget")

	// ErrCipherValueBytesExceeded is returned when an EncryptedData payload
	// CipherValue exceeds the Decryptor's effective byte budget. The payload
	// may arrive as many text or CDATA nodes, so this limit is separate from
	// the parser's per-node content limit. Decryptor.MaxCipherValueBytes owns
	// the budget, including its effective-limit rules. See also
	// DefaultMaxCipherValueBytes.
	ErrCipherValueBytesExceeded = errors.New("xmlenc1: EncryptedData CipherValue exceeds the byte budget")

	// ErrReferenceNotFound is returned when a ds:RetrievalMethod or an
	// xenc:CipherReference names something this package will not resolve: a
	// same-document reference matching no element in the document the
	// EncryptedData belongs to, or a URI that is not a same-document reference
	// at all and cannot be dereferenced. The shipped FSReferenceResolver wraps
	// it for every URI shape it refuses and for a resource it cannot read.
	//
	// The two constructs differ in whether an external URI can ever be
	// resolved:
	//
	//   - a ds:RetrievalMethod naming another resource is refused whatever it
	//     names, with no setting that lifts the refusal. Only the
	//     same-document form is REQUIRED (xmlenc-core1 §3.5), no external form
	//     is mandated anywhere, and an external key location decides which key
	//     material the recipient trial-decrypts — not a decision a document
	//     gets to make for a caller.
	//   - an xenc:CipherReference naming another resource is refused until the
	//     caller supplies a Decryptor.CipherReferenceResolver. §3.3.1 imports
	//     XMLDSig's dereferencing model, which makes the same-document forms
	//     MUSTs and HTTP dereferencing RECOMMENDED, so the external form is an
	//     opt-in capability, and no obligation.
	//
	// The reference is resolved while the document is read, so this precedes
	// the Decryptor.SessionKey early return: a pre-shared session key does not
	// decrypt past a reference that was refused. A ds:RetrievalMethod whose
	// Type this package does not implement is never resolved at all and so
	// never reaches this error. Match with errors.Is.
	ErrReferenceNotFound = errors.New("xmlenc1: reference not found")

	// ErrAmbiguousReference is returned when a same-document
	// ds:RetrievalMethod or xenc:CipherReference URI matches more than one
	// element.
	//
	// This is XML Signature Wrapping applied to encryption. An attacker who
	// can inject an element carrying an Id already in use would otherwise
	// choose which of the two the recipient resolves, and so which key it
	// unwraps and trial-decrypts with, or which octets it takes as the cipher
	// text. Resolution therefore collects every match and refuses on more than
	// one, taking neither. Match with errors.Is.
	ErrAmbiguousReference = errors.New("xmlenc1: ambiguous reference")

	// ErrInvalidPadding names invalid AES-CBC padding. Decryption never
	// returns it: distinguishing a padding failure from any other CBC
	// failure is exactly what a padding oracle needs, so decryptCBC
	// collapses every cause to ErrDecryptionFailed. It remains exported so
	// that decryptCipherValue can defensively squash it should any future
	// code path start returning it, and so tests can assert it never
	// escapes.
	//
	// Deprecated: matching on this sentinel never succeeds. Test for
	// ErrDecryptionFailed instead.
	ErrInvalidPadding = errors.New("xmlenc1: invalid AES-CBC padding")

	// ErrKeyUnwrapFailed is returned when AES key unwrap integrity check
	// fails. It is always wrapped in ErrDecryptionFailed, so a caller that
	// tests only for ErrDecryptionFailed catches a failed key unwrap the
	// same way it catches a failed RSA key transport.
	ErrKeyUnwrapFailed = errors.New("xmlenc1: AES key unwrap integrity check failed")

	// ErrMalformedEncrypted is returned when an EncryptedData element is malformed.
	ErrMalformedEncrypted = errors.New("xmlenc1: malformed EncryptedData element")

	// ErrOpaquePayload is returned by Decryptor.Decrypt when the
	// EncryptedData's @Type does not declare XML content: it is absent, empty,
	// or a URI other than TypeElement and TypeContent. Decrypt is the XML
	// path, so it refuses such a payload, and every message wrapping this
	// sentinel names Decryptor.DecryptBytes, which returns the plaintext
	// octets without parsing them.
	//
	// @Type sits OUTSIDE the ciphertext and is authenticated by nothing, not
	// even AES-GCM. Treating an absent or unrecognized value as TypeElement
	// would let anyone who can edit the document delete the attribute and have
	// an opaque octet stream parsed as XML and handed back as nodes to graft
	// into a tree — type confusion decided by an attribute the recipient
	// cannot verify. xmlenc-core1 §4.2 asks a decryptor to take an unknown or
	// empty Type as a signal that the cleartext is an opaque octet stream, and
	// §3.1 puts the absent case in the same position.
	//
	// It is deliberately NOT ErrMalformedEncrypted: such a document is
	// well-formed, and a caller must be able to tell "this payload is opaque,
	// retry with DecryptBytes" from a document it should reject outright.
	// DecryptBytes never returns it — it does not read @Type at all. Match
	// with errors.Is.
	ErrOpaquePayload = errors.New("xmlenc1: EncryptedData payload is not XML")

	// ErrMissingConfig is returned when required encryption config is missing.
	ErrMissingConfig = errors.New("xmlenc1: missing required configuration")

	// ErrConflictingKeyConfig is returned when an Encryptor configures two of
	// the three ways to protect the session key: RSA key transport
	// (KeyTransportAlgorithm + RecipientPublicKey), ECDH-ES key agreement
	// (KeyWrapAlgorithm + RecipientECPublicKey), and AES key wrapping
	// (KeyWrapAlgorithm + KeyEncryptionKey). Any pair of them fails with this
	// error, naming both of the configured things so the caller knows which
	// two to choose between. A SessionKey alongside a single mechanism is
	// not a pair: it supplies the key that mechanism protects.
	//
	// An EncryptedData carries a single EncryptedKey here, so honoring one
	// mechanism means silently discarding the other — and a recipient
	// holding only the discarded key then fails to decrypt with an error
	// that points nowhere near the real mistake. The caller must pick one.
	ErrConflictingKeyConfig = errors.New("xmlenc1: conflicting key protection configured")

	// ErrConflictingBlockAlgorithm is returned when an EncryptedData declares a
	// block algorithm in its EncryptionMethod and the Decryptor was given a
	// different one through Decryptor.BlockAlgorithm. The message names both
	// URIs so the caller knows which of the two to change.
	//
	// The two are matched on purpose, and never ordered. Decryptor.BlockAlgorithm
	// exists for an EncryptedData that carries no EncryptionMethod at all, where
	// the algorithm is known out of band (W3C xmlenc-core1 §3.1, §4.4); letting a
	// document's declaration win over a caller who stated the algorithm would let
	// the document choose the cipher the recipient runs, which is exactly the
	// algorithm confusion the setter must not introduce. Under a strict match,
	// setting it can only narrow what a decrypt accepts.
	ErrConflictingBlockAlgorithm = errors.New("xmlenc1: conflicting block algorithm")

	// ErrCBCRequiresOptIn is returned when a Decryptor is asked to
	// decrypt an AES-CBC ciphertext but the caller has not opted in
	// to unauthenticated CBC via Decryptor.AllowUnauthenticatedCBC(true).
	//
	// AES-CBC under XML Encryption 1.0 is unauthenticated and is
	// vulnerable to padding-oracle attacks (Jager/Somorovsky 2011).
	// XML Encryption 1.1 deprecated CBC in favor of AES-GCM. Callers
	// that must interoperate with legacy CBC ciphertexts can opt in
	// after evaluating the attack surface (e.g. ensuring decryption
	// errors are not exposed to remote attackers).
	ErrCBCRequiresOptIn = errors.New("xmlenc1: AES-CBC decryption requires AllowUnauthenticatedCBC(true)")

	// ErrCBCEncryptionRequiresOptIn is returned when an Encryptor is
	// configured to emit a new AES-CBC ciphertext (via a CBC
	// BlockAlgorithm) but the caller has not opted in to legacy CBC
	// encryption via Encryptor.AllowLegacyCBC(true).
	//
	// The Encryptor defaults to AES-256-GCM (authenticated). AES-CBC
	// under XML Encryption 1.0 is unauthenticated and vulnerable to
	// padding-oracle attacks (Jager/Somorovsky 2011); XML Encryption
	// 1.1 deprecated it in favor of AES-GCM. Emitting new CBC
	// ciphertext therefore requires an explicit acknowledgement.
	ErrCBCEncryptionRequiresOptIn = errors.New("xmlenc1: AES-CBC encryption requires AllowLegacyCBC(true)")
)

// Parameter roles reported by UnsupportedAlgorithmError and KeySizeError.
// They name the configuration knob (or the wire attribute it maps to) that
// the rejected value came from, so an error identifies which setter to fix,
// and not only which URI was refused. They are diagnostic text: match
// on the error type and its Algorithm field, not on these strings.
const (
	paramBlockAlgorithm = "block algorithm"
	paramKeyTransport   = "key transport algorithm"
	paramKeyWrap        = "key wrap algorithm"
	paramOAEPDigest     = "OAEP digest algorithm"
	paramMGF            = "MGF algorithm"
	paramKeyAgreement   = "key agreement algorithm"
	paramKeyDerivation  = "key derivation algorithm"
	paramConcatKDF      = "ConcatKDF digest algorithm"
	paramSessionKey     = "session key"
	paramKEK            = "key-encryption key"

	// paramCipherReferenceTransform names the ds:Transform slot inside an
	// xenc:CipherReference. Only the XMLDSig #base64 transform is accepted
	// there; parseCipherReferenceTransforms owns why.
	paramCipherReferenceTransform = "CipherReference transform algorithm"

	// paramEncryptedKey names the EncryptedKey's own declared algorithm
	// without saying which class of key protection it was meant to be. It is
	// for the slot that dispatches on that URI and reached none of the
	// recognized branches, where naming a class would point the caller at a
	// setter the URI may have nothing to do with.
	paramEncryptedKey = "EncryptedKey algorithm"
)

// UnsupportedAlgorithmError is returned for unrecognized algorithm URIs.
//
// Construct it with keyed fields only, as in
// &UnsupportedAlgorithmError{Algorithm: uri}. The field set grows as the
// diagnostics improve, and every such addition stops an unkeyed composite
// literal from compiling. That break is accepted under this package's
// EXPERIMENTAL, pre-1.0 API posture; keyed literals are unaffected and are
// the supported form.
type UnsupportedAlgorithmError struct {
	// Parameter names the algorithm slot that rejected the URI, e.g.
	// "block algorithm" or "MGF algorithm". It is diagnostic text and may
	// be empty when the slot is not known at the point of failure.
	Parameter string
	Algorithm string
}

func (e *UnsupportedAlgorithmError) Error() string {
	if e.Parameter == "" {
		return fmt.Sprintf("xmlenc1: unsupported algorithm %q", e.Algorithm)
	}
	return fmt.Sprintf("xmlenc1: unsupported %s %q", e.Parameter, e.Algorithm)
}

// KeySizeError is returned when a key (session key or key-encryption key)
// does not match the exact length required by its declared algorithm URI.
// It guards against algorithm/key-size confusion, e.g. declaring AES-256
// on the wire while supplying a 16-byte key that crypto/aes would silently
// treat as AES-128.
//
// Construct it with keyed fields only, as in
// &KeySizeError{Algorithm: uri, Want: want, Got: got}; see
// UnsupportedAlgorithmError for why unkeyed literals are unsupported here.
type KeySizeError struct {
	// Key names the key that was the wrong length, e.g. "session key" or
	// "key-encryption key". It is diagnostic text and may be empty when
	// the role is not known at the point of failure.
	Key       string
	Algorithm string
	Want      int
	Got       int
}

func (e *KeySizeError) Error() string {
	key := e.Key
	if key == "" {
		key = "key"
	}
	return fmt.Sprintf("xmlenc1: algorithm %q requires a %d-byte %s, got %d bytes", e.Algorithm, e.Want, key, e.Got)
}
