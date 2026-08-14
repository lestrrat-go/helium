package xmlenc1

import (
	"crypto/ecdh"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	mrand "math/rand"
	"strconv"
	"strings"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

func TestConcatKDFNonByteAlignedOtherInfo(t *testing.T) {
	doc, err := helium.NewParser().Parse(t.Context(), []byte(
		`<xenc11:ConcatKDFParams xmlns:xenc11="`+NamespaceXMLEnc11+`" xmlns:ds="`+NamespaceDSig+`" PartyUInfo="03D8" PartyVInfo="0780"><ds:DigestMethod Algorithm="`+DigestSHA256+`"/></xenc11:ConcatKDFParams>`,
	))
	require.NoError(t, err)

	params, err := parseConcatKDFParams(t.Context(), doc.DocumentElement())
	require.NoError(t, err)
	require.Equal(t, []byte{0xd8}, params.PartyUInfo)
	require.Equal(t, uint8(3), params.partyUInfoUnusedBits)
	require.Equal(t, []byte{0x80}, params.PartyVInfo)
	require.Equal(t, uint8(7), params.partyVInfoUnusedBits)

	sharedSecret := []byte{0x01, 0x02, 0x03, 0x04}
	got, err := deriveConcatKDF(sharedSecret, params, sha256.Size)
	require.NoError(t, err)

	input := []byte{0, 0, 0, 1}
	input = append(input, sharedSecret...)
	input = append(input, 0xdc) // 11011 || 1, repacked with two trailing zero bits.
	want := sha256.Sum256(input)
	require.Equal(t, want[:], got)
}

func TestConcatKDFSHA384AcceptsXMLDSigMoreURI(t *testing.T) {
	sharedSecret := []byte{0x01, 0x02, 0x03, 0x04}

	canonical, err := deriveConcatKDF(sharedSecret, &ConcatKDFParams{DigestMethod: DigestSHA384}, 48)
	require.NoError(t, err)
	alias, err := deriveConcatKDF(sharedSecret, &ConcatKDFParams{DigestMethod: DigestSHA384DSigMore}, 48)
	require.NoError(t, err)
	require.Equal(t, canonical, alias)
}

func TestDecryptECDHSessionKeyAllowsOnlyAESKeyWrap(t *testing.T) {
	tests := []struct {
		name      string
		algorithm string
		wantError bool
	}{
		{name: "AES-128 key wrap", algorithm: AES128KeyWrap},
		{name: "AES-192 key wrap", algorithm: AES192KeyWrap},
		{name: "AES-256 key wrap", algorithm: AES256KeyWrap},
		{name: "AES-128 GCM", algorithm: AES128GCM, wantError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			priv, ek, want := newECDHEncryptedKey(t, tt.algorithm)
			// newECDHEncryptedKey wraps a 16-byte session key.
			got, err := decryptECDHSessionKey(priv, ek, 16)
			if tt.wantError {
				require.ErrorIs(t, err, ErrDecryptionFailed)
				var unsupported *UnsupportedAlgorithmError
				require.ErrorAs(t, err, &unsupported)
				return
			}
			require.NoError(t, err)
			require.Equal(t, want, got)
		})
	}
}

func newECDHEncryptedKey(t *testing.T, algorithm string) (*ecdsa.PrivateKey, *encryptedKey, []byte) {
	t.Helper()

	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	originatorKey, err := ecdh.P256().GenerateKey(rand.Reader)
	require.NoError(t, err)
	recipientECDH, err := privateKey.ECDH()
	require.NoError(t, err)
	sharedSecret, err := recipientECDH.ECDH(originatorKey.PublicKey())
	require.NoError(t, err)

	kekSize, err := keySizeForAlgorithm(paramKeyWrap, algorithm)
	require.NoError(t, err)
	params := &ConcatKDFParams{DigestMethod: DigestSHA256}
	kek, err := deriveConcatKDF(sharedSecret, params, kekSize)
	require.NoError(t, err)

	want := make([]byte, 16)
	_, err = rand.Read(want)
	require.NoError(t, err)
	cipherValue, err := aesKeyWrap(kek, want)
	require.NoError(t, err)

	return privateKey, &encryptedKey{
		EncryptionMethod: &encryptionMethod{Algorithm: algorithm},
		CipherValue:      cipherValue,
		AgreementMethod: &agreementMethod{
			Algorithm: ECDHES,
			KeyDerivationMethod: &keyDerivationMethod{
				Algorithm: ConcatKDF,
				ConcatKDF: params,
			},
			OriginatorKey: &ecKeyValue{
				curve:     ecdh.P256(),
				PublicKey: originatorKey.PublicKey().Bytes(),
			},
		},
	}, want
}

