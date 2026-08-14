package xmlenc1_test

import (
	"context"
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"testing"
	"time"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xmlenc1"
	"github.com/stretchr/testify/require"
)

// The decryptor's key material is fixed, and never generated per run, because
// it belongs to the target and not to the input: a crasher the fuzzing engine
// writes under testdata/fuzz has to reproduce, and every path past a successful
// key unwrap depends on which key was configured. These keys exist only here
// and protect nothing.
const fuzzRSAPrivateKeyPEM = `-----BEGIN PRIVATE KEY-----
MIIEvgIBADANBgkqhkiG9w0BAQEFAASCBKgwggSkAgEAAoIBAQDoFLvbdAca1sbr
D7R9OCtRT++XJ/hORxbafSgJ+1yPnPSQU23Ryy2MNnqPkRiNIoIaxdDRkeyOnmp9
7LflLJ6VlCQquSMzNmNc2vhhfGclsrA0Sl7zHYTHqCxdbVxoT1onRDUHOjyhflZs
Cm7LeIlI0+gN4a5pIf9V5BehIDCD3SdpqwgUYQL4AXHdXn49jOhtIGwCSW9v6MU+
T0yE7iP19eeyuMVKRpNX3Q2YmHGvZ9hALJ7Ap87rxA0ZlKsV/KoxZ/79JMBOQKD0
aL3AogO5fbLIiMa8zGRnSpvr0krzRMFd/R+lEvKRf68hpqMtuIIk1H+dXi58ZzBF
G8aNo4LNAgMBAAECggEAAfWEj5VRpuRGhpXebKq+ySZJYInFMZo/4MePii11p+vV
ePPX6IwY4EJ65aiiraJQJvwhx3sZpU/7KpNieJVF6SQKxviwrk40Azu39b603+iE
4PixLBCggqqpQTYRwo0U7artFarkHEZN9E9gznotpMc8lCjZCURMRdZokYL6+m6I
uIYVX7QS7bf3KJFySjGY4c9Sal5Wh8YUq6Jl951Gm9TWIEQQrBXzA4ZTTZHHUQy5
wGcOUNKFlMiC44Aw6k6GqjTbgIIz3kajtvofl6k70x9t1BnCp4xZdUbj3CfofoSh
1ntHyyrQmDQC+YfbYSUf/NkjEmlk7ShlyKe37vIE8wKBgQD6x5cS+CW8jhC1O8eZ
qnAud3rCkNB8Zv4ghHsDViMoEwftMxuwRjr8xnzMAv/i+TQSt6nC3wxPSBxcYvmk
B+uf03cWN7yFT9dxpiCOoDX4OqxZkYlOYISzYCFt9WJGFgr/JRiLSdxb3QCeALWx
nlP3LKC6ubCzDA7vFyDOX3x4xwKBgQDs6X+HNOFwwJb8tOCRpPr6BwFIAyeNV+Vy
Yok1DBs75QmQnRM2ENBX2B51yqRpC0M7xdDDrSCd8MJiVnLoBMXKZlwBbMuDp86Q
ftix/IHVc/2yJfeeGNlSmf0wH3cW0DXK9jw1naq3jU/RLve6ZI2iFh/GjXTexhng
X9L3efdbywKBgHGRV6I4jGZic8CPTOoTHHB+nTJlgHUF80nolQjCxnMMg0dxILXo
aCg2/ycoqJcyQdnEIPXmKt3wix9vlxwolhUwH7sJDK/Wo3uNPys39JjwgUKivOqo
nQ/alekE+jdBHkPDmeTiUw+q+u+S5LWGPQIvzK4jD5lV+aFe+PVcmrLbAoGBAOaK
5sYNCKDvWT67SZmRgYYTkQShxTh/Y1GnX7vWdx4W6PLoV8ySGhyRvDqGIu3xvtCI
1HnGnOn1Y0PMum7cThmC+F+OnpEUmCf2uCqj/ThZcnSNC+S2a609GqxcwkfZ/67t
ZXQLZRjPk++NFBc3SLiFbRCLkUJEZuP4e9TFxJd3AoGBAOrq489tTL3m+4Y64i6g
rXKuGSTpZqUQqqijf1Mx5b3ifHla0BjijIMiua1Bi3+ZvhDjc7nhfFJ8Pn0Gk6AQ
OBBxedpxNxE9hTVWn3vi1spvPIGGIVIl8lqZOAyL8Lw3pCZBHLGest4hRvFF/dEw
3cU8XZZDGb+U5swyh0vS0myU
-----END PRIVATE KEY-----
`

