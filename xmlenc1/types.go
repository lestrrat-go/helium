package xmlenc1

import "crypto/ecdh"

// EncryptionMethod represents the <EncryptionMethod> element.
type EncryptionMethod struct {
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
	// Empty means no label. Applies to RSA-OAEP key transport only.
	OAEPParams []byte
}

// EncryptedData represents the <EncryptedData> element.
type EncryptedData struct {
	// ID is the @Id attribute, used to reference this EncryptedData from
	// elsewhere in the document.
	ID string
	// Type is the @Type attribute: TypeElement when the plaintext is one
	// element, TypeContent when it is the children of an element. An empty
	// Type means arbitrary octets, which Decryptor.Decrypt treats as
	// TypeElement and Decryptor.DecryptBytes returns unparsed.
	Type string
	// EncryptionMethod describes how the content itself is encrypted: a
	// block encryption URI (AES-CBC or AES-GCM). Decryption needs it, so
	// an EncryptedData without one fails with [ErrMalformedEncrypted].
	EncryptionMethod *EncryptionMethod
	// EncryptedKey is the first EncryptedKey candidate, kept for backward
	// compatibility with callers written against the old single-key field.
	//
	// Deprecated: use EncryptedKeys. When both are set, EncryptedKeys takes
	// precedence and this field is ignored; when only this field is set it
	// is treated as a single-element EncryptedKeys.
	EncryptedKey *EncryptedKey
	// EncryptedKeys holds every EncryptedKey candidate found in KeyInfo
	// (one per recipient). Decryption tries each in turn, so a
	// multi-recipient document, or one with a bogus EncryptedKey
	// prepended to a legitimate one, still resolves. A Decryptor with a
	// non-empty SessionKey uses none of them — see
	// [Decryptor.SessionKey].
	EncryptedKeys []*EncryptedKey
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
func (ed *EncryptedData) effectiveEncryptedKeys() []*EncryptedKey {
	if len(ed.EncryptedKeys) > 0 {
		return ed.EncryptedKeys
	}
	if ed.EncryptedKey != nil {
		return []*EncryptedKey{ed.EncryptedKey}
	}
	return nil
}

// EncryptedKey represents the <EncryptedKey> element: the session key of an
// EncryptedData, protected for one recipient.
type EncryptedKey struct {
	// ID is the @Id attribute.
	ID string
	// Recipient is the @Recipient hint naming who the key is intended for.
	// It is populated when parsing and carried for inspection only —
	// encryption does not serialize it.
	Recipient string
	// EncryptionMethod describes how the session key is protected: an
	// RSA-OAEP key transport URI, an AES key wrap URI, or (with
	// AgreementMethod set) the key wrap applied to the agreed key.
	EncryptionMethod *EncryptionMethod
	// CipherValue is the protected session key, base64-decoded.
	CipherValue []byte
	// CarriedKeyName is the xenc:CarriedKeyName text, a name for the key
	// this element carries. It is populated when parsing and carried for
	// inspection only — encryption does not serialize it.
	CarriedKeyName string
	// AgreementMethod, when set, means the key that protects CipherValue is
	// derived by key agreement rather than supplied directly.
	AgreementMethod *AgreementMethod
}

// AgreementMethod describes a key agreement used to derive the key that
// protects an EncryptedKey. XML Encryption 1.1 places this element inside a
// ds:KeyInfo child of the EncryptedKey.
type AgreementMethod struct {
	// Algorithm is the @Algorithm URI of the agreement. ECDHES is the
	// supported value.
	Algorithm string
	// KeyDerivationMethod turns the agreed shared secret into a key of the
	// length the EncryptionMethod requires.
	KeyDerivationMethod *KeyDerivationMethod
	// OriginatorKey is the sender's ephemeral public key, from
	// xenc:OriginatorKeyInfo. The recipient combines it with its own
	// private key to reach the shared secret.
	OriginatorKey *ECKeyValue
}

// KeyDerivationMethod describes the explicit KDF parameters carried by an
// AgreementMethod.
type KeyDerivationMethod struct {
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

// ECKeyValue contains an XML Signature 1.1 elliptic-curve public key.
type ECKeyValue struct {
	// Curve is the named curve resolved from dsig11:NamedCurve/@URI:
	// P-256, P-384, or P-521.
	Curve ecdh.Curve
	// PublicKey is the base64-decoded dsig11:PublicKey point, in the
	// uncompressed SEC 1 form (0x04 || X || Y) that
	// ecdh.Curve.NewPublicKey accepts. Parsing validates it against Curve.
	PublicKey []byte
}
