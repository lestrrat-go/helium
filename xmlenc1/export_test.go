package xmlenc1

import (
	"context"

	helium "github.com/lestrrat-go/helium"
)

// EncryptBytesForTest encrypts an arbitrary byte slice with the given
// algorithm and key and returns the resulting CipherValue (IV/nonce
// prefix included). It exists solely so security tests can construct
// EncryptedData whose plaintext is not a well-formed XML element
// serializable through the public Encryptor (e.g. payloads beginning
// with a DOCTYPE).
func EncryptBytesForTest(algorithm string, key, plaintext []byte) ([]byte, error) {
	return blockEncrypt(algorithm, key, plaintext)
}

// MarshalEncryptedDataForTest is a test-only re-export of the package
// internal marshaler so security tests can assemble EncryptedData
// elements from raw fields.
func MarshalEncryptedDataForTest(doc *helium.Document, ed *EncryptedData) (*helium.Element, error) {
	return marshalEncryptedData(doc, ed)
}

// HardenedParserForTest returns the parser configuration the decryptor
// uses for inner decrypted-XML parsing. Tests use this to assert that
// XXE-class inputs are rejected or stripped.
func HardenedParserForTest() helium.Parser {
	return newHardenedInnerParser()
}

// ParseEncryptedDataForTest is a test-only re-export of the package
// internal EncryptedData parser so tests can assert namespace-aware
// element matching directly against a parsed DOM. It parses under a
// default Decryptor's CipherValue budgets, as Decrypt does, and under an
// uncancelled context, so what these callers see is the parse's own verdict
// rather than a cancellation. Cancellation of the parse is covered through the
// public terminals instead.
func ParseEncryptedDataForTest(elem *helium.Element) (*EncryptedData, error) {
	cfg := &decryptConfig{}
	ps := &parseState{cfg: cfg, keys: newEncryptedKeyBudget(cfg), payload: newPayloadCipherValueBudget(cfg)}
	return parseEncryptedData(context.Background(), elem, ps)
}

// MaxOAEPParamsBytesForTest re-exports the xenc:OAEPparams decoded-size limit
// so a test can pin the boundary against the constant itself rather than a
// copy of its value, which would drift from it.
const MaxOAEPParamsBytesForTest = maxOAEPParamsBytes

// MaxECPublicKeyBytesForTest re-exports the dsig11:PublicKey decoded-size limit
// so a test can pin the boundary against the constant itself rather than a copy
// of its value, which would drift from it.
const MaxECPublicKeyBytesForTest = maxECPublicKeyBytes

// MaxConcatKDFOtherInfoBytesForTest re-exports the cumulative ConcatKDF
// OtherInfo budget so a test can size a parameter set against the constant
// itself rather than a copy of its value, which would drift from it.
const MaxConcatKDFOtherInfoBytesForTest = maxConcatKDFOtherInfoBytes

// AESKeyWrapForTest wraps key material under a KEK using RFC 3394 AES Key
// Wrap. It exists so security tests can assemble an EncryptedKey whose
// wrapped session-key length does not match the declared block algorithm,
// exercising the post-unwrap key-size binding.
func AESKeyWrapForTest(kek, plaintext []byte) ([]byte, error) {
	return aesKeyWrap(kek, plaintext)
}
