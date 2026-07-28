package xmlenc1_test

import (
	"encoding/base64"
	"runtime"
	"strings"
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

// TestEncryptionMethodCardinality verifies that parseEncryptionMethod is
// fail-closed: an empty Algorithm and duplicate DigestMethod/MGF/OAEPparams
// children are rejected as ErrMalformedEncrypted, while a well-formed single
// occurrence still parses.
func TestEncryptionMethodCardinality(t *testing.T) {
	const head = `<xenc:EncryptedData xmlns:xenc="http://www.w3.org/2001/04/xmlenc#" xmlns:ds="http://www.w3.org/2000/09/xmldsig#" xmlns:xenc11="http://www.w3.org/2009/xmlenc11#">`
	const cipher = `<xenc:CipherData><xenc:CipherValue>AAAA</xenc:CipherValue></xenc:CipherData></xenc:EncryptedData>`

	parse := func(t *testing.T, em string) (*xmlenc1.EncryptedData, error) {
		t.Helper()
		doc := mustParseXML(t, head+em+cipher)
		elem, ok := helium.AsNode[*helium.Element](doc.DocumentElement())
		require.True(t, ok)
		return xmlenc1.ParseEncryptedDataForTest(elem)
	}

	t.Run("empty Algorithm rejected", func(t *testing.T) {
		_, err := parse(t, `<xenc:EncryptionMethod Algorithm=""/>`)
		require.ErrorIs(t, err, xmlenc1.ErrMalformedEncrypted)
	})

	t.Run("duplicate DigestMethod rejected", func(t *testing.T) {
		_, err := parse(t, `<xenc:EncryptionMethod Algorithm="`+xmlenc1.RSAOAEP11+`">`+
			`<ds:DigestMethod Algorithm="`+xmlenc1.DigestSHA256+`"/>`+
			`<ds:DigestMethod Algorithm="`+xmlenc1.DigestSHA1+`"/>`+
			`</xenc:EncryptionMethod>`)
		require.ErrorIs(t, err, xmlenc1.ErrMalformedEncrypted)
	})

	t.Run("duplicate MGF rejected", func(t *testing.T) {
		_, err := parse(t, `<xenc:EncryptionMethod Algorithm="`+xmlenc1.RSAOAEP11+`">`+
			`<xenc11:MGF Algorithm="`+xmlenc1.MGFSHA256+`"/>`+
			`<xenc11:MGF Algorithm="`+xmlenc1.MGFSHA1+`"/>`+
			`</xenc:EncryptionMethod>`)
		require.ErrorIs(t, err, xmlenc1.ErrMalformedEncrypted)
	})

	t.Run("DigestMethod missing Algorithm rejected", func(t *testing.T) {
		_, err := parse(t, `<xenc:EncryptionMethod Algorithm="`+xmlenc1.RSAOAEP11+`">`+
			`<ds:DigestMethod/>`+
			`</xenc:EncryptionMethod>`)
		require.ErrorIs(t, err, xmlenc1.ErrMalformedEncrypted)
	})

	t.Run("MGF empty Algorithm rejected", func(t *testing.T) {
		_, err := parse(t, `<xenc:EncryptionMethod Algorithm="`+xmlenc1.RSAOAEP11+`">`+
			`<xenc11:MGF Algorithm=""/>`+
			`</xenc:EncryptionMethod>`)
		require.ErrorIs(t, err, xmlenc1.ErrMalformedEncrypted)
	})

	t.Run("duplicate KeySize rejected", func(t *testing.T) {
		_, err := parse(t, `<xenc:EncryptionMethod Algorithm="`+xmlenc1.AES256GCM+`">`+
			`<xenc:KeySize>256</xenc:KeySize>`+
			`<xenc:KeySize>256</xenc:KeySize>`+
			`</xenc:EncryptionMethod>`)
		require.ErrorIs(t, err, xmlenc1.ErrMalformedEncrypted)
	})

	t.Run("duplicate OAEPparams rejected", func(t *testing.T) {
		_, err := parse(t, `<xenc:EncryptionMethod Algorithm="`+xmlenc1.RSAOAEP11+`">`+
			`<xenc:OAEPparams>AAAA</xenc:OAEPparams>`+
			`<xenc:OAEPparams>BBBB</xenc:OAEPparams>`+
			`</xenc:EncryptionMethod>`)
		require.ErrorIs(t, err, xmlenc1.ErrMalformedEncrypted)
	})

	t.Run("well-formed single still parses", func(t *testing.T) {
		ed, err := parse(t, `<xenc:EncryptionMethod Algorithm="`+xmlenc1.RSAOAEP11+`">`+
			`<ds:DigestMethod Algorithm="`+xmlenc1.DigestSHA256+`"/>`+
			`<xenc11:MGF Algorithm="`+xmlenc1.MGFSHA256+`"/>`+
			`<xenc:OAEPparams>AAAA</xenc:OAEPparams>`+
			`</xenc:EncryptionMethod>`)
		require.NoError(t, err)
		require.NotNil(t, ed.EncryptionMethod)
		require.Equal(t, xmlenc1.RSAOAEP11, ed.EncryptionMethod.Algorithm)
		require.Equal(t, xmlenc1.DigestSHA256, ed.EncryptionMethod.DigestMethod)
		require.Equal(t, xmlenc1.MGFSHA256, ed.EncryptionMethod.MGFAlgorithm)
	})
}

// oaepParamsOverLimit is the fragment the OAEPparams size limit puts in its
// error. Every parse failure in this package wraps ErrMalformedEncrypted, so
// the sentinel alone cannot tell a value refused by the limit from one the
// decoder or a later check refused.
const oaepParamsOverLimit = "OAEPparams is over the"

// oaepParamsEncryptedData builds an EncryptedData whose own EncryptionMethod
// carries params as raw OAEPparams markup — text, CDATA sections, or any mix.
// Writing the markup straight into the document is the only way to put a
// chosen lexical form in front of the size limit.
func oaepParamsEncryptedData(t *testing.T, params string) *helium.Element {
	t.Helper()
	doc := mustParseXML(t, `<xenc:EncryptedData xmlns:xenc="`+xmlenc1.NamespaceXMLEnc+`">`+
		`<xenc:EncryptionMethod Algorithm="`+xmlenc1.RSAOAEP11+`">`+
		`<xenc:OAEPparams>`+params+`</xenc:OAEPparams>`+
		`</xenc:EncryptionMethod>`+
		`<xenc:CipherData><xenc:CipherValue>AAAA</xenc:CipherValue></xenc:CipherData>`+
		`</xenc:EncryptedData>`)
	return doc.DocumentElement()
}

// oaepParamsKeyEncryptedData is oaepParamsEncryptedData for the other call
// site: the EncryptionMethod of an EncryptedKey carried in KeyInfo.
func oaepParamsKeyEncryptedData(t *testing.T, params string) *helium.Element {
	t.Helper()
	doc := mustParseXML(t, `<xenc:EncryptedData xmlns:xenc="`+xmlenc1.NamespaceXMLEnc+`" xmlns:ds="`+xmlenc1.NamespaceDSig+`">`+
		`<xenc:EncryptionMethod Algorithm="`+xmlenc1.AES256GCM+`"/>`+
		`<ds:KeyInfo><xenc:EncryptedKey>`+
		`<xenc:EncryptionMethod Algorithm="`+xmlenc1.RSAOAEP11+`">`+
		`<xenc:OAEPparams>`+params+`</xenc:OAEPparams>`+
		`</xenc:EncryptionMethod>`+
		`<xenc:CipherData><xenc:CipherValue>AAAA</xenc:CipherValue></xenc:CipherData>`+
		`</xenc:EncryptedKey></ds:KeyInfo>`+
		`<xenc:CipherData><xenc:CipherValue>AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA</xenc:CipherValue></xenc:CipherData>`+
		`</xenc:EncryptedData>`)
	return doc.DocumentElement()
}

// TestOAEPParamsBound covers the decoded-size limit on xenc:OAEPparams. The
// element is read while the document is parsed — once for the EncryptedData's
// own EncryptionMethod and once for each EncryptedKey's — so it is reached
// before any key is resolved and before anything the document says has been
// authenticated. On the EncryptedData side nothing ever reads the decoded
// bytes back, so without the limit the whole cost is pure amplification.
func TestOAEPParamsBound(t *testing.T) {
	atLimit := base64.StdEncoding.EncodeToString(make([]byte, xmlenc1.MaxOAEPParamsBytesForTest))
	overLimit := base64.StdEncoding.EncodeToString(make([]byte, xmlenc1.MaxOAEPParamsBytesForTest+1))

	t.Run("a value at the limit parses", func(t *testing.T) {
		ed, err := xmlenc1.ParseEncryptedDataForTest(oaepParamsEncryptedData(t, atLimit))
		require.NoError(t, err)
		require.Len(t, ed.EncryptionMethod.OAEPParams, xmlenc1.MaxOAEPParamsBytesForTest)
	})

	t.Run("one byte over the limit is rejected", func(t *testing.T) {
		_, err := xmlenc1.ParseEncryptedDataForTest(oaepParamsEncryptedData(t, overLimit))
		require.ErrorIs(t, err, xmlenc1.ErrMalformedEncrypted)
		require.Contains(t, err.Error(), oaepParamsOverLimit)
	})

	// Splitting the value into CDATA sections is what defeats the XML parser's
	// per-node content cap, which bounds one indivisible run of characters and
	// not their concatenation. The limit is charged on the whole value, so the
	// split buys nothing.
	t.Run("an over-limit value split across CDATA sections is rejected", func(t *testing.T) {
		_, err := xmlenc1.ParseEncryptedDataForTest(oaepParamsEncryptedData(t, splitIntoNodes(overLimit, 64)))
		require.ErrorIs(t, err, xmlenc1.ErrMalformedEncrypted)
		require.Contains(t, err.Error(), oaepParamsOverLimit)
	})

	// Sub-quantum children are the sharper form of the same trick: counting
	// each child alone would report zero for every three-character piece.
	t.Run("an over-limit value split into sub-quantum children is rejected", func(t *testing.T) {
		_, err := xmlenc1.ParseEncryptedDataForTest(oaepParamsEncryptedData(t, splitIntoNodes(overLimit, 3)))
		require.ErrorIs(t, err, xmlenc1.ErrMalformedEncrypted)
		require.Contains(t, err.Error(), oaepParamsOverLimit)
	})

	// The limit measures decoded octets, so the XML whitespace xs:base64Binary
	// permits between characters neither counts against a legal value nor
	// lets an oversized one through.
	t.Run("whitespace does not count against the limit", func(t *testing.T) {
		ed, err := xmlenc1.ParseEncryptedDataForTest(oaepParamsEncryptedData(t, splitIntoNodes(atLimit, 4)))
		require.NoError(t, err)
		require.Len(t, ed.EncryptionMethod.OAEPParams, xmlenc1.MaxOAEPParamsBytesForTest)
	})

	// An EncryptedKey's EncryptionMethod is the other call site, and it is
	// reached once per candidate, so the same limit has to hold there.
	t.Run("an EncryptedKey OAEPparams is bounded too", func(t *testing.T) {
		_, err := xmlenc1.ParseEncryptedDataForTest(oaepParamsKeyEncryptedData(t, splitIntoNodes(overLimit, 64)))
		require.ErrorIs(t, err, xmlenc1.ErrMalformedEncrypted)
		require.Contains(t, err.Error(), oaepParamsOverLimit)

		ed, err := xmlenc1.ParseEncryptedDataForTest(oaepParamsKeyEncryptedData(t, atLimit))
		require.NoError(t, err)
		require.Len(t, ed.EncryptedKeys, 1)
		require.Len(t, ed.EncryptedKeys[0].EncryptionMethod.OAEPParams, xmlenc1.MaxOAEPParamsBytesForTest)
	})

	// A real label is a handful of octets, and it must survive the round trip
	// byte for byte: it is hashed into the RSA-OAEP encoding, so a label that
	// changed shape would make every decrypt fail.
	t.Run("a real label round-trips unchanged", func(t *testing.T) {
		label := []byte("label-params")
		ed, err := xmlenc1.ParseEncryptedDataForTest(oaepParamsEncryptedData(t, base64.StdEncoding.EncodeToString(label)))
		require.NoError(t, err)
		require.Equal(t, label, ed.EncryptionMethod.OAEPParams)

		// And through the split shapes a document may choose freely.
		for _, chunk := range []int{1, 2, 3, 5} {
			ed, err := xmlenc1.ParseEncryptedDataForTest(oaepParamsEncryptedData(t, splitIntoNodes(base64.StdEncoding.EncodeToString(label), chunk)))
			require.NoError(t, err)
			require.Equal(t, label, ed.EncryptionMethod.OAEPParams)
		}
	})
}

// TestOAEPParamsAllocation pins what an xenc:OAEPparams element may ALLOCATE,
// which the error assertions above cannot see. The limit governs decoded
// octets, but the lexical text an attacker wraps around them is unbounded:
// xs:base64Binary permits XML whitespace between characters, and the value may
// be spread over as many text and CDATA children as the document likes.
// Joining that text into one string before the limit is applied makes the
// limit an accounting formality — the memory is allocated by the time the
// error is returned — and for a value the limit ACCEPTS it is never refused at
// all, so a test that only checks for the rejection would miss half of it.
//
// Each case reads the process-wide TotalAlloc delta across Decrypt, so these
// subtests must NOT run in parallel: a concurrent test's allocations would
// pollute the delta.
func TestOAEPParamsAllocation(t *testing.T) {
	// no t.Parallel(): isolated so each delta reflects only its own Decrypt.

	// lexical is the text the attacker writes, and every bound below is a
	// multiple of it, because the defect being pinned is exactly a cost that
	// follows the lexical length.
	// cdataChunk cuts that text into sixteen CDATA sections.
	const (
		lexical    = 4 << 20
		cdataChunk = lexical / 16
	)

	// Reading each child's content is the floor a bound has to clear: a DOM
	// hands out a copy per node, and no value can be weighed without looking at
	// it. A rejected value is looked at once, and an accepted one twice — once
	// to weigh it and once to build the characters the limit approved.
	const (
		countOnly     = lexical * 3 / 2
		countAndBuild = lexical * 5 / 2
	)

	// Only space and tab are used as whitespace: an XML parser folds CRLF to
	// LF, which would make the text the DOM holds shorter than the text written
	// here and every multiple above harder to read.
	padding := strings.Repeat(" \t", lexical/2)
	// oversized decodes to 3 MiB, far over the limit however it is laid out.
	oversized := strings.Repeat("A", lexical)

	for _, tc := range []struct {
		name     string
		params   string
		rejected bool
		maxAlloc uint64
	}{
		{
			name:     "over-limit value in one text node",
			params:   oversized,
			rejected: true,
			maxAlloc: countOnly,
		},
		{
			// CDATA is how the value evades the parser's per-node content cap:
			// the cap bounds one indivisible run, and every section is its own.
			name:     "over-limit value split across CDATA sections",
			params:   splitIntoNodes(oversized, cdataChunk),
			rejected: true,
			maxAlloc: countOnly,
		},
		{
			// Nothing here is ever refused: one quantum of label is far under
			// the limit, and the whitespace after it is not counted at all.
			// Allocating for it would be unbounded amplification with no error
			// anywhere to notice it, which a rejection-only test would miss.
			name:     "under-limit value with trailing whitespace",
			params:   "AA==" + padding,
			rejected: false,
			maxAlloc: countAndBuild,
		},
		{
			// The same accepted value spread over CDATA sections, where joining
			// the children costs the most.
			name:     "under-limit value with whitespace split across CDATA sections",
			params:   "AA==" + splitIntoNodes(padding, cdataChunk),
			rejected: false,
			maxAlloc: countAndBuild,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			elem := oaepParamsEncryptedData(t, tc.params)
			decryptor := xmlenc1.NewDecryptor().PrivateKey(generateRSAKey(t))

			var before, after runtime.MemStats
			runtime.GC()
			runtime.ReadMemStats(&before)
			_, err := decryptor.Decrypt(t.Context(), elem)
			runtime.ReadMemStats(&after)

			require.Error(t, err)
			require.Equal(t, tc.rejected, strings.Contains(err.Error(), oaepParamsOverLimit), "err=%v", err)

			allocated := after.TotalAlloc - before.TotalAlloc
			require.Less(t, allocated, tc.maxAlloc, "decrypting %d lexical bytes allocated %d bytes", len(tc.params), allocated)
		})
	}
}
