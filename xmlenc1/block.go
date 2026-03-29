package xmlenc1

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"
)

func keySizeForAlgorithm(algorithm string) (int, error) {
	switch algorithm {
	case AES128CBC, AES128GCM, AES128KeyWrap:
		return 16, nil
	case AES256CBC, AES256GCM, AES256KeyWrap:
		return 32, nil
	default:
		return 0, &UnsupportedAlgorithmError{Algorithm: algorithm}
	}
}

func blockEncrypt(algorithm string, key, plaintext []byte) ([]byte, error) {
	switch algorithm {
	case AES128CBC, AES256CBC:
		return encryptCBC(key, plaintext)
	case AES128GCM, AES256GCM:
		return encryptGCM(key, plaintext)
	default:
		return nil, &UnsupportedAlgorithmError{Algorithm: algorithm}
	}
}

func blockDecrypt(algorithm string, key, ciphertext []byte) ([]byte, error) {
	switch algorithm {
	case AES128CBC, AES256CBC:
		return decryptCBC(key, ciphertext)
	case AES128GCM, AES256GCM:
		return decryptGCM(key, ciphertext)
	default:
		return nil, &UnsupportedAlgorithmError{Algorithm: algorithm}
	}
}

// AES-CBC

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
		return nil, fmt.Errorf("%w: %v", ErrDecryptionFailed, err)
	}

	if len(data) < aes.BlockSize*2 {
		return nil, fmt.Errorf("%w: ciphertext too short", ErrDecryptionFailed)
	}
	if len(data)%aes.BlockSize != 0 {
		return nil, fmt.Errorf("%w: ciphertext not block-aligned", ErrDecryptionFailed)
	}

	iv := data[:aes.BlockSize]
	ciphertext := data[aes.BlockSize:]

	plaintext := make([]byte, len(ciphertext))
	mode := cipher.NewCBCDecrypter(block, iv)
	mode.CryptBlocks(plaintext, ciphertext)

	return pkcs7Unpad(plaintext, aes.BlockSize)
}

// AES-GCM

func encryptGCM(key, plaintext []byte) ([]byte, error) {
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

	// nonce || ciphertext || tag
	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

func decryptGCM(key, data []byte) ([]byte, error) {
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

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
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

func pkcs7Unpad(data []byte, blockSize int) ([]byte, error) {
	if len(data) == 0 {
		return nil, ErrInvalidPadding
	}
	padding := int(data[len(data)-1])
	if padding == 0 || padding > blockSize || padding > len(data) {
		return nil, ErrInvalidPadding
	}
	for i := len(data) - padding; i < len(data); i++ {
		if data[i] != byte(padding) {
			return nil, ErrInvalidPadding
		}
	}
	return data[:len(data)-padding], nil
}
