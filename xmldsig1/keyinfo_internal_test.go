package xmldsig1

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"fmt"
	"math/big"
	"testing"
	"time"

	helium "github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

func ecElem(t *testing.T, inner string) *helium.Element {
	t.Helper()
	full := `<dsig11:ECKeyValue xmlns:dsig11="` + NamespaceDSig11 + `">` + inner + `</dsig11:ECKeyValue>`
	d := mustParse(t, full)
	return d.DocumentElement()
}

// newKeyInfoCertElem builds a <KeyInfo><X509Data><X509Certificate>... subtree
// whose elements are placed in the namespace ns (declared under prefix px). The
// returned element is the KeyInfo root, ready to hand to parseKeyInfo.
func newKeyInfoCertElem(t *testing.T, doc *helium.Document, px, ns string, certDER []byte) *helium.Element {
	t.Helper()

	keyInfo, err := doc.CreateElement("KeyInfo")
	require.NoError(t, err)
	require.NoError(t, keyInfo.DeclareNamespace(px, ns))
	require.NoError(t, keyInfo.SetActiveNamespace(px, ns))

	x509Data, err := doc.CreateElement("X509Data")
	require.NoError(t, err)
	require.NoError(t, x509Data.SetActiveNamespace(px, ns))
	require.NoError(t, keyInfo.AddChild(x509Data))

	certElem, err := doc.CreateElement("X509Certificate")
	require.NoError(t, err)
	require.NoError(t, certElem.SetActiveNamespace(px, ns))
	require.NoError(t, certElem.AddChild(
		doc.CreateText([]byte(base64.StdEncoding.EncodeToString(certDER)))))
	require.NoError(t, x509Data.AddChild(certElem))

	return keyInfo
}

// selfSignedCertDER returns a throwaway self-signed certificate's DER bytes.
func selfSignedCertDER(t *testing.T) []byte {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "test"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	require.NoError(t, err)
	return der
}

// rsaKeyInfoXML builds a ds:KeyInfo document carrying an RSAKeyValue with the
// given base64-encoded modulus and exponent.
func rsaKeyInfoXML(modulus, exponent string) string {
	return fmt.Sprintf(`<ds:KeyInfo xmlns:ds="%s">`+
		`<ds:KeyValue><ds:RSAKeyValue>`+
		`<ds:Modulus>%s</ds:Modulus>`+
		`<ds:Exponent>%s</ds:Exponent>`+
		`</ds:RSAKeyValue></ds:KeyValue></ds:KeyInfo>`, NamespaceDSig, modulus, exponent)
}

func TestParseECKeyValueErrors(t *testing.T) {
	// unsupported curve covers the unsupported-curve branch.
	t.Run("unsupported curve", func(t *testing.T) {
		elem := ecElem(t, `<dsig11:NamedCurve xmlns:dsig11="`+NamespaceDSig11+`" URI="urn:oid:bogus"/>`)
		var data KeyInfoData
		err := parseECKeyValue(elem, &data)
		require.ErrorIs(t, err, ErrInvalidKeyInfo)
		require.Contains(t, err.Error(), "unsupported EC curve")
	})

	// missing curve covers the PublicKey-before-NamedCurve branch.
	t.Run("missing curve", func(t *testing.T) {
		elem := ecElem(t, `<dsig11:PublicKey xmlns:dsig11="`+NamespaceDSig11+`">BBBB</dsig11:PublicKey>`)
		var data KeyInfoData
		err := parseECKeyValue(elem, &data)
		require.ErrorIs(t, err, ErrInvalidKeyInfo)
		require.Contains(t, err.Error(), "missing NamedCurve")
	})

	// only NamedCurve (no PublicKey point) must be rejected as an incomplete
	// ECKeyValue rather than yielding a partial key with a nil point.
	t.Run("named curve without public key", func(t *testing.T) {
		elem := ecElem(t, `<dsig11:NamedCurve xmlns:dsig11="`+NamespaceDSig11+`" URI="urn:oid:1.2.840.10045.3.1.7"/>`)
		var data KeyInfoData
		err := parseECKeyValue(elem, &data)
		require.ErrorIs(t, err, ErrInvalidKeyInfo)
		require.Nil(t, data.ECKeyValue,
			"an ECKeyValue without a PublicKey point must not produce a partial key")
	})

	// invalid point covers the invalid-point branch: a valid curve but a
	// PublicKey blob that does not unmarshal to a point.
	t.Run("invalid point", func(t *testing.T) {
		inner := `<dsig11:NamedCurve xmlns:dsig11="` + NamespaceDSig11 + `" URI="urn:oid:1.2.840.10045.3.1.7"/>` +
			`<dsig11:PublicKey xmlns:dsig11="` + NamespaceDSig11 + `">AAAA</dsig11:PublicKey>`
		elem := ecElem(t, inner)
		var data KeyInfoData
		err := parseECKeyValue(elem, &data)
		require.ErrorIs(t, err, ErrInvalidKeyInfo)
		require.Contains(t, err.Error(), "invalid EC public key point")
	})

	// bad base64 covers the base64-decode error branch.
	t.Run("bad base64", func(t *testing.T) {
		inner := `<dsig11:NamedCurve xmlns:dsig11="` + NamespaceDSig11 + `" URI="urn:oid:1.2.840.10045.3.1.7"/>` +
			`<dsig11:PublicKey xmlns:dsig11="` + NamespaceDSig11 + `">!!!!notbase64</dsig11:PublicKey>`
		elem := ecElem(t, inner)
		var data KeyInfoData
		err := parseECKeyValue(elem, &data)
		require.ErrorIs(t, err, ErrInvalidKeyInfo)
	})
}

