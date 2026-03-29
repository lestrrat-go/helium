package xmlenc1

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/sha256"
	"fmt"
	"hash"
)

func encryptSessionKey(algorithm string, pub *rsa.PublicKey, sessionKey []byte, oaepDigest, oaepMGF string, oaepParams []byte) ([]byte, error) {
	h := oaepHashFunc(algorithm, oaepDigest)
	opts := &rsa.OAEPOptions{
		Hash:    oaepCryptoHash(algorithm, oaepDigest),
		Label:   oaepParams,
		MGFHash: oaepMGFHash(algorithm, oaepMGF),
	}
	return rsa.EncryptOAEP(h(), rand.Reader, pub, sessionKey, opts.Label)
}

func decryptSessionKey(algorithm string, priv *rsa.PrivateKey, ciphertext []byte, oaepDigest, oaepMGF string, oaepParams []byte) ([]byte, error) {
	h := oaepHashFunc(algorithm, oaepDigest)
	plaintext, err := rsa.DecryptOAEP(h(), rand.Reader, priv, ciphertext, oaepParams)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDecryptionFailed, err)
	}
	return plaintext, nil
}

func oaepHashFunc(algorithm, digest string) func() hash.Hash {
	if algorithm == RSAOAEP {
		// xmlenc 1.0: always SHA-1
		return sha1.New
	}
	// xmlenc 1.1: configurable
	switch digest {
	case DigestSHA256:
		return sha256.New
	default:
		return sha1.New
	}
}

func oaepCryptoHash(algorithm, digest string) crypto.Hash {
	if algorithm == RSAOAEP {
		return crypto.SHA1
	}
	switch digest {
	case DigestSHA256:
		return crypto.SHA256
	default:
		return crypto.SHA1
	}
}

func oaepMGFHash(algorithm, mgf string) crypto.Hash {
	if algorithm == RSAOAEP {
		return crypto.SHA1
	}
	switch mgf {
	case MGFSHA256:
		return crypto.SHA256
	default:
		return crypto.SHA1
	}
}
