package xmlenc1

import (
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

// AESKeyWrapForTest wraps key material under a KEK using RFC 3394 AES Key
// Wrap. It exists so security tests can assemble an EncryptedKey whose
// wrapped session-key length does not match the declared block algorithm,
// exercising the post-unwrap key-size binding.
func AESKeyWrapForTest(kek, plaintext []byte) ([]byte, error) {
	return aesKeyWrap(kek, plaintext)
}
