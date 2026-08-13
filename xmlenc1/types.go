package xmlenc1

import "crypto/ecdh"

// encryptionMethod represents the <EncryptionMethod> element.
type encryptionMethod struct {
	// Algorithm is the algorithm URI carried by @Algorithm. Its meaning
	// depends on where the EncryptionMethod sits: a block encryption URI
	// on an EncryptedData, a key transport or key wrap URI on an
	// EncryptedKey.
	Algorithm string
	// DigestMethod is the RSA-OAEP label digest URI from the ds:DigestMethod
	// child. Empty means the XML Encryption default, SHA-1. Applies to
	// RSA-OAEP key transport only.
	DigestMethod string
	// MGFAlgorithm is the mask generation function URI from the xenc11:MGF
	// child. Parsing accepts and stores it on any EncryptionMethod; it is
	// resolved only on the RSA-OAEP key transport paths, where RSAOAEP11
	// takes an explicit value and RSAOAEP rejects one because its
	// MGF1-SHA-1 is implicit. Empty resolves to MGF1 with SHA-1,
	// independent of DigestMethod. On any other algorithm the value is
	// carried and never read.
	MGFAlgorithm string
	// OAEPParams is the decoded OAEP label from the xenc:OAEPparams child.
	// Empty means no label. Applies to RSA-OAEP key transport only. The 1 KiB
	// decoded limit on the label holds in both directions: parsing refuses a
	// larger one with [ErrMalformedEncrypted], and an [Encryptor] configured
	// with a larger one through [Encryptor.OAEPParams] fails the encryption
	// with [ErrEncryptionFailed] before any payload work, so a label this
	// package writes is a label it reads back. That limit is a policy ceiling,
	// not a conformance boundary: the label is hashed before it is used, so a
	// real one is a handful of octets, while a larger one is still valid per
	// the xenc schema and refused anyway.
	// Parsing reads only character data: a text or CDATA child, or an entity
	// reference, whose declared replacement text is taken as written and not
	// expanded further. An xenc:OAEPparams holding an element child is refused
	// with the same sentinel, while a comment or processing instruction is
	// ignored.
	OAEPParams []byte
}

// encryptedData represents the <EncryptedData> element.
type encryptedData struct {
	// ID is the @Id attribute, used to reference this EncryptedData from
	// elsewhere in the document.
	ID string
	// Type is the @Type attribute. For Decryptor.Decrypt, an empty Type or
	// TypeElement means the plaintext is one element, and TypeContent means it
	// is the children of an element. Any other non-empty Type is rejected as
	// malformed. Decryptor.DecryptBytes returns the plaintext octets without
	// interpreting Type, so use it for opaque or application-defined payloads.
	Type string
	// EncryptionMethod describes how the content itself is encrypted: a
	// block encryption URI (AES-CBC or AES-GCM). Decryption needs it, so
	// an EncryptedData without one fails with [ErrMalformedEncrypted].
	EncryptionMethod *encryptionMethod
	// EncryptedKey is the first EncryptedKey candidate, kept for backward
	// compatibility with callers written against the old single-key field.
	//
	// Deprecated: use EncryptedKeys. When both are set, EncryptedKeys takes
	// precedence and this field is ignored; when only this field is set it
	// is treated as a single-element EncryptedKeys.
	EncryptedKey *encryptedKey
	// EncryptedKeys holds every EncryptedKey candidate found in KeyInfo
	// (one per recipient). Decryption tries each in turn, so a
	// multi-recipient document, or one with a bogus EncryptedKey
	// prepended to a legitimate one, still resolves. A Decryptor with a
	// non-empty SessionKey uses none of them — see
	// [Decryptor.SessionKey].
	EncryptedKeys []*encryptedKey
	// CipherValue is the encrypted content, base64-decoded. Its internal
	// layout belongs to the block algorithm: the AES modes used here all
	// prefix the ciphertext with the IV, and GCM appends the
	// authentication tag.
	CipherValue []byte
}

// effectiveEncryptedKeys returns the EncryptedKey candidates to use,
// reconciling the EncryptedKeys slice with the deprecated single
// EncryptedKey field: EncryptedKeys wins when non-empty; otherwise the
// deprecated field, if set, is treated as a single-element list.
func (ed *encryptedData) effectiveEncryptedKeys() []*encryptedKey {
	if len(ed.EncryptedKeys) > 0 {
		return ed.EncryptedKeys
	}
	if ed.EncryptedKey != nil {
		return []*encryptedKey{ed.EncryptedKey}
	}
	return nil
}

// encryptedKey represents the <EncryptedKey> element: the session key of an
// EncryptedData, protected for one recipient.
type encryptedKey struct {
	// ID is the @Id attribute.
	ID string
	// recipient is the @Recipient hint naming who the key is intended for.
	// It is populated when parsing and carried for inspection only —
	// encryption does not serialize it.
	recipient string
	// EncryptionMethod describes how the session key is protected: an
	// RSA-OAEP key transport URI, an AES key wrap URI, or (with
	// AgreementMethod set) the key wrap applied to the agreed key.
	EncryptionMethod *encryptionMethod
	// CipherValue is the protected session key, base64-decoded.
	CipherValue []byte
	// CarriedKeyName is the xenc:CarriedKeyName text, a name for the key this
	// element carries. It is part of the parsed shape and nothing fills it:
	// parsing steps over the element, and encryption does not serialize it.
	//
	// Reading it would cost an unbounded copy of attacker-supplied metadata no
	// decrypt path consults — the name is joined from every child of the
	// element at every depth that join reaches, and no budget charges it, so a
	// document may spread one over as many CDATA sections as it likes.
	//
	// That join belongs to the DOM, not to this package: helium's
	// aggregateOwnedContent (node.go) answers a container's Content() by
	// recursing through its whole descendant subtree, and
	// internal/domutil.TextContent concatenates Content() over every direct
	// child. xmlenc1 implements neither, so the cost above is what reading the
	// field through them would take rather than work any xmlenc1 code path
	// performs.
	CarriedKeyName string
	// AgreementMethod, when set, means the key that protects CipherValue is
	// derived by key agreement rather than supplied directly.
	AgreementMethod *agreementMethod
}

