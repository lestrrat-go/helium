package xmlenc1

import (
	"crypto/ecdsa"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/binary"
	"fmt"
	"hash"
)

func decryptECDHSessionKey(priv *ecdsa.PrivateKey, ek *EncryptedKey) ([]byte, error) {
	agreement := ek.AgreementMethod
	if agreement == nil || agreement.Algorithm != ECDHES {
		return nil, fmt.Errorf("%w: %w", ErrDecryptionFailed, &UnsupportedAlgorithmError{Algorithm: agreementAlgorithm(agreement)})
	}
	if agreement.OriginatorKey == nil {
		return nil, fmt.Errorf("%w: ECDH-ES missing OriginatorKeyInfo", ErrMalformedEncrypted)
	}
	if agreement.KeyDerivationMethod == nil {
		return nil, fmt.Errorf("%w: ECDH-ES missing KeyDerivationMethod", ErrMalformedEncrypted)
	}
	method := agreement.KeyDerivationMethod
	if method.Algorithm != ConcatKDF {
		return nil, fmt.Errorf("%w: %w", ErrDecryptionFailed, &UnsupportedAlgorithmError{Algorithm: method.Algorithm})
	}
	if method.ConcatKDF == nil {
		return nil, fmt.Errorf("%w: ConcatKDF missing parameters", ErrMalformedEncrypted)
	}

	if ek.EncryptionMethod == nil {
		return nil, fmt.Errorf("%w: EncryptedKey missing EncryptionMethod", ErrMalformedEncrypted)
	}
	kekSize, err := keySizeForAlgorithm(ek.EncryptionMethod.Algorithm)
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
		return nil, err
	}
	return aesKeyUnwrap(kek, ek.CipherValue)
}

func agreementAlgorithm(agreement *AgreementMethod) string {
	if agreement == nil {
		return ""
	}
	return agreement.Algorithm
}

func deriveConcatKDF(sharedSecret []byte, params *ConcatKDFParams, keySize int) ([]byte, error) {
	newHash, err := concatKDFHash(params.DigestMethod)
	if err != nil {
		return nil, err
	}
	otherInfo := make([]byte, 0,
		len(params.AlgorithmID)+len(params.PartyUInfo)+len(params.PartyVInfo)+
			len(params.SuppPubInfo)+len(params.SuppPrivInfo))
	otherInfo = append(otherInfo, params.AlgorithmID...)
	otherInfo = append(otherInfo, params.PartyUInfo...)
	otherInfo = append(otherInfo, params.PartyVInfo...)
	otherInfo = append(otherInfo, params.SuppPubInfo...)
	otherInfo = append(otherInfo, params.SuppPrivInfo...)

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
			return nil, fmt.Errorf("%w: ConcatKDF output is too large", ErrDecryptionFailed)
		}
	}
	return result[:keySize], nil
}

func concatKDFHash(uri string) (func() hash.Hash, error) {
	switch uri {
	case DigestSHA1:
		return sha1.New, nil
	case DigestSHA256:
		return sha256.New, nil
	case DigestSHA384:
		return sha512.New384, nil
	case DigestSHA512:
		return sha512.New, nil
	default:
		return nil, fmt.Errorf("%w: %w", ErrDecryptionFailed, &UnsupportedAlgorithmError{Algorithm: uri})
	}
}
