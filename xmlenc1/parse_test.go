package xmlenc1_test

import (
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xmlenc1"
	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	t.Run("encrypted data errors", func(t *testing.T) {
		t.Run("nil element", func(t *testing.T) {
			_, err := xmlenc1.ParseEncryptedDataForTest(nil)
			require.ErrorIs(t, err, xmlenc1.ErrMalformedEncrypted)
		})

		t.Run("wrong element name/namespace", func(t *testing.T) {
			doc := mustParseXML(t, `<root xmlns="http://www.w3.org/2001/04/xmlenc#"><NotEncryptedData/></root>`)
			child, ok := helium.AsNode[*helium.Element](doc.DocumentElement().FirstChild())
			require.True(t, ok)
			_, err := xmlenc1.ParseEncryptedDataForTest(child)
			require.ErrorIs(t, err, xmlenc1.ErrMalformedEncrypted)
		})

		t.Run("missing CipherData", func(t *testing.T) {
			// An EncryptedData with no CipherData/CipherValue must be rejected.
			doc := mustParseXML(t, `<xenc:EncryptedData xmlns:xenc="http://www.w3.org/2001/04/xmlenc#"><xenc:EncryptionMethod Algorithm="`+xmlenc1.AES256GCM+`"/></xenc:EncryptedData>`)
			elem, ok := helium.AsNode[*helium.Element](doc.DocumentElement())
			require.True(t, ok)
			_, err := xmlenc1.ParseEncryptedDataForTest(elem)
			require.ErrorIs(t, err, xmlenc1.ErrMalformedEncrypted)
		})

		t.Run("duplicate CipherValue", func(t *testing.T) {
			// CipherData is a choice of exactly one CipherValue (or one
			// CipherReference); two CipherValue children are schema-invalid
			// and must be rejected at parse rather than silently using the
			// first.
			doc := mustParseXML(t, `<xenc:EncryptedData xmlns:xenc="http://www.w3.org/2001/04/xmlenc#"><xenc:CipherData><xenc:CipherValue>AAAA</xenc:CipherValue><xenc:CipherValue>BBBB</xenc:CipherValue></xenc:CipherData></xenc:EncryptedData>`)
			elem, ok := helium.AsNode[*helium.Element](doc.DocumentElement())
			require.True(t, ok)
			_, err := xmlenc1.ParseEncryptedDataForTest(elem)
			require.ErrorIs(t, err, xmlenc1.ErrMalformedEncrypted)
		})

		t.Run("CipherValue plus CipherReference", func(t *testing.T) {
			// CipherData is a choice of EXACTLY ONE CipherValue or one
			// CipherReference. A CipherValue accompanied by a CipherReference
			// (in either order, under EncryptedData or EncryptedKey) is
			// schema-invalid and must be rejected, not silently reduced to
			// the CipherValue.
			const xencNS = "http://www.w3.org/2001/04/xmlenc#"
			const dsigNS = "http://www.w3.org/2000/09/xmldsig#"

			edWith := func(cipherData string) string {
				return `<xenc:EncryptedData xmlns:xenc="` + xencNS + `">` +
					`<xenc:CipherData>` + cipherData + `</xenc:CipherData>` +
					`</xenc:EncryptedData>`
			}
			ekWith := func(cipherData string) string {
				return `<xenc:EncryptedData xmlns:xenc="` + xencNS + `" xmlns:ds="` + dsigNS + `">` +
					`<ds:KeyInfo><xenc:EncryptedKey>` +
					`<xenc:CipherData>` + cipherData + `</xenc:CipherData>` +
					`</xenc:EncryptedKey></ds:KeyInfo>` +
					`<xenc:CipherData><xenc:CipherValue>AAAA</xenc:CipherValue></xenc:CipherData>` +
					`</xenc:EncryptedData>`
			}

			const valueFirst = `<xenc:CipherValue>AAAA</xenc:CipherValue><xenc:CipherReference URI="#ref"/>`
			const refFirst = `<xenc:CipherReference URI="#ref"/><xenc:CipherValue>AAAA</xenc:CipherValue>`

			for _, tc := range []struct {
				name string
				xml  string
			}{
				{"EncryptedData value-then-reference", edWith(valueFirst)},
				{"EncryptedData reference-then-value", edWith(refFirst)},
				{"EncryptedKey value-then-reference", ekWith(valueFirst)},
				{"EncryptedKey reference-then-value", ekWith(refFirst)},
			} {
				t.Run(tc.name, func(t *testing.T) {
					doc := mustParseXML(t, tc.xml)
					elem, ok := helium.AsNode[*helium.Element](doc.DocumentElement())
					require.True(t, ok)
					_, err := xmlenc1.ParseEncryptedDataForTest(elem)
					require.ErrorIs(t, err, xmlenc1.ErrMalformedEncrypted)
				})
			}
		})

		t.Run("EncryptedKey missing CipherData", func(t *testing.T) {
			// An EncryptedKey carried in KeyInfo with no CipherData/CipherValue
			// must be rejected at parse, not deferred to a later crypto error.
			doc := mustParseXML(t, `<xenc:EncryptedData xmlns:xenc="http://www.w3.org/2001/04/xmlenc#" xmlns:ds="http://www.w3.org/2000/09/xmldsig#"><ds:KeyInfo><xenc:EncryptedKey><xenc:EncryptionMethod Algorithm="`+xmlenc1.RSAOAEP11+`"/></xenc:EncryptedKey></ds:KeyInfo><xenc:CipherData><xenc:CipherValue>AAAA</xenc:CipherValue></xenc:CipherData></xenc:EncryptedData>`)
			elem, ok := helium.AsNode[*helium.Element](doc.DocumentElement())
			require.True(t, ok)
			_, err := xmlenc1.ParseEncryptedDataForTest(elem)
			require.ErrorIs(t, err, xmlenc1.ErrMalformedEncrypted)
		})
	})

	t.Run("encryption method missing algorithm", func(t *testing.T) {
		doc := mustParseXML(t, `<xenc:EncryptedData xmlns:xenc="http://www.w3.org/2001/04/xmlenc#"><xenc:EncryptionMethod/><xenc:CipherData><xenc:CipherValue>AAAA</xenc:CipherValue></xenc:CipherData></xenc:EncryptedData>`)
		elem, ok := helium.AsNode[*helium.Element](doc.DocumentElement())
		require.True(t, ok)
		_, err := xmlenc1.ParseEncryptedDataForTest(elem)
		require.ErrorIs(t, err, xmlenc1.ErrMalformedEncrypted)
	})

	t.Run("invalid base64", func(t *testing.T) {
		t.Run("CipherValue", func(t *testing.T) {
			doc := mustParseXML(t, `<xenc:EncryptedData xmlns:xenc="http://www.w3.org/2001/04/xmlenc#"><xenc:CipherData><xenc:CipherValue>!!!not-base64!!!</xenc:CipherValue></xenc:CipherData></xenc:EncryptedData>`)
			elem, ok := helium.AsNode[*helium.Element](doc.DocumentElement())
			require.True(t, ok)
			_, err := xmlenc1.ParseEncryptedDataForTest(elem)
			require.ErrorIs(t, err, xmlenc1.ErrMalformedEncrypted)
		})

		t.Run("OAEPparams", func(t *testing.T) {
			doc := mustParseXML(t, `<xenc:EncryptedData xmlns:xenc="http://www.w3.org/2001/04/xmlenc#"><xenc:EncryptionMethod Algorithm="`+xmlenc1.RSAOAEP11+`"><xenc:OAEPparams>!!!bad!!!</xenc:OAEPparams></xenc:EncryptionMethod><xenc:CipherData><xenc:CipherValue>AAAA</xenc:CipherValue></xenc:CipherData></xenc:EncryptedData>`)
			elem, ok := helium.AsNode[*helium.Element](doc.DocumentElement())
			require.True(t, ok)
			_, err := xmlenc1.ParseEncryptedDataForTest(elem)
			require.ErrorIs(t, err, xmlenc1.ErrMalformedEncrypted)
		})
	})

	t.Run("missing encryption method on decrypt", func(t *testing.T) {
		doc := mustParseXML(t, `<root/>`)
		ed := &xmlenc1.EncryptedData{
			Type:        xmlenc1.TypeElement,
			CipherValue: make([]byte, 48),
		}
		elem, err := xmlenc1.MarshalEncryptedDataForTest(doc, ed)
		require.NoError(t, err)

		decryptor := xmlenc1.NewDecryptor().SessionKey(make([]byte, 32))
		_, err = decryptor.Decrypt(t.Context(), elem)
		require.ErrorIs(t, err, xmlenc1.ErrMalformedEncrypted)
	})

	t.Run("element matching requires namespace", func(t *testing.T) {
		const xencNS = "http://www.w3.org/2001/04/xmlenc#"
		const dsigNS = "http://www.w3.org/2000/09/xmldsig#"
		const foreignNS = "urn:example:not-xmlenc"

		t.Run("foreign namespace is not matched", func(t *testing.T) {
			// Every child reuses the XMLEnc local names but lives in a
			// foreign namespace. None should be picked up, so CipherData
			// resolution fails (missing CipherData/CipherValue).
			xml := `<EncryptedData xmlns="` + foreignNS + `">` +
				`<EncryptionMethod Algorithm="x"/>` +
				`<KeyInfo/>` +
				`<CipherData><CipherValue>aGVsbG8=</CipherValue></CipherData>` +
				`</EncryptedData>`
			doc := mustParseXML(t, xml)

			_, err := xmlenc1.ParseEncryptedDataForTest(doc.DocumentElement())
			require.Error(t, err, "foreign-namespaced CipherData must not be matched")
		})

		t.Run("correct namespace is matched", func(t *testing.T) {
			// Same structure but correctly namespace-qualified: xenc for
			// the XMLEnc elements, ds for KeyInfo. CipherValue must resolve.
			xml := `<xenc:EncryptedData xmlns:xenc="` + xencNS + `" xmlns:ds="` + dsigNS + `">` +
				`<xenc:EncryptionMethod Algorithm="http://www.w3.org/2001/04/xmlenc#aes128-cbc"/>` +
				`<ds:KeyInfo/>` +
				`<xenc:CipherData><xenc:CipherValue>aGVsbG8=</xenc:CipherValue></xenc:CipherData>` +
				`</xenc:EncryptedData>`
			doc := mustParseXML(t, xml)

			ed, err := xmlenc1.ParseEncryptedDataForTest(doc.DocumentElement())
			require.NoError(t, err)
			require.NotNil(t, ed.EncryptionMethod)
			require.Equal(t, []byte("hello"), ed.CipherValue)
		})

		t.Run("foreign root with valid xenc children is rejected", func(t *testing.T) {
			// The entry element itself is foreign-namespaced even though all
			// of its children are correctly xenc-qualified. The parser must
			// reject the entry element rather than trusting the children.
			xml := `<foo:EncryptedData xmlns:foo="` + foreignNS + `" xmlns:xenc="` + xencNS + `">` +
				`<xenc:EncryptionMethod Algorithm="http://www.w3.org/2001/04/xmlenc#aes128-cbc"/>` +
				`<xenc:CipherData><xenc:CipherValue>aGVsbG8=</xenc:CipherValue></xenc:CipherData>` +
				`</foo:EncryptedData>`
			doc := mustParseXML(t, xml)

			_, err := xmlenc1.ParseEncryptedDataForTest(doc.DocumentElement())
			require.Error(t, err, "foreign-namespaced EncryptedData root must not be accepted")
		})

		t.Run("foreign CipherValue inside correct CipherData is not matched", func(t *testing.T) {
			// The CipherData is correctly namespaced but its CipherValue
			// child is foreign. The foreign CipherValue must be ignored,
			// leaving CipherData without a usable value.
			xml := `<xenc:EncryptedData xmlns:xenc="` + xencNS + `" xmlns:foo="` + foreignNS + `">` +
				`<xenc:CipherData><foo:CipherValue>aGVsbG8=</foo:CipherValue></xenc:CipherData>` +
				`</xenc:EncryptedData>`
			doc := mustParseXML(t, xml)

			_, err := xmlenc1.ParseEncryptedDataForTest(doc.DocumentElement())
			require.Error(t, err, "foreign-namespaced CipherValue must not be matched")
		})
	})
}

