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

	t.Run("CipherValue character data", func(t *testing.T) {
		parse := func(t *testing.T, value string) (*xmlenc1.EncryptedData, error) {
			t.Helper()
			doc := mustParseXML(t, `<xenc:EncryptedData xmlns:xenc="http://www.w3.org/2001/04/xmlenc#"><xenc:CipherData><xenc:CipherValue>`+value+`</xenc:CipherValue></xenc:CipherData></xenc:EncryptedData>`)
			return xmlenc1.ParseEncryptedDataForTest(doc.DocumentElement())
		}

		t.Run("comments and processing instructions are ignored", func(t *testing.T) {
			for _, value := range []string{
				`AA<!-- a comment -->==`,
				`AA<?target data?>==`,
				`<!--c-->AA<?t?>==<!--c-->`,
			} {
				ed, err := parse(t, value)
				require.NoError(t, err, "value=%s", value)
				require.Equal(t, []byte{0x00}, ed.CipherValue, "value=%s", value)
			}
		})

		t.Run("element children are rejected", func(t *testing.T) {
			_, err := parse(t, `AA==<junk>   </junk>`)
			require.ErrorIs(t, err, xmlenc1.ErrMalformedEncrypted)
			require.Contains(t, err.Error(), "CipherValue holds a child")
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

// oaepParamsNotCharacterData is the fragment the child-kind rule puts in its
// error, and it is a DIFFERENT refusal from the size limit: an element child is
// refused for what it is, before its size is ever weighed.
const oaepParamsNotCharacterData = "which is not character data"

// oaepParamsInvalid is the fragment the base64 decoder's refusal carries. An
// entity reference contributes its DECLARED replacement text and nothing below
// it, so an entity holding another reference or markup lands here rather than
// being expanded into something decodable.
const oaepParamsInvalid = "invalid OAEPparams"

// oaepParamsNoEntityDecl is the fragment an entity reference whose first child
// is present but is not an Entity carries. Only a caller-built tree holds that
// shape; a CHILDLESS entity reference is an ordinary parser output and is not
// refused at all.
const oaepParamsNoEntityDecl = "entity reference with no entity declaration"

// oaepParamsEncryptedData builds an EncryptedData whose own EncryptionMethod
// carries params as raw OAEPparams markup — text, CDATA sections, elements, or
// any mix. Writing the markup straight into the document is the only way to
// put a chosen lexical form in front of the size limit.
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

// oaepParamsEntityEncryptedData is oaepParamsEncryptedData with an internal DTD
// in front of it: decls declares the entities params may reference. The parser
// this package's tests use does not substitute entities, so a reference in
// params survives into the DOM as an EntityRefNode whose first child is the
// declared Entity, which is the shape the parse path has to read through.
func oaepParamsEntityEncryptedData(t *testing.T, decls, params string) *helium.Element {
	t.Helper()
	doc := mustParseXML(t, `<?xml version="1.0"?>`+"\n"+
		`<!DOCTYPE xenc:EncryptedData [`+"\n"+decls+"\n"+`]>`+"\n"+
		`<xenc:EncryptedData xmlns:xenc="`+xmlenc1.NamespaceXMLEnc+`">`+
		`<xenc:EncryptionMethod Algorithm="`+xmlenc1.RSAOAEP11+`">`+
		`<xenc:OAEPparams>`+params+`</xenc:OAEPparams>`+
		`</xenc:EncryptionMethod>`+
		`<xenc:CipherData><xenc:CipherValue>AAAA</xenc:CipherValue></xenc:CipherData>`+
		`</xenc:EncryptedData>`)
	return doc.DocumentElement()
}

// oaepParamsDoctypeEncryptedData is oaepParamsEntityEncryptedData with the
// WHOLE doctype declaration chosen by the caller rather than just its internal
// subset. Which DTD shape a document carries is what decides whether the parser
// refuses an undeclared general entity reference outright or keeps it in the
// DOM, so a test of that behaviour has to write the declaration itself.
func oaepParamsDoctypeEncryptedData(t *testing.T, doctype, params string) *helium.Element {
	t.Helper()
	doc := mustParseXML(t, `<?xml version="1.0"?>`+"\n"+doctype+"\n"+
		`<xenc:EncryptedData xmlns:xenc="`+xmlenc1.NamespaceXMLEnc+`">`+
		`<xenc:EncryptionMethod Algorithm="`+xmlenc1.RSAOAEP11+`">`+
		`<xenc:OAEPparams>`+params+`</xenc:OAEPparams>`+
		`</xenc:EncryptionMethod>`+
		`<xenc:CipherData><xenc:CipherValue>AAAA</xenc:CipherValue></xenc:CipherData>`+
		`</xenc:EncryptedData>`)
	return doc.DocumentElement()
}

// oaepParamsElement returns the OAEPparams element of an EncryptedData built by
// the helpers above, so a test can attach a child no XML document can produce.
func oaepParamsElement(t *testing.T, encryptedData *helium.Element) *helium.Element {
	t.Helper()
	method, ok := helium.AsNode[*helium.Element](encryptedData.FirstChild())
	require.True(t, ok)
	params, ok := helium.AsNode[*helium.Element](method.FirstChild())
	require.True(t, ok)
	require.Equal(t, "OAEPparams", params.LocalName())
	return params
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

	// An element child is refused for what it is, before its size is weighed.
	// It makes the content invalid xs:base64Binary, and reading it would cost
	// its whole subtree: helium aggregates an element's Content from every
	// descendant, so the text of an arbitrarily deep subtree would be spent
	// against a limit that counts decoded octets.
	t.Run("an element child is rejected", func(t *testing.T) {
		_, err := xmlenc1.ParseEncryptedDataForTest(oaepParamsEncryptedData(t, `<junk>AAAA</junk>`))
		require.ErrorIs(t, err, xmlenc1.ErrMalformedEncrypted)
		require.Contains(t, err.Error(), oaepParamsNotCharacterData)
	})

	// The shape the size limit alone can never catch: one legal quantum, then
	// an element holding whitespace. Whitespace decodes to nothing, so the
	// value weighs one byte and the limit has no reason to fire — a walk that
	// read the element would return no error at all, having already spent the
	// subtree. TestOAEPParamsAllocation pins that cost; this pins the verdict.
	t.Run("an element child beside a legal value is rejected", func(t *testing.T) {
		_, err := xmlenc1.ParseEncryptedDataForTest(oaepParamsEncryptedData(t, "AA==<junk>   \t   </junk>"))
		require.ErrorIs(t, err, xmlenc1.ErrMalformedEncrypted)
		require.Contains(t, err.Error(), oaepParamsNotCharacterData)
	})

	// And an EncryptedKey's OAEPparams refuses it the same way.
	t.Run("an element child in an EncryptedKey OAEPparams is rejected", func(t *testing.T) {
		_, err := xmlenc1.ParseEncryptedDataForTest(oaepParamsKeyEncryptedData(t, "AA==<junk>   \t   </junk>"))
		require.ErrorIs(t, err, xmlenc1.ErrMalformedEncrypted)
		require.Contains(t, err.Error(), oaepParamsNotCharacterData)
	})

	// Comments and processing instructions are not character data either, but
	// unlike an element they may appear inside any element and say nothing
	// about its value, so they are ignored rather than refused. Reading one
	// would splice its text into the base64 and decode a label the document
	// never wrote.
	t.Run("comments and processing instructions are ignored", func(t *testing.T) {
		for _, params := range []string{
			`AA<!-- a comment -->==`,
			`AA<?target data?>==`,
			`<!--c-->AA<?t?>==<!--c-->`,
		} {
			ed, err := xmlenc1.ParseEncryptedDataForTest(oaepParamsEncryptedData(t, params))
			require.NoError(t, err, "params=%s", params)
			require.Equal(t, []byte{0x00}, ed.EncryptionMethod.OAEPParams, "params=%s", params)
		}
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

// TestOAEPParamsEntityReference covers the entity-reference child, which a
// conforming document may write anywhere character data is allowed and which
// the parse path therefore has to read rather than refuse.
//
// Reading it is bounded because it goes exactly one hop: the EntityRef's first
// child is the declared Entity, and helium.Entity.Content is a leaf accessor
// that returns the declared replacement literal without recursing. So the cost
// is one copy of that literal — the same floor a text child already costs — and
// the value is whatever the DTD literally declares, never what it would expand
// to. Every case below distinguishes those two: an entity whose replacement is
// another reference or markup contributes that text verbatim and fails the
// decode, exactly as it must.
func TestOAEPParamsEntityReference(t *testing.T) {
	label := []byte("label-params")
	encodedLabel := base64.StdEncoding.EncodeToString(label)

	// A label written as an entity reference decodes to the same bytes as the
	// same label written literally. This is the shape a conforming document is
	// free to use and the one refusing entity references would break.
	t.Run("an entity reference contributes its replacement text", func(t *testing.T) {
		ed, err := xmlenc1.ParseEncryptedDataForTest(oaepParamsEntityEncryptedData(t,
			`<!ENTITY label "`+encodedLabel+`">`, `&label;`))
		require.NoError(t, err)
		require.Equal(t, label, ed.EncryptionMethod.OAEPParams)
	})

	// The counting state carries across child boundaries, so a quantum split
	// between text and an entity reference decodes like the joined value.
	t.Run("an entity reference beside text decodes as one value", func(t *testing.T) {
		ed, err := xmlenc1.ParseEncryptedDataForTest(oaepParamsEntityEncryptedData(t,
			`<!ENTITY half "A">`, `A&half;==`))
		require.NoError(t, err)
		require.Equal(t, []byte{0x00}, ed.EncryptionMethod.OAEPParams)
	})

	// The limit governs the value however it arrives, and an entity reference
	// is charged the declared replacement text like any other child.
	t.Run("an over-limit entity reference is rejected", func(t *testing.T) {
		over := base64.StdEncoding.EncodeToString(make([]byte, xmlenc1.MaxOAEPParamsBytesForTest+1))
		_, err := xmlenc1.ParseEncryptedDataForTest(oaepParamsEntityEncryptedData(t,
			`<!ENTITY big "`+over+`">`, `&big;`))
		require.ErrorIs(t, err, xmlenc1.ErrMalformedEncrypted)
		require.Contains(t, err.Error(), oaepParamsOverLimit)
	})

	// A nested entity is where the one-hop bound shows: the value is the
	// literal "&inner;", not what inner would expand to, so it is not base64
	// and the decode refuses it.
	t.Run("a nested entity is not expanded", func(t *testing.T) {
		_, err := xmlenc1.ParseEncryptedDataForTest(oaepParamsEntityEncryptedData(t,
			`<!ENTITY inner "QUJD">`+"\n"+`<!ENTITY nested "&inner;">`, `&nested;`))
		require.ErrorIs(t, err, xmlenc1.ErrMalformedEncrypted)
		require.Contains(t, err.Error(), oaepParamsInvalid)
	})

	// An entity holding markup is the same story: the replacement text is read
	// as text, never built into nodes and never walked.
	t.Run("an entity holding markup is not expanded", func(t *testing.T) {
		_, err := xmlenc1.ParseEncryptedDataForTest(oaepParamsEntityEncryptedData(t,
			`<!ENTITY markup "<x/>">`, `&markup;`))
		require.ErrorIs(t, err, xmlenc1.ErrMalformedEncrypted)
		require.Contains(t, err.Error(), oaepParamsInvalid)
	})

	// Only the Entity is read. Its NextSibling is the next declaration in the
	// DTD — unrelated to this value — so a walk that followed siblings would
	// splice the other declarations' text into the label. Two more entities are
	// declared after the referenced one purely to make that visible.
	t.Run("the entity's DTD siblings are not read", func(t *testing.T) {
		ed, err := xmlenc1.ParseEncryptedDataForTest(oaepParamsEntityEncryptedData(t,
			`<!ENTITY label "`+encodedLabel+`">`+"\n"+
				`<!ENTITY after1 "QUJD">`+"\n"+
				`<!ENTITY after2 "REVG">`, `&label;`))
		require.NoError(t, err)
		require.Equal(t, label, ed.EncryptionMethod.OAEPParams)
	})

	// A reference with no entity behind it carries no character data at all, so
	// it contributes none and the rest of the value decodes untouched. Nothing
	// is read through it, so the bound is not at stake either.
	t.Run("an entity reference with no entity contributes nothing", func(t *testing.T) {
		encryptedData := oaepParamsEncryptedData(t, `AA==`)
		ref, err := encryptedData.OwnerDocument().CreateReference("undeclared")
		require.NoError(t, err)
		require.NoError(t, oaepParamsElement(t, encryptedData).AddChild(ref))

		ed, err := xmlenc1.ParseEncryptedDataForTest(encryptedData)
		require.NoError(t, err)
		require.Equal(t, []byte{0x00}, ed.EncryptionMethod.OAEPParams)
	})

	// The refusal that remains covers the hazardous caller-built shape: an EntityRef
	// carrying an element child. Asking such a node for its content aggregates
	// that subtree, which is the cost the bound exists to refuse.
	t.Run("an entity reference holding an element is rejected", func(t *testing.T) {
		encryptedData := oaepParamsEncryptedData(t, `AA==`)
		doc := encryptedData.OwnerDocument()
		ref, err := doc.CreateReference("undeclared")
		require.NoError(t, err)
		junk, err := doc.CreateElement("junk")
		require.NoError(t, err)
		require.NoError(t, junk.AddChild(doc.CreateText([]byte("QUJD"))))
		require.NoError(t, ref.AddChild(junk))
		require.NoError(t, oaepParamsElement(t, encryptedData).AddChild(ref))

		_, err = xmlenc1.ParseEncryptedDataForTest(encryptedData)
		require.ErrorIs(t, err, xmlenc1.ErrMalformedEncrypted)
		require.Contains(t, err.Error(), oaepParamsNoEntityDecl)
	})
}

// TestOAEPParamsUndeclaredEntityReference covers the entity reference the
// PARSER leaves with no Entity under it, which is where the one-hop walk meets
// a real document rather than a tree a test built by hand.
//
// XML 1.0's "Entity Declared" constraint makes an undeclared general entity a
// fatal well-formedness error only when the document is standalone="yes", or
// when it has neither an external subset nor a parameter-entity reference.
// Given either of those two DTD shapes and no standalone="yes", it is a
// VALIDITY error instead, so helium's non-validating parse keeps the reference
// in the DOM as an EntityRefNode with no child at all. Every case below drives
// helium.NewParser over such a document, which is the only way to reach that
// shape — hand-attaching a child through Document.CreateReference produces a
// DIFFERENT one and would not exercise this at all.
//
// Such a reference carries no character data, so it contributes none: the rest
// of the value decodes exactly as if it were written on its own. Refusing it
// would reject documents the parser accepts and the c14n canonicalizer already
// reads as empty, since c14n canonicalizes an EntityRef by walking its
// children.
func TestOAEPParamsUndeclaredEntityReference(t *testing.T) {
	label := []byte("label-params")
	encodedLabel := base64.StdEncoding.EncodeToString(label)
	require.Len(t, encodedLabel, 16)
	// The reference sits BETWEEN two halves of one base64 value, so a walk that
	// contributed anything for it — even a single character — would corrupt the
	// quantum boundary and change the decoded bytes rather than merely fail.
	split := encodedLabel[:8] + `&undeclared;` + encodedLabel[8:]

	// An external subset is one of the two DTD shapes that downgrade the
	// undeclared entity to a validity error. The system identifier is never
	// dereferenced — the default parser blocks external entities and denies
	// filesystem access — so its mere presence is what counts.
	t.Run("an external subset keeps an undeclared reference in the tree", func(t *testing.T) {
		ed, err := xmlenc1.ParseEncryptedDataForTest(oaepParamsDoctypeEncryptedData(t,
			`<!DOCTYPE xenc:EncryptedData SYSTEM "no-such.dtd">`, split))
		require.NoError(t, err)
		require.Equal(t, label, ed.EncryptionMethod.OAEPParams)
	})

	// A parameter-entity reference is the other shape, and it needs NO external
	// subset: a wholly internal DTD that references one parameter entity is
	// enough for the parser to keep going.
	t.Run("a parameter entity reference keeps an undeclared reference in the tree", func(t *testing.T) {
		ed, err := xmlenc1.ParseEncryptedDataForTest(oaepParamsDoctypeEncryptedData(t,
			`<!DOCTYPE xenc:EncryptedData [<!ENTITY % pe "<!ENTITY unused 'x'>"> %pe;]>`, split))
		require.NoError(t, err)
		require.Equal(t, label, ed.EncryptionMethod.OAEPParams)
	})

	// And the shape itself, asserted on the DOM rather than inferred from the
	// decode: the parser really does hand the walk an EntityRef with a nil
	// first child, so the guard on that case is load-bearing for parsed input.
	t.Run("the parser produces an entity reference with no child", func(t *testing.T) {
		encryptedData := oaepParamsDoctypeEncryptedData(t,
			`<!DOCTYPE xenc:EncryptedData SYSTEM "no-such.dtd">`, split)
		var seen int
		for child := oaepParamsElement(t, encryptedData).FirstChild(); child != nil; child = child.NextSibling() {
			ref, ok := helium.AsNode[*helium.EntityRef](child)
			if !ok {
				continue
			}
			seen++
			require.Equal(t, "undeclared", ref.Name())
			require.Nil(t, ref.FirstChild())
		}
		require.Equal(t, 1, seen)
	})

	// The guard must not swallow the declared path it sits in front of: with
	// the same external subset present, a DECLARED entity still contributes its
	// replacement text.
	t.Run("a declared entity still decodes beside the external subset", func(t *testing.T) {
		ed, err := xmlenc1.ParseEncryptedDataForTest(oaepParamsDoctypeEncryptedData(t,
			`<!DOCTYPE xenc:EncryptedData SYSTEM "no-such.dtd" [<!ENTITY label "`+encodedLabel+`">]>`,
			`&label;`))
		require.NoError(t, err)
		require.Equal(t, label, ed.EncryptionMethod.OAEPParams)
	})

	// Neither DTD shape means the parser refuses the document outright, so the
	// reference never reaches this package at all. Pinning that keeps the cases
	// above honest about WHY they need their doctype.
	t.Run("without either shape the parser rejects the document", func(t *testing.T) {
		for name, head := range map[string]string{
			"no doctype":                      `<?xml version="1.0"?>`,
			"internal subset, no PE":          `<?xml version="1.0"?>` + "\n" + `<!DOCTYPE xenc:EncryptedData [<!ENTITY unused "x">]>`,
			"standalone yes, external subset": `<?xml version="1.0" standalone="yes"?>` + "\n" + `<!DOCTYPE xenc:EncryptedData SYSTEM "no-such.dtd">`,
		} {
			_, err := helium.NewParser().Parse(t.Context(), []byte(head+"\n"+
				`<xenc:EncryptedData xmlns:xenc="`+xmlenc1.NamespaceXMLEnc+`">`+
				`<xenc:EncryptionMethod Algorithm="`+xmlenc1.RSAOAEP11+`">`+
				`<xenc:OAEPparams>`+split+`</xenc:OAEPparams>`+
				`</xenc:EncryptionMethod>`+
				`<xenc:CipherData><xenc:CipherValue>AAAA</xenc:CipherValue></xenc:CipherData>`+
				`</xenc:EncryptedData>`))
			require.Error(t, err, "doctype=%s", name)
		}
	})
}

// oaepAllocLexical is the lexical text an attacker writes around an
// xs:base64Binary value in the allocation cases below, and oaepAllocSlack is
// what those cases allow on top of reading it.
//
// The slack is a CONSTANT, deliberately: a bound stated as a multiple of the
// attacker's own text can never fail on a cost that follows that text, which is
// the only cost being pinned. It covers the rest of the document, the failed
// decrypt, and the runtime's own bookkeeping, all of which are fixed.
const (
	oaepAllocLexical = 4 << 20
	oaepAllocSlack   = 1 << 20
)

// TestOAEPParamsAllocation pins what an xenc:OAEPparams element may ALLOCATE,
// which the error assertions above cannot see. The limit governs decoded
// octets, but the lexical text an attacker wraps around them is unbounded:
// xs:base64Binary permits XML whitespace between characters, the value may be
// spread over as many text and CDATA children as the document likes, and an
// element child can hide a whole subtree of it behind a value that decodes to
// a single byte. Joining that text into one string before the limit is applied
// makes the limit an accounting formality — the memory is allocated by the time
// the error is returned — and for a value the limit ACCEPTS it is never refused
// at all, so a test that only checks for the rejection would miss half of it.
//
// The bounds are one copy of the lexical text plus a fixed slack, or the slack
// alone. One copy is the floor and cannot be removed here: a DOM hands out a
// copy per node and offers no read-only view, so weighing a value costs one
// copy of its text. Everything above that floor is what this test refuses —
// a second pass over the value, or reading a subtree an element child hides,
// each cost another whole copy and blow the bound. The element cases carry no
// lexical term at all, because the child is refused before it is read.
//
// Each case reads the process-wide TotalAlloc delta across Decrypt, so these
// subtests must NOT run in parallel: a concurrent test's allocations would
// pollute the delta.
func TestOAEPParamsAllocation(t *testing.T) {
	// no t.Parallel(): isolated so each delta reflects only its own Decrypt.

	// cdataChunk cuts the attacker's text into sixteen CDATA sections.
	const (
		cdataChunk = oaepAllocLexical / 16
		oneRead    = oaepAllocLexical + oaepAllocSlack
	)

	// Only space and tab are used as whitespace: an XML parser folds CRLF to
	// LF, which would make the text the DOM holds shorter than the text written
	// here and the bounds above harder to read.
	padding := strings.Repeat(" \t", oaepAllocLexical/2)
	// oversized decodes to 3 MiB, far over the limit however it is laid out.
	oversized := strings.Repeat("A", oaepAllocLexical)

	for _, tc := range []struct {
		name string
		// params is the raw OAEPparams markup, and errFragment is what the
		// OAEPparams path must say about it — empty for a value that path
		// accepts, where Decrypt still fails later for want of a usable key.
		params      string
		errFragment string
		maxAlloc    uint64
	}{
		{
			name:        "over-limit value in one text node",
			params:      oversized,
			errFragment: oaepParamsOverLimit,
			maxAlloc:    oneRead,
		},
		{
			// CDATA is how the value evades the parser's per-node content cap:
			// the cap bounds one indivisible run, and every section is its own.
			name:        "over-limit value split across CDATA sections",
			params:      splitIntoNodes(oversized, cdataChunk),
			errFragment: oaepParamsOverLimit,
			maxAlloc:    oneRead,
		},
		{
			// Nothing here is ever refused: one quantum of label is far under
			// the limit, and the whitespace after it is not counted at all.
			// Allocating twice for it would be amplification with no error
			// anywhere to notice it, which a rejection-only test would miss.
			name:     "under-limit value with trailing whitespace",
			params:   "AA==" + padding,
			maxAlloc: oneRead,
		},
		{
			// The same accepted value spread over CDATA sections, where joining
			// the children costs the most.
			name:     "under-limit value with whitespace split across CDATA sections",
			params:   "AA==" + splitIntoNodes(padding, cdataChunk),
			maxAlloc: oneRead,
		},
		{
			// The sharpest shape: a legal one-byte value, then an element
			// holding the attacker's text. The text decodes to nothing, so no
			// size limit has any reason to fire — but helium builds an
			// element's Content from its whole subtree, so a walk that read the
			// child would spend every byte of it and return no error at all.
			// The bound carries no lexical term: the child is refused for what
			// it is, before anything asks it how big it is.
			name:        "element child hiding whitespace",
			params:      "AA==<junk>" + padding + "</junk>",
			errFragment: oaepParamsNotCharacterData,
			maxAlloc:    oaepAllocSlack,
		},
		{
			// The same subtree built from many small CDATA leaves, so no single
			// node is large and only the aggregate is.
			name:        "element child hiding CDATA leaves",
			params:      "AA==<junk>" + splitIntoNodes(padding, 1024) + "</junk>",
			errFragment: oaepParamsNotCharacterData,
			maxAlloc:    oaepAllocSlack,
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
			// Both refusals are checked every time, so a case that expects one
			// also proves the other did not fire and a case that expects
			// neither proves the value really was accepted.
			for _, fragment := range []string{oaepParamsOverLimit, oaepParamsNotCharacterData} {
				if fragment == tc.errFragment {
					require.Contains(t, err.Error(), fragment, "err=%v", err)
					continue
				}
				require.NotContains(t, err.Error(), fragment, "err=%v", err)
			}

			allocated := after.TotalAlloc - before.TotalAlloc
			require.Less(t, allocated, tc.maxAlloc, "decrypting %d lexical bytes allocated %d bytes", len(tc.params), allocated)
		})
	}
}

// TestOAEPParamsAllocationSlope pins the MARGINAL cost of the attacker's text:
// doubling it must cost at most one more copy of what was added. The bounds in
// TestOAEPParamsAllocation carry a lexical term because one copy is the floor,
// so a slope of two would satisfy them if the slack were ever loosened; this
// measures the slope directly and no amount of slack hides it.
func TestOAEPParamsAllocationSlope(t *testing.T) {
	// no t.Parallel(): both measurements read the process-wide TotalAlloc.
	single := oaepParamsDecryptAlloc(t, "AA=="+strings.Repeat(" \t", oaepAllocLexical/2))
	double := oaepParamsDecryptAlloc(t, "AA=="+strings.Repeat(" \t", oaepAllocLexical))

	require.Less(t, double-single, uint64(oaepAllocLexical+oaepAllocSlack),
		"doubling the lexical text cost %d bytes on top of %d", double-single, single)
}

// oaepParamsDecryptAlloc reports the TotalAlloc delta across one Decrypt of an
// EncryptedData carrying params as its OAEPparams markup. The document and the
// key are built outside the measurement, so the delta is the decrypt alone.
func oaepParamsDecryptAlloc(t *testing.T, params string) uint64 {
	t.Helper()
	elem := oaepParamsEncryptedData(t, params)
	decryptor := xmlenc1.NewDecryptor().PrivateKey(generateRSAKey(t))

	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	_, err := decryptor.Decrypt(t.Context(), elem)
	runtime.ReadMemStats(&after)

	require.Error(t, err)
	require.NotContains(t, err.Error(), oaepParamsOverLimit, "err=%v", err)
	return after.TotalAlloc - before.TotalAlloc
}
