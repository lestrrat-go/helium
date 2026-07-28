package xmlenc1

import (
	"crypto/ecdh"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/binary"
	"errors"
	"fmt"
	"hash"
)

// decryptECDHSessionKey derives the KEK from the agreement and unwraps the
// session key with it. sessionKeySize is the length the EncryptedData's block
// algorithm requires; aesKeyUnwrap owns what it binds.
func decryptECDHSessionKey(priv *ecdsa.PrivateKey, ek *EncryptedKey, sessionKeySize int) ([]byte, error) {
	agreement := ek.AgreementMethod
	if agreement == nil || agreement.Algorithm != ECDHES {
		return nil, fmt.Errorf("%w: %w", ErrDecryptionFailed, &UnsupportedAlgorithmError{Parameter: paramKeyAgreement, Algorithm: agreementAlgorithm(agreement)})
	}
	if agreement.OriginatorKey == nil {
		return nil, fmt.Errorf("%w: ECDH-ES missing OriginatorKeyInfo", ErrMalformedEncrypted)
	}
	if agreement.KeyDerivationMethod == nil {
		return nil, fmt.Errorf("%w: ECDH-ES missing KeyDerivationMethod", ErrMalformedEncrypted)
	}
	method := agreement.KeyDerivationMethod
	if method.Algorithm != ConcatKDF {
		return nil, fmt.Errorf("%w: %w", ErrDecryptionFailed, &UnsupportedAlgorithmError{Parameter: paramKeyDerivation, Algorithm: method.Algorithm})
	}
	if method.ConcatKDF == nil {
		return nil, fmt.Errorf("%w: ConcatKDF missing parameters", ErrMalformedEncrypted)
	}

	if ek.EncryptionMethod == nil {
		return nil, fmt.Errorf("%w: EncryptedKey missing EncryptionMethod", ErrMalformedEncrypted)
	}
	switch ek.EncryptionMethod.Algorithm {
	case AES128KeyWrap, AES192KeyWrap, AES256KeyWrap:
	default:
		return nil, fmt.Errorf("%w: %w", ErrDecryptionFailed, &UnsupportedAlgorithmError{Parameter: paramKeyWrap, Algorithm: ek.EncryptionMethod.Algorithm})
	}
	kekSize, err := keySizeForAlgorithm(paramKeyWrap, ek.EncryptionMethod.Algorithm)
	if err != nil {
		return nil, err
	}

	privateKey, err := priv.ECDH()
	if err != nil {
		return nil, fmt.Errorf("%w: invalid ECDH private key: %v", ErrDecryptionFailed, err)
	}
	originatorKey, err := agreement.OriginatorKey.Curve.NewPublicKey(agreement.OriginatorKey.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid ECDH public key: %v", ErrDecryptionFailed, err)
	}
	sharedSecret, err := privateKey.ECDH(originatorKey)
	if err != nil {
		return nil, fmt.Errorf("%w: ECDH exchange failed: %v", ErrDecryptionFailed, err)
	}

	kek, err := deriveConcatKDF(sharedSecret, method.ConcatKDF, kekSize)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrDecryptionFailed, err)
	}
	return aesKeyUnwrap(kek, ek.CipherValue, sessionKeySize)
}

// ecdhRecipientKey converts a recipient's ECDSA public key into the
// crypto/ecdh form ECDH-ES needs, and rejects any recipient key the package
// cannot see an encryption through. Three distinct failures live here: the
// key may carry no point at all (an ecdsa.PublicKey with unset coordinates,
// which crypto/ecdsa dereferences rather than reporting), crypto/ecdh refuses
// some curves outright (P-224 has no ECDH form at all), and a curve it does
// accept may still have no dsig11:NamedCurve URI, which would yield an
// EncryptedKey no recipient can parse.
//
// It is the single gate for all three, and resolveEncryptConfig calls it
// before any entry point serializes plaintext, generates a session key, or
// block encrypts, so an unusable recipient key is reported as an error and
// costs nothing proportional to the payload.
func ecdhRecipientKey(recipient *ecdsa.PublicKey) (*ecdh.PublicKey, error) {
	// crypto/ecdsa reads the affine coordinates before validating them, so a
	// public key whose X or Y was never set panics inside (*PublicKey).ECDH
	// instead of erroring. This gate exists to decide whether the recipient
	// key is usable at all, and "carries no point" is the most basic way it
	// can be unusable; the caller's contract is an error, not a panic.
	if recipient.X == nil || recipient.Y == nil {
		return nil, fmt.Errorf("%w: invalid recipient EC public key: missing curve point", ErrEncryptionFailed)
	}

	recipientKey, err := recipient.ECDH()
	if err != nil {
		return nil, fmt.Errorf("%w: invalid recipient EC public key: %v", ErrEncryptionFailed, err)
	}
	if _, err := ecdhURIForCurve(recipientKey.Curve()); err != nil {
		return nil, err
	}
	return recipientKey, nil
}

