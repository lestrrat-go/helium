package xmlenc1_test

import (
	"crypto/ecdh"
	"crypto/rand"
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

// TestMarshalParseRoundTrip exercises the serialize and parse paths for the
// optional fields both directions carry: the EncryptedData's ID and Type, its
// EncryptionMethod's DigestMethod, MGFAlgorithm and OAEPParams, and an
// EncryptedKey with its own ID and EncryptionMethod. The marshaler's own
// in-memory DOM is fed straight back to the internal EncryptedData parser, with
// no serialization to bytes and no reparse through the public XML parser in
// between, so what is pinned is that the two internal directions agree on the
// same tree, not a byte-level round trip.
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

// TestEncryptionMethodKeySize verifies that parseEncryptionMethod checks a
// present xenc:KeySize against the length its algorithm URI already implies
// (xmlenc-core1 §3.2, §5.3, §5.6.2.2), rather than silently ignoring it: a
// KeySize consistent with what the algorithm fixes parses, one inconsistent
// with it is ErrMalformedEncrypted, and one under a URI that implies no
// length at all (RSA key transport) is accepted and ignored.
func TestEncryptionMethodKeySize(t *testing.T) {
	const head = `<xenc:EncryptedData xmlns:xenc="http://www.w3.org/2001/04/xmlenc#" xmlns:ds="http://www.w3.org/2000/09/xmldsig#">`
	const cipher = `<xenc:CipherData><xenc:CipherValue>AAAA</xenc:CipherValue></xenc:CipherData></xenc:EncryptedData>`

	parse := func(t *testing.T, em string) (*xmlenc1.EncryptedData, error) {
		t.Helper()
		doc := mustParseXML(t, head+em+cipher)
		elem, ok := helium.AsNode[*helium.Element](doc.DocumentElement())
		require.True(t, ok)
		return xmlenc1.ParseEncryptedDataForTest(elem)
	}

	t.Run("agreeing KeySize parses", func(t *testing.T) {
		ed, err := parse(t, `<xenc:EncryptionMethod Algorithm="`+xmlenc1.AES256GCM+`">`+
			`<xenc:KeySize>256</xenc:KeySize>`+
			`</xenc:EncryptionMethod>`)
		require.NoError(t, err)
		require.Equal(t, xmlenc1.AES256GCM, ed.EncryptionMethod.Algorithm)
	})

	t.Run("contradicting KeySize rejected", func(t *testing.T) {
		_, err := parse(t, `<xenc:EncryptionMethod Algorithm="`+xmlenc1.AES256GCM+`">`+
			`<xenc:KeySize>128</xenc:KeySize>`+
			`</xenc:EncryptionMethod>`)
		require.ErrorIs(t, err, xmlenc1.ErrMalformedEncrypted)
		require.Contains(t, err.Error(), "128")
		require.Contains(t, err.Error(), "256")
	})

	t.Run("byte-valued KeySize rejected", func(t *testing.T) {
		// KeySize is in BITS (xmlenc-core1 §5.6.2.2). A producer that wrote
		// the byte count (32, for AES-256) contradicts the algorithm rather
		// than matching it — this pins the bits-not-bytes decision.
		_, err := parse(t, `<xenc:EncryptionMethod Algorithm="`+xmlenc1.AES256GCM+`">`+
			`<xenc:KeySize>32</xenc:KeySize>`+
			`</xenc:EncryptionMethod>`)
		require.ErrorIs(t, err, xmlenc1.ErrMalformedEncrypted)
		require.Contains(t, err.Error(), "32")
		require.Contains(t, err.Error(), "256")
	})

	t.Run("KeySize contradicting AES key wrap on EncryptedKey rejected", func(t *testing.T) {
		doc := mustParseXML(t, head+
			`<ds:KeyInfo><xenc:EncryptedKey>`+
			`<xenc:EncryptionMethod Algorithm="`+xmlenc1.AES128KeyWrap+`">`+
			`<xenc:KeySize>256</xenc:KeySize>`+
			`</xenc:EncryptionMethod>`+
			`<xenc:CipherData><xenc:CipherValue>BBBB</xenc:CipherValue></xenc:CipherData>`+
			`</xenc:EncryptedKey></ds:KeyInfo>`+
			cipher)
		elem, ok := helium.AsNode[*helium.Element](doc.DocumentElement())
		require.True(t, ok)
		_, err := xmlenc1.ParseEncryptedDataForTest(elem)
		require.ErrorIs(t, err, xmlenc1.ErrMalformedEncrypted)
		require.Contains(t, err.Error(), "256")
		require.Contains(t, err.Error(), "128")
	})

	t.Run("KeySize under RSA-OAEP key transport accepted", func(t *testing.T) {
		ed, err := parse(t, `<xenc:EncryptionMethod Algorithm="`+xmlenc1.RSAOAEP11+`">`+
			`<xenc:KeySize>256</xenc:KeySize>`+
			`</xenc:EncryptionMethod>`)
		require.NoError(t, err)
		require.Equal(t, xmlenc1.RSAOAEP11, ed.EncryptionMethod.Algorithm)
	})

	t.Run("whitespace and leading zeros accepted", func(t *testing.T) {
		ed, err := parse(t, `<xenc:EncryptionMethod Algorithm="`+xmlenc1.AES256GCM+`">`+
			`<xenc:KeySize>  00256  </xenc:KeySize>`+
			`</xenc:EncryptionMethod>`)
		require.NoError(t, err)
		require.Equal(t, xmlenc1.AES256GCM, ed.EncryptionMethod.Algorithm)
	})

	t.Run("non-breaking space rejected", func(t *testing.T) {
		// xs:integer's whiteSpace='collapse' facet covers exactly #x20, #x9,
		// #xD and #xA, so the ASCII spaces the case above accepts are in the
		// lexical space but U+00A0 is not. A value wrapped in NBSP is not an
		// xs:integer and must be refused rather than trimmed into one.
		_, err := parse(t, `<xenc:EncryptionMethod Algorithm="`+xmlenc1.AES256GCM+`">`+
			"<xenc:KeySize>\u00a0256\u00a0</xenc:KeySize>"+
			`</xenc:EncryptionMethod>`)
		require.ErrorIs(t, err, xmlenc1.ErrMalformedEncrypted)
		require.Contains(t, err.Error(), "invalid KeySize")
	})

	t.Run("out-of-range value is a consistency question, not a syntax one", func(t *testing.T) {
		// xs:integer is unbounded, so a 64-digit value is well formed even
		// though no int can hold it. It therefore reaches the consistency
		// check like any other value: ignored under a URI that implies no
		// key length, and inconsistent under one that does.
		digits := strings.Repeat("9", 64)

		ed, err := parse(t, `<xenc:EncryptionMethod Algorithm="`+xmlenc1.RSAOAEP11+`">`+
			`<xenc:KeySize>`+digits+`</xenc:KeySize>`+
			`</xenc:EncryptionMethod>`)
		require.NoError(t, err)
		require.Equal(t, xmlenc1.RSAOAEP11, ed.EncryptionMethod.Algorithm)

		_, err = parse(t, `<xenc:EncryptionMethod Algorithm="`+xmlenc1.AES256GCM+`">`+
			`<xenc:KeySize>`+digits+`</xenc:KeySize>`+
			`</xenc:EncryptionMethod>`)
		require.ErrorIs(t, err, xmlenc1.ErrMalformedEncrypted)
		require.Contains(t, err.Error(), "inconsistent")
		require.Contains(t, err.Error(), "256")
	})

	t.Run("value spread over text and CDATA accepted", func(t *testing.T) {
		// Proves the character-data walk, not Content(): a value split
		// across a text node and a CDATA section must still be joined.
		ed, err := parse(t, `<xenc:EncryptionMethod Algorithm="`+xmlenc1.AES256GCM+`">`+
			`<xenc:KeySize>2<![CDATA[56]]></xenc:KeySize>`+
			`</xenc:EncryptionMethod>`)
		require.NoError(t, err)
		require.Equal(t, xmlenc1.AES256GCM, ed.EncryptionMethod.Algorithm)
	})

	t.Run("element child rejected", func(t *testing.T) {
		_, err := parse(t, `<xenc:EncryptionMethod Algorithm="`+xmlenc1.AES256GCM+`">`+
			`<xenc:KeySize><bogus/></xenc:KeySize>`+
			`</xenc:EncryptionMethod>`)
		require.ErrorIs(t, err, xmlenc1.ErrMalformedEncrypted)
	})

	t.Run("comment ignored", func(t *testing.T) {
		ed, err := parse(t, `<xenc:EncryptionMethod Algorithm="`+xmlenc1.AES256GCM+`">`+
			`<xenc:KeySize>2<!-- comment -->56</xenc:KeySize>`+
			`</xenc:EncryptionMethod>`)
		require.NoError(t, err)
		require.Equal(t, xmlenc1.AES256GCM, ed.EncryptionMethod.Algorithm)
	})

	t.Run("over-length value refused before conversion", func(t *testing.T) {
		_, err := parse(t, `<xenc:EncryptionMethod Algorithm="`+xmlenc1.AES256GCM+`">`+
			`<xenc:KeySize>`+strings.Repeat("2", 65)+`</xenc:KeySize>`+
			`</xenc:EncryptionMethod>`)
		require.ErrorIs(t, err, xmlenc1.ErrMalformedEncrypted)
	})

	t.Run("non-integer rejected", func(t *testing.T) {
		_, err := parse(t, `<xenc:EncryptionMethod Algorithm="`+xmlenc1.AES256GCM+`">`+
			`<xenc:KeySize>not-a-number</xenc:KeySize>`+
			`</xenc:EncryptionMethod>`)
		require.ErrorIs(t, err, xmlenc1.ErrMalformedEncrypted)
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

// oaepParamsChildOfType is the fragment a child-kind refusal puts in front of
// oaepParamsNotCharacterData: the NAME of the value being read. The tail is
// shared with every other bounded base64 value, so the name is the only part of
// the message that says which value the document got wrong.
const oaepParamsChildOfType = "OAEPparams holds a child of type"

// oaepParamsBadEntityChild is the fragment an entity reference whose first child
// is present but is not an Entity carries. Only a caller-built tree holds that
// shape; a CHILDLESS entity reference is an ordinary parser output and is not
// refused at all, so the message must name the first child rather than a missing
// declaration.
const oaepParamsBadEntityChild = "entity reference whose first child is not an entity declaration"

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
		require.Contains(t, err.Error(), oaepParamsBadEntityChild)
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

// cipherValueBadEntityChild is the whole refusal an entity reference whose first
// child is present but is not an Entity carries on the CipherValue path. It
// names the value, so asserting the entire fragment — not just its tail — is
// what pins that base64CharacterData reports the call site it was reached from
// rather than the OAEPparams one it shares its implementation with.
const cipherValueBadEntityChild = "CipherValue holds an entity reference whose first child is not an entity declaration"

// cipherValueEncryptedData builds an EncryptedData whose payload CipherValue
// carries value as raw markup — text, CDATA sections, elements, or any mix.
// Writing the markup straight into the document is the only way to put a
// chosen lexical form in front of the CipherValue walk.
func cipherValueEncryptedData(t *testing.T, value string) *helium.Element {
	t.Helper()
	doc := mustParseXML(t, `<xenc:EncryptedData xmlns:xenc="`+xmlenc1.NamespaceXMLEnc+`">`+
		`<xenc:EncryptionMethod Algorithm="`+xmlenc1.AES256GCM+`"/>`+
		`<xenc:CipherData><xenc:CipherValue>`+value+`</xenc:CipherValue></xenc:CipherData>`+
		`</xenc:EncryptedData>`)
	return doc.DocumentElement()
}

// cipherValueDoctypeEncryptedData is cipherValueEncryptedData with the WHOLE
// doctype declaration chosen by the caller. Which DTD shape a document carries
// decides both whether an entity reference in the value resolves to a declared
// Entity and whether the parser keeps an undeclared reference in the DOM at
// all, so a test of either has to write the declaration itself.
func cipherValueDoctypeEncryptedData(t *testing.T, doctype, value string) *helium.Element {
	t.Helper()
	doc := mustParseXML(t, `<?xml version="1.0"?>`+"\n"+doctype+"\n"+
		`<xenc:EncryptedData xmlns:xenc="`+xmlenc1.NamespaceXMLEnc+`">`+
		`<xenc:EncryptionMethod Algorithm="`+xmlenc1.AES256GCM+`"/>`+
		`<xenc:CipherData><xenc:CipherValue>`+value+`</xenc:CipherValue></xenc:CipherData>`+
		`</xenc:EncryptedData>`)
	return doc.DocumentElement()
}

// cipherValueElement returns the payload CipherValue element of an
// EncryptedData built by the helpers above, so a test can attach a child no XML
// document can produce.
func cipherValueElement(t *testing.T, encryptedData *helium.Element) *helium.Element {
	t.Helper()
	data, ok := helium.AsNode[*helium.Element](encryptedData.FirstChild().NextSibling())
	require.True(t, ok)
	require.Equal(t, "CipherData", data.LocalName())
	value, ok := helium.AsNode[*helium.Element](data.FirstChild())
	require.True(t, ok)
	require.Equal(t, "CipherValue", value.LocalName())
	return value
}

// TestCipherValueEntityReference covers the entity-reference child on the
// CipherValue path.
//
// base64CharacterData is shared by xenc:OAEPparams and xenc:CipherValue, and
// its entity branch is otherwise reached only through the OAEPparams tests.
// The two call sites differ in what they hand it and in what its refusals must
// say, so pinning the branch at one of them leaves the other inferred: the
// value name travels into every error message, and the CipherValue walk reads
// each child TWICE — once to count and once to build — where the OAEPparams
// walk reads it once. An entity contributing on the counting pass but not on
// the building pass, or the reverse, would corrupt the value on this path
// while leaving the OAEPparams cases green.
//
// The bound is the same one hop the OAEPparams cases pin: the EntityRef's
// first child is the declared Entity and helium.Entity.Content returns the
// declared replacement literal without recursing.
func TestCipherValueEntityReference(t *testing.T) {
	payload := []byte("cipher-payload")
	encoded := base64.StdEncoding.EncodeToString(payload)
	require.Len(t, encoded, 20)

	// A ciphertext written as an entity reference decodes to the same bytes as
	// the same ciphertext written literally, which is the shape a conforming
	// document is free to use.
	t.Run("a declared entity reference contributes its replacement text", func(t *testing.T) {
		ed, err := xmlenc1.ParseEncryptedDataForTest(cipherValueDoctypeEncryptedData(t,
			`<!DOCTYPE xenc:EncryptedData [<!ENTITY payload "`+encoded+`">]>`, `&payload;`))
		require.NoError(t, err)
		require.Equal(t, payload, ed.CipherValue)
	})

	// The entity supplies the MIDDLE of the value, with text on both sides and
	// both cuts away from a quantum boundary. So the entity's characters have
	// to be concatenated with the siblings' in order, not substituted for them
	// and not appended after them: any other arrangement changes the decoded
	// bytes rather than merely failing.
	t.Run("an entity reference between text siblings decodes as one value", func(t *testing.T) {
		ed, err := xmlenc1.ParseEncryptedDataForTest(cipherValueDoctypeEncryptedData(t,
			`<!DOCTYPE xenc:EncryptedData [<!ENTITY middle "`+encoded[2:9]+`">]>`,
			encoded[:2]+`&middle;`+encoded[9:]))
		require.NoError(t, err)
		require.Equal(t, payload, ed.CipherValue)
	})

	// The same split with CDATA siblings instead of text, since the walk
	// classifies the two child kinds separately.
	t.Run("an entity reference between CDATA siblings decodes as one value", func(t *testing.T) {
		ed, err := xmlenc1.ParseEncryptedDataForTest(cipherValueDoctypeEncryptedData(t,
			`<!DOCTYPE xenc:EncryptedData [<!ENTITY middle "`+encoded[2:9]+`">]>`,
			`<![CDATA[`+encoded[:2]+`]]>&middle;<![CDATA[`+encoded[9:]+`]]>`))
		require.NoError(t, err)
		require.Equal(t, payload, ed.CipherValue)
	})

	// A reference with no entity behind it carries no character data, so it
	// contributes none and the rest of the value decodes untouched. This is a
	// PARSER-produced shape, not a caller-built one: an external subset with no
	// standalone="yes" downgrades XML 1.0's "Entity Declared" constraint to a
	// validity error, so helium's non-validating parse keeps the reference in
	// the DOM with no child at all. The system identifier is never dereferenced
	// — the default parser blocks external entities and denies filesystem
	// access — so its mere presence is what counts. The reference sits between
	// two halves of one value, so contributing even one character for it would
	// shift the quantum boundary and change the decoded bytes.
	t.Run("an entity reference with no entity contributes nothing", func(t *testing.T) {
		split := encoded[:8] + `&undeclared;` + encoded[8:]
		encryptedData := cipherValueDoctypeEncryptedData(t,
			`<!DOCTYPE xenc:EncryptedData SYSTEM "no-such.dtd">`, split)

		// The shape itself, asserted on the DOM rather than inferred from the
		// decode: the parser really does hand this walk a childless EntityRef.
		var seen int
		for child := cipherValueElement(t, encryptedData).FirstChild(); child != nil; child = child.NextSibling() {
			ref, ok := helium.AsNode[*helium.EntityRef](child)
			if !ok {
				continue
			}
			seen++
			require.Equal(t, "undeclared", ref.Name())
			require.Nil(t, ref.FirstChild())
		}
		require.Equal(t, 1, seen)

		ed, err := xmlenc1.ParseEncryptedDataForTest(encryptedData)
		require.NoError(t, err)
		require.Equal(t, payload, ed.CipherValue)
	})

	// The refusal covers the hazardous shape: an EntityRef whose first child is
	// present and is not an Entity. Asking such a node for its content
	// aggregates that subtree, which is the cost the one-hop bound exists to
	// refuse. No XML document produces this, so the tree is BUILT BY HAND here
	// — Document.CreateReference plus an attached element — exactly as the
	// OAEPparams counterpart does.
	//
	// The message must name CipherValue: base64CharacterData takes the value
	// name from its caller, and nothing else pins that this call site passes
	// its own rather than the OAEPparams one.
	t.Run("an entity reference holding an element is rejected", func(t *testing.T) {
		encryptedData := cipherValueEncryptedData(t, `AA==`)
		doc := encryptedData.OwnerDocument()
		ref, err := doc.CreateReference("undeclared")
		require.NoError(t, err)
		junk, err := doc.CreateElement("junk")
		require.NoError(t, err)
		require.NoError(t, junk.AddChild(doc.CreateText([]byte("QUJD"))))
		require.NoError(t, ref.AddChild(junk))
		require.NoError(t, cipherValueElement(t, encryptedData).AddChild(ref))

		_, err = xmlenc1.ParseEncryptedDataForTest(encryptedData)
		require.ErrorIs(t, err, xmlenc1.ErrMalformedEncrypted)
		require.Contains(t, err.Error(), cipherValueBadEntityChild)
		require.NotContains(t, err.Error(), "OAEPparams")
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

// ecPublicKeyOverLimit is the fragment the dsig11:PublicKey size limit puts in
// its error. Every parse failure in this package wraps ErrMalformedEncrypted,
// so the sentinel alone cannot tell a value refused by the limit from one
// crypto/ecdh or a later check refused.
const ecPublicKeyOverLimit = "ECKeyValue PublicKey is over the"

// ecPublicKeyNotCharacterData is the fragment the child-kind rule puts in its
// error, and it is a DIFFERENT refusal from the size limit: an element child is
// refused for what it is, before its size is ever weighed.
const ecPublicKeyNotCharacterData = "which is not character data"

// ecPublicKeyChildOfType is oaepParamsChildOfType for the other value: the name
// a child-kind refusal puts in front of the shared tail.
const ecPublicKeyChildOfType = "ECKeyValue PublicKey holds a child of type"

// ecPublicKeyInvalid is the fragment the base64 decoder's refusal carries for a
// dsig11:PublicKey. It is deliberately not the phrase OAEPparams uses: one walk
// decodes both values, so the phrase is the only thing in the error that says
// which of them the document got wrong.
const ecPublicKeyInvalid = "invalid ECKeyValue base64"

// The dsig11:NamedCurve URIs of the three curves this package supports. A
// curve is named by the OID URN of its named-curve object identifier.
const (
	ecCurveURIP256 = "urn:oid:1.2.840.10045.3.1.7"
	ecCurveURIP384 = "urn:oid:1.3.132.0.34"
	ecCurveURIP521 = "urn:oid:1.3.132.0.35"
)

// ecKeyValueEncryptedData builds an EncryptedData whose EncryptedKey protects
// its session key by ECDH-ES, with children written verbatim as the child
// markup of the dsig11:ECKeyValue. Writing that markup straight into the
// document is the only way to put a chosen child ORDER, and a chosen lexical
// form of the point, in front of the parse.
//
// The wrapped key and the payload ciphertext are junk: every case below is
// decided while the document is read, so none of them reaches any crypto.
func ecKeyValueEncryptedData(t *testing.T, children string) *helium.Element {
	t.Helper()
	doc := mustParseXML(t, `<xenc:EncryptedData xmlns:xenc="`+xmlenc1.NamespaceXMLEnc+`" xmlns:ds="`+xmlenc1.NamespaceDSig+`" xmlns:dsig11="`+xmlenc1.NamespaceDSig11+`" xmlns:xenc11="`+xmlenc1.NamespaceXMLEnc11+`">`+
		`<xenc:EncryptionMethod Algorithm="`+xmlenc1.AES128GCM11+`"/>`+
		`<ds:KeyInfo><xenc:EncryptedKey><xenc:EncryptionMethod Algorithm="`+xmlenc1.AES128KeyWrap+`"/>`+
		`<ds:KeyInfo><xenc:AgreementMethod Algorithm="`+xmlenc1.ECDHES+`">`+
		`<xenc11:KeyDerivationMethod Algorithm="`+xmlenc1.ConcatKDF+`"><xenc11:ConcatKDFParams><ds:DigestMethod Algorithm="`+xmlenc1.DigestSHA256+`"/></xenc11:ConcatKDFParams></xenc11:KeyDerivationMethod>`+
		`<xenc:OriginatorKeyInfo><ds:KeyValue><dsig11:ECKeyValue>`+children+`</dsig11:ECKeyValue></ds:KeyValue></xenc:OriginatorKeyInfo>`+
		`</xenc:AgreementMethod></ds:KeyInfo>`+
		`<xenc:CipherData><xenc:CipherValue>AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA</xenc:CipherValue></xenc:CipherData>`+
		`</xenc:EncryptedKey></ds:KeyInfo>`+
		`<xenc:CipherData><xenc:CipherValue>AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA</xenc:CipherValue></xenc:CipherData>`+
		`</xenc:EncryptedData>`)
	return doc.DocumentElement()
}

// ecKeyValueChildren renders the two children of a dsig11:ECKeyValue with the
// NamedCurve either before or after the PublicKey. The schema puts no order on
// them, so both orders are documents the parse has to read, and which one the
// document chose must not decide what the PublicKey walk may spend.
func ecKeyValueChildren(curveURI, publicKey string, namedCurveFirst bool) string {
	curve := `<dsig11:NamedCurve URI="` + curveURI + `"/>`
	key := `<dsig11:PublicKey>` + publicKey + `</dsig11:PublicKey>`
	if namedCurveFirst {
		return curve + key
	}
	return key + curve
}

// ecPublicKeyPoint returns the SEC1 uncompressed encoding of a freshly
// generated public key on curve, which is the octet string a conforming
// dsig11:PublicKey carries.
func ecPublicKeyPoint(t *testing.T, curve ecdh.Curve) []byte {
	t.Helper()
	key, err := curve.GenerateKey(rand.Reader)
	require.NoError(t, err)
	return key.PublicKey().Bytes()
}

// parseECPublicKey parses an ECDH-ES EncryptedData built from children and
// returns the originator public key the parse kept.
func parseECPublicKey(t *testing.T, children string) []byte {
	t.Helper()
	ed, err := xmlenc1.ParseEncryptedDataForTest(ecKeyValueEncryptedData(t, children))
	require.NoError(t, err)
	require.Len(t, ed.EncryptedKeys, 1)
	require.NotNil(t, ed.EncryptedKeys[0].AgreementMethod)
	require.NotNil(t, ed.EncryptedKeys[0].AgreementMethod.OriginatorKey)
	return ed.EncryptedKeys[0].AgreementMethod.OriginatorKey.PublicKey
}

// TestECPublicKeyBound covers the decoded-size limit on the dsig11:PublicKey of
// an ECDH-ES originator key. The element is read while the document is parsed,
// so it is reached before any key is resolved and before anything the document
// says has been authenticated.
//
// The limit is the largest encoding across ALL THREE supported curves and not
// the selected curve's own size, because the document decides the child order:
// NamedCurve may legally follow PublicKey, and a document carrying no
// NamedCurve at all still has its PublicKey read before the missing-curve error
// is raised. So at the moment the value is weighed there may be no curve to
// size it by.
func TestECPublicKeyBound(t *testing.T) {
	for _, tc := range []struct {
		name  string
		curve ecdh.Curve
		uri   string
	}{
		{name: "P-256", curve: ecdh.P256(), uri: ecCurveURIP256},
		{name: "P-384", curve: ecdh.P384(), uri: ecCurveURIP384},
		// P-521's 133-octet encoding is the boundary the limit is set at, so
		// this case is what proves the limit admits every conforming key.
		{name: "P-521", curve: ecdh.P521(), uri: ecCurveURIP521},
	} {
		t.Run("a "+tc.name+" key parses", func(t *testing.T) {
			point := ecPublicKeyPoint(t, tc.curve)
			require.LessOrEqual(t, len(point), xmlenc1.MaxECPublicKeyBytesForTest)
			encoded := base64.StdEncoding.EncodeToString(point)

			// Both child orders, because the value is weighed against the
			// all-curves limit either way.
			for _, namedCurveFirst := range []bool{true, false} {
				require.Equal(t, point, parseECPublicKey(t, ecKeyValueChildren(tc.uri, encoded, namedCurveFirst)),
					"namedCurveFirst=%v", namedCurveFirst)
			}
		})
	}

	// A genuine key whose NamedCurve follows it is the ordering that makes the
	// all-curves limit necessary, and it must still parse to the same point.
	t.Run("a NamedCurve after the PublicKey parses", func(t *testing.T) {
		point := ecPublicKeyPoint(t, ecdh.P256())
		children := ecKeyValueChildren(ecCurveURIP256, base64.StdEncoding.EncodeToString(point), false)
		require.Equal(t, point, parseECPublicKey(t, children))
	})

	// Splitting the value into CDATA sections is what defeats the XML parser's
	// per-node content cap, which bounds one indivisible run of characters and
	// not their concatenation. The limit is charged on the whole value, so the
	// split buys nothing.
	t.Run("an over-limit value split across CDATA sections is rejected", func(t *testing.T) {
		oversized := splitIntoNodes(strings.Repeat("A", 64<<10), 64)
		for _, namedCurveFirst := range []bool{true, false} {
			_, err := xmlenc1.ParseEncryptedDataForTest(ecKeyValueEncryptedData(t, ecKeyValueChildren(ecCurveURIP256, oversized, namedCurveFirst)))
			require.ErrorIs(t, err, xmlenc1.ErrMalformedEncrypted)
			require.Contains(t, err.Error(), ecPublicKeyOverLimit, "namedCurveFirst=%v", namedCurveFirst)
		}
	})

	// Sub-quantum children are the sharper form of the same trick: counting
	// each child alone would report zero for every three-character piece.
	t.Run("an over-limit value split into sub-quantum children is rejected", func(t *testing.T) {
		oversized := splitIntoNodes(strings.Repeat("A", 64<<10), 3)
		_, err := xmlenc1.ParseEncryptedDataForTest(ecKeyValueEncryptedData(t, ecKeyValueChildren(ecCurveURIP256, oversized, true)))
		require.ErrorIs(t, err, xmlenc1.ErrMalformedEncrypted)
		require.Contains(t, err.Error(), ecPublicKeyOverLimit)
	})

	// With no NamedCurve at all there is no curve to size the value by, and the
	// missing-curve error is only raised once the whole child list has been
	// read — so the limit is what has to refuse this document.
	t.Run("an over-limit value with no NamedCurve is refused by the limit", func(t *testing.T) {
		oversized := splitIntoNodes(strings.Repeat("A", 64<<10), 64)
		_, err := xmlenc1.ParseEncryptedDataForTest(ecKeyValueEncryptedData(t, `<dsig11:PublicKey>`+oversized+`</dsig11:PublicKey>`))
		require.ErrorIs(t, err, xmlenc1.ErrMalformedEncrypted)
		require.Contains(t, err.Error(), ecPublicKeyOverLimit)
	})

	// The limit measures decoded octets, so the XML whitespace
	// xs:base64Binary permits between characters neither counts against a legal
	// key nor lets an oversized value through.
	t.Run("a whitespace-wrapped key parses", func(t *testing.T) {
		point := ecPublicKeyPoint(t, ecdh.P521())
		encoded := base64.StdEncoding.EncodeToString(point)
		wrapped := " \t\n" + strings.Join(strings.Split(encoded, ""), " ") + "\n\t "
		require.Equal(t, point, parseECPublicKey(t, ecKeyValueChildren(ecCurveURIP521, wrapped, true)))
	})

	// The same value spread over CDATA sections, including sub-quantum ones
	// where a quantum and even padding straddle a child boundary.
	t.Run("a CDATA-split key parses", func(t *testing.T) {
		point := ecPublicKeyPoint(t, ecdh.P521())
		encoded := base64.StdEncoding.EncodeToString(point)
		for _, chunk := range []int{1, 2, 3, 5, 64} {
			require.Equal(t, point, parseECPublicKey(t, ecKeyValueChildren(ecCurveURIP521, splitIntoNodes(encoded, chunk), true)),
				"chunk=%d", chunk)
		}
	})

	// A comment carries no character information item, so it is skipped rather
	// than spliced into the base64 stream — which is what would decode a point
	// the document never wrote. A processing instruction is skipped for the
	// same reason.
	t.Run("comments and processing instructions are ignored", func(t *testing.T) {
		point := ecPublicKeyPoint(t, ecdh.P256())
		encoded := base64.StdEncoding.EncodeToString(point)
		for _, value := range []string{
			encoded[:8] + `<!-- a comment -->` + encoded[8:],
			encoded[:8] + `<?target data?>` + encoded[8:],
			`<!--c-->` + encoded + `<?t?>`,
		} {
			require.Equal(t, point, parseECPublicKey(t, ecKeyValueChildren(ecCurveURIP256, value, true)), "value=%s", value)
		}
	})

	// An element child is refused for what it is, before its size is weighed.
	// dsig11:PublicKey is xs:base64Binary, a simple type, so one makes the
	// content invalid — and reading it would cost its whole subtree, since
	// helium builds an element's Content from every descendant.
	t.Run("an element child is rejected", func(t *testing.T) {
		point := base64.StdEncoding.EncodeToString(ecPublicKeyPoint(t, ecdh.P256()))
		for _, value := range []string{
			`<junk>` + point + `</junk>`,
			point + `<junk>   \t   </junk>`,
		} {
			_, err := xmlenc1.ParseEncryptedDataForTest(ecKeyValueEncryptedData(t, ecKeyValueChildren(ecCurveURIP256, value, true)))
			require.ErrorIs(t, err, xmlenc1.ErrMalformedEncrypted, "value=%s", value)
			require.Contains(t, err.Error(), ecPublicKeyNotCharacterData, "value=%s", value)
		}
	})
}

// TestECPublicKeyAllocation pins what a dsig11:PublicKey may ALLOCATE, which
// the error assertions above cannot see. The limit governs decoded octets, but
// the lexical text an attacker wraps around them is unbounded: xs:base64Binary
// permits XML whitespace between characters and the value may be spread over as
// many text and CDATA children as the document likes. Joining that text into
// one string before the limit is applied makes the limit an accounting
// formality — the memory is allocated by the time the error is returned — and
// for a value the limit ACCEPTS it is never refused at all, so a test that only
// checked for the rejection would miss half of it.
//
// The bounds are one copy of the lexical text plus a fixed slack. One copy is
// the floor and cannot be removed here: a DOM hands out a copy per node and
// offers no read-only view, so weighing a value costs one copy of its text.
// Everything above that floor is what this test refuses.
//
// Each case reads the process-wide TotalAlloc delta across a parse, so these
// subtests must NOT run in parallel: a concurrent test's allocations would
// pollute the delta.
func TestECPublicKeyAllocation(t *testing.T) {
	// no t.Parallel(): isolated so each delta reflects only its own parse.

	// oaepAllocLexical and oaepAllocSlack describe the lexical envelope an
	// attacker writes around any xs:base64Binary value, so the same figures
	// apply here. The text is split into CDATA sections so no single child comes
	// near the XML parser's own per-node content cap, which is not the limit
	// under test here.
	const (
		chunk   = oaepAllocLexical / 16
		oneRead = oaepAllocLexical + oaepAllocSlack
	)

	// Only space and tab are used as whitespace: an XML parser folds CRLF to
	// LF, which would make the text the DOM holds shorter than the text written
	// here and the bounds above harder to read.
	padding := strings.Repeat(" \t", oaepAllocLexical/2)
	oversized := strings.Repeat("A", oaepAllocLexical)
	accepted := base64.StdEncoding.EncodeToString(ecPublicKeyPoint(t, ecdh.P256()))

	for _, tc := range []struct {
		name string
		// value is the raw PublicKey markup, and rejected says whether the
		// limit must refuse it — a value it accepts still leaves the parse
		// succeeding, since the point is a real one.
		value    string
		rejected bool
	}{
		{
			name:     "over-limit value split across CDATA sections",
			value:    splitIntoNodes(oversized, chunk),
			rejected: true,
		},
		{
			// Nothing here is ever refused: the point is a conforming one and
			// the whitespace after it is not counted at all. Allocating twice
			// for it would be amplification with no error anywhere to notice
			// it, which a rejection-only test would miss.
			name:  "accepted key with whitespace split across CDATA sections",
			value: accepted + splitIntoNodes(padding, chunk),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			elem := ecKeyValueEncryptedData(t, ecKeyValueChildren(ecCurveURIP256, tc.value, true))

			var before, after runtime.MemStats
			runtime.GC()
			runtime.ReadMemStats(&before)
			_, err := xmlenc1.ParseEncryptedDataForTest(elem)
			runtime.ReadMemStats(&after)

			if tc.rejected {
				require.ErrorIs(t, err, xmlenc1.ErrMalformedEncrypted)
				require.Contains(t, err.Error(), ecPublicKeyOverLimit, "err=%v", err)
			}
			if !tc.rejected {
				require.NoError(t, err)
			}

			allocated := after.TotalAlloc - before.TotalAlloc
			require.Less(t, allocated, uint64(oneRead), "parsing %d lexical bytes allocated %d bytes", len(tc.value), allocated)
		})
	}
}

// TestBoundedBase64ErrorsNameTheirOwnValue pins that xenc:OAEPparams and an
// ECDH-ES dsig11:PublicKey each name THEMSELVES in all three refusals the
// bounded base64 walk they share can produce: the size limit, the child-kind
// rule, and the decoder.
//
// One walk reads both values and is told them apart only by the value name, the
// byte ceiling, and the decode-error phrase it is given. Nothing else in the
// error says which value a document got wrong — every parse failure here wraps
// ErrMalformedEncrypted — so the wording is all a caller diagnosing a rejected
// document has. Each case therefore asserts the OTHER value's wording is absent
// too: a walk handed one value's wording for the other would refuse exactly the
// same documents, so every size, child-kind and decoder assertion elsewhere
// would still pass.
func TestBoundedBase64ErrorsNameTheirOwnValue(t *testing.T) {
	// One octet over either ceiling encodes to a whole number of quanta, which
	// is never past the character ceiling, so both values are refused by the
	// exact decoded-length test that follows the walk.
	overLimitLabel := base64.StdEncoding.EncodeToString(make([]byte, xmlenc1.MaxOAEPParamsBytesForTest+1))
	overLimitPoint := base64.StdEncoding.EncodeToString(make([]byte, xmlenc1.MaxECPublicKeyBytesForTest+1))

	// notBase64 is one quantum of characters outside the base64 alphabet, so it
	// is far inside both ceilings and reaches the decoder, which refuses it.
	const notBase64 = "!!!!"

	// elementChild is a legal value followed by an element, which the child-kind
	// rule refuses for what it is, before its size is weighed.
	const elementChild = "AA==<junk>AAAA</junk>"

	for _, tc := range []struct {
		name string
		elem *helium.Element
		// want is the fragment naming the value the document got wrong, and
		// notWant is the same fragment for the OTHER value.
		want    string
		notWant string
	}{
		{
			name:    "an over-limit OAEPparams",
			elem:    oaepParamsEncryptedData(t, overLimitLabel),
			want:    oaepParamsOverLimit,
			notWant: ecPublicKeyOverLimit,
		},
		{
			name:    "an over-limit PublicKey",
			elem:    ecKeyValueEncryptedData(t, ecKeyValueChildren(ecCurveURIP256, overLimitPoint, true)),
			want:    ecPublicKeyOverLimit,
			notWant: oaepParamsOverLimit,
		},
		{
			name:    "an undecodable OAEPparams",
			elem:    oaepParamsEncryptedData(t, notBase64),
			want:    oaepParamsInvalid,
			notWant: ecPublicKeyInvalid,
		},
		{
			name:    "an undecodable PublicKey",
			elem:    ecKeyValueEncryptedData(t, ecKeyValueChildren(ecCurveURIP256, notBase64, true)),
			want:    ecPublicKeyInvalid,
			notWant: oaepParamsInvalid,
		},
		{
			name:    "an element child of OAEPparams",
			elem:    oaepParamsEncryptedData(t, elementChild),
			want:    oaepParamsChildOfType,
			notWant: ecPublicKeyChildOfType,
		},
		{
			name:    "an element child of PublicKey",
			elem:    ecKeyValueEncryptedData(t, ecKeyValueChildren(ecCurveURIP256, elementChild, true)),
			want:    ecPublicKeyChildOfType,
			notWant: oaepParamsChildOfType,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := xmlenc1.ParseEncryptedDataForTest(tc.elem)
			require.ErrorIs(t, err, xmlenc1.ErrMalformedEncrypted)
			require.Contains(t, err.Error(), tc.want, "err=%v", err)
			require.NotContains(t, err.Error(), tc.notWant, "err=%v", err)
		})
	}
}

// carriedKeyNameLexical is the lexical text a document writes as one
// xenc:CarriedKeyName below, and carriedKeyNameSlack is what the bound allows
// on top of a decrypt of the same document without that element.
//
// The slack is a CONSTANT for the same reason the OAEPparams slack is: a bound
// stated as a multiple of the attacker's own text could never fail on a cost
// that follows that text, and that cost is the only thing being pinned. It
// covers the rest of the document, the block decryption, and the runtime's own
// bookkeeping, all of which are fixed. Both figures stay far under the memory
// guard this package's tests run within.
const (
	carriedKeyNameLexical = 4 << 20
	carriedKeyNameSlack   = 1 << 20
)

// carriedKeyNamePayload is the plaintext every fixture below decrypts to, so a
// measurement that lost its success is caught rather than measured.
const carriedKeyNamePayload = "payload"

// TestCarriedKeyNameIsNotRead pins that the decrypt parse never materializes an
// EncryptedKey's xenc:CarriedKeyName.
//
// That name is metadata no decrypt path consults, nothing serializes, and no
// budget charges — neither the cumulative EncryptedKey ciphertext allowance nor
// the payload allowance covers it. It is also a value joined from every child at
// every depth the join reaches, so a document may spread one across as many CDATA
// sections as it likes and the XML parser's per-node content cap, which bounds
// one indivisible run of characters, says nothing about the total. Reading it
// would therefore cost one unbounded copy of attacker-supplied text on the
// SUCCESS path, and ahead of the SessionKey early return that makes the whole
// EncryptedKey irrelevant to the decrypt.
//
// The allocation case reads the process-wide TotalAlloc delta across
// DecryptBytes, so these subtests must NOT run in parallel: a concurrent test's
// allocations would pollute the delta.
func TestCarriedKeyNameIsNotRead(t *testing.T) {
	// no t.Parallel(): the deltas below must reflect only their own DecryptBytes.
	sessionKey := randKey(t, 32)

	t.Run("a document carrying one still decrypts", func(t *testing.T) {
		elem := carriedKeyNameEncryptedData(t, sessionKey, "session-key-1")

		plaintext, err := xmlenc1.NewDecryptor().SessionKey(sessionKey).DecryptBytes(t.Context(), elem)
		require.NoError(t, err)
		require.Equal(t, []byte(carriedKeyNamePayload), plaintext)

		// The element is SKIPPED, not refused: the EncryptedKey around it is
		// still a candidate the parse returns, and the field it would have
		// filled stays at its zero value.
		ed, err := xmlenc1.ParseEncryptedDataForTest(elem)
		require.NoError(t, err)
		require.Len(t, ed.EncryptedKeys, 1)
		require.Empty(t, ed.EncryptedKeys[0].CarriedKeyName)
	})

	t.Run("a large one costs the decrypt nothing", func(t *testing.T) {
		// Sixteen CDATA sections, which is how the name evades the parser's
		// per-node content cap: the cap bounds one run and every section is its
		// own run.
		name := splitIntoNodes(strings.Repeat("n", carriedKeyNameLexical), carriedKeyNameLexical/16)
		large := carriedKeyNameEncryptedData(t, sessionKey, name)
		absent := carriedKeyNameEncryptedData(t, sessionKey, "")

		largeAlloc := carriedKeyNameDecryptAlloc(t, sessionKey, large)
		absentAlloc := carriedKeyNameDecryptAlloc(t, sessionKey, absent)

		require.Less(t, largeAlloc, absentAlloc+carriedKeyNameSlack,
			"a %d byte CarriedKeyName cost the decrypt %d bytes against %d for the same document without one",
			carriedKeyNameLexical, largeAlloc, absentAlloc)
	})
}

// carriedKeyNameEncryptedData builds an EncryptedData that decrypts to
// carriedKeyNamePayload under sessionKey and whose single EncryptedKey candidate
// carries name as its raw xenc:CarriedKeyName markup. The markup is written
// straight into the document because the lexical form the parse is handed is the
// whole point; an empty name leaves the element out, which is the baseline the
// large case is measured against.
//
// The EncryptedKey's own ciphertext is never unwrapped: the decryptor is given
// the session key directly and returns before candidate key resolution, so the
// document is still parsed in full and no key-encryption key is needed.
func carriedKeyNameEncryptedData(t *testing.T, sessionKey []byte, name string) *helium.Element {
	t.Helper()
	cipher, err := xmlenc1.EncryptBytesForTest(xmlenc1.AES256GCM, sessionKey, []byte(carriedKeyNamePayload))
	require.NoError(t, err)

	var carried string
	if name != "" {
		carried = `<xenc:CarriedKeyName>` + name + `</xenc:CarriedKeyName>`
	}

	doc := mustParseXML(t, `<xenc:EncryptedData xmlns:xenc="`+xmlenc1.NamespaceXMLEnc+`" xmlns:ds="`+xmlenc1.NamespaceDSig+`">`+
		`<xenc:EncryptionMethod Algorithm="`+xmlenc1.AES256GCM+`"/>`+
		`<ds:KeyInfo><xenc:EncryptedKey>`+
		`<xenc:EncryptionMethod Algorithm="`+xmlenc1.AES256KeyWrap+`"/>`+
		`<xenc:CipherData><xenc:CipherValue>`+base64.StdEncoding.EncodeToString(make([]byte, 40))+`</xenc:CipherValue></xenc:CipherData>`+
		carried+
		`</xenc:EncryptedKey></ds:KeyInfo>`+
		`<xenc:CipherData><xenc:CipherValue>`+base64.StdEncoding.EncodeToString(cipher)+`</xenc:CipherValue></xenc:CipherData>`+
		`</xenc:EncryptedData>`)
	return doc.DocumentElement()
}

// carriedKeyNameDecryptAlloc reports the TotalAlloc delta across one successful
// DecryptBytes of elem. The document and the decryptor are built outside the
// measurement, so the delta is the decrypt alone.
func carriedKeyNameDecryptAlloc(t *testing.T, sessionKey []byte, elem *helium.Element) uint64 {
	t.Helper()
	decryptor := xmlenc1.NewDecryptor().SessionKey(sessionKey)

	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	plaintext, err := decryptor.DecryptBytes(t.Context(), elem)
	runtime.ReadMemStats(&after)

	require.NoError(t, err)
	require.Equal(t, []byte(carriedKeyNamePayload), plaintext)
	return after.TotalAlloc - before.TotalAlloc
}
