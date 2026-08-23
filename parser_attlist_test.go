package helium_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/enum"
	"github.com/stretchr/testify/require"
)

func parseWithDTDAttributeType(t *testing.T, typ enum.AttributeType, value string) error {
	t.Helper()

	var docDecl string
	var extraDecl string
	var body string
	var typeName string

	switch typ {
	case enum.AttrID:
		docDecl = "<!ELEMENT doc EMPTY>"
		body = fmt.Sprintf(`<doc attr=%q/>`, value)
		typeName = "ID"
	case enum.AttrNmtoken:
		docDecl = "<!ELEMENT doc EMPTY>"
		body = fmt.Sprintf(`<doc attr=%q/>`, value)
		typeName = "NMTOKEN"
	case enum.AttrNmtokens:
		docDecl = "<!ELEMENT doc EMPTY>"
		body = fmt.Sprintf(`<doc attr=%q/>`, value)
		typeName = "NMTOKENS"
	case enum.AttrIDRefs:
		docDecl = "<!ELEMENT doc (item*)>"
		extraDecl = "<!ELEMENT item EMPTY>\n  <!ATTLIST item id ID #IMPLIED>"
		body = fmt.Sprintf(`<doc attr=%q><item id="id1"/><item id="id2"/></doc>`, value)
		typeName = "IDREFS"
	case enum.AttrCDATA:
		docDecl = "<!ELEMENT doc EMPTY>"
		body = fmt.Sprintf(`<doc attr=%q/>`, value)
		typeName = "CDATA"
	default:
		t.Fatalf("unsupported attr type: %v", typ)
	}

	input := fmt.Sprintf(`<?xml version="1.0"?>
<!DOCTYPE doc [
  %s
  %s
  <!ATTLIST doc attr %s #IMPLIED>
]>
%s`, docDecl, extraDecl, typeName, body)

	p := helium.NewParser().ValidateDTD(true)
	_, err := p.Parse(t.Context(), []byte(input))
	return err
}