func TestParseRSAKeyValue(t *testing.T) {
	// exponent out of range covers the exponent-range guard.
	t.Run("exponent out of range", func(t *testing.T) {
		// Exponent of 0 (base64 "AA==" decodes to a single 0 byte -> Sign() == 0).
		inner := `<ds:Modulus xmlns:ds="` + NamespaceDSig + `">AQAB</ds:Modulus>` +
			`<ds:Exponent xmlns:ds="` + NamespaceDSig + `">AA==</ds:Exponent>`
		d := mustParse(t, `<ds:RSAKeyValue xmlns:ds="`+NamespaceDSig+`">`+inner+`</ds:RSAKeyValue>`)
		var data KeyInfoData
		err := parseRSAKeyValue(d.DocumentElement(), &data)
		require.ErrorIs(t, err, ErrInvalidKeyInfo)
		require.Contains(t, err.Error(), "out of range")
	})

	// bad base64 covers the base64 error branch.
	t.Run("bad base64", func(t *testing.T) {
		d := mustParse(t, `<ds:RSAKeyValue xmlns:ds="`+NamespaceDSig+`"><ds:Modulus xmlns:ds="`+NamespaceDSig+`">!!!</ds:Modulus></ds:RSAKeyValue>`)
		var data KeyInfoData
		err := parseRSAKeyValue(d.DocumentElement(), &data)
		require.ErrorIs(t, err, ErrInvalidKeyInfo)
	})

	// only Exponent (no Modulus) must be rejected as an incomplete RSAKeyValue
	// rather than yielding a partial key with a nil modulus.
	t.Run("exponent without modulus", func(t *testing.T) {
		d := mustParse(t, `<ds:RSAKeyValue xmlns:ds="`+NamespaceDSig+`"><ds:Exponent xmlns:ds="`+NamespaceDSig+`">AQAB</ds:Exponent></ds:RSAKeyValue>`)
		var data KeyInfoData
		err := parseRSAKeyValue(d.DocumentElement(), &data)
		require.ErrorIs(t, err, ErrInvalidKeyInfo)
		require.Nil(t, data.RSAKeyValue,
			"an RSAKeyValue without a Modulus must not produce a partial key")
	})

	// only Modulus (no Exponent) must likewise be rejected.
	t.Run("modulus without exponent", func(t *testing.T) {
		d := mustParse(t, `<ds:RSAKeyValue xmlns:ds="`+NamespaceDSig+`"><ds:Modulus xmlns:ds="`+NamespaceDSig+`">AQAB</ds:Modulus></ds:RSAKeyValue>`)
		var data KeyInfoData
		err := parseRSAKeyValue(d.DocumentElement(), &data)
		require.ErrorIs(t, err, ErrInvalidKeyInfo)
		require.Nil(t, data.RSAKeyValue,
			"an RSAKeyValue without an Exponent must not produce a partial key")
	})
}

func TestParseX509Data(t *testing.T) {
	// bad cert covers the invalid-base64 and invalid-cert branches.
	t.Run("bad cert", func(t *testing.T) {
		// well-formed base64 but not a certificate.
		d := mustParse(t, `<ds:X509Data xmlns:ds="`+NamespaceDSig+`"><ds:X509Certificate xmlns:ds="`+NamespaceDSig+`">aGVsbG8=</ds:X509Certificate></ds:X509Data>`)
		var data KeyInfoData
		err := parseX509Data(d.DocumentElement(), &data)
		require.ErrorIs(t, err, ErrInvalidKeyInfo)

		d2 := mustParse(t, `<ds:X509Data xmlns:ds="`+NamespaceDSig+`"><ds:X509Certificate xmlns:ds="`+NamespaceDSig+`">!!!bad</ds:X509Certificate></ds:X509Data>`)
		var data2 KeyInfoData
		err = parseX509Data(d2.DocumentElement(), &data2)
		require.ErrorIs(t, err, ErrInvalidKeyInfo)
		require.Contains(t, err.Error(), "base64")
	})

	// foreign cert covers parseX509Data's foreign-namespace look-alike continue
	// branch.
	t.Run("foreign cert", func(t *testing.T) {
		doc := mustParse(t, `<ds:X509Data xmlns:ds="`+NamespaceDSig+`"><evil:X509Certificate xmlns:evil="urn:evil">AA==</evil:X509Certificate></ds:X509Data>`)
		var data KeyInfoData
		require.NoError(t, parseX509Data(doc.DocumentElement(), &data))
		require.Empty(t, data.X509Certificates)
	})
}

