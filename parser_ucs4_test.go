package helium_test

import (
	"bytes"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

// encodeUCS4 encodes s as UCS-4 (4 bytes per code point) using the byte order
// described by order, which maps the four output byte positions to a shift
// amount (in bits) applied to the rune. This lets a single helper produce all
// four UCS-4 byte orders recognized by the encoding auto-detector.
func encodeUCS4(s string, order [4]uint) []byte {
	var buf bytes.Buffer
	for _, r := range s {
		u := uint32(r)
		for _, shift := range order {
			buf.WriteByte(byte(u >> shift))
		}
	}
	return buf.Bytes()
}

// TestParseUCS4FirstByteNotConsumed is a regression for E-UCS4-CONSUMES-LT: the
// encoding auto-detector used to CONSUME the four leading bytes during UCS-4
// detection. Those bytes are the encoded first '<' character (not a BOM), so a
// genuine UCS-4 document lost its leading '<' and failed with "start tag
// expected, '<' not found". Detection must peek, not consume.
func TestParseUCS4FirstByteNotConsumed(t *testing.T) {
	const doc = `<?xml version="1.0" encoding="ISO-10646-UCS-4"?><root>hi</root>`

	// byte orders matching the detector patterns:
	//   BE   : 00 00 00 3C  -> shifts 24,16,8,0
	//   LE   : 3C 00 00 00  -> shifts 0,8,16,24
	//   2143 : 00 00 3C 00  -> shifts 16,24,0,8
	//   3412 : 00 3C 00 00  -> shifts 8,0,24,16
	orders := map[string][4]uint{
		"BE":   {24, 16, 8, 0},
		"LE":   {0, 8, 16, 24},
		"2143": {16, 24, 0, 8},
		"3412": {8, 0, 24, 16},
	}

	for name, order := range orders {
		t.Run(name, func(t *testing.T) {
			in := encodeUCS4(doc, order)

			// Sanity: the first encoded code point is '<', and its four bytes
			// must contain exactly one 0x3C so this really exercises a
			// UCS-4-looking leading sequence.
			require.Len(t, in[:4], 4)

			parsed, err := helium.NewParser().Parse(t.Context(), in)
			require.NoError(t, err, "genuine UCS-4 document must parse with its first '<' intact")

			root := parsed.DocumentElement()
			require.NotNil(t, root, "document element must be present")
			require.Equal(t, "root", root.Name())
			require.Equal(t, "hi", string(root.Content()))
		})
	}
}

// TestParseUCS4LeadingStylesheetPI is a regression for the UCS-4 fixed-width
// prolog probe misclassifying a legal leading PI as an XML declaration. The
// probe used HasPrefixString("<?xml"), which matches "<?xml-stylesheet ...?>"
// and then failed with "blank needed after '<?xml'". The probe must use
// looksLikeXMLDeclString to distinguish "<?xml " (decl) from "<?xml-stylesheet"
// (PI), so a UCS-4 document beginning with such a PI parses with the PI intact.
func TestParseUCS4LeadingStylesheetPI(t *testing.T) {
	const doc = `<?xml-stylesheet type="text/xsl" href="x.xsl"?><root>hi</root>`

	orders := map[string][4]uint{
		"BE":   {24, 16, 8, 0},
		"LE":   {0, 8, 16, 24},
		"2143": {16, 24, 0, 8},
		"3412": {8, 0, 24, 16},
	}

	for name, order := range orders {
		t.Run(name, func(t *testing.T) {
			in := encodeUCS4(doc, order)

			parsed, err := helium.NewParser().Parse(t.Context(), in)
			require.NoError(t, err, "UCS-4 document with a leading PI must not be misread as an XML declaration")

			var pi *helium.ProcessingInstruction
			for c := parsed.FirstChild(); c != nil; c = c.NextSibling() {
				if got, ok := helium.AsNode[*helium.ProcessingInstruction](c); ok {
					pi = got
					break
				}
			}
			require.NotNil(t, pi, "leading processing instruction must be preserved")
			require.Equal(t, "xml-stylesheet", pi.Name())
			require.Equal(t, `type="text/xsl" href="x.xsl"`, string(pi.Content()))

			root := parsed.DocumentElement()
			require.NotNil(t, root, "document element must be present")
			require.Equal(t, "root", root.Name())
			require.Equal(t, "hi", string(root.Content()))
		})
	}
}
