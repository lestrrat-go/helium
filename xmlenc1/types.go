package xmlenc1

import "crypto/ecdh"

// EncryptionMethod represents the <EncryptionMethod> element.
type EncryptionMethod struct {
	Algorithm    string
	DigestMethod string // optional (for RSA-OAEP 1.1)
	MGFAlgorithm string // optional (for RSA-OAEP 1.1)
	OAEPParams   []byte // optional
}

// EncryptedData represents the <EncryptedData> element.
type EncryptedData struct {
	ID               string
	Type             string // TypeElement or TypeContent
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
	// prepended to a legitimate one, still resolves.
	EncryptedKeys []*EncryptedKey
	CipherValue   []byte // base64-decoded cipher bytes
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

// EncryptedKey represents the <EncryptedKey> element.
type EncryptedKey struct {
	ID               string
	Recipient        string
	EncryptionMethod *EncryptionMethod
	CipherValue      []byte // base64-decoded cipher bytes
	CarriedKeyName   string
	AgreementMethod  *AgreementMethod
}

// AgreementMethod describes a key agreement used to derive the key that
// protects an EncryptedKey. XML Encryption 1.1 places this element inside a
// ds:KeyInfo child of the EncryptedKey.
type AgreementMethod struct {
	Algorithm           string
	KeyDerivationMethod *KeyDerivationMethod
	OriginatorKey       *ECKeyValue
}

// KeyDerivationMethod describes the explicit KDF parameters carried by an
// AgreementMethod.
type KeyDerivationMethod struct {
	Algorithm string
	ConcatKDF *ConcatKDFParams
}

// ConcatKDFParams contains the XML Encryption 1.1 ConcatKDF parameters.
// The parameter attributes are decoded from their hexBinary representation.
type ConcatKDFParams struct {
	AlgorithmID  []byte
	PartyUInfo   []byte
	PartyVInfo   []byte
	SuppPubInfo  []byte
	SuppPrivInfo []byte
	DigestMethod string
}

// ECKeyValue contains an XML Signature 1.1 elliptic-curve public key.
type ECKeyValue struct {
	Curve     ecdh.Curve
	PublicKey []byte
}
