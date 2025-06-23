package helium

import (
	"bytes"
	"testing"

	"github.com/lestrrat-go/pdebug"
	"github.com/stretchr/testify/require"
)

func TestDetectBOM(t *testing.T) {
	data := map[string][][]byte{
		encUCS4BE:   {{0x00, 0x00, 0x00, 0x3C}},
		encUCS4LE:   {{0x3C, 0x00, 0x00, 0x00}},
		encUCS42143: {{0x00, 0x00, 0x3C, 0x00}},
		encUCS43412: {{0x00, 0x3C, 0x00, 0x00}},
		encEBCDIC:   {{0x4C, 0x6F, 0xA7, 0x94}},
		encUTF8:     {{0x3C, 0x3F, 0x78, 0x6D}, {0xEF, 0xBB, 0xBF}},
		encUTF16LE:  {{0x3C, 0x00, 0x3F, 0x00}, {0xFF, 0xFE}},
		encUTF16BE:  {{0x00, 0x3C, 0x00, 0x3F}, {0xFE, 0xFF}},
		"":          {{0xde, 0xad, 0xbe, 0xef}},
	}

	p := NewParser() // DUMMY
	for expected, inputs := range data {
		for i, input := range inputs {
			ctx := &parserCtx{}
			require.NoError(t, ctx.init(p, bytes.NewReader(input)))
			enc, err := ctx.detectEncoding()
			if expected == "" {
				t.Logf("checking [invalid] (%d)", i)
				require.Error(t, err, "detectEncoding should fail for sequence %#v", input)
			} else {
				t.Logf("checking %s (%d)", expected, i)
				require.NoError(t, err, "detectEncoding should succeed for sequence %#v", input)

				require.Equal(t, expected, enc, "detectEncoding returns as expected")
			}
		}
	}
}

func TestEmptyDocument(t *testing.T) {
	p := NewParser()
	// BOM only
	_, err := p.Parse([]byte{0x00, 0x00, 0x00, 0x3C})
	require.Error(t, err, "Parsing BOM only should fail")
}

func TestParseXMLDecl(t *testing.T) {
	const content = `<root />`
	inputs := map[string]struct {
		version    string
		encoding   string
		standalone int
	}{
		`<?xml version="1.0"?>` + content:                                   {"1.0", "utf8", int(StandaloneImplicitNo)},
		`<?xml version="1.0" encoding="euc-jp"?>` + content:                 {"1.0", "euc-jp", int(StandaloneImplicitNo)},
		`<?xml version="1.0" encoding="cp932" standalone='yes'?>` + content: {"1.0", "cp932", int(StandaloneExplicitYes)},
	}

	for input, expect := range inputs {
		p := NewParser()
		doc, err := p.Parse([]byte(input))
		require.NoError(t, err, "Parse should succeed for '%s'", input)

		require.Equal(t, expect.version, doc.Version(), "version matches")
		require.Equal(t, expect.encoding, doc.Encoding(), "encoding matches")
		require.Equal(t, expect.standalone, int(doc.Standalone()), "standalone matches")
	}
}

func TestParseMisc(t *testing.T) {
	const decl = `<?xml version="1.0"?>` + "\n"
	const content = `<root />`
	inputs := []string{
		decl + `<?xml-stylesheet type="text/xsl" href="style.xsl"?>` + content,
		decl + `<?xml-stylesheet type="text/css" href="style.css"?>` + content,
	}

	for _, input := range inputs {
		p := NewParser()
		doc, err := p.Parse([]byte(input))
		require.NoError(t, err, "Parse should succeed for '%s'", input)

		// XXX Not sure if this is right, but I'm going to assume it's ok
		// to have a DTD in the child list
		n := doc.FirstChild()
	LOOP:
		for {
			t.Logf("%#v", n)
			if n == nil {
				t.Errorf("Could not find ProcessingInstruction node")
				return
			}

			switch n.Type() {
			case ProcessingInstructionNode:
				require.IsType(t, &ProcessingInstruction{}, n, "First child should be a processing instruction")

				require.IsType(t, &Element{}, n.NextSibling(), "NextSibling of PI should be Element node")
				break LOOP
			}
			n = n.NextSibling()
		}
	}
}

func TestParse(t *testing.T) {
	const input = `<?xml version="1.0"?>
<root foo="bar">
	<!-- this is a sample comment -->
  <child>foo</child>
	<child><![CDATA[
H
E
L
L
O!]]></child>
</root>`
	p := NewParser()
	_, err := p.Parse([]byte(input))
	require.NoError(t, err, "Parse should succeed for '%s'", input)
}

func TestParseBad(t *testing.T) {
	inputs := []string{
		`<?xml version="1.0"?>
<root foo="bar">
  <child>foo</chld>
</root>`,
		`<?xml version="abc">`,
		`<?xml varsion="1.0">`,
	}
	p := NewParser()
	for _, input := range inputs {
		_, err := p.Parse([]byte(input))
		require.Error(t, err, "Parse should fail for '%s'", input)
	}
}

func TestParseNamespace(t *testing.T) {
	const input = `<?xml version="1.0"?>
<helium:root xmlns:helium="https://github.com/lestrrat-go/helium">
  <helium:child>foo</helium:child>
</helium:root>`
	p := NewParser()
	doc, err := p.Parse([]byte(input))
	require.NoError(t, err, "Parse should succeed for '%s'", input)

	if pdebug.Enabled {
		pdebug.Dump(doc)
	}
}