const fuzzECPrivateKeyPEM = `-----BEGIN PRIVATE KEY-----
MIGHAgEAMBMGByqGSM49AgEGCCqGSM49AwEHBG0wawIBAQQgCbki1SXiqB/A1m4Y
KTDSGV72K6ftxpY1qyG+6Eis+bKhRANCAAS1cejqV2eUnMv/5k+sRaReLzEYNJNF
d7z7EbsPM10zgCshtqn/9s6SyPMcYbp74ChoyfMBU1Jg9NsH3N3UBcdA
-----END PRIVATE KEY-----
`

// fuzzKEK is the fixed AES-256 key-encryption key the AES key-wrap branch uses.
var fuzzKEK = []byte("xmlenc1-fuzz-key-encryption-key!")

// FuzzDecrypt drives malformed EncryptedData, EncryptedKey, EncryptionMethod,
// and AgreementMethod trees through the public decryptor, which is the whole
// pre-authentication surface: everything it reads comes from the document.
//
// One Decryptor carries an RSA private key, an EC private key, and an AES
// key-encryption key at once — a configuration Decryptor.PrivateKey documents
// as supported — and the candidate branch is dispatched on what the DOCUMENT
// declares. So the fuzz input alone selects the key protection that runs, and a
// single target reaches all three: RSA-OAEP key transport, ECDH-ES key
// agreement with ConcatKDF, and AES key wrap. CBC is opted in so the
// unauthenticated block path is reachable too. SessionKey is deliberately left
// unset: a non-empty one is an early return past candidate selection and key
// resolution, which is most of what is being fuzzed.
//
// The decryptor has no resolver, and the inner plaintext parse runs through the
// hardened parser, so no fuzz input can cause filesystem, network, or DTD work.
func FuzzDecrypt(f *testing.F) {
	rsaKey := fuzzRSAKey(f)
	ecKey := fuzzECKey(f)

	// Three documents this package produced itself, one per key-protection
	// mechanism, so the engine starts from inputs that decrypt all the way
	// through, and never from shapes that fail at the parse gate.
	f.Add(fuzzSeedDocument(f, xmlenc1.NewEncryptor().
		BlockAlgorithm(xmlenc1.AES256CBC).
		AllowLegacyCBC(true).
		KeyWrapAlgorithm(xmlenc1.AES256KeyWrap).
		KeyEncryptionKey(fuzzKEK)))
	f.Add(fuzzSeedDocument(f, xmlenc1.NewEncryptor().
		BlockAlgorithm(xmlenc1.AES256GCM).
		KeyTransportAlgorithm(xmlenc1.RSAOAEP11).
		OAEPDigest(xmlenc1.DigestSHA256).
		OAEPMGF(xmlenc1.MGFSHA256).
		OAEPParams([]byte("xmlenc1-fuzz-oaep-label")).
		RecipientPublicKey(&rsaKey.PublicKey)))
	f.Add(fuzzSeedDocument(f, xmlenc1.NewEncryptor().
		BlockAlgorithm(xmlenc1.AES256GCM11).
		KeyWrapAlgorithm(xmlenc1.AES256KeyWrap).
		RecipientECPublicKey(&ecKey.PublicKey).
		KeyDerivationParams(&xmlenc1.ConcatKDFParams{
			DigestMethod: xmlenc1.DigestSHA256,
			AlgorithmID:  []byte{0x00, 0xa0},
			PartyUInfo:   []byte{0x00, 0x55},
			PartyVInfo:   []byte{0x00, 0xaa},
		})))

	// Three same-document xenc:CipherReference shapes, one per way the octets
	// are produced: canonicalization of a named subtree, a #base64 transform
	// over that subtree's character data, and the null URI naming the whole
	// resolution root. Each names a target INSIDE the EncryptedData, because
	// the target is the document element here and the resolution root is its
	// topmost element ancestor. No seed carries an external URI: the decryptor
	// below has no resolver, so every external form is refused before any I/O
	// could be attempted, and the target stays offline whatever the engine
	// mutates these into.
	for _, cipherReference := range []string{
		`<xenc:CipherReference URI="#ct"/>`,
		`<xenc:CipherReference URI="#ct"><xenc:Transforms><ds:Transform Algorithm="` + xmlenc1.NamespaceDSig + `base64"/></xenc:Transforms></xenc:CipherReference>`,
		`<xenc:CipherReference URI=""/>`,
	} {
		f.Add([]byte(`<xenc:EncryptedData xmlns:xenc="` + xmlenc1.NamespaceXMLEnc + `" xmlns:ds="` + xmlenc1.NamespaceDSig + `" Type="` + xmlenc1.TypeElement + `">` +
			`<xenc:EncryptionMethod Algorithm="` + xmlenc1.AES256GCM + `"/>` +
			`<xenc:EncryptionProperties><ct Id="ct">AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA</ct></xenc:EncryptionProperties>` +
			`<xenc:CipherData>` + cipherReference + `</xenc:CipherData>` +
			`</xenc:EncryptedData>`))
	}

	// An EncryptedData with nothing in it, and one whose shape is complete but
	// whose every value is junk: the two ends the mutation engine works
	// outward from.
	f.Add([]byte(`<xenc:EncryptedData xmlns:xenc="` + xmlenc1.NamespaceXMLEnc + `"/>`))
	f.Add([]byte(`<xenc:EncryptedData xmlns:xenc="` + xmlenc1.NamespaceXMLEnc + `" xmlns:ds="` + xmlenc1.NamespaceDSig + `" Type="` + xmlenc1.TypeElement + `"><xenc:EncryptionMethod Algorithm="` + xmlenc1.AES256GCM + `"/><ds:KeyInfo><xenc:EncryptedKey><xenc:EncryptionMethod Algorithm="` + xmlenc1.RSAOAEP + `"><xenc:OAEPparams>not-base64</xenc:OAEPparams></xenc:EncryptionMethod><xenc:CipherData><xenc:CipherValue>AA==</xenc:CipherValue></xenc:CipherData></xenc:EncryptedKey></ds:KeyInfo><xenc:CipherData><xenc:CipherValue>not-base64</xenc:CipherValue></xenc:CipherData></xenc:EncryptedData>`))

	f.Fuzz(func(t *testing.T, input []byte) {
		if len(input) > 1<<20 {
			return
		}
		ctx, cancel := context.WithTimeout(t.Context(), time.Second)
		defer cancel()

		doc, err := helium.NewParser().Parse(ctx, input)
		if err != nil {
			return
		}
		root := doc.DocumentElement()
		if root == nil {
			return
		}

		decryptor := xmlenc1.NewDecryptor().
			PrivateKey(rsaKey).
			ECPrivateKey(ecKey).
			KeyEncryptionKey(fuzzKEK).
			AllowUnauthenticatedCBC(true).
			MaxEncryptedKeys(32).
			MaxEncryptedKeyBytes(1 << 20).
			MaxCipherValueBytes(1 << 20)
		_, _ = decryptor.Decrypt(ctx, root)
	})
}

