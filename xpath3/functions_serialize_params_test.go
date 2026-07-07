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
		{
			// Element-form cdata-section-elements accepts an EQName Q{uri}local
			// and matches by exact expanded name: the namespaced <p:b> is CDATA,
			// the no-namespace <b> is not.
			name:      "element-form cdata EQName Q{uri}local matches namespaced element",
			sourceXML: `<doc><b>plain</b><p:b xmlns:p="urn:p">nsd</p:b></doc>`,
			paramsXML: `<output:serialization-parameters xmlns:output="http://www.w3.org/2010/xslt-xquery-serialization">` +
				`<output:method value="xml"/><output:cdata-section-elements value="Q{urn:p}b"/>` +
				`</output:serialization-parameters>`,
			expr:     exprSerializeWithParams,
			contains: []string{"CDATA[nsd]"},
			absent:   []string{"CDATA[plain]"},
		},
		{
			// Element-form use-character-maps: an empty @map-string maps the
			// character to an empty replacement (deletion).
			name:      "element-form use-character-maps empty map-string deletes char",
			sourceXML: `<e>xax</e>`,
			paramsXML: `<output:serialization-parameters xmlns:output="http://www.w3.org/2010/xslt-xquery-serialization">` +
				`<output:method value="xml"/><output:omit-xml-declaration value="yes"/>` +
				`<output:use-character-maps><output:character-map character="x" map-string=""/></output:use-character-maps>` +
				`</output:serialization-parameters>`,
			expr:     exprSerializeWithParams,
			contains: []string{"<e>a</e>"},
			absent:   []string{"x"},
		},
		{
			// An ABSENT @map-string is equivalent to an empty one (deletion).
			name:      "element-form use-character-maps absent map-string deletes char",
			sourceXML: `<e>xax</e>`,
			paramsXML: `<output:serialization-parameters xmlns:output="http://www.w3.org/2010/xslt-xquery-serialization">` +
				`<output:method value="xml"/><output:omit-xml-declaration value="yes"/>` +
				`<output:use-character-maps><output:character-map character="x"/></output:use-character-maps>` +
				`</output:serialization-parameters>`,
			expr:     exprSerializeWithParams,
			contains: []string{"<e>a</e>"},
			absent:   []string{"x"},
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
				eval = eval.Variables(map[string]xpath3.Sequence{paramsVar: paramsRes.Sequence()})
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

// TestSerialize_ParamNegativeCases locks in the spec-correctness boundaries the
// applied serialization parameters must respect (gauntlet findings on
// fix-qt3-serialize-params):
//   - standalone: an explicit "omit" (and the Serialization default) forces the
//     standalone pseudo-attribute OFF even when the source declaration carried
//     standalone="yes".
//   - cdata-section-elements / suppress-indentation match by EXACT expanded
//     name: QName("","b") must not match a namespaced <p:b>.
//   - undeclare-prefixes at output version 1.0 is a SEPM0010 static error.
//   - method="html" emits the HTML5 DOCTYPE + Content-Type meta ONLY when the
//     document element is <html>; an <article><head/> root gets neither.
func TestSerialize_ParamNegativeCases(t *testing.T) {
	t.Run("standalone omit drops a source standalone=yes", func(t *testing.T) {
		doc := mustParseXML(t, `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><root>text</root>`)
		res, err := evaluate(t.Context(), doc,
			`serialize(., map{"omit-xml-declaration":false(),"standalone":"omit"})`)
		require.NoError(t, err)
		require.NotContains(t, res.StringValue(), "standalone", "output:\n%s", res.StringValue())
	})

	t.Run("standalone default drops a source standalone=yes", func(t *testing.T) {
		doc := mustParseXML(t, `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><root>text</root>`)
		res, err := evaluate(t.Context(), doc, `serialize(., map{"omit-xml-declaration":false()})`)
		require.NoError(t, err)
		require.NotContains(t, res.StringValue(), "standalone", "output:\n%s", res.StringValue())
	})

	t.Run("cdata QName no-namespace does not match namespaced element", func(t *testing.T) {
		const src = `<doc><b>plain</b><p:b xmlns:p="urn:p">nsd</p:b></doc>`
		doc := mustParseXML(t, src)
		res, err := evaluate(t.Context(), doc,
			`serialize(., map{"method":"xml","cdata-section-elements":QName("","b")})`)
		require.NoError(t, err)
		out := res.StringValue()
		// The no-namespace <b> is CDATA-wrapped; the namespaced <p:b> is not.
		require.Contains(t, out, "CDATA[plain]", "output:\n%s", out)
		require.NotContains(t, out, "CDATA[nsd]", "output:\n%s", out)
	})

	t.Run("undeclare-prefixes at version 1.0 is SEPM0010", func(t *testing.T) {
		doc := mustParseXML(t, `<root xmlns:p="urn:p"/>`)
		_, err := evaluate(t.Context(), doc,
			`serialize(., map{"method":"xml","version":"1.0","undeclare-prefixes":true()})`)
		require.Error(t, err)
		var xerr *xpath3.XPathError
		require.ErrorAs(t, err, &xerr)
		require.Equal(t, "SEPM0010", xerr.Code)

		// Absent version defaults to 1.0, so it is also SEPM0010.
		_, err = evaluate(t.Context(), doc,
			`serialize(., map{"method":"xml","undeclare-prefixes":true()})`)
		require.Error(t, err)
		require.ErrorAs(t, err, &xerr)
		require.Equal(t, "SEPM0010", xerr.Code)
	})

	t.Run("html method on non-html root emits no doctype or meta", func(t *testing.T) {
		doc := mustParseXML(t, `<?xml version="1.0" encoding="UTF-8"?><article><head/><body><p>Hi</p></body></article>`)
		res, err := evaluate(t.Context(), doc, `serialize(., map{"method":"html","html-version":5})`)
		require.NoError(t, err)
		out := res.StringValue()
		require.NotRegexp(t, `(?i)DOCTYPE`, out, "output:\n%s", out)
		require.NotContains(t, out, "<meta", "output:\n%s", out)
	})
}

// TestSerialize_OutputVersionAndNsResolution covers the effective-output-version
// override and the element-form QName-resolver namespace semantics (gauntlet
// findings on fix-qt3-serialize-params):
//   - the version serialization parameter drives the XML declaration text
//     regardless of the source document's own version;
//   - an unspecified/1.0 version with undeclare-prefixes is SEPM0010, while
//     version 1.1 emits a 1.1 declaration and allows undeclarations;
//   - the reserved xml prefix is always bound;
//   - xmlns:p="" undeclares (masks) the prefix p, leaving p:local unbound.
func TestSerialize_OutputVersionAndNsResolution(t *testing.T) {
	t.Run("version param overrides a 1.0 source declaration to 1.1", func(t *testing.T) {
		doc := mustParseXML(t, `<?xml version="1.0" encoding="UTF-8"?><root/>`)
		res, err := evaluate(t.Context(), doc,
			`serialize(., map{"method":"xml","omit-xml-declaration":false(),"version":"1.1"})`)
		require.NoError(t, err)
		require.Contains(t, res.StringValue(), `version="1.1"`, "output:\n%s", res.StringValue())
	})

	t.Run("version param overrides a 1.1 source declaration to 1.0", func(t *testing.T) {
		doc := mustParseXML(t, `<?xml version="1.1" encoding="UTF-8"?><root/>`)
		res, err := evaluate(t.Context(), doc,
			`serialize(., map{"method":"xml","omit-xml-declaration":false(),"version":"1.0"})`)
		require.NoError(t, err)
		out := res.StringValue()
		require.Contains(t, out, `version="1.0"`, "output:\n%s", out)
		require.NotContains(t, out, `version="1.1"`, "output:\n%s", out)
	})

	t.Run("version 1.1 emits 1.1 declaration and allows undeclarations", func(t *testing.T) {
		doc := mustParseXML(t, `<?xml version="1.1" encoding="UTF-8"?>`+
			`<p:chapter xmlns:p="urn:p"><section xmlns:p=""><para/></section></p:chapter>`)
		res, err := evaluate(t.Context(), doc,
			`serialize(., map{"method":"xml","omit-xml-declaration":false(),"version":"1.1","undeclare-prefixes":true()})`)
		require.NoError(t, err)
		out := res.StringValue()
		require.Contains(t, out, `version="1.1"`, "output:\n%s", out)
		require.Regexp(t, `section xmlns:p=["']["']`, out, "output:\n%s", out)
	})

	t.Run("element-form xml:foo resolves to the XML namespace", func(t *testing.T) {
		doc := mustParseXML(t, `<doc><xml:foo>hi</xml:foo><bar>no</bar></doc>`)
		params := mustParseXML(t, `<output:serialization-parameters xmlns:output="http://www.w3.org/2010/xslt-xquery-serialization">`+
			`<output:method value="xml"/><output:cdata-section-elements value="xml:foo"/>`+
			`</output:serialization-parameters>`)
		res0, err := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
			Evaluate(t.Context(), mustCompile(t, `.`), params.DocumentElement())
		require.NoError(t, err)
		eval := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
			Variables(map[string]xpath3.Sequence{paramsVar: res0.Sequence()})
		res, err := eval.Evaluate(t.Context(), mustCompile(t, `serialize(., $params)`), doc)
		require.NoError(t, err)
		out := res.StringValue()
		require.Contains(t, out, "CDATA[hi]", "output:\n%s", out)
		require.NotContains(t, out, "CDATA[no]", "output:\n%s", out)
	})

	t.Run("element-form xmlns:p= undeclaration leaves p unbound", func(t *testing.T) {
		// The params document is XML 1.1 so xmlns:p="" (undeclaring a non-default
		// prefix) is well-formed; the masking makes p:foo an unbound-prefix error.
		params := mustParseXML(t, `<?xml version="1.1"?>`+
			`<output:serialization-parameters xmlns:output="http://www.w3.org/2010/xslt-xquery-serialization" xmlns:p="urn:p">`+
			`<output:cdata-section-elements xmlns:p="" value="p:foo"/>`+
			`</output:serialization-parameters>`)
		res0, err := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
			Evaluate(t.Context(), mustCompile(t, `.`), params.DocumentElement())
		require.NoError(t, err)
		eval := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
			Variables(map[string]xpath3.Sequence{paramsVar: res0.Sequence()})
		doc := mustParseXML(t, `<root/>`)
		_, err = eval.Evaluate(t.Context(), mustCompile(t, `serialize(., $params)`), doc)
		require.Error(t, err)
		var xerr *xpath3.XPathError
		require.ErrorAs(t, err, &xerr)
		require.Equal(t, "XPTY0004", xerr.Code)
	})
}

// TestSerialize_SEPM0009 covers Serialization 3.1 §5.1.6: requesting
// omit-xml-declaration=yes together with a standalone value of yes/no is a
// static error (a standalone declaration is impossible without an XML
// declaration). The map-form default of omit-xml-declaration=true makes an
// explicit standalone yes/no conflict; "omit"/absent standalone, or an explicit
// omit-xml-declaration=false, do not.
func TestSerialize_SEPM0009(t *testing.T) {
	doc := mustParseXML(t, `<?xml version="1.0" encoding="UTF-8"?><root/>`)

	requireSEPM0009 := func(t *testing.T, expr string) {
		t.Helper()
		_, err := evaluate(t.Context(), doc, expr)
		require.Error(t, err)
		var xerr *xpath3.XPathError
		require.ErrorAs(t, err, &xerr)
		require.Equal(t, "SEPM0009", xerr.Code)
	}

	requireNoError := func(t *testing.T, expr string) {
		t.Helper()
		_, err := evaluate(t.Context(), doc, expr)
		require.NoError(t, err)
	}

	t.Run("map default omit + standalone true is SEPM0009", func(t *testing.T) {
		requireSEPM0009(t, `serialize(., map{"standalone": true()})`)
	})
	t.Run("map default omit + standalone false is SEPM0009", func(t *testing.T) {
		requireSEPM0009(t, `serialize(., map{"standalone": false()})`)
	})
	t.Run("map default omit + standalone omit is allowed", func(t *testing.T) {
		requireNoError(t, `serialize(., map{"standalone": "omit"})`)
	})
	t.Run("map default omit + absent standalone is allowed", func(t *testing.T) {
		requireNoError(t, `serialize(., map{})`)
	})
	t.Run("explicit omit=false + standalone true is allowed", func(t *testing.T) {
		requireNoError(t, `serialize(., map{"omit-xml-declaration": false(), "standalone": true()})`)
	})

	t.Run("element form omit=yes + standalone yes is SEPM0009", func(t *testing.T) {
		params := mustParseXML(t, `<output:serialization-parameters xmlns:output="http://www.w3.org/2010/xslt-xquery-serialization">`+
			`<output:omit-xml-declaration value="yes"/><output:standalone value="yes"/>`+
			`</output:serialization-parameters>`)
		res0, err := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
			Evaluate(t.Context(), mustCompile(t, `.`), params.DocumentElement())
		require.NoError(t, err)
		eval := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
			Variables(map[string]xpath3.Sequence{paramsVar: res0.Sequence()})
		_, err = eval.Evaluate(t.Context(), mustCompile(t, `serialize(., $params)`), doc)
		require.Error(t, err)
		var xerr *xpath3.XPathError
		require.ErrorAs(t, err, &xerr)
		require.Equal(t, "SEPM0009", xerr.Code)
	})
}
