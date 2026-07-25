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
	switch ek.EncryptionMethod.Algorithm {
	case AES128KeyWrap, AES192KeyWrap, AES256KeyWrap:
	default:
		return nil, fmt.Errorf("%w: %w", ErrDecryptionFailed, &UnsupportedAlgorithmError{Algorithm: ek.EncryptionMethod.Algorithm})
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
			return nil, fmt.Errorf("%w: ConcatKDF output is too large", ErrDecryptionFailed)
		}
	}
	return result[:keySize], nil
}

func concatKDFOtherInfo(params *ConcatKDFParams) ([]byte, error) {
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
		return nil, fmt.Errorf("%w: %w", ErrDecryptionFailed, &UnsupportedAlgorithmError{Algorithm: uri})
	}
}
