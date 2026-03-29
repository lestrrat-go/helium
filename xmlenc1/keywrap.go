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

// aesKeyUnwrap unwraps a key using the AES Key Wrap algorithm (RFC 3394).
func aesKeyUnwrap(kek, ciphertext []byte) ([]byte, error) {
	if len(ciphertext)%8 != 0 || len(ciphertext) < 24 {
		return nil, fmt.Errorf("%w: ciphertext must be at least 24 bytes and a multiple of 8", ErrKeyUnwrapFailed)
	}

	block, err := aes.NewCipher(kek)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrKeyUnwrapFailed, err)
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
		return nil, ErrKeyUnwrapFailed
	}

	out := make([]byte, n*8)
	for i := range n {
		copy(out[i*8:], r[i])
	}
	return out, nil
}