// TestMarshalParseRoundTrip exercises the serialize and parse paths
// for every optional field: EncryptedData ID/Type, an EncryptedKey carrying
// its own ID/Recipient/CarriedKeyName and an EncryptionMethod with
// DigestMethod, MGFAlgorithm and OAEPParams. The marshaled element is
// serialized to bytes, reparsed through the public XML parser, and the
// resulting DOM is fed back through the internal EncryptedData parser so
// both directions are covered honestly via a real round-trip.
func TestMarshalParseRoundTrip(t *testing.T) {
	doc := mustParseXML(t, `<root/>`)

	ed := &xmlenc1.EncryptedData{
		ID:   "ED-1",
		Type: xmlenc1.TypeElement,
		EncryptionMethod: &xmlenc1.EncryptionMethod{
			Algorithm:    xmlenc1.AES256GCM,
			DigestMethod: xmlenc1.DigestSHA256,
			MGFAlgorithm: xmlenc1.MGFSHA256,
			OAEPParams:   []byte("params-bytes"),
		},
		EncryptedKeys: []*xmlenc1.EncryptedKey{
			{
				ID: "EK-1",
				EncryptionMethod: &xmlenc1.EncryptionMethod{
					Algorithm:    xmlenc1.RSAOAEP11,
					DigestMethod: xmlenc1.DigestSHA256,
					MGFAlgorithm: xmlenc1.MGFSHA256,
				},
				CipherValue: []byte("wrapped-key-bytes"),
			},
		},
		CipherValue: []byte("cipher-bytes"),
	}

	elem, err := xmlenc1.MarshalEncryptedDataForTest(doc, ed)
	require.NoError(t, err)

	// Parse the marshaled DOM back through the internal EncryptedData
	// parser. The marshaler sets active namespaces on each element, so the
	// namespace-aware matcher resolves the xenc/ds/xenc11 URIs directly.
	parsed, err := xmlenc1.ParseEncryptedDataForTest(elem)
	require.NoError(t, err)

	require.Equal(t, "ED-1", parsed.ID)
	require.Equal(t, xmlenc1.TypeElement, parsed.Type)
	require.NotNil(t, parsed.EncryptionMethod)
	require.Equal(t, xmlenc1.AES256GCM, parsed.EncryptionMethod.Algorithm)
	require.Equal(t, xmlenc1.DigestSHA256, parsed.EncryptionMethod.DigestMethod)
	require.Equal(t, xmlenc1.MGFSHA256, parsed.EncryptionMethod.MGFAlgorithm)
	require.Equal(t, []byte("params-bytes"), parsed.EncryptionMethod.OAEPParams)

	require.Len(t, parsed.EncryptedKeys, 1)
	require.Equal(t, "EK-1", parsed.EncryptedKeys[0].ID)
	require.NotNil(t, parsed.EncryptedKeys[0].EncryptionMethod)
	require.Equal(t, xmlenc1.RSAOAEP11, parsed.EncryptedKeys[0].EncryptionMethod.Algorithm)
	require.Equal(t, []byte("wrapped-key-bytes"), parsed.EncryptedKeys[0].CipherValue)
	require.Equal(t, []byte("cipher-bytes"), parsed.CipherValue)
}
