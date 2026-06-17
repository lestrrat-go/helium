package xmlenc1

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/sha256"
	"fmt"
	"hash"
)

func encryptSessionKey(algorithm string, pub *rsa.PublicKey, sessionKey []byte, oaepDigest, oaepMGF string, oaepParams []byte) ([]byte, error) {
	h, err := oaepHashFunc(algorithm, oaepDigest, oaepMGF)
	if err != nil {
		return nil, err
	}
	ciphertext, err := rsa.EncryptOAEP(h(), rand.Reader, pub, sessionKey, oaepParams)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrEncryptionFailed, err)
	}
	return ciphertext, nil
}

func decryptSessionKey(algorithm string, priv *rsa.PrivateKey, ciphertext []byte, oaepDigest, oaepMGF string, oaepParams []byte) ([]byte, error) {
	h, err := oaepHashFunc(algorithm, oaepDigest, oaepMGF)
	if err != nil {
		return nil, err
	}
	plaintext, err := rsa.DecryptOAEP(h(), rand.Reader, priv, ciphertext, oaepParams)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDecryptionFailed, err)
	}
	return plaintext, nil
}

// oaepHashFunc resolves the hash used for both the OAEP digest and the
// MGF1 mask generation function from the declared digest and MGF URIs.
//
// Go's crypto/rsa OAEP API (rsa.EncryptOAEP / rsa.DecryptOAEP) uses a
// SINGLE hash for both the label digest and the MGF1 mask generation.
// XML Encryption, by contrast, lets the DigestMethod and MGF advertise
// different hashes. To avoid serializing metadata that lies about the
// crypto actually performed, this function rejects any combination Go
// cannot faithfully represent (digest hash != MGF hash). It NEVER
// silently falls back to SHA-1: an unknown or empty digest/MGF URI is a
// hard error.
func oaepHashFunc(algorithm, digest, mgf string) (func() hash.Hash, error) {
	// Resolve the digest hash. An empty digest defaults to SHA-1, which
	// matches the XML Encryption default for RSA-OAEP. An unrecognized
	// (non-empty) digest URI is rejected rather than downgraded.
	var digestHash func() hash.Hash
	switch digest {
	case "", DigestSHA1:
		digestHash = sha1.New
	case DigestSHA256:
		digestHash = sha256.New
	default:
		return nil, &UnsupportedAlgorithmError{Algorithm: digest}
	}

	// Resolve the MGF hash. The legacy RSAOAEP (rsa-oaep-mgf1p) URI fixes
	// MGF1 to SHA-1; an explicit MGF URI is not permitted for it. The
	// RSAOAEP11 URI carries an explicit MGF, defaulting to MGF1-SHA1.
	var mgfHash func() hash.Hash
	switch algorithm {
	case RSAOAEP:
		if mgf != "" && mgf != MGFSHA1 {
			return nil, &UnsupportedAlgorithmError{Algorithm: mgf}
		}
		mgfHash = sha1.New
	default: // RSAOAEP11 and any RSA-OAEP variant carrying an explicit MGF.
		switch mgf {
		case "", MGFSHA1:
			mgfHash = sha1.New
		case MGFSHA256:
			mgfHash = sha256.New
		default:
			return nil, &UnsupportedAlgorithmError{Algorithm: mgf}
		}
	}

	// Go uses one hash for both digest and MGF1. If the declared hashes
	// differ, the metadata cannot honestly describe what would be done,
	// so reject rather than silently using one for both.
	if hashName(digestHash) != hashName(mgfHash) {
		return nil, fmt.Errorf("%w: RSA-OAEP digest hash and MGF1 hash must match (crypto/rsa uses a single hash for both); got digest %q and MGF %q",
			ErrEncryptionFailed, oaepDigestName(digest), oaepMGFName(algorithm, mgf))
	}

	return digestHash, nil
}

// hashName returns a stable identifier for a hash constructor so two
// constructors can be compared for equality.
func hashName(h func() hash.Hash) string {
	switch h().Size() {
	case sha256.Size:
		return "sha256"
	default:
		return "sha1"
	}
}

func oaepDigestName(digest string) string {
	if digest == "" {
		return DigestSHA1 + " (default)"
	}
	return digest
}

func oaepMGFName(algorithm, mgf string) string {
	if mgf != "" {
		return mgf
	}
	if algorithm == RSAOAEP {
		return MGFSHA1 + " (implied by rsa-oaep-mgf1p)"
	}
	return MGFSHA1 + " (default)"
}