// encryptECDHSessionKey performs the originator side of XML Encryption 1.1
// ECDH-ES. It generates an ephemeral key pair on the recipient's own curve,
// agrees a shared secret with the recipient's static public key, derives a
// key-encryption key from it with ConcatKDF, and wraps the session key under
// that KEK with AES Key Wrap.
//
// The ephemeral key is generated per call and never retained, which is what
// makes the scheme forward-secret; the resulting EncryptedKey carries the
// ephemeral PUBLIC key so the recipient can reach the same shared secret.
//
// It takes the recipient key already resolved by ecdhRecipientKey, so the
// curve is known to be usable and nameable before the caller does any
// payload-proportional work.
func encryptECDHSessionKey(recipientKey *ecdh.PublicKey, keyWrapAlgorithm string, params *ConcatKDFParams, sessionKey []byte) (*EncryptedKey, error) {
	switch keyWrapAlgorithm {
	case AES128KeyWrap, AES192KeyWrap, AES256KeyWrap:
	default:
		// ECDH-ES derives a KEK, so the EncryptedKey algorithm must be a
		// key wrap. Anything else would declare a mechanism we are not
		// performing.
		return nil, fmt.Errorf("%w: %w", ErrEncryptionFailed, &UnsupportedAlgorithmError{Parameter: paramKeyWrap, Algorithm: keyWrapAlgorithm})
	}
	kekSize, err := keySizeForAlgorithm(paramKeyWrap, keyWrapAlgorithm)
	if err != nil {
		return nil, err
	}

	curve := recipientKey.Curve()
	ephemeral, err := curve.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrEncryptionFailed, err)
	}
	sharedSecret, err := ephemeral.ECDH(recipientKey)
	if err != nil {
		return nil, fmt.Errorf("%w: ECDH exchange failed: %v", ErrEncryptionFailed, err)
	}

	kek, err := deriveConcatKDF(sharedSecret, params, kekSize)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrEncryptionFailed, err)
	}
	wrapped, err := aesKeyWrap(kek, sessionKey)
	if err != nil {
		return nil, err
	}

	return &EncryptedKey{
		EncryptionMethod: &EncryptionMethod{Algorithm: keyWrapAlgorithm},
		CipherValue:      wrapped,
		AgreementMethod: &AgreementMethod{
			Algorithm: ECDHES,
			KeyDerivationMethod: &KeyDerivationMethod{
				Algorithm: ConcatKDF,
				ConcatKDF: params,
			},
			OriginatorKey: &ECKeyValue{
				Curve:     curve,
				PublicKey: ephemeral.PublicKey().Bytes(),
			},
		},
	}, nil
}

// effectiveKDFParams applies the fallback [Encryptor.KeyDerivationParams]
// documents, for an Encryptor that selected ECDH-ES without spelling the
// derivation out. SHA-256 with no OtherInfo is the neutral choice: it commits
// to no application-specific AlgorithmID/PartyInfo the recipient would have
// to guess, and the emitted xenc11:ConcatKDFParams states it explicitly on
// the wire either way. The fallback is all-or-nothing so that caller
// OtherInfo is never silently paired with a digest the caller never chose.
func effectiveKDFParams(params *ConcatKDFParams) *ConcatKDFParams {
	if params == nil || params.DigestMethod == "" {
		return &ConcatKDFParams{DigestMethod: DigestSHA256}
	}
	return params
}

func agreementAlgorithm(agreement *AgreementMethod) string {
	if agreement == nil {
		return ""
	}
	return agreement.Algorithm
}

// deriveConcatKDF runs the NIST SP 800-56A concatenation KDF over the agreed
// shared secret. Its errors are UNWRAPPED so the caller classifies them:
// decryptECDHSessionKey wraps them with ErrDecryptionFailed and
// encryptECDHSessionKey with ErrEncryptionFailed, mirroring oaepHashes.
func deriveConcatKDF(sharedSecret []byte, params *ConcatKDFParams, keySize int) ([]byte, error) {
	newHash, err := concatKDFHash(params.DigestMethod)
	if err != nil {
		return nil, err
	}
	otherInfo, err := concatKDFOtherInfo(params)
	if err != nil {
		return nil, err
	}

	result := make([]byte, 0, keySize)
	for counter := uint32(1); len(result) < keySize; counter++ {
		h := newHash()
		var counterBytes [4]byte
		binary.BigEndian.PutUint32(counterBytes[:], counter)
		_, _ = h.Write(counterBytes[:])
		_, _ = h.Write(sharedSecret)
		_, _ = h.Write(otherInfo)
		result = append(result, h.Sum(nil)...)
		if counter == ^uint32(0) && len(result) < keySize {
			return nil, errors.New("xmlenc1: ConcatKDF output is too large")
		}
	}
	return result[:keySize], nil
}

