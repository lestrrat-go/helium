package xmlenc1

import (
	"crypto/aes"
	"encoding/binary"
	"fmt"
)

// AES Key Wrap (RFC 3394)

var defaultIV = [8]byte{0xA6, 0xA6, 0xA6, 0xA6, 0xA6, 0xA6, 0xA6, 0xA6}

// aesKeyWrap wraps a key encryption key (KEK) around plaintext key material
// using the AES Key Wrap algorithm (RFC 3394).
func aesKeyWrap(kek, plaintext []byte) ([]byte, error) {
	if len(plaintext)%8 != 0 || len(plaintext) < 16 {
		return nil, fmt.Errorf("%w: plaintext must be at least 16 bytes and a multiple of 8", ErrEncryptionFailed)
	}

	block, err := aes.NewCipher(kek)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrEncryptionFailed, err)
	}

	n := len(plaintext) / 8
	// Initialize A and R
	var a [8]byte
	copy(a[:], defaultIV[:])

	r := make([][]byte, n)
	for i := range n {
		r[i] = make([]byte, 8)
		copy(r[i], plaintext[i*8:(i+1)*8])
	}

	// Wrap
	for j := range 6 {
		for i := range n {
			// B = AES(K, A | R[i])
			var input [16]byte
			copy(input[:8], a[:])
			copy(input[8:], r[i])

			var b [16]byte
			block.Encrypt(b[:], input[:])

			// A = MSB(64, B) ^ t where t = (n*j)+i+1
			t := uint64(n*j + i + 1)
			copy(a[:], b[:8])
			xorA := binary.BigEndian.Uint64(a[:])
			xorA ^= t
			binary.BigEndian.PutUint64(a[:], xorA)

			// R[i] = LSB(64, B)
			copy(r[i], b[8:])
		}
	}

	// Output: A || R[0] || R[1] || ... || R[n-1]
	out := make([]byte, 8+n*8)
	copy(out[:8], a[:])
	for i := range n {
		copy(out[8+i*8:], r[i])
	}
	return out, nil
}

// aesKeyUnwrap unwraps keySize bytes of key material using the AES Key Wrap
// algorithm (RFC 3394). keySize is the length the recipient requires — the
// session-key length fixed by the EncryptedData's declared block algorithm,
// which every caller resolves through keySizeForAlgorithm.
//
// RFC 3394 §2.2.2 wraps a keySize-byte key into exactly keySize+8 bytes, so
// any other length is provably not a wrap of the key being unwrapped. It is
// rejected here, before the six unwrap rounds, which run AES over every
// 8-byte block: accepting merely "a multiple of 8" would let a document
// declaring a 32-byte session key spend those rounds on a multi-megabyte
// CipherValue. The unwrapped key still passes the key-size binding in
// blockDecrypt, which is what catches a wrong-length key that arrived by RSA
// key transport instead.
func aesKeyUnwrap(kek, ciphertext []byte, keySize int) ([]byte, error) {
	if len(ciphertext) != keySize+8 {
		return nil, fmt.Errorf("%w: %w: ciphertext must be exactly %d bytes to unwrap a %d-byte key, got %d", ErrDecryptionFailed, ErrKeyUnwrapFailed, keySize+8, keySize, len(ciphertext))
	}

	block, err := aes.NewCipher(kek)
	if err != nil {
		return nil, fmt.Errorf("%w: %w: %v", ErrDecryptionFailed, ErrKeyUnwrapFailed, err)
	}

	n := len(ciphertext)/8 - 1
	// Initialize A and R
	var a [8]byte
	copy(a[:], ciphertext[:8])

	r := make([][]byte, n)
	for i := range n {
		r[i] = make([]byte, 8)
		copy(r[i], ciphertext[(i+1)*8:(i+2)*8])
	}

	// Unwrap
	for j := 5; j >= 0; j-- {
		for i := n - 1; i >= 0; i-- {
			// A ^ t
			t := uint64(n*j + i + 1)
			xorA := binary.BigEndian.Uint64(a[:])
			xorA ^= t
			binary.BigEndian.PutUint64(a[:], xorA)

			// B = AES-1(K, (A ^ t) | R[i])
			var input [16]byte
			copy(input[:8], a[:])
			copy(input[8:], r[i])

			var b [16]byte
			block.Decrypt(b[:], input[:])

			copy(a[:], b[:8])
			copy(r[i], b[8:])
		}
	}

	// Check integrity
	if a != defaultIV {
		// Wrap in ErrDecryptionFailed so a caller testing only that
		// sentinel catches a failed key unwrap the same way it catches a
		// failed RSA key transport, which decryptSessionKey wraps.
		return nil, fmt.Errorf("%w: %w", ErrDecryptionFailed, ErrKeyUnwrapFailed)
	}

	out := make([]byte, n*8)
	for i := range n {
		copy(out[i*8:], r[i])
	}
	return out, nil
}
