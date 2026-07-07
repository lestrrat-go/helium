package xpath3_test

import (
	"testing"

	"github.com/lestrrat-go/helium/xpath3"
	"github.com/stretchr/testify/require"
)

// serializeParamCase drives fn:serialize the way the W3C QT3 fn/serialize tests
// do and asserts on the produced string. When paramsXML is set, it is parsed and
// its document element is bound to $params (the element-options form, invoked as
// serialize(., $params)); otherwise expr is a self-contained call using an
// inline options map.
// exprSerializeWithParams is the element-options invocation form used by the
// QT3 *b test variants: the serialization parameters come from the bound
// $params element.
const exprSerializeWithParams = `serialize(., $params)`

type serializeParamCase struct {
	name      string
	sourceXML string
	paramsXML string
	expr      string
	contains  []string
	absent    []string
	matches   []string
	unmatched []string
}

// TestSerialize_ApplyParams replicates the QT3 fn:serialize cases whose
// serialization parameters must actually take effect (not merely parse):
// suppress-indentation, standalone, use-character-maps, undeclare-prefixes,
// cdata-section-elements, the map-form omit-xml-declaration default, and the
// html output method's DOCTYPE/meta emission.
func TestSerialize_ApplyParams(t *testing.T) {
	const src008 = `<doc>
  <title>A document</title>
  <p>A paragraph containing some <code>code</code> which should not be indented</p>
</doc>`
	const params008 = `<output:serialization-parameters xmlns:output="http://www.w3.org/2010/xslt-xquery-serialization">` +
		`<output:method value="xml"/><output:indent value="yes"/><output:suppress-indentation value="p"/>` +
		`</output:serialization-parameters>`

	const srcAtomic = `<?xml version="1.0" encoding="UTF-8"?><root>text</root>`
	const params029 = `<output:serialization-parameters xmlns:output="http://www.w3.org/2010/xslt-xquery-serialization">` +
		`<output:omit-xml-declaration value="no"/><output:standalone value=" no "/>` +
		`</output:serialization-parameters>`
	const params030 = `<output:serialization-parameters xmlns:output="http://www.w3.org/2010/xslt-xquery-serialization">` +
		`<output:omit-xml-declaration value="no"/><output:standalone value=" yes "/>` +
		`</output:serialization-parameters>`

	const src032 = `<doc><title>A document</title><p>A paragraph containing a character $ which should be mapped</p></doc>`
	const params032 = `<output:serialization-parameters xmlns:output="http://www.w3.org/2010/xslt-xquery-serialization">` +
		`<output:use-character-maps><output:character-map character="$" map-string="` + "£" + `"/></output:use-character-maps>` +
		`</output:serialization-parameters>`

	const src035 = `<?xml version="1.1" encoding="UTF-8"?>` +
		`<p:chapter xmlns:p="http://example.com/p"><section xmlns:p=""><para/></section></p:chapter>`
	const params035 = `<output:serialization-parameters xmlns:output="http://www.w3.org/2010/xslt-xquery-serialization">` +
		`<output:method value="xml"/><output:version value="1.1"/><output:undeclare-prefixes value="yes"/>` +
		`</output:serialization-parameters>`

	const src110 = `<?xml version="1.0" encoding="UTF-8"?><doc>` +
		`<chapter xmlns:p="http://www.example.org/ns/p"><section><para>` +
		`<b>bold</b><i>italic</i><p:b>BOLD</p:b><p:i>ITALIC</p:i></para></section></chapter></doc>`

	const src138 = `<?xml version="1.0" encoding="UTF-8"?><e>xml</e>`

	const srcHTML = `<?xml version="1.0" encoding="UTF-8"?><html><head/><body><p>Hello World!</p></body></html>`

	cases := []serializeParamCase{
		{
			name:      "xml-008b suppress-indentation (element form)",
			sourceXML: src008,
			paramsXML: params008,
			expr:      exprSerializeWithParams,
			matches:   []string{`\n\s+<title>`, `\n\s+<p>`},
			unmatched: []string{`\n\s+<code>`},
		},
		{
			name:      "xml-108 suppress-indentation (map form)",
			sourceXML: src008,
			expr:      `serialize(., map{"method":"xml","indent":true(),"suppress-indentation":QName("","p")})`,
			matches:   []string{`\n\s+<title>`, `\n\s+<p>`},
			unmatched: []string{`\n\s+<code>`},
		},
		{
			name:      "xml-029b standalone no (element form)",
			sourceXML: srcAtomic,
			paramsXML: params029,
			expr:      exprSerializeWithParams,
			contains:  []string{`standalone="no"`},
		},
		{
			name:      "xml-030b standalone yes (element form)",
			sourceXML: srcAtomic,
			paramsXML: params030,
			expr:      exprSerializeWithParams,
			contains:  []string{`standalone="yes"`},
		},
		{
			name:      "xml-129 standalone false (map form)",
			sourceXML: srcAtomic,
			expr:      `serialize(., map{"omit-xml-declaration":false(),"standalone":false()})`,
			contains:  []string{`standalone="no"`},
		},
		{
			name:      "xml-130 standalone true (map form)",
			sourceXML: srcAtomic,
			expr:      `serialize(., map{"omit-xml-declaration":false(),"standalone":true()})`,
			contains:  []string{`standalone="yes"`},
		},
		{
			name:      "xml-032b use-character-maps (element form)",
			sourceXML: src032,
			paramsXML: params032,
			expr:      exprSerializeWithParams,
			contains:  []string{"£"},
			absent:    []string{"$"},
		},
		{
			name:      "xml-132 use-character-maps (map form)",
			sourceXML: src032,
			expr:      `serialize(., map{"use-character-maps": map{"$":"` + "£" + `"}})`,
			contains:  []string{"£"},
			absent:    []string{"$"},
		},
		{
			name:      "xml-138b use-character-maps from JSON options",
			sourceXML: src138,
			expr:      `serialize(., parse-json('{"method" : "xml", "indent" : true, "use-character-maps" : { "x" : "j", "m" : "so", "l" : "n" } }'))`,
			contains:  []string{`<e>json</e>`},
		},
		{
			name:      "xml-035b undeclare-prefixes (element form)",
			sourceXML: src035,
			paramsXML: params035,
			expr:      exprSerializeWithParams,
			matches:   []string{`section xmlns:p=["']["']`},
		},
		{
			name:      "xml-135 undeclare-prefixes (map form)",
			sourceXML: src035,
			expr:      `serialize(., map{"method":"xml","version":"1.1","undeclare-prefixes":true()})`,
			matches:   []string{`section xmlns:p=["']["']`},
		},
		{
			name:      "xml-110 cdata-section-elements (map form)",
			sourceXML: src110,
			expr: `serialize(., map{"method":"xml",` +
				`"cdata-section-elements":(QName("","b"), QName("http://www.example.org/ns/p","b")),` +
				`"suppress-indentation":QName("","para")})`,
			contains: []string{"CDATA[bold]", "CDATA[BOLD]"},
			absent:   []string{"CDATA[italic]", "CDATA[ITALIC]"},
		},
		{
			name:      "xml-127a omit-xml-declaration default for empty map",
			sourceXML: srcAtomic,
			expr:      `serialize(., map{})`,
			absent:    []string{"<?xml"},
		},
		{
			name:      "html-001b method html DOCTYPE + meta",
			sourceXML: srcHTML,
			expr:      `serialize(., map{"method":"html","html-version":5.0})`,
			matches:   []string{`DOCTYPE (HTML|html)`, `<meta http-equiv`},
		},
		{
			name:      "html-002b method html html-version as integer",
			sourceXML: srcHTML,
			expr:      `serialize(., map{"method":"html","html-version":5})`,
			matches:   []string{`DOCTYPE (HTML|html)`, `<meta http-equiv`},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			doc := mustParseXML(t, tc.sourceXML)

			eval := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions)
			if tc.paramsXML != "" {
				paramsDoc := mustParseXML(t, tc.paramsXML)
				paramsRes, err := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
					Evaluate(t.Context(), mustCompile(t, `.`), paramsDoc.DocumentElement())
				require.NoError(t, err)
				eval = eval.Variables(map[string]xpath3.Sequence{"params": paramsRes.Sequence()})
			}

			res, err := eval.Evaluate(t.Context(), mustCompile(t, tc.expr), doc)
			require.NoError(t, err)
			out := res.StringValue()

			for _, want := range tc.contains {
				require.Contains(t, out, want, "output:\n%s", out)
			}
			for _, notWant := range tc.absent {
				require.NotContains(t, out, notWant, "output:\n%s", out)
			}
			for _, pat := range tc.matches {
				require.Regexp(t, pat, out, "output:\n%s", out)
			}
			for _, pat := range tc.unmatched {
				require.NotRegexp(t, pat, out, "output:\n%s", out)
			}
		})
	}
}