// concatKDFOtherInfoPerBit is the bit-at-a-time reference packing, kept here
// verbatim so the production octet-oriented path can be pinned against it.
// The derived KEK is what two implementations have to agree on, so any change
// to the packing that is not byte-identical is an interoperability break.
func concatKDFOtherInfoPerBit(params *ConcatKDFParams) ([]byte, error) {
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

func TestConcatKDFOtherInfoMatchesPerBitPacking(t *testing.T) {
	// Hand-picked shapes first: every unused-bit count, empty fields in
	// leading/trailing/interior position, a field long enough for an
	// unused-bit count past a whole octet, and the all-empty case.
	fixed := []*ConcatKDFParams{
		{},
		{PartyUInfo: []byte{0xb9, 0xe1, 0x3a, 0x70}},
		{AlgorithmID: []byte{0xff}, algorithmIDUnusedBits: 7},
		{AlgorithmID: []byte{0xd8}, algorithmIDUnusedBits: 3, PartyUInfo: []byte{0x80}, partyUInfoUnusedBits: 7},
		{PartyUInfo: []byte{0xaa, 0x55}, partyUInfoUnusedBits: 1, PartyVInfo: []byte{0x0f, 0xf0, 0x3c}, partyVInfoUnusedBits: 5, SuppPubInfo: []byte{0x01}},
		{AlgorithmID: []byte{0x12, 0x34}, PartyVInfo: []byte{0x56}, partyVInfoUnusedBits: 2, SuppPrivInfo: []byte{0x78, 0x9a}},
		{SuppPrivInfo: []byte{0xff, 0xff, 0xff}, suppPrivInfoUnusedBits: 10},
		{AlgorithmID: []byte{0xc3}, algorithmIDUnusedBits: 8},
		{PartyUInfo: []byte{}, PartyVInfo: nil, SuppPubInfo: []byte{0x00}},
	}
	for _, unused := range []uint8{0, 1, 2, 3, 4, 5, 6, 7} {
		fixed = append(fixed, &ConcatKDFParams{
			AlgorithmID:            []byte{0x9c, 0x3f, 0xa1},
			algorithmIDUnusedBits:  unused,
			PartyUInfo:             []byte{0x5a, 0xa5},
			partyUInfoUnusedBits:   unused,
			PartyVInfo:             []byte{0x0f},
			partyVInfoUnusedBits:   unused,
			SuppPubInfo:            []byte{0xde, 0xad, 0xbe, 0xef},
			suppPubInfoUnusedBits:  unused,
			SuppPrivInfo:           []byte{0x01, 0x80},
			suppPrivInfoUnusedBits: unused,
		})
	}

	for i, params := range fixed {
		want, wantErr := concatKDFOtherInfoPerBit(params)
		got, gotErr := concatKDFOtherInfo(params)
		require.Equal(t, wantErr == nil, gotErr == nil, "case %d error disagreement: %v vs %v", i, wantErr, gotErr)
		require.Equal(t, want, got, "case %d packing differs", i)
	}

	// Then a randomized spread, deterministic so a failure is reproducible.
	rng := mrand.New(mrand.NewSource(20260728))
	for iter := range 20000 {
		params := &ConcatKDFParams{}
		dests := []struct {
			value  *[]byte
			unused *uint8
		}{
			{&params.AlgorithmID, &params.algorithmIDUnusedBits},
			{&params.PartyUInfo, &params.partyUInfoUnusedBits},
			{&params.PartyVInfo, &params.partyVInfoUnusedBits},
			{&params.SuppPubInfo, &params.suppPubInfoUnusedBits},
			{&params.SuppPrivInfo, &params.suppPrivInfoUnusedBits},
		}
		for _, d := range dests {
			n := rng.Intn(9)
			if n > 0 {
				buf := make([]byte, n)
				_, err := rng.Read(buf)
				require.NoError(t, err)
				*d.value = buf
				// Mostly a legal 0-7, occasionally a whole-octet-or-more
				// count so the general path sees bitCount < len(value)*8-7.
				*d.unused = uint8(rng.Intn(8))
				if rng.Intn(8) == 0 {
					*d.unused = uint8(rng.Intn(n*8 + 1))
				}
			}
		}
		want, wantErr := concatKDFOtherInfoPerBit(params)
		got, gotErr := concatKDFOtherInfo(params)
		require.Equal(t, wantErr == nil, gotErr == nil, "iteration %d error disagreement: %v vs %v", iter, wantErr, gotErr)
		require.Equal(t, want, got, "iteration %d packing differs for %+v", iter, params)
	}
}

func TestConcatKDFOtherInfoBudget(t *testing.T) {
	// The budget is cumulative: five fields each comfortably under it still
	// add up to a rejection, and a set landing exactly on the limit is
	// accepted.
	overflowing := &ConcatKDFParams{
		DigestMethod: DigestSHA256,
		AlgorithmID:  make([]byte, maxConcatKDFOtherInfoBytes/4),
		PartyUInfo:   make([]byte, maxConcatKDFOtherInfoBytes/4),
		PartyVInfo:   make([]byte, maxConcatKDFOtherInfoBytes/4),
		SuppPubInfo:  make([]byte, maxConcatKDFOtherInfoBytes/4),
		SuppPrivInfo: []byte{0x01},
	}
	_, err := concatKDFOtherInfo(overflowing)
	require.ErrorIs(t, err, ErrMalformedEncrypted)

	atLimit := &ConcatKDFParams{
		DigestMethod: DigestSHA256,
		AlgorithmID:  make([]byte, maxConcatKDFOtherInfoBytes-1),
		PartyUInfo:   []byte{0x01},
	}
	packed, err := concatKDFOtherInfo(atLimit)
	require.NoError(t, err)
	require.Len(t, packed, maxConcatKDFOtherInfoBytes)

	// One byte past the limit is the first set that must be refused, and it
	// is the case an off-by-one in the comparison would let through.
	overLimit := &ConcatKDFParams{
		DigestMethod: DigestSHA256,
		AlgorithmID:  make([]byte, maxConcatKDFOtherInfoBytes-1),
		PartyUInfo:   []byte{0x01, 0x02},
	}
	_, err = concatKDFOtherInfo(overLimit)
	require.ErrorIs(t, err, ErrMalformedEncrypted)
}

// parseConcatKDFHexAttribute refuses a hex attribute from its ENCODED length
// alone, before hex.DecodeString allocates half of it. That ceiling coincides
// exactly with the cumulative OtherInfo budget, so the precheck is an
// allocation-avoidance guard, and no distinct semantic gate: every value
// it refuses the set-wide check in parseConcatKDFParams would refuse too, only
// after paying for the decode. Its whole value is the allocation it never
// makes, which is why it is not redundant and must not be simplified away.
// Both guards wrap ErrMalformedEncrypted, so the message is what distinguishes
// which one fired.
func TestConcatKDFHexAttributePrecheckBoundary(t *testing.T) {
	// Two hex characters per octet, over the field's own bytes plus the
	// leading unused-bit octet the encoded form carries.
	const limit = 2 * (maxConcatKDFOtherInfoBytes + 1)

	tests := []struct {
		name      string
		hexLen    int
		wantError bool
	}{
		{name: "at the limit", hexLen: limit},
		{name: "one octet past the limit", hexLen: limit + 2, wantError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := helium.NewParser().Parse(t.Context(), []byte(
				`<xenc11:ConcatKDFParams xmlns:xenc11="`+NamespaceXMLEnc11+`" PartyUInfo="`+strings.Repeat("0", tt.hexLen)+`"/>`,
			))
			require.NoError(t, err)

			field, unusedBits, err := parseConcatKDFHexAttribute(doc.DocumentElement(), "PartyUInfo")
			if tt.wantError {
				require.ErrorIs(t, err, ErrMalformedEncrypted)
				require.Contains(t, err.Error(), fmt.Sprintf("ConcatKDF PartyUInfo alone is over the %d byte OtherInfo limit", maxConcatKDFOtherInfoBytes))
				return
			}
			require.NoError(t, err)
			require.Zero(t, unusedBits)
			require.Len(t, field, tt.hexLen/2-1)
		})
	}
}

