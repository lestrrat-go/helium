package xmlenc1

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"fmt"

	_ "crypto/sha1"   // register SHA-1 for crypto.Hash.New (OAEP digest/MGF)
	_ "crypto/sha256" // register SHA-256 for crypto.Hash.New (OAEP digest/MGF)
)

func encryptSessionKey(algorithm string, pub *rsa.PublicKey, sessionKey []byte, oaepDigest, oaepMGF string, oaepParams []byte) ([]byte, error) {
	digestHash, mgfHash, err := oaepHashes(algorithm, oaepDigest, oaepMGF)
	if err != nil {
		// oaepHashes returns an unwrapped parameter error; classify it
		// for the encrypt path so callers see ErrEncryptionFailed while
		// preserving the typed error in the chain for errors.As.
		return nil, fmt.Errorf("%w: %w", ErrEncryptionFailed, err)
	}
	ciphertext, err := rsa.EncryptOAEPWithOptions(rand.Reader, pub, sessionKey, &rsa.OAEPOptions{
		Hash:    digestHash,
		MGFHash: mgfHash,
		Label:   oaepParams,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrEncryptionFailed, err)
	}
	return ciphertext, nil
}

func decryptSessionKey(algorithm string, priv *rsa.PrivateKey, ciphertext []byte, oaepDigest, oaepMGF string, oaepParams []byte) ([]byte, error) {
	digestHash, mgfHash, err := oaepHashes(algorithm, oaepDigest, oaepMGF)
	if err != nil {
		// oaepHashes returns an unwrapped parameter error; classify it
		// for the decrypt path so callers see ErrDecryptionFailed while
		// preserving the typed error in the chain for errors.As.
		return nil, fmt.Errorf("%w: %w", ErrDecryptionFailed, err)
	}
	plaintext, err := priv.Decrypt(rand.Reader, ciphertext, &rsa.OAEPOptions{
		Hash:    digestHash,
		MGFHash: mgfHash,
		Label:   oaepParams,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDecryptionFailed, err)
	}
	return plaintext, nil
}

// oaepHashes resolves the OAEP label digest and the MGF1 mask-generation
// hash from the declared digest and MGF URIs.
//
// crypto/rsa's option-bearing OAEP API (rsa.EncryptOAEPWithOptions and
// rsa.PrivateKey.Decrypt with *rsa.OAEPOptions) can represent a distinct
// label digest and MGF1 hash, which mirrors XML Encryption's separate
// DigestMethod and MGF elements. The two hashes need NOT agree, so this
// function returns them independently.
//
// It NEVER silently falls back to SHA-1 for an unrecognized URI: an unknown
// (non-empty) digest/MGF URI is a hard error. Empty URIs resolve to the
// XML-Encryption default for the algorithm.
//
// All errors returned here are UNWRAPPED parameter-validation errors
// (e.g. *UnsupportedAlgorithmError). Callers on the encrypt path wrap them
// with ErrEncryptionFailed and callers on the decrypt path wrap them with
// ErrDecryptionFailed, so errors.Is reflects which path failed.
func oaepHashes(algorithm, digest, mgf string) (crypto.Hash, crypto.Hash, error) {
	// Whitelist the key-transport algorithm itself. Only the two supported
	// RSA-OAEP variants may proceed; any other (or empty) URI is rejected
	// here rather than silently performing RSA-OAEP and serializing a
	// metadata @Algorithm that lies about the crypto actually performed.
	switch algorithm {
	case RSAOAEP, RSAOAEP11:
	default:
		return 0, 0, &UnsupportedAlgorithmError{Algorithm: algorithm}
	}

	// Resolve the label digest. An empty digest defaults to SHA-1, which
	// matches the XML Encryption default for RSA-OAEP. An unrecognized
	// (non-empty) digest URI is rejected rather than downgraded.
	var digestHash crypto.Hash
	switch digest {
	case "", DigestSHA1:
		digestHash = crypto.SHA1
	case DigestSHA256:
		digestHash = crypto.SHA256
	default:
		return 0, 0, &UnsupportedAlgorithmError{Algorithm: digest}
	}

	// Resolve the MGF1 hash. The legacy RSAOAEP (rsa-oaep-mgf1p) URI fixes
	// MGF1 to SHA-1; an explicit MGF URI is not permitted for it. The
	// RSAOAEP11 URI carries an explicit MGF; when absent it defaults to
	// MGF1 with SHA-1 per W3C xmlenc-core1 §5.5.2, INDEPENDENT of the
	// DigestMethod (a distinct DigestMethod is still honored via OAEPOptions).
	var mgfHash crypto.Hash
	switch algorithm {
	case RSAOAEP:
		// XML-Enc 1.1: an xenc11:MGF element MUST NOT be provided for
		// rsa-oaep-mgf1p; its MGF1-SHA-1 is implicit. Reject any MGF.
		if mgf != "" {
			return 0, 0, &UnsupportedAlgorithmError{Algorithm: mgf}
		}
		mgfHash = crypto.SHA1
	default: // RSAOAEP11, which carries an explicit MGF.
		switch mgf {
		case "":
			// W3C xmlenc-core1 §5.5.2: an absent MGF defaults to MGF1
			// with SHA-1, independent of the DigestMethod. Conforming
			// implementations assume SHA-1 MGF when no MGF is present,
			// so defaulting to digestHash here would break interop.
			mgfHash = crypto.SHA1
		case MGFSHA1:
			mgfHash = crypto.SHA1
		case MGFSHA256:
			mgfHash = crypto.SHA256
		default:
			return 0, 0, &UnsupportedAlgorithmError{Algorithm: mgf}
		}
	}

	return digestHash, mgfHash, nil
}
