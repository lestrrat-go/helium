package xmlenc1

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/sha256"
	"fmt"
	"hash"
)

func encryptSessionKey(algorithm string, pub *rsa.PublicKey, sessionKey []byte, oaepDigest, _ string, oaepParams []byte) ([]byte, error) {
	h := oaepHashFunc(algorithm, oaepDigest)
	return rsa.EncryptOAEP(h(), rand.Reader, pub, sessionKey, oaepParams)
}

func decryptSessionKey(algorithm string, priv *rsa.PrivateKey, ciphertext []byte, oaepDigest, _ string, oaepParams []byte) ([]byte, error) {
	h := oaepHashFunc(algorithm, oaepDigest)
	plaintext, err := rsa.DecryptOAEP(h(), rand.Reader, priv, ciphertext, oaepParams)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDecryptionFailed, err)
	}
	return plaintext, nil
}

func oaepHashFunc(algorithm, digest string) func() hash.Hash {
	if algorithm == RSAOAEP {
		return sha1.New
	}
	switch digest {
	case DigestSHA256:
		return sha256.New
	default:
		return sha1.New
	}
}