// fuzzSeedDocument encrypts one small element and returns the serialized
// EncryptedData, so a seed is always whatever the current Encryptor emits, and never a
// transcript that can drift from it.
func fuzzSeedDocument(f *testing.F, encryptor xmlenc1.Encryptor) []byte {
	f.Helper()
	doc, err := helium.NewParser().Parse(f.Context(), []byte(`<root id="r"><inner attr="v">secret</inner>tail</root>`))
	require.NoError(f, err)
	edElem, err := encryptor.EncryptElement(f.Context(), doc.DocumentElement())
	require.NoError(f, err)
	serialized, err := helium.WriteString(edElem)
	require.NoError(f, err)
	return []byte(serialized)
}

func fuzzRSAKey(f *testing.F) *rsa.PrivateKey {
	f.Helper()
	key := fuzzPrivateKey(f, fuzzRSAPrivateKeyPEM)
	rsaKey, ok := key.(*rsa.PrivateKey)
	require.True(f, ok, "fuzz key PEM must hold an RSA private key, got %T", key)
	return rsaKey
}

func fuzzECKey(f *testing.F) *ecdsa.PrivateKey {
	f.Helper()
	key := fuzzPrivateKey(f, fuzzECPrivateKeyPEM)
	ecKey, ok := key.(*ecdsa.PrivateKey)
	require.True(f, ok, "fuzz key PEM must hold an EC private key, got %T", key)
	return ecKey
}

func fuzzPrivateKey(f *testing.F, keyPEM string) any {
	f.Helper()
	block, _ := pem.Decode([]byte(keyPEM))
	require.NotNil(f, block, "fuzz key PEM must decode")
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	require.NoError(f, err)
	return key
}
