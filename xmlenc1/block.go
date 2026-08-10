package xmlenc1

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/subtle"
	"errors"
	"fmt"
	"io"
)

// keySizeForAlgorithm returns the exact key length the algorithm URI
// requires. algParam names the slot the URI came from (paramBlockAlgorithm,
// paramKeyWrap, ...) so a rejected URI reports which setter to fix.
func keySizeForAlgorithm(algParam, algorithm string) (int, error) {
	switch algorithm {
	case AES128CBC, AES128GCM, AES128GCM11, AES128KeyWrap:
		return 16, nil
	case AES192GCM11, AES192KeyWrap:
		return 24, nil
	case AES256CBC, AES256GCM, AES256GCM11, AES256KeyWrap:
		return 32, nil
	default:
		return 0, &UnsupportedAlgorithmError{Parameter: algParam, Algorithm: algorithm}
	}
}

// validateBlockAlgorithm requires algorithm to name one of the supported
// block encryption URIs. keySizeForAlgorithm also accepts key-wrap URIs, so it
// cannot validate the role of a block algorithm on its own.
func validateBlockAlgorithm(algorithm string) error {
	switch algorithm {
	case AES128CBC, AES256CBC, AES128GCM, AES256GCM,
		AES128GCM11, AES192GCM11, AES256GCM11:
		return nil
	default:
		return &UnsupportedAlgorithmError{Parameter: paramBlockAlgorithm, Algorithm: algorithm}
	}
}

// validateKeySize ensures key is exactly the length required by the
// declared algorithm URI. This binds the wire-declared algorithm to the
// real key length: requesting, e.g., an AES-256 algorithm with a 16-byte
// key (which crypto/aes silently accepts as AES-128) is rejected instead
// of producing data that DECLARES AES-256 but uses AES-128. Enforced on
// both encrypt and decrypt, including after key unwrap / key transport.
//
// algParam and keyParam name the algorithm slot and the key the caller is
// binding together, so the resulting error says which one to fix.
func validateKeySize(algParam, algorithm, keyParam string, key []byte) error {
	want, err := keySizeForAlgorithm(algParam, algorithm)
	if err != nil {
		return err
	}
	if len(key) != want {
		return &KeySizeError{Key: keyParam, Algorithm: algorithm, Want: want, Got: len(key)}
	}
	return nil
}

func wrapBlockAlgorithmError(operation, err error) error {
	var unsupported *UnsupportedAlgorithmError
	if !errors.As(err, &unsupported) {
		return err
	}
	return fmt.Errorf("%w: %w", operation, err)
}

// blockEncrypt encrypts plaintext with the given algorithm. For AEAD
// algorithms (GCM) the algorithm URI is bound into the additional
// authenticated data so that an attacker cannot substitute a different
// EncryptionMethod/@Algorithm on the wire.
func blockEncrypt(algorithm string, key, plaintext []byte) ([]byte, error) {
	if err := validateKeySize(paramBlockAlgorithm, algorithm, paramSessionKey, key); err != nil {
		return nil, wrapBlockAlgorithmError(ErrEncryptionFailed, err)
	}
	switch algorithm {
	case AES128CBC, AES256CBC:
		return encryptCBC(key, plaintext)
	case AES128GCM, AES256GCM:
		return encryptGCM(key, plaintext, []byte(algorithm))
	case AES128GCM11, AES192GCM11, AES256GCM11:
		// XML Encryption 1.1 defines the GCM input as IV || ciphertext ||
		// authentication tag and does not define additional authenticated
		// data, so these three bind none. The two 2001-namespace identifiers
		// above bind the algorithm URI instead: no XML Security specification
		// defines AES-GCM in that namespace, so the binding is this package's
		// own measure against an algorithm substitution on the wire, kept so
		// documents it has already emitted still decrypt — not specification
		// compatibility.
		return encryptGCM(key, plaintext, nil)
	default:
		return nil, wrapBlockAlgorithmError(ErrEncryptionFailed, &UnsupportedAlgorithmError{Parameter: paramBlockAlgorithm, Algorithm: algorithm})
	}
}

// blockDecrypt decrypts ciphertext. For AEAD algorithms (GCM) the
// algorithm URI is verified as additional authenticated data. For CBC
// the function returns ErrDecryptionFailed for ANY failure (cipher,
// padding, or downstream parse) so callers cannot mount a padding
// oracle by distinguishing the cause.
func blockDecrypt(algorithm string, key, ciphertext []byte) ([]byte, error) {
	// Bind the declared algorithm URI to the real key length before
	// touching the ciphertext. The key length is not attacker-controlled
	// (it is the recipient's configured / unwrapped key), so reporting a
	// distinguishable KeySizeError here is not a padding-oracle signal.
	if err := validateKeySize(paramBlockAlgorithm, algorithm, paramSessionKey, key); err != nil {
		return nil, wrapBlockAlgorithmError(ErrDecryptionFailed, err)
	}
	switch algorithm {
	case AES128CBC, AES256CBC:
		return decryptCBC(key, ciphertext)
	case AES128GCM, AES256GCM:
		return decryptGCM(key, ciphertext, []byte(algorithm))
	case AES128GCM11, AES192GCM11, AES256GCM11:
		return decryptGCM(key, ciphertext, nil)
	default:
		return nil, wrapBlockAlgorithmError(ErrDecryptionFailed, &UnsupportedAlgorithmError{Parameter: paramBlockAlgorithm, Algorithm: algorithm})
	}
}