// A 32-bit int overflows far below the address space a caller can fill,
// because the five OtherInfo fields are caller-supplied and may all point at
// ONE slice. Five aliases of a 512 MiB slice total 2.5 GiB, which wraps
// negative in a 32-bit int, so a check that adds the five lengths together
// accepts the set. The packing arithmetic wraps the same way — len(value)*8
// comes out 0 — and the KDF would derive from an EMPTY OtherInfo that no
// 64-bit peer agrees with, instead of failing.
func TestConcatKDFOtherInfoBudgetIsOverflowSafe(t *testing.T) {
	if strconv.IntSize != 32 {
		t.Skip("int is wider than 32 bits here; run with GOARCH=386 or GOARCH=arm to exercise the wrap")
	}

	const per = 512 << 20
	aliased := make([]byte, per)
	params := &ConcatKDFParams{
		DigestMethod: DigestSHA256,
		AlgorithmID:  aliased,
		PartyUInfo:   aliased,
		PartyVInfo:   aliased,
		SuppPubInfo:  aliased,
		SuppPrivInfo: aliased,
	}
	_, err := concatKDFOtherInfo(params)
	require.ErrorIs(t, err, ErrMalformedEncrypted)
}

func benchmarkParams() *ConcatKDFParams {
	const per = maxConcatKDFOtherInfoBytes / 5
	return &ConcatKDFParams{
		DigestMethod: DigestSHA256,
		AlgorithmID:  make([]byte, per),
		PartyUInfo:   make([]byte, per),
		PartyVInfo:   make([]byte, per),
		SuppPubInfo:  make([]byte, per),
		SuppPrivInfo: make([]byte, per),
	}
}

// BenchmarkConcatKDFOtherInfo contrasts the octet-oriented packing with the
// bit-at-a-time reference on the same byte-aligned input.
func BenchmarkConcatKDFOtherInfo(b *testing.B) {
	params := benchmarkParams()
	b.Run("octets", func(b *testing.B) {
		for range b.N {
			if _, err := concatKDFOtherInfo(params); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("per-bit", func(b *testing.B) {
		for range b.N {
			if _, err := concatKDFOtherInfoPerBit(params); err != nil {
				b.Fatal(err)
			}
		}
	})
}
