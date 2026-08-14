package xmlenc1

import (
	"context"

	helium "github.com/lestrrat-go/helium"
)

// The xmlenc-core1 wire structs are unexported, so nothing outside this
// package can name them and none of their fields is an API promise. These
// aliases give the external test package a name for each one, so a test can
// assemble or inspect a wire shape directly without the type appearing in the
// shipped API. They exist only under test.
type (
	EncryptedData       = encryptedData
	EncryptedKey        = encryptedKey
	EncryptionMethod    = encryptionMethod
	AgreementMethod     = agreementMethod
	KeyDerivationMethod = keyDerivationMethod
	ECKeyValue          = ecKeyValue
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
func MarshalEncryptedDataForTest(doc *helium.Document, ed *encryptedData) (*helium.Element, error) {
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
// default Decryptor's CipherValue budgets and the same same-document
// reference scope Decrypt builds, and under an uncancelled context, so what
// these callers see is the parse's own verdict rather than a cancellation.
// Cancellation of the parse is covered through the public terminals instead.
func ParseEncryptedDataForTest(elem *helium.Element) (*encryptedData, error) {
	cfg := &decryptConfig{}
	ps := &parseState{cfg: cfg, keys: newEncryptedKeyBudget(cfg), payload: newPayloadCipherValueBudget(cfg), refs: newRefScope(elem)}
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

// MaxCipherReferenceTransformsForTest re-exports the cap on how many
// ds:Transform children one xenc:CipherReference may declare, so a test can
// size an over-cap list against the constant itself rather than a copy of its
// value, which would drift from it.
const MaxCipherReferenceTransformsForTest = maxCipherReferenceTransforms

// MaxConcatKDFOtherInfoBytesForTest re-exports the cumulative ConcatKDF
// OtherInfo budget so a test can size a parameter set against the constant
// itself rather than a copy of its value, which would drift from it.
const MaxConcatKDFOtherInfoBytesForTest = maxConcatKDFOtherInfoBytes

// EncryptCBCArbitraryPaddingForTest AES-CBC encrypts plaintext under key,
// padding it the way xmlenc-core1 §5.2.1 allows and this package never writes:
// the final octet names the padding length N, and the N-1 octets ahead of it
// all carry filler chosen independently of N. It exists so a test can build the
// conforming ciphertext a third-party implementation emits, which is the shape
// PKCS#7 unpadding refuses.
//
// filler must differ from the padding length for the ciphertext to be
// distinguishable from a PKCS#7-padded one; a caller passing the length itself
// gets ordinary PKCS#7 padding back.
func EncryptCBCArbitraryPaddingForTest(key, plaintext []byte, filler byte) ([]byte, error) {
	blockSize := 16
	n := blockSize - len(plaintext)%blockSize
	padded := make([]byte, len(plaintext)+n)
	copy(padded, plaintext)
	for i := len(plaintext); i < len(padded)-1; i++ {
		padded[i] = filler
	}
	padded[len(padded)-1] = byte(n)
	return cbcSeal(key, padded)
}

// AESKeyWrapForTest wraps key material under a KEK using RFC 3394 AES Key
// Wrap. It exists so security tests can assemble an EncryptedKey whose
// wrapped session-key length does not match the declared block algorithm,
// exercising the post-unwrap key-size binding.
func AESKeyWrapForTest(kek, plaintext []byte) ([]byte, error) {
	return aesKeyWrap(kek, plaintext)
}