func TestAttlistDecl(t *testing.T) {
	t.Parallel()

	// XML 1.0 §3.3: AttlistDecl ::= '<!ATTLIST' S Name AttDef* S? '>'. AttDef* is
	// zero-or-more, so an empty attribute-list declaration `<!ATTLIST name>` (with
	// or without trailing whitespace) declares no attributes and is well-formed.
	t.Run("an empty ATTLIST", func(t *testing.T) {
		testcases := []struct {
			name string
			doc  string
		}{
			{
				name: "no trailing space",
				doc: `<!DOCTYPE r [
<!ELEMENT r EMPTY>
<!ATTLIST r>
]>
<r/>`,
			},
			{
				name: "trailing space",
				doc: `<!DOCTYPE r [
<!ELEMENT r EMPTY>
<!ATTLIST r >
]>
<r/>`,
			},
			{
				name: "empty then non-empty for same element",
				doc: `<!DOCTYPE r [
<!ELEMENT r (#PCDATA)>
<!ATTLIST r>
<!ATTLIST r a CDATA #IMPLIED>
]>
<r a="x">y</r>`,
			},
		}

		for _, tc := range testcases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				parsed, err := helium.NewParser().Parse(t.Context(), []byte(tc.doc))
				require.NoError(t, err, "an empty <!ATTLIST> is well-formed and must parse")
				require.NotNil(t, parsed)
				require.NotNil(t, parsed.DocumentElement())
			})
		}
	})

	// A non-empty ATTLIST still parses and its declared default is applied, proving
	// the empty-list handling did not regress the AttDef loop.
	t.Run("a non-empty ATTLIST still works", func(t *testing.T) {
		const doc = `<!DOCTYPE r [
<!ELEMENT r (#PCDATA)>
<!ATTLIST r a CDATA "def">
]>
<r>x</r>`

		parsed, err := helium.NewParser().
			DefaultDTDAttributes(true).
			Parse(t.Context(), []byte(doc))
		require.NoError(t, err)
		a, ok := parsed.DocumentElement().GetAttribute("a")
		require.True(t, ok, "the declared default attribute must be present")
		require.Equal(t, "def", a)
	})

	// An ATTLIST with no element Name at all is malformed and must still be
	// rejected — the mandatory `S Name` after `<!ATTLIST` is unaffected by allowing
	// an empty AttDef list.
	t.Run("an ATTLIST without an element name is rejected", func(t *testing.T) {
		const doc = `<!DOCTYPE r [
<!ELEMENT r EMPTY>
<!ATTLIST>
]>
<r/>`

		_, err := helium.NewParser().Parse(t.Context(), []byte(doc))
		require.Error(t, err, "an <!ATTLIST> with no element name must be rejected")
	})

	// XML 1.0 §3.3: "When more than one definition is provided for the same
	// attribute of a given element type, the first declaration is binding and later
	// declarations are ignored." A repeated <!ATTLIST> for the same attribute is a
	// validity warning, not a fatal error — libxml2 accepts such documents. helium
	// must accept them too and keep the first declaration's default.
	t.Run("a duplicate ATTLIST declaration", func(t *testing.T) {
		const doc = `<!DOCTYPE doc [
<!ELEMENT doc (#PCDATA)>
<!ATTLIST doc a1 CDATA "first">
<!ATTLIST doc a1 CDATA "second">
]>
<doc></doc>`

		parsed, err := helium.NewParser().
			DefaultDTDAttributes(true).
			Parse(t.Context(), []byte(doc))
		require.NoError(t, err, "a duplicate ATTLIST attribute definition must not be a fatal error")
		require.NotNil(t, parsed)

		root := parsed.DocumentElement()
		require.NotNil(t, root)
		// The first declaration is binding: the defaulted value is "first".
		a1, ok := root.GetAttribute("a1")
		require.True(t, ok, "the defaulted attribute from the first ATTLIST must be present")
		require.Equal(t, "first", a1)
	})

	// A later, ignored duplicate declaration must not have its (possibly invalid)
	// default value validated — §3.3 ignores the whole declaration, so an invalid
	// default in the duplicate cannot abort the parse.
	t.Run("a duplicate ignores an invalid duplicate default", func(t *testing.T) {
		const doc = `<!DOCTYPE doc [
<!ELEMENT doc (#PCDATA)>
<!ATTLIST doc a1 CDATA "ok">
<!ATTLIST doc a1 NMTOKEN "not a token">
]>
<doc></doc>`

		parsed, err := helium.NewParser().
			DefaultDTDAttributes(true).
			Parse(t.Context(), []byte(doc))
		require.NoError(t, err, "the ignored duplicate's invalid default must not abort the parse")
		require.NotNil(t, parsed)
		a1, ok := parsed.DocumentElement().GetAttribute("a1")
		require.True(t, ok)
		require.Equal(t, "ok", a1, "the first (CDATA) declaration is binding")
	})

	// The first declaration's type governs attribute-value normalization; a later
	// duplicate with a different type must not change it. First CDATA keeps explicit
	// whitespace; a duplicate NMTOKEN must not cause collapsing.
	t.Run("a duplicate type does not affect normalization", func(t *testing.T) {
		const doc = `<!DOCTYPE doc [
<!ELEMENT doc (#PCDATA)>
<!ATTLIST doc a2 CDATA #IMPLIED>
<!ATTLIST doc a2 NMTOKEN #IMPLIED>
]>
<doc a2="  x   y  "></doc>`

		parsed, err := helium.NewParser().Parse(t.Context(), []byte(doc))
		require.NoError(t, err)
		a2, ok := parsed.DocumentElement().GetAttribute("a2")
		require.True(t, ok)
		// CDATA (the first, binding type) preserves the explicit whitespace; had the
		// duplicate NMTOKEN won, the value would have been collapsed to "x y".
		require.Equal(t, "  x   y  ", a2)
	})

	t.Run("duplicate enumeration tokens", func(t *testing.T) {
		t.Run("enumeration with duplicate token", func(t *testing.T) {
			t.Parallel()

			const input = `<?xml version="1.0"?>
<!DOCTYPE r [<!ATTLIST r color (red|red) "red">]>
<r/>`

			p := helium.NewParser()
			_, err := p.Parse(t.Context(), []byte(input))

			require.Error(t, err, "duplicate enumeration token should be rejected")
			var dup helium.DTDDupTokenError
			require.True(t, errors.As(err, &dup), "error should be DTDDupTokenError")
			require.Equal(t, "red", dup.Name)
		})

		t.Run("notation with duplicate token", func(t *testing.T) {
			t.Parallel()

			const input = `<?xml version="1.0"?>
<!DOCTYPE r [
  <!NOTATION n PUBLIC "pub-n">
  <!ATTLIST r kind NOTATION (n|n) #IMPLIED>
]>
<r/>`

			p := helium.NewParser()
			_, err := p.Parse(t.Context(), []byte(input))

			require.Error(t, err, "duplicate notation token should be rejected")
			var dup helium.DTDDupTokenError
			require.True(t, errors.As(err, &dup), "error should be DTDDupTokenError")
			require.Equal(t, "n", dup.Name)
		})

		t.Run("enumeration with distinct tokens", func(t *testing.T) {
			t.Parallel()

			const input = `<?xml version="1.0"?>
<!DOCTYPE r [<!ATTLIST r color (red|green) "red">]>
<r/>`

			p := helium.NewParser()
			_, err := p.Parse(t.Context(), []byte(input))
			require.NoError(t, err, "distinct enumeration tokens should parse")
		})
	})
}

func TestValidateAttributeValue(t *testing.T) {
	t.Parallel()

	// Each case is validated against the type declared for it in the DTD.
	tests := []struct {
		name    string
		typ     enum.AttributeType
		value   string
		wantErr bool
	}{
		{name: "ID valid", typ: enum.AttrID, value: "myid"},
		{name: "ID invalid", typ: enum.AttrID, value: "123", wantErr: true},
		{name: "NMTOKEN valid", typ: enum.AttrNmtoken, value: "hello-world"},
		{name: "NMTOKEN valid digits", typ: enum.AttrNmtoken, value: "123"},
		{name: "NMTOKEN invalid", typ: enum.AttrNmtoken, value: "hello world", wantErr: true},
		{name: "NMTOKENS valid", typ: enum.AttrNmtokens, value: "hello world"},
		{name: "IDREFS valid", typ: enum.AttrIDRefs, value: "id1 id2"},
		{name: "IDREFS invalid", typ: enum.AttrIDRefs, value: "id1 123", wantErr: true},
		{name: "CDATA anything", typ: enum.AttrCDATA, value: "anything goes here!"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := parseWithDTDAttributeType(t, tc.typ, tc.value)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}