// AES-CBC
//
// CBC ciphertexts produced by this package are NOT authenticated. The
// XML-Encryption 1.0 CBC mode is vulnerable to padding-oracle attacks
// (Jager/Somorovsky 2011) and has been deprecated by XML-Encryption 1.1.
// Decryption with CBC requires an explicit caller opt-in via
// Decryptor.AllowUnauthenticatedCBC(true).

func encryptCBC(key, plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrEncryptionFailed, err)
	}

	padded := pkcs7Pad(plaintext, aes.BlockSize)

	// IV || ciphertext
	out := make([]byte, aes.BlockSize+len(padded))
	iv := out[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrEncryptionFailed, err)
	}

	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(out[aes.BlockSize:], padded)
	return out, nil
}

func decryptCBC(key, data []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		// Caller error (wrong key size), not an oracle signal.
		return nil, ErrDecryptionFailed
	}

	if len(data) < aes.BlockSize*2 || len(data)%aes.BlockSize != 0 {
		return nil, ErrDecryptionFailed
	}

	iv := data[:aes.BlockSize]
	ciphertext := data[aes.BlockSize:]

	plaintext := make([]byte, len(ciphertext))
	mode := cipher.NewCBCDecrypter(block, iv)
	mode.CryptBlocks(plaintext, ciphertext)

	out, ok := pkcs7UnpadConstantTime(plaintext, aes.BlockSize)
	if !ok {
		// Do not distinguish "bad padding" from any other downstream
		// failure: surface the same opaque ErrDecryptionFailed so the
		// caller cannot build a padding oracle.
		return nil, ErrDecryptionFailed
	}
	return out, nil
}

// AES-GCM

func encryptGCM(key, plaintext, aad []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrEncryptionFailed, err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrEncryptionFailed, err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrEncryptionFailed, err)
	}

	// nonce || ciphertext || tag, with aad bound into the tag.
	return gcm.Seal(nonce, nonce, plaintext, aad), nil
}

func decryptGCM(key, data, aad []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDecryptionFailed, err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDecryptionFailed, err)
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize+gcm.Overhead() {
		return nil, fmt.Errorf("%w: ciphertext too short", ErrDecryptionFailed)
	}

	nonce := data[:nonceSize]
	ciphertext := data[nonceSize:]

	plaintext, err := gcm.Open(nil, nonce, ciphertext, aad)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDecryptionFailed, err)
	}
	return plaintext, nil
}

// PKCS#7 padding

func pkcs7Pad(data []byte, blockSize int) []byte {
	padding := blockSize - len(data)%blockSize
	padded := make([]byte, len(data)+padding)
	copy(padded, data)
	for i := len(data); i < len(padded); i++ {
		padded[i] = byte(padding)
	}
	return padded
}

// pkcs7UnpadConstantTime removes PKCS#7 padding without short-circuiting
// on the first invalid byte. The return value ok is false iff the
// padding is invalid; valid is computed by walking every padding byte
// regardless of intermediate mismatches.
//
// Note: the work performed depends on the *padding length byte*, which
// is observable in cache-timing studies but is the same trade-off as
// stdlib `crypto/tls`'s legacy CBC unpadding. This is a defense in
// depth on top of the primary mitigation (require opt-in for CBC). For
// strong authentication use AES-GCM.
func pkcs7UnpadConstantTime(data []byte, blockSize int) ([]byte, bool) {
	if len(data) == 0 || len(data)%blockSize != 0 {
		return nil, false
	}
	padding := int(data[len(data)-1])
	// Bounds check on the padding length.
	good := 1
	if padding == 0 || padding > blockSize || padding > len(data) {
		good = 0
		// Continue with a safe value so the inner loop still runs to
		// completion on a fixed range.
		padding = blockSize
	}
	// Compare every byte of the (assumed) padding region in constant
	// time relative to `padding`.
	want := byte(padding)
	for i := len(data) - padding; i < len(data); i++ {
		good &= subtle.ConstantTimeByteEq(data[i], want)
	}
	if good != 1 {
		return nil, false
	}
	return data[:len(data)-int(data[len(data)-1])], true
}