// maxConcatKDFOtherInfoBytes bounds the five xenc11:ConcatKDFParams OtherInfo
// fields — AlgorithmID, PartyUInfo, PartyVInfo, SuppPubInfo, SuppPrivInfo —
// taken TOGETHER, as the sum of their decoded octet lengths. It is the single
// statement of that limit; checkConcatKDFOtherInfoBudget applies it.
//
// The fields are NIST SP 800-56A OtherInfo: identifiers, party names, and
// nonces. The W3C xmlenc-core1 examples use a 16-octet PartyUInfo and leave
// the rest empty, and the largest value a real deployment carries is on the
// order of an algorithm URI or a certificate digest — tens of octets. 4 KiB
// is two orders of magnitude above that, so no interoperable document is
// rejected, while a hostile one can no longer hand the KDF an input sized
// only by what the XML parser happened to accept: without this ceiling the
// packing below runs one iteration per input BIT, and five fields at the
// parser's own 10 MiB attribute cap are ~210 million of them for a single
// EncryptedKey.
const maxConcatKDFOtherInfoBytes = 4096

// checkConcatKDFOtherInfoBudget rejects a ConcatKDFParams whose OtherInfo
// fields exceed maxConcatKDFOtherInfoBytes in total.
//
// It runs at both ends of the parameters' life: parseConcatKDFParams applies
// it to wire data at the point the whole set is first known, so an oversized
// document is refused before any ECDH or KDF work, and concatKDFOtherInfo
// applies it again immediately before the packing loop, which is what makes
// it hold for a caller that hands xmlenc1 a DOM or a ConcatKDFParams it built
// itself rather than one this package parsed.
func checkConcatKDFOtherInfoBudget(params *ConcatKDFParams) error {
	total := len(params.AlgorithmID) + len(params.PartyUInfo) + len(params.PartyVInfo) +
		len(params.SuppPubInfo) + len(params.SuppPrivInfo)
	if total > maxConcatKDFOtherInfoBytes {
		return fmt.Errorf("%w: ConcatKDF OtherInfo is %d bytes, over the %d byte limit", ErrMalformedEncrypted, total, maxConcatKDFOtherInfoBytes)
	}
	return nil
}

// concatKDFOtherInfo packs the five OtherInfo fields into the single bit
// string ConcatKDF hashes. Each field is a bit string carrying an unused-bit
// count, so a field whose bit length is not a multiple of eight leaves every
// field after it straddling octet boundaries.
func concatKDFOtherInfo(params *ConcatKDFParams) ([]byte, error) {
	if err := checkConcatKDFOtherInfoBudget(params); err != nil {
		return nil, err
	}

	fields := []struct {
		value      []byte
		unusedBits uint8
	}{
		{value: params.AlgorithmID, unusedBits: params.algorithmIDUnusedBits},
		{value: params.PartyUInfo, unusedBits: params.partyUInfoUnusedBits},
		{value: params.PartyVInfo, unusedBits: params.partyVInfoUnusedBits},
		{value: params.SuppPubInfo, unusedBits: params.suppPubInfoUnusedBits},
		{value: params.SuppPrivInfo, unusedBits: params.suppPrivInfoUnusedBits},
	}

	totalBits := 0
	for _, field := range fields {
		if int(field.unusedBits) > len(field.value)*8 {
			return nil, fmt.Errorf("%w: invalid ConcatKDF bitstring", ErrMalformedEncrypted)
		}
		totalBits += len(field.value)*8 - int(field.unusedBits)
	}

	otherInfo := make([]byte, (totalBits+7)/8)
	bitOffset := 0
	for _, field := range fields {
		bitCount := len(field.value)*8 - int(field.unusedBits)
		// A field landing on an octet boundary keeps its own alignment: its
		// whole octets go across unshifted and only a trailing partial octet
		// needs masking. Every preceding field has a bit length that is a
		// multiple of eight in that case, which is what every W3C example and
		// every hexBinary-with-zero-unused-bits document looks like.
		if bitOffset%8 == 0 {
			fullBytes := bitCount / 8
			copy(otherInfo[bitOffset/8:], field.value[:fullBytes])
			if rem := bitCount % 8; rem != 0 {
				otherInfo[bitOffset/8+fullBytes] |= field.value[fullBytes] & (^byte(0) << (8 - rem))
			}
			bitOffset += bitCount
			continue
		}
		// Straddling the boundary: fall back to moving one bit at a time.
		// maxConcatKDFOtherInfoBytes bounds how long that can run.
		for i := range bitCount {
			if field.value[i/8]&(1<<uint(7-i%8)) != 0 {
				otherInfo[bitOffset/8] |= 1 << uint(7-bitOffset%8)
			}
			bitOffset++
		}
	}
	return otherInfo, nil
}

func concatKDFHash(uri string) (func() hash.Hash, error) {
	switch uri {
	case DigestSHA1:
		return sha1.New, nil
	case DigestSHA256:
		return sha256.New, nil
	case DigestSHA384, DigestSHA384DSigMore:
		return sha512.New384, nil
	case DigestSHA512:
		return sha512.New, nil
	default:
		return nil, &UnsupportedAlgorithmError{Parameter: paramConcatKDF, Algorithm: uri}
	}
}