// agreementMethod describes a key agreement used to derive the key that
// protects an EncryptedKey. XML Encryption 1.1 places this element inside a
// ds:KeyInfo child of the EncryptedKey.
type agreementMethod struct {
	// Algorithm is the @Algorithm URI of the agreement. ECDHES is the
	// supported value.
	Algorithm string
	// KeyDerivationMethod turns the agreed shared secret into a key of the
	// length the EncryptionMethod requires.
	KeyDerivationMethod *keyDerivationMethod
	// OriginatorKey is the sender's ephemeral public key, from
	// xenc:OriginatorKeyInfo. The recipient combines it with its own
	// private key to reach the shared secret.
	OriginatorKey *ecKeyValue
}

// keyDerivationMethod describes the explicit KDF parameters carried by an
// AgreementMethod.
type keyDerivationMethod struct {
	// Algorithm is the @Algorithm URI of the derivation function.
	// ConcatKDF is the supported value.
	Algorithm string
	// ConcatKDF holds the parameters when Algorithm is ConcatKDF.
	ConcatKDF *ConcatKDFParams
}

// ConcatKDFParams contains the XML Encryption 1.1 ConcatKDF parameters.
// The parameter attributes are decoded from their hexBinary representation;
// their unused-bit counts are retained internally for KDF bit-string packing.
type ConcatKDFParams struct {
	// AlgorithmID, PartyUInfo, PartyVInfo, SuppPubInfo, and SuppPrivInfo
	// are the NIST SP 800-56A OtherInfo fields, decoded from the hexBinary
	// attributes of the same names. They are concatenated, in this order,
	// into the KDF input, so both parties must agree on them exactly.
	//
	// The five fields TOGETHER are limited to 4096 bytes, since the document
	// under decryption is attacker-supplied and the concatenation costs work
	// proportional to its size. Real OtherInfo is identifiers and nonces —
	// tens of bytes — so the limit is far above any interoperable value.
	// Exceeding it is an error wrapping [ErrMalformedEncrypted], raised when
	// parsing a document and when deriving from these parameters.
	//
	// One parameter set never reaches a derivation and so is never measured:
	// the fallback [Encryptor.KeyDerivationParams] documents replaces a set
	// whose DigestMethod is empty, wholesale, with the SHA-256 default
	// carrying empty OtherInfo. These five fields are discarded there rather
	// than checked, so an oversized set paired with an empty DigestMethod
	// encrypts successfully and emits no OtherInfo attributes at all.
	AlgorithmID  []byte
	PartyUInfo   []byte
	PartyVInfo   []byte
	SuppPubInfo  []byte
	SuppPrivInfo []byte
	// DigestMethod is the hash driving the KDF, taken from the @Algorithm
	// of the ds:DigestMethod child. Parsed wire parameters must carry one:
	// a xenc11:ConcatKDFParams without it is rejected as malformed. On an
	// Encryptor these parameters are configuration rather than wire data,
	// and [Encryptor.KeyDerivationParams] states what an empty DigestMethod
	// means there.
	DigestMethod string

	algorithmIDUnusedBits  uint8
	partyUInfoUnusedBits   uint8
	partyVInfoUnusedBits   uint8
	suppPubInfoUnusedBits  uint8
	suppPrivInfoUnusedBits uint8
}

// clone returns a deep copy, detaching every OtherInfo byte slice from the
// caller's arrays. A shallow struct copy would leave them aliased, and these
// values feed both the derived KEK and the emitted xenc11:ConcatKDFParams, so
// a later mutation of the caller's array would change what gets encrypted.
// It is nil-safe, and an empty field stays empty rather than becoming a
// zero-length non-nil slice.
func (p *ConcatKDFParams) clone() *ConcatKDFParams {
	if p == nil {
		return nil
	}
	cp := *p
	cp.AlgorithmID = append([]byte(nil), p.AlgorithmID...)
	cp.PartyUInfo = append([]byte(nil), p.PartyUInfo...)
	cp.PartyVInfo = append([]byte(nil), p.PartyVInfo...)
	cp.SuppPubInfo = append([]byte(nil), p.SuppPubInfo...)
	cp.SuppPrivInfo = append([]byte(nil), p.SuppPrivInfo...)
	return &cp
}

// ecKeyValue contains an XML Signature 1.1 elliptic-curve public key.
type ecKeyValue struct {
	// curve is the named curve resolved from dsig11:NamedCurve/@URI:
	// P-256, P-384, or P-521.
	curve ecdh.Curve
	// PublicKey is the base64-decoded dsig11:PublicKey point, in the
	// uncompressed SEC 1 form (0x04 || X || Y) that
	// ecdh.Curve.NewPublicKey accepts. Parsing validates it against the
	// curve field.
	PublicKey []byte
}