// TestParseForeignChild covers the foreign-namespace continue branches across
// the KeyInfo parse helpers.
func TestParseForeignChild(t *testing.T) {
	// key value rsa covers parseKeyValue's continue branch: an RSAKeyValue
	// look-alike in a foreign namespace must be skipped, leaving no RSAKeyValue
	// parsed.
	t.Run("key value rsa", func(t *testing.T) {
		doc := mustParse(t, `<ds:KeyValue xmlns:ds="`+NamespaceDSig+`"><evil:RSAKeyValue xmlns:evil="urn:evil"/></ds:KeyValue>`)
		var data KeyInfoData
		require.NoError(t, parseKeyValue(doc.DocumentElement(), &data))
		require.Nil(t, data.RSAKeyValue)
	})

	// key value ec covers parseKeyValue's ECKeyValue continue branch: an
	// ECKeyValue look-alike in a non-dsig11 namespace must be skipped.
	t.Run("key value ec", func(t *testing.T) {
		doc := mustParse(t, `<ds:KeyValue xmlns:ds="`+NamespaceDSig+`"><evil:ECKeyValue xmlns:evil="urn:evil"/></ds:KeyValue>`)
		var data KeyInfoData
		require.NoError(t, parseKeyValue(doc.DocumentElement(), &data))
		require.Nil(t, data.ECKeyValue)
	})

	// rsa key value child covers parseRSAKeyValue's foreign-namespace child
	// continue branch: the foreign Modulus is skipped, leaving an incomplete
	// RSAKeyValue that must be rejected rather than emitted as a partial key.
	t.Run("rsa key value child", func(t *testing.T) {
		doc := mustParse(t, `<ds:RSAKeyValue xmlns:ds="`+NamespaceDSig+`"><evil:Modulus xmlns:evil="urn:evil">AQAB</evil:Modulus></ds:RSAKeyValue>`)
		var data KeyInfoData
		err := parseRSAKeyValue(doc.DocumentElement(), &data)
		require.ErrorIs(t, err, ErrInvalidKeyInfo)
		require.Nil(t, data.RSAKeyValue) // foreign Modulus was skipped -> incomplete
	})

	// key info child covers parseKeyInfo's continue branch for a foreign-namespace
	// X509Data look-alike.
	t.Run("key info child", func(t *testing.T) {
		doc := mustParse(t, `<ds:KeyInfo xmlns:ds="`+NamespaceDSig+`"><evil:X509Data xmlns:evil="urn:evil"/></ds:KeyInfo>`)
		data, err := parseKeyInfo(doc.DocumentElement())
		require.NoError(t, err)
		require.Empty(t, data.X509Certificates)
	})
}

