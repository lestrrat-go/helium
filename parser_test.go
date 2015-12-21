package helium

import (
	"testing"

	"github.com/lestrrat/helium/internal/debug"
	"github.com/stretchr/testify/assert"
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
			ctx.init(p, input)
			enc, err := ctx.detectEncoding()
			if expected == "" {
				t.Logf("checking [invalid] (%d)", i)
				if !assert.Error(t, err, "detectEncoding should fail for sequence %#v", input) {
					return
				}
			} else {
				t.Logf("checking %s (%d)", expected, i)
				if !assert.NoError(t, err, "detectEncoding should succeed for sequence %#v", input) {
					return
				}

				if !assert.Equal(t, expected, enc, "detectEncoding returns as expected") {
					return
				}
			}
		}
	}
}

func TestEmptyDocument(t *testing.T) {
	p := NewParser()
	// BOM only
	_, err := p.Parse([]byte{0x00, 0x00, 0x00, 0x3C})
	if !assert.Error(t, err, "Parsing BOM only should fail") {
		return
	}
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
		if !assert.NoError(t, err, "Parse should succeed for '%s'", input) {
			return
		}

		if !assert.Equal(t, expect.version, doc.Version(), "version matches") {
			return
		}
		if !assert.Equal(t, expect.encoding, doc.Encoding(), "encoding matches") {
			return
		}
		if !assert.Equal(t, expect.standalone, int(doc.Standalone()), "standalone matches") {
			return
		}
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
		if !assert.NoError(t, err, "Parse should succeed for '%s'", input) {
			return
		}

		if debug.Enabled {
			debug.Dump(doc)
		}

		pi := doc.FirstChild()
		if !assert.IsType(t, &ProcessingInstruction{}, pi, "First child should be a processing instruction") {
			return
		}

		if !assert.IsType(t, &Element{}, pi.NextSibling(), "NextSibling of PI should be Element node") {
			return
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
	if !assert.NoError(t, err, "Parse should succeed for '%s'", input) {
		return
	}
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
		if !assert.Error(t, err, "Parse should fail for '%s'", input) {
			return
		}
	}
}

func TestParseNamespace(t *testing.T) {
	const input = `<?xml version="1.0"?>
<helium:root xmlns:helium="https://github.com/lestrrat/helium">
  <helium:child>foo</helium:child>
</helium:root>`
	p := NewParser()
	doc, err := p.Parse([]byte(input))
	if !assert.NoError(t, err, "Parse should succeed for '%s'", input) {
		return
	}

	if debug.Enabled {
		debug.Dump(doc)
	}
}