// TestKeyInfoNamespace guards against namespace confusion in KeyInfo parsing.
func TestKeyInfoNamespace(t *testing.T) {
	// foreign namespace x509 ignored guards against namespace confusion in
	// KeyInfo parsing. The parser fed key material (certificates, key values) into
	// key resolution; matching those elements on local name alone would let a
	// foreign-namespace <evil:X509Data>/<evil:X509Certificate> masquerade as the
	// core ds:X509Data and supply an attacker-chosen verification certificate. A
	// foreign-namespace KeyInfo subtree must therefore yield no parsed key material.
	t.Run("foreign namespace x509 ignored", func(t *testing.T) {
		doc, err := helium.NewParser().Parse(context.Background(), []byte(`<root/>`))
		require.NoError(t, err)
		certDER := selfSignedCertDER(t)

		const evilNS = "urn:example:evil"
		keyInfo := newKeyInfoCertElem(t, doc, "evil", evilNS, certDER)
		require.Equal(t, evilNS, elementNamespaceURI(keyInfo))

		data, err := parseKeyInfo(keyInfo)
		require.NoError(t, err)
		require.Empty(t, data.X509Certificates,
			"a foreign-namespace X509Data look-alike must not supply a certificate")
	})

	// core namespace x509 parsed is the positive control: a correctly
	// ds-namespaced KeyInfo/X509Data/X509Certificate subtree must still parse.
	t.Run("core namespace x509 parsed", func(t *testing.T) {
		doc, err := helium.NewParser().Parse(context.Background(), []byte(`<root/>`))
		require.NoError(t, err)
		certDER := selfSignedCertDER(t)

		keyInfo := newKeyInfoCertElem(t, doc, nsPrefix, NamespaceDSig, certDER)
		require.Equal(t, NamespaceDSig, elementNamespaceURI(keyInfo))

		data, err := parseKeyInfo(keyInfo)
		require.NoError(t, err)
		require.Len(t, data.X509Certificates, 1,
			"a correctly ds-namespaced X509Certificate must still parse")
	})

	// foreign namespace rsa key value ignored ensures the same guard covers the
	// RSAKeyValue path: a foreign-namespace KeyValue/RSAKeyValue look-alike must
	// not supply key material.
	t.Run("foreign namespace rsa key value ignored", func(t *testing.T) {
		doc, err := helium.NewParser().Parse(context.Background(), []byte(`<root/>`))
		require.NoError(t, err)

		const evilNS = "urn:example:evil"
		keyInfo, err := doc.CreateElement("KeyInfo")
		require.NoError(t, err)
		require.NoError(t, keyInfo.DeclareNamespace("evil", evilNS))
		require.NoError(t, keyInfo.SetActiveNamespace("evil", evilNS))

		keyValue, err := doc.CreateElement("KeyValue")
		require.NoError(t, err)
		require.NoError(t, keyValue.SetActiveNamespace("evil", evilNS))
		require.NoError(t, keyInfo.AddChild(keyValue))

		rsaKV, err := doc.CreateElement("RSAKeyValue")
		require.NoError(t, err)
		require.NoError(t, rsaKV.SetActiveNamespace("evil", evilNS))
		require.NoError(t, keyValue.AddChild(rsaKV))

		mod, err := doc.CreateElement("Modulus")
		require.NoError(t, err)
		require.NoError(t, mod.SetActiveNamespace("evil", evilNS))
		require.NoError(t, mod.AddChild(doc.CreateText([]byte(base64.StdEncoding.EncodeToString([]byte{1, 2, 3})))))
		require.NoError(t, rsaKV.AddChild(mod))

		data, err := parseKeyInfo(keyInfo)
		require.NoError(t, err)
		require.Nil(t, data.RSAKeyValue,
			"a foreign-namespace RSAKeyValue look-alike must not supply key material")
	})
}

func TestParseKeyInfoRSAExponentRange(t *testing.T) {
	modulus := base64.StdEncoding.EncodeToString(big.NewInt(3233).Bytes())

	encode := func(n *big.Int) string {
		return base64.StdEncoding.EncodeToString(n.Bytes())
	}

	t.Run("valid 65537", func(t *testing.T) {
		xml := rsaKeyInfoXML(modulus, "AQAB") // 65537
		doc, err := helium.NewParser().Parse(t.Context(), []byte(xml))
		require.NoError(t, err)

		data, err := parseKeyInfo(doc.DocumentElement())
		require.NoError(t, err)
		require.NotNil(t, data.RSAKeyValue)
		require.Equal(t, 65537, data.RSAKeyValue.Exponent)
	})

	oversized := map[string]*big.Int{
		"2^63": new(big.Int).Lsh(big.NewInt(1), 63),
	}
	// 2^64 + 65537
	bigVal := new(big.Int).Lsh(big.NewInt(1), 64)
	bigVal.Add(bigVal, big.NewInt(65537))
	oversized["2^64+65537"] = bigVal

	for name, val := range oversized {
		t.Run("oversized "+name, func(t *testing.T) {
			xml := rsaKeyInfoXML(modulus, encode(val))
			doc, err := helium.NewParser().Parse(t.Context(), []byte(xml))
			require.NoError(t, err)

			_, err = parseKeyInfo(doc.DocumentElement())
			require.Error(t, err)
			require.ErrorIs(t, err, ErrInvalidKeyInfo)
		})
	}

	t.Run("zero exponent rejected", func(t *testing.T) {
		xml := rsaKeyInfoXML(modulus, encode(big.NewInt(0)))
		doc, err := helium.NewParser().Parse(t.Context(), []byte(xml))
		require.NoError(t, err)

		_, err = parseKeyInfo(doc.DocumentElement())
		require.Error(t, err)
		require.ErrorIs(t, err, ErrInvalidKeyInfo)
	})
}

// TestKeySourceFuncNil covers the typed-nil guard in KeySourceFunc.ResolveKey.
func TestKeySourceFuncNil(t *testing.T) {
	var f KeySourceFunc
	_, err := f.ResolveKey(t.Context(), nil, "")
	require.ErrorIs(t, err, ErrNoKeySource)
}
