package xpath3_test

import (
	"strings"
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
//   - undeclare-prefixes at output version 1.0 is a SEPM0010 static error under
//     the xml/xhtml methods, but is silently ignored (no error) for html/text/
//     json/adaptive, where the parameter is not applicable.
//   - the doctype-system SystemLiteral both-quotes SEPM0016 check fires only for
//     the methods that emit a DOCTYPE (xml/xhtml/html), and is ignored by
//     text/json/adaptive.
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

	t.Run("undeclare-prefixes ignored for non-applicable method (html)", func(t *testing.T) {
		// undeclare-prefixes is applicable only to the xml/xhtml methods
		// (Serialization 3.1 §5.1.8); for html it is silently ignored, so an
		// effective 1.0 version raises NO SEPM0010 and serialization proceeds.
		doc := mustParseXML(t, `<html xmlns:p="urn:p"><body><p>Hi</p></body></html>`)
		res, err := evaluate(t.Context(), doc,
			`serialize(., map{"method":"html","undeclare-prefixes":true()})`)
		require.NoError(t, err)
		require.Contains(t, res.StringValue(), "<body>", "output:\n%s", res.StringValue())
	})

	t.Run("undeclare-prefixes ignored for non-applicable method (text)", func(t *testing.T) {
		doc := mustParseXML(t, `<root xmlns:p="urn:p">hi</root>`)
		res, err := evaluate(t.Context(), doc,
			`serialize(., map{"method":"text","undeclare-prefixes":true()})`)
		require.NoError(t, err)
		require.Equal(t, "hi", res.StringValue())
	})

	t.Run("doctype-system both-quotes SEPM0016 only for applicable methods", func(t *testing.T) {
		doc := mustParseXML(t, `<root>hi</root>`)
		// A doctype-system value that could not be written as a SystemLiteral (it
		// contains both a " and a ') is SEPM0016 under the xml method, which emits a
		// DOCTYPE from the parameter (§5.1.7).
		_, err := evaluate(t.Context(), doc,
			`serialize(., map{"method":"xml","doctype-system":concat('a', codepoints-to-string(34), 'b', codepoints-to-string(39), 'c')})`)
		require.Error(t, err)
		var xerr *xpath3.XPathError
		require.ErrorAs(t, err, &xerr)
		require.Equal(t, "SEPM0016", xerr.Code)

		// text/adaptive ignore doctype-system, so the same value raises no error
		// and serialization proceeds.
		for _, method := range []string{"text", "adaptive"} {
			expr := `serialize(., map{"method":"` + method +
				`","doctype-system":concat('a', codepoints-to-string(34), 'b', codepoints-to-string(39), 'c')})`
			_, err := evaluate(t.Context(), doc, expr)
			require.NoErrorf(t, err, "method %q must ignore doctype-system", method)
		}
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

// runElementFormParams parses an output:serialization-parameters element, binds
// it to $params, and runs serialize(., $params) over the given source, returning
// the serialized string and any error.
func runElementFormParams(t *testing.T, sourceXML, paramsXML string) (string, error) {
	t.Helper()
	doc := mustParseXML(t, sourceXML)
	params := mustParseXML(t, paramsXML)
	res0, err := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
		Evaluate(t.Context(), mustCompile(t, `.`), params.DocumentElement())
	require.NoError(t, err)
	eval := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
		Variables(map[string]xpath3.Sequence{paramsVar: res0.Sequence()})
	r, err := eval.Evaluate(t.Context(), mustCompile(t, exprSerializeWithParams), doc)
	if err != nil {
		return "", err
	}
	return r.StringValue(), nil
}

// TestSerialize_ElementFormSchemaTypes locks in the element-form option parser's
// conformance to the Serialization 3.1 schema types (gauntlet findings): the
// boolean/yes-no-omit lexical space (true/false/1/0 accepted, uppercase
// rejected), XSD-list-whitespace splitting of QName lists (NBSP stays in the
// token and fails validation), and EQName brace rejection.
func TestSerialize_ElementFormSchemaTypes(t *testing.T) {
	const paramsOpen = `<output:serialization-parameters xmlns:output="http://www.w3.org/2010/xslt-xquery-serialization">`
	const paramsClose = `</output:serialization-parameters>`

	requireXPTY0004 := func(t *testing.T, paramsXML string) {
		t.Helper()
		_, err := runElementFormParams(t, `<root/>`, paramsXML)
		require.Error(t, err)
		var xerr *xpath3.XPathError
		require.ErrorAs(t, err, &xerr)
		require.Equal(t, "XPTY0004", xerr.Code)
	}

	requireSEPM0009 := func(t *testing.T, standaloneValue string) {
		t.Helper()
		p := paramsOpen +
			`<output:omit-xml-declaration value="yes"/>` +
			`<output:standalone value="` + standaloneValue + `"/>` + paramsClose
		_, err := runElementFormParams(t, `<root/>`, p)
		require.Error(t, err)
		var xerr *xpath3.XPathError
		require.ErrorAs(t, err, &xerr)
		require.Equal(t, "SEPM0009", xerr.Code, "standalone=%q", standaloneValue)
	}

	// Finding 1: element-form standalone now accepts the boolean lexical families
	// true/1 (→yes) and false/0 (→no); with omit-xml-declaration=yes each reaches
	// the SEPM0009 conflict check.
	for _, v := range []string{"true", "1", "false", "0", "yes", "no"} {
		t.Run("standalone "+v+" + omit=yes is SEPM0009", func(t *testing.T) {
			requireSEPM0009(t, v)
		})
	}

	t.Run("standalone true normalizes to yes in the declaration", func(t *testing.T) {
		out, err := runElementFormParams(t, `<root/>`, paramsOpen+
			`<output:omit-xml-declaration value="no"/><output:standalone value="true"/>`+paramsClose)
		require.NoError(t, err)
		require.Contains(t, out, `standalone="yes"`, "output:\n%s", out)
	})

	// A yes-no boolean accepts true/1 but rejects uppercase (xs:boolean is
	// lowercase-only).
	t.Run("indent true is accepted", func(t *testing.T) {
		_, err := runElementFormParams(t, `<a><b>x</b></a>`, paramsOpen+
			`<output:method value="xml"/><output:indent value="true"/>`+paramsClose)
		require.NoError(t, err)
	})
	t.Run("indent YES uppercase is XPTY0004", func(t *testing.T) {
		requireXPTY0004(t, paramsOpen+`<output:indent value="YES"/>`+paramsClose)
	})
	t.Run("standalone TRUE uppercase is XPTY0004", func(t *testing.T) {
		requireXPTY0004(t, paramsOpen+`<output:standalone value="TRUE"/>`+paramsClose)
	})

	// Finding 2: an NBSP inside a QName-list token is not a list separator, so the
	// token "a b" stays whole and fails NCName validation.
	t.Run("NBSP in QName-list token is XPTY0004", func(t *testing.T) {
		requireXPTY0004(t, paramsOpen+
			"<output:cdata-section-elements value=\"a b\"/>"+paramsClose)
	})

	// Finding 3: an EQName with a brace inside the URI part is not well-formed.
	t.Run("EQName with brace in URI is XPTY0004", func(t *testing.T) {
		requireXPTY0004(t, paramsOpen+
			`<output:cdata-section-elements value="Q{a{b}c"/>`+paramsClose)
	})

	// Previously-unrecognized but schema-valid parameters are accepted (validated
	// and ignored), not rejected as unsupported.
	t.Run("escape-uri-attributes and include-content-type are accepted", func(t *testing.T) {
		_, err := runElementFormParams(t, `<root/>`, paramsOpen+
			`<output:method value="xml"/><output:escape-uri-attributes value="yes"/>`+
			`<output:include-content-type value="no"/><output:html-version value="5.0"/>`+paramsClose)
		require.NoError(t, err)
	})
}

// TestSerialize_HTMLParamsApplied verifies the html output-method parameters are
// APPLIED (not merely recognized): include-content-type, escape-uri-attributes,
// and html-version (doctype selection).
func TestSerialize_HTMLParamsApplied(t *testing.T) {
	const srcHTMLHead = `<?xml version="1.0" encoding="UTF-8"?><html><head/><body><p>Hi</p></body></html>`

	t.Run("include-content-type=no suppresses the meta element", func(t *testing.T) {
		doc := mustParseXML(t, srcHTMLHead)
		res, err := evaluate(t.Context(), doc,
			`serialize(., map{"method":"html","include-content-type":false()})`)
		require.NoError(t, err)
		require.NotContains(t, res.StringValue(), "<meta", "output:\n%s", res.StringValue())
	})

	t.Run("include-content-type default injects the meta element", func(t *testing.T) {
		doc := mustParseXML(t, srcHTMLHead)
		res, err := evaluate(t.Context(), doc, `serialize(., map{"method":"html"})`)
		require.NoError(t, err)
		require.Contains(t, res.StringValue(), "<meta", "output:\n%s", res.StringValue())
	})

	t.Run("escape-uri-attributes=no leaves URI attributes unescaped", func(t *testing.T) {
		doc := mustParseXML(t, `<?xml version="1.0" encoding="UTF-8"?>`+
			`<html><head/><body><a href="a b|c">x</a></body></html>`)
		res, err := evaluate(t.Context(), doc,
			`serialize(., map{"method":"html","escape-uri-attributes":false()})`)
		require.NoError(t, err)
		require.Contains(t, res.StringValue(), `href="a b|c"`, "output:\n%s", res.StringValue())

		res2, err := evaluate(t.Context(), doc,
			`serialize(., map{"method":"html","escape-uri-attributes":true()})`)
		require.NoError(t, err)
		require.Contains(t, res2.StringValue(), `href="a%20b|c"`, "output:\n%s", res2.StringValue())
	})

	t.Run("html-version 5 emits the HTML5 doctype", func(t *testing.T) {
		doc := mustParseXML(t, srcHTMLHead)
		res, err := evaluate(t.Context(), doc, `serialize(., map{"method":"html","html-version":5.0})`)
		require.NoError(t, err)
		require.Contains(t, res.StringValue(), "<!DOCTYPE html>", "output:\n%s", res.StringValue())
	})

	t.Run("html-version 4.0 emits the HTML 4.01 doctype", func(t *testing.T) {
		doc := mustParseXML(t, srcHTMLHead)
		res, err := evaluate(t.Context(), doc, `serialize(., map{"method":"html","html-version":4.0})`)
		require.NoError(t, err)
		out := res.StringValue()
		require.Contains(t, out, "DTD HTML 4.01", "output:\n%s", out)
		require.NotContains(t, out, "<!DOCTYPE html>", "output:\n%s", out)
	})
}

// TestSerialize_ElementFormValueValidation verifies the element-form value
// schema-type validation for previously shape-only-checked parameters, and the
// XSD-whitespace content checks (NBSP is significant).
func TestSerialize_ElementFormValueValidation(t *testing.T) {
	const paramsOpen = `<output:serialization-parameters xmlns:output="http://www.w3.org/2010/xslt-xquery-serialization">`
	const paramsClose = `</output:serialization-parameters>`

	requireXPTY0004 := func(t *testing.T, paramsXML string) {
		t.Helper()
		_, err := runElementFormParams(t, `<root/>`, paramsXML)
		require.Error(t, err)
		var xerr *xpath3.XPathError
		require.ErrorAs(t, err, &xerr)
		require.Equal(t, "XPTY0004", xerr.Code)
	}
	requireOK := func(t *testing.T, paramsXML string) {
		t.Helper()
		_, err := runElementFormParams(t, `<root/>`, paramsXML)
		require.NoError(t, err)
	}

	t.Run("html-version bogus is XPTY0004", func(t *testing.T) {
		requireXPTY0004(t, paramsOpen+`<output:html-version value="bogus"/>`+paramsClose)
	})
	t.Run("html-version decimal is accepted", func(t *testing.T) {
		requireOK(t, paramsOpen+`<output:html-version value="4.01"/>`+paramsClose)
	})
	t.Run("json-node-output-method bad value is XPTY0004", func(t *testing.T) {
		requireXPTY0004(t, paramsOpen+`<output:json-node-output-method value="not a qname"/>`+paramsClose)
	})
	t.Run("json-node-output-method xml is accepted", func(t *testing.T) {
		requireOK(t, paramsOpen+`<output:json-node-output-method value="xml"/>`+paramsClose)
	})
	t.Run("normalization-form bogus is XPTY0004", func(t *testing.T) {
		requireXPTY0004(t, paramsOpen+`<output:normalization-form value="bogus"/>`+paramsClose)
	})
	t.Run("normalization-form none is accepted (no-op)", func(t *testing.T) {
		requireOK(t, paramsOpen+`<output:normalization-form value="none"/>`+paramsClose)
	})
	t.Run("normalization-form NFC is applied (supported)", func(t *testing.T) {
		requireOK(t, paramsOpen+`<output:method value="xml"/><output:normalization-form value="NFC"/>`+paramsClose)
	})
	t.Run("normalization-form fully-normalized is SESU0011 (unsupported)", func(t *testing.T) {
		_, err := runElementFormParams(t, `<root/>`, paramsOpen+`<output:method value="xml"/><output:normalization-form value="fully-normalized"/>`+paramsClose)
		require.Error(t, err)
		var xerr *xpath3.XPathError
		require.ErrorAs(t, err, &xerr)
		require.Equal(t, "SESU0011", xerr.Code)
	})

	// Finding 3: NBSP is not XSD whitespace, so NBSP-only content is significant.
	t.Run("NBSP-only text in serialization-parameters is XPTY0004", func(t *testing.T) {
		requireXPTY0004(t, paramsOpen+" "+paramsClose)
	})
	t.Run("NBSP-only content in a parameter element is XPTY0004", func(t *testing.T) {
		requireXPTY0004(t, paramsOpen+"<output:method value=\"xml\"> </output:method>"+paramsClose)
	})
}

// TestSerialize_TextMethodAndHonesty covers the newly-implemented text output
// method, map-form validation parity, the media-type meta value, and the
// spec-honest normalization-form (SESU0011) behavior.
func TestSerialize_TextMethodAndHonesty(t *testing.T) {
	t.Run("text method concatenates string values with no markup", func(t *testing.T) {
		doc := mustParseXML(t, `<doc><a>Hello</a><b> World</b></doc>`)
		res, err := evaluate(t.Context(), doc, `serialize(., map{"method":"text"})`)
		require.NoError(t, err)
		require.Equal(t, "Hello World", res.StringValue())
	})

	t.Run("text method on atomics uses item-separator", func(t *testing.T) {
		res, err := evaluate(t.Context(), nil, `serialize((1, 2, 3), map{"method":"text","item-separator":"|"})`)
		require.NoError(t, err)
		require.Equal(t, "1|2|3", res.StringValue())
	})

	t.Run("text method applies character maps", func(t *testing.T) {
		doc := mustParseXML(t, `<e>xml</e>`)
		res, err := evaluate(t.Context(), doc, `serialize(., map{"method":"text","use-character-maps":map{"x":"j","m":"so","l":"n"}})`)
		require.NoError(t, err)
		require.Equal(t, "json", res.StringValue())
	})

	t.Run("map-form normalization-form fully-normalized is SESU0011", func(t *testing.T) {
		doc := mustParseXML(t, `<root/>`)
		_, err := evaluate(t.Context(), doc, `serialize(., map{"normalization-form":"fully-normalized"})`)
		require.Error(t, err)
		var xerr *xpath3.XPathError
		require.ErrorAs(t, err, &xerr)
		require.Equal(t, "SESU0011", xerr.Code)
	})

	t.Run("map-form normalization-form bogus is XPTY0004", func(t *testing.T) {
		doc := mustParseXML(t, `<root/>`)
		_, err := evaluate(t.Context(), doc, `serialize(., map{"normalization-form":"bogus"})`)
		require.Error(t, err)
		var xerr *xpath3.XPathError
		require.ErrorAs(t, err, &xerr)
		require.Equal(t, "XPTY0004", xerr.Code)
	})

	t.Run("map-form byte-order-mark bad value is XPTY0004", func(t *testing.T) {
		doc := mustParseXML(t, `<root/>`)
		_, err := evaluate(t.Context(), doc, `serialize(., map{"byte-order-mark":"maybe"})`)
		require.Error(t, err)
		var xerr *xpath3.XPathError
		require.ErrorAs(t, err, &xerr)
		require.Equal(t, "XPTY0004", xerr.Code)
	})

	t.Run("map-form json-node-output-method bad value is XPTY0004", func(t *testing.T) {
		doc := mustParseXML(t, `<root/>`)
		_, err := evaluate(t.Context(), doc, `serialize(., map{"json-node-output-method":"not a qname"})`)
		require.Error(t, err)
		var xerr *xpath3.XPathError
		require.ErrorAs(t, err, &xerr)
		require.Equal(t, "XPTY0004", xerr.Code)
	})

	t.Run("media-type is used in the html Content-Type meta", func(t *testing.T) {
		doc := mustParseXML(t, `<?xml version="1.0" encoding="UTF-8"?><html><head/><body/></html>`)
		res, err := evaluate(t.Context(), doc,
			`serialize(., map{"method":"html","media-type":"application/xhtml+xml"})`)
		require.NoError(t, err)
		require.Contains(t, res.StringValue(), "application/xhtml+xml; charset=UTF-8", "output:\n%s", res.StringValue())
	})

	t.Run("xhtml method serializes as XML", func(t *testing.T) {
		doc := mustParseXML(t, `<?xml version="1.0" encoding="UTF-8"?><html><head/><body><p>Hi</p></body></html>`)
		res, err := evaluate(t.Context(), doc, `serialize(., map{"method":"xhtml"})`)
		require.NoError(t, err)
		// XML serialization emits an explicit end tag and no HTML DOCTYPE.
		require.Contains(t, res.StringValue(), "<p>Hi</p>", "output:\n%s", res.StringValue())
		require.NotContains(t, res.StringValue(), "<!DOCTYPE", "output:\n%s", res.StringValue())
	})
}

// TestSerialize_DoctypeMethodAndMeta covers the three correctness fixes:
// media-type updating an existing Content-Type meta, map-form method validation,
// the SEPM0009 doctype-system half, and XML doctype-system emission.
func TestSerialize_DoctypeMethodAndMeta(t *testing.T) {
	t.Run("media-type updates an existing Content-Type meta", func(t *testing.T) {
		doc := mustParseXML(t, `<?xml version="1.0" encoding="UTF-8"?>`+
			`<html><head><meta http-equiv="Content-Type" content="text/html; charset=iso-8859-1"/></head><body/></html>`)
		res, err := evaluate(t.Context(), doc,
			`serialize(., map{"method":"html","media-type":"application/xhtml+xml"})`)
		require.NoError(t, err)
		out := res.StringValue()
		require.Contains(t, out, "application/xhtml+xml; charset=UTF-8", "output:\n%s", out)
		require.NotContains(t, out, "text/html; charset=iso-8859-1", "output:\n%s", out)
	})

	t.Run("map-form method bad value errors, not silent XML", func(t *testing.T) {
		doc := mustParseXML(t, `<root/>`)
		_, err := evaluate(t.Context(), doc, `serialize(., map{"method":"not a qname"})`)
		require.Error(t, err)
		var xerr *xpath3.XPathError
		require.ErrorAs(t, err, &xerr)
		require.Equal(t, "XPTY0004", xerr.Code)
	})

	t.Run("map-form extension-method QName is unsupported (SEPM0016)", func(t *testing.T) {
		// An extension method is a valid method-type value, but helium implements
		// no extension output methods, so it is an unsupported value rather than a
		// silent fall-through to the xml method.
		doc := mustParseXML(t, `<root/>`)
		_, err := evaluate(t.Context(), doc, `serialize(., map{"method":"ex:custom","omit-xml-declaration":true()})`)
		require.Error(t, err)
		var xerr *xpath3.XPathError
		require.ErrorAs(t, err, &xerr)
		require.Equal(t, "SEPM0016", xerr.Code)
	})

	// Per Serialization 3.1 SEPM0009, the doctype-system sub-condition is gated on
	// version != "1.0": a DOCTYPE without an XML declaration is well-formed in XML
	// 1.0, so omit-xml-declaration=yes + doctype-system at version 1.0 is NOT an
	// error and emits the DOCTYPE.
	t.Run("omit=yes + doctype-system at version 1.0 emits DOCTYPE, no error", func(t *testing.T) {
		doc := mustParseXML(t, `<root/>`)
		res, err := evaluate(t.Context(), doc,
			`serialize(., map{"omit-xml-declaration":true(),"doctype-system":"x.dtd"})`)
		require.NoError(t, err)
		out := res.StringValue()
		require.Contains(t, out, `<!DOCTYPE root SYSTEM "x.dtd">`, "output:\n%s", out)
		require.NotContains(t, out, "<?xml", "output:\n%s", out)
	})

	t.Run("map default omit + doctype-system at version 1.0 emits DOCTYPE, no error", func(t *testing.T) {
		doc := mustParseXML(t, `<root/>`)
		res, err := evaluate(t.Context(), doc, `serialize(., map{"doctype-system":"x.dtd"})`)
		require.NoError(t, err)
		require.Contains(t, res.StringValue(), `<!DOCTYPE root SYSTEM "x.dtd">`, "output:\n%s", res.StringValue())
	})

	t.Run("omit=yes + doctype-system at version 1.1 is SEPM0009", func(t *testing.T) {
		doc := mustParseXML(t, `<root/>`)
		_, err := evaluate(t.Context(), doc,
			`serialize(., map{"omit-xml-declaration":true(),"version":"1.1","doctype-system":"x.dtd"})`)
		require.Error(t, err)
		var xerr *xpath3.XPathError
		require.ErrorAs(t, err, &xerr)
		require.Equal(t, "SEPM0009", xerr.Code)
	})

	t.Run("XML doctype-system is emitted", func(t *testing.T) {
		doc := mustParseXML(t, `<?xml version="1.0" encoding="UTF-8"?><root><a>x</a></root>`)
		res, err := evaluate(t.Context(), doc,
			`serialize(., map{"method":"xml","omit-xml-declaration":false(),"doctype-system":"http://example.com/x.dtd"})`)
		require.NoError(t, err)
		out := res.StringValue()
		require.Contains(t, out, `<!DOCTYPE root SYSTEM "http://example.com/x.dtd">`, "output:\n%s", out)
	})

	t.Run("XML doctype-public+system is emitted", func(t *testing.T) {
		doc := mustParseXML(t, `<?xml version="1.0" encoding="UTF-8"?><root><a>x</a></root>`)
		res, err := evaluate(t.Context(), doc,
			`serialize(., map{"method":"xml","omit-xml-declaration":false(),"doctype-public":"-//EX//DTD//EN","doctype-system":"http://example.com/x.dtd"})`)
		require.NoError(t, err)
		out := res.StringValue()
		require.Contains(t, out, `<!DOCTYPE root PUBLIC "-//EX//DTD//EN" "http://example.com/x.dtd">`, "output:\n%s", out)
	})

	t.Run("XML doctype-system replaces a pre-existing internal subset", func(t *testing.T) {
		doc := mustParseXML(t, `<?xml version="1.0"?><!DOCTYPE root SYSTEM "old.dtd" [<!ELEMENT root EMPTY>]><root/>`)
		res, err := evaluate(t.Context(), doc,
			`serialize(., map{"method":"xml","doctype-system":"new.dtd"})`)
		require.NoError(t, err)
		out := res.StringValue()
		require.Contains(t, out, `<!DOCTYPE root SYSTEM "new.dtd">`, "output:\n%s", out)
		require.NotContains(t, out, "old.dtd", "output:\n%s", out)
		require.NotContains(t, out, "<!ELEMENT", "output:\n%s", out)
	})

	t.Run("html method with doctype-system does not raise SEPM0009", func(t *testing.T) {
		doc := mustParseXML(t, `<?xml version="1.0" encoding="UTF-8"?><html><head/><body/></html>`)
		_, err := evaluate(t.Context(), doc, `serialize(., map{"method":"html","doctype-system":"about:legacy-compat"})`)
		require.NoError(t, err)
	})

	// fn:serialize applies sequence normalization (Serialization 3.1 §2), which
	// wraps the node sequence in a document node, so doctype-system MUST apply to
	// an ELEMENT input too — not only a document node. The requested DOCTYPE is
	// emitted immediately before the first element, named after that element, with
	// an empty internal subset. The caller's tree is never mutated.
	t.Run("element input + doctype-system emits DOCTYPE", func(t *testing.T) {
		doc := mustParseXML(t, `<?xml version="1.0" encoding="UTF-8"?><root><a>x</a></root>`)
		res, err := evaluate(t.Context(), doc,
			`serialize(/*, map{"method":"xml","omit-xml-declaration":true(),"doctype-system":"x.dtd"})`)
		require.NoError(t, err)
		out := res.StringValue()
		require.Contains(t, out, `<!DOCTYPE root SYSTEM "x.dtd">`, "output:\n%s", out)
		require.Contains(t, out, `<root>`, "output:\n%s", out)
	})

	t.Run("element input + doctype-public+system emits PUBLIC form", func(t *testing.T) {
		doc := mustParseXML(t, `<?xml version="1.0" encoding="UTF-8"?><root><a>x</a></root>`)
		res, err := evaluate(t.Context(), doc,
			`serialize(/*, map{"method":"xml","omit-xml-declaration":true(),"doctype-public":"-//EX//DTD//EN","doctype-system":"http://example.com/x.dtd"})`)
		require.NoError(t, err)
		out := res.StringValue()
		require.Contains(t, out, `<!DOCTYPE root PUBLIC "-//EX//DTD//EN" "http://example.com/x.dtd">`, "output:\n%s", out)
	})

	t.Run("element input + doctype-system with both quote kinds is SEPM0016", func(t *testing.T) {
		doc := mustParseXML(t, `<root/>`)
		_, err := evaluate(t.Context(), doc,
			`serialize(/*, map{"method":"xml","omit-xml-declaration":true(),"doctype-system":concat('a"b',"'",'c')})`)
		require.Error(t, err)
		var xerr *xpath3.XPathError
		require.ErrorAs(t, err, &xerr)
		require.Equal(t, "SEPM0016", xerr.Code)
	})

	t.Run("element input serialize does not mutate the caller's tree", func(t *testing.T) {
		doc := mustParseXML(t, `<?xml version="1.0" encoding="UTF-8"?><root><a>x</a></root>`)
		res, err := evaluate(t.Context(), doc,
			`serialize(/*, map{"method":"xml","omit-xml-declaration":true(),"doctype-system":"x.dtd"})`)
		require.NoError(t, err)
		require.Contains(t, res.StringValue(), `<!DOCTYPE root SYSTEM "x.dtd">`)
		// Re-serializing the same source without doctype-system must show no DOCTYPE
		// leaked back onto the caller's document.
		plain, err := evaluate(t.Context(), doc, `serialize(/*, map{"method":"xml","omit-xml-declaration":true()})`)
		require.NoError(t, err)
		require.NotContains(t, plain.StringValue(), "<!DOCTYPE", "output:\n%s", plain.StringValue())
	})
}

// TestSerialize_MethodVersionAndCharMapAudit locks in the spec-correctness fixes
// from the exhaustive param audit: method-value validation (bare non-built-in
// NCName and unsupported extension methods both error, never silent XML), the
// narrower json-node-output-method domain, XML output-version validation
// (SESU0013), doctype-public ignored without doctype-system, character maps
// applied to the html output method, and full replacement of stale Content-Type
// metas.
func TestSerialize_MethodVersionAndCharMapAudit(t *testing.T) {
	requireCode := func(t *testing.T, expr, code string) {
		t.Helper()
		doc := mustParseXML(t, `<root/>`)
		_, err := evaluate(t.Context(), doc, expr)
		require.Error(t, err)
		var xerr *xpath3.XPathError
		require.ErrorAs(t, err, &xerr)
		require.Equal(t, code, xerr.Code)
	}

	t.Run("map-form method bogus bare NCName errors", func(t *testing.T) {
		requireCode(t, `serialize(., map{"method":"bogus"})`, "XPTY0004")
	})
	t.Run("map-form method extension QName is unsupported SEPM0016", func(t *testing.T) {
		requireCode(t, `serialize(., map{"method":"ex:custom","omit-xml-declaration":true()})`, "SEPM0016")
	})
	t.Run("map-form method EQName extension is unsupported SEPM0016", func(t *testing.T) {
		requireCode(t, `serialize(., map{"method":"Q{urn:x}custom","omit-xml-declaration":true()})`, "SEPM0016")
	})
	t.Run("element-form method bogus bare NCName is XPTY0004", func(t *testing.T) {
		_, err := runElementFormParams(t, `<root/>`,
			`<output:serialization-parameters xmlns:output="http://www.w3.org/2010/xslt-xquery-serialization">`+
				`<output:method value="bogus"/></output:serialization-parameters>`)
		require.Error(t, err)
		var xerr *xpath3.XPathError
		require.ErrorAs(t, err, &xerr)
		require.Equal(t, "XPTY0004", xerr.Code)
	})

	t.Run("map-form json-node-output-method rejects json", func(t *testing.T) {
		requireCode(t, `serialize(., map{"json-node-output-method":"json"})`, "XPTY0004")
	})
	t.Run("map-form json-node-output-method rejects adaptive", func(t *testing.T) {
		requireCode(t, `serialize(., map{"json-node-output-method":"adaptive"})`, "XPTY0004")
	})
	t.Run("map-form json-node-output-method accepts text", func(t *testing.T) {
		doc := mustParseXML(t, `<root/>`)
		_, err := evaluate(t.Context(), doc, `serialize(., map{"json-node-output-method":"text"})`)
		require.NoError(t, err)
	})

	t.Run("map-form version bogus is SESU0013", func(t *testing.T) {
		requireCode(t, `serialize(., map{"method":"xml","omit-xml-declaration":false(),"version":"bogus"})`, "SESU0013")
	})
	t.Run("map-form version 1.2 unsupported is SESU0013", func(t *testing.T) {
		requireCode(t, `serialize(., map{"method":"xml","omit-xml-declaration":false(),"version":"1.2"})`, "SESU0013")
	})
	t.Run("element-form version bogus is SESU0013", func(t *testing.T) {
		_, err := runElementFormParams(t, `<root/>`,
			`<output:serialization-parameters xmlns:output="http://www.w3.org/2010/xslt-xquery-serialization">`+
				`<output:method value="xml"/><output:version value="bogus"/></output:serialization-parameters>`)
		require.Error(t, err)
		var xerr *xpath3.XPathError
		require.ErrorAs(t, err, &xerr)
		require.Equal(t, "SESU0013", xerr.Code)
	})
	t.Run("map-form version 1.1 is accepted", func(t *testing.T) {
		doc := mustParseXML(t, `<root/>`)
		res, err := evaluate(t.Context(), doc,
			`serialize(., map{"method":"xml","omit-xml-declaration":false(),"version":"1.1"})`)
		require.NoError(t, err)
		require.Contains(t, res.StringValue(), `version="1.1"`, "output:\n%s", res.StringValue())
	})

	t.Run("XML doctype-public WITHOUT doctype-system injects no DTD", func(t *testing.T) {
		doc := mustParseXML(t, `<?xml version="1.0" encoding="UTF-8"?><root><a>x</a></root>`)
		res, err := evaluate(t.Context(), doc,
			`serialize(., map{"method":"xml","omit-xml-declaration":false(),"doctype-public":"-//EX//DTD//EN"})`)
		require.NoError(t, err)
		out := res.StringValue()
		require.NotContains(t, out, "<!DOCTYPE", "output:\n%s", out)
	})

	t.Run("html output applies character maps to text and attributes", func(t *testing.T) {
		doc := mustParseXML(t, `<?xml version="1.0" encoding="UTF-8"?>`+
			`<html><head/><body><p title="x">xml</p></body></html>`)
		res, err := evaluate(t.Context(), doc,
			`serialize(., map{"method":"html","include-content-type":false(),"use-character-maps":map{"x":"j"}})`)
		require.NoError(t, err)
		out := res.StringValue()
		require.Contains(t, out, "<p title=\"j\">jml</p>", "output:\n%s", out)
	})

	t.Run("html replaces every stale Content-Type meta (trimmed, case-insensitive)", func(t *testing.T) {
		doc := mustParseXML(t, `<?xml version="1.0" encoding="UTF-8"?><html><head>`+
			`<meta http-equiv=" Content-Type " content="text/html; charset=iso-8859-1"/>`+
			`<meta http-equiv="CONTENT-TYPE" content="text/html; charset=us-ascii"/>`+
			`</head><body/></html>`)
		res, err := evaluate(t.Context(), doc,
			`serialize(., map{"method":"html","media-type":"application/xhtml+xml"})`)
		require.NoError(t, err)
		out := res.StringValue()
		require.Contains(t, out, "application/xhtml+xml; charset=UTF-8", "output:\n%s", out)
		require.NotContains(t, out, "iso-8859-1", "output:\n%s", out)
		require.NotContains(t, out, "us-ascii", "output:\n%s", out)
		// Exactly one Content-Type meta remains.
		require.Equal(t, 1, strings.Count(strings.ToLower(out), "http-equiv"), "output:\n%s", out)
	})
}

// TestSerialize_NormalizationJSONNodeAndQNameMethod locks in the round-2 fixes:
// normalization-form NFC/NFD/NFKC/NFKD actually normalize the serialized output
// (json/adaptive ignore it, fully-normalized is SESU0011); json-node-output-method
// non-default over a serialized node is an honest unsupported error (not silent
// xml); and a map-form namespaced-QName method is treated as an unsupported
// extension (SEPM0016) rather than losing its namespace to XPTY0004.
func TestSerialize_NormalizationJSONNodeAndQNameMethod(t *testing.T) {
	// U+00E9 (composed é) vs "e" + U+0301 (decomposed).
	const composed = "\u00e9"
	const decomposed = "e\u0301"

	t.Run("NFD decomposes the serialized text output", func(t *testing.T) {
		res, err := evaluate(t.Context(), nil,
			`serialize("`+composed+`", map{"method":"text","normalization-form":"NFD"})`)
		require.NoError(t, err)
		require.Equal(t, decomposed, res.StringValue())
	})
	t.Run("NFC composes the serialized text output", func(t *testing.T) {
		res, err := evaluate(t.Context(), nil,
			`serialize("`+decomposed+`", map{"method":"text","normalization-form":"NFC"})`)
		require.NoError(t, err)
		require.Equal(t, composed, res.StringValue())
	})
	t.Run("NFKC folds a compatibility ligature", func(t *testing.T) {
		// U+FB01 (ﬁ ligature) → "fi" under NFKC.
		res, err := evaluate(t.Context(), nil,
			`serialize("ﬁ", map{"method":"text","normalization-form":"NFKC"})`)
		require.NoError(t, err)
		require.Equal(t, "fi", res.StringValue())
	})
	t.Run("NFKD folds a compatibility ligature", func(t *testing.T) {
		res, err := evaluate(t.Context(), nil,
			`serialize("ﬁ", map{"method":"text","normalization-form":"NFKD"})`)
		require.NoError(t, err)
		require.Equal(t, "fi", res.StringValue())
	})
	t.Run("NFC in an XML element normalizes text content", func(t *testing.T) {
		doc := mustParseXML(t, `<e>`+decomposed+`</e>`)
		res, err := evaluate(t.Context(), doc,
			`serialize(., map{"method":"xml","normalization-form":"NFC"})`)
		require.NoError(t, err)
		// Normalization is scoped to the text node and runs BEFORE the XML writer's
		// escaping: the decomposed "e" + U+0301 composes to é (U+00E9), which the
		// writer's default non-ASCII escaping then emits as &#xE9; — exactly as a
		// literal composed é serializes, so normalization is consistent with the
		// rest of the writer rather than slipping a literal through a post-escape
		// pass.
		require.Equal(t, `<e>&#xE9;</e>`, res.StringValue())
	})
	t.Run("json applies normalization-form to string content", func(t *testing.T) {
		// normalization-form IS applicable to the json output method (§9.1.9): the
		// decomposed sequence composes under NFC.
		res, err := evaluate(t.Context(), nil,
			`serialize("`+decomposed+`", map{"method":"json","normalization-form":"NFC"})`)
		require.NoError(t, err)
		require.Equal(t, `"`+composed+`"`, res.StringValue())
	})

	t.Run("json-node-output-method text over a node is unsupported SEPM0016", func(t *testing.T) {
		doc := mustParseXML(t, `<a>x</a>`)
		_, err := evaluate(t.Context(), doc,
			`serialize(., map{"method":"json","json-node-output-method":"text"})`)
		require.Error(t, err)
		var xerr *xpath3.XPathError
		require.ErrorAs(t, err, &xerr)
		require.Equal(t, "SEPM0016", xerr.Code)
	})
	t.Run("json-node-output-method xml (default) over a node is fine", func(t *testing.T) {
		doc := mustParseXML(t, `<a>x</a>`)
		res, err := evaluate(t.Context(), doc,
			`serialize(., map{"method":"json","json-node-output-method":"xml"})`)
		require.NoError(t, err)
		// The node is xml-serialized and embedded as a JSON string (its markup
		// stays literal inside the string).
		require.Contains(t, res.StringValue(), "<a>x", "output:\n%s", res.StringValue())
	})
	t.Run("json-node-output-method text with no node serialized is a harmless no-op", func(t *testing.T) {
		res, err := evaluate(t.Context(), nil,
			`serialize("hi", map{"method":"json","json-node-output-method":"text"})`)
		require.NoError(t, err)
		require.Equal(t, `"hi"`, res.StringValue())
	})

	t.Run("map-form namespaced-QName method is unsupported extension SEPM0016", func(t *testing.T) {
		doc := mustParseXML(t, `<root/>`)
		_, err := evaluate(t.Context(), doc,
			`serialize(., map{"method":QName("urn:x","custom"),"omit-xml-declaration":true()})`)
		require.Error(t, err)
		var xerr *xpath3.XPathError
		require.ErrorAs(t, err, &xerr)
		require.Equal(t, "SEPM0016", xerr.Code)
	})
	t.Run("map-form no-namespace non-builtin QName method is invalid XPTY0004", func(t *testing.T) {
		doc := mustParseXML(t, `<root/>`)
		_, err := evaluate(t.Context(), doc,
			`serialize(., map{"method":QName("","custom")})`)
		require.Error(t, err)
		var xerr *xpath3.XPathError
		require.ErrorAs(t, err, &xerr)
		require.Equal(t, "XPTY0004", xerr.Code)
	})
	t.Run("map-form no-namespace QName method (even builtin local) is invalid XPTY0004", func(t *testing.T) {
		// F&O 3.1: an xs:QName method value must have a non-absent namespace;
		// built-in methods are supplied as strings, so QName("","text") is invalid.
		doc := mustParseXML(t, `<root>hi</root>`)
		_, err := evaluate(t.Context(), doc,
			`serialize(., map{"method":QName("","text")})`)
		require.Error(t, err)
		var xerr *xpath3.XPathError
		require.ErrorAs(t, err, &xerr)
		require.Equal(t, "XPTY0004", xerr.Code)
	})
	t.Run("map-form string builtin method is accepted", func(t *testing.T) {
		doc := mustParseXML(t, `<root>hi</root>`)
		res, err := evaluate(t.Context(), doc, `serialize(., map{"method":"text"})`)
		require.NoError(t, err)
		require.Equal(t, "hi", res.StringValue())
	})
}

// TestSerialize_NormalizationScope covers Serialization 3.1 §4: the
// normalization-form parameter is a step of the character-expansion phase, which
// is scoped to TEXT and ATTRIBUTE node character content only. Element/attribute
// NAMES, comment and PI markup, the DOCTYPE, and the XML declaration are NOT
// normalized even when a non-none normalization-form is set.
func TestSerialize_NormalizationScope(t *testing.T) {
	// U+00E9 (composed é) vs "e" + U+0301 (decomposed). The XML writer's default
	// non-ASCII escaping emits a composed é (U+00E9, < U+0100) as &#xE9;.
	const decomposed = "e\u0301"
	const composed = "\u00e9"
	const composedRef = "&#xE9;"

	t.Run("xml: names, comment, and PI are not normalized; text and attr value are", func(t *testing.T) {
		src := `<caf` + decomposed + ` at` + decomposed + `="` + decomposed + `">` +
			`<!--` + decomposed + `--><?p ` + decomposed + `?>` + decomposed + `</caf` + decomposed + `>`
		doc := mustParseXML(t, src)
		res, err := evaluate(t.Context(), doc,
			`serialize(., map{"method":"xml","omit-xml-declaration":true(),"normalization-form":"NFC"})`)
		require.NoError(t, err)
		out := res.StringValue()
		// Element/attribute names, comment content, and PI data stay decomposed.
		want := `<caf` + decomposed + ` at` + decomposed + `="` + composedRef + `">` +
			`<!--` + decomposed + `--><?p ` + decomposed + `?>` + composedRef + `</caf` + decomposed + `>`
		require.Equal(t, want, out)
		// A whole-string normalization would have composed the name/comment/PI too.
		require.NotContains(t, out, "caf"+composed, "element name must not be normalized")
		require.NotContains(t, out, "<!--"+composed, "comment must not be normalized")
	})

	t.Run("xml: attribute value is normalized", func(t *testing.T) {
		doc := mustParseXML(t, `<e a="`+decomposed+`"/>`)
		res, err := evaluate(t.Context(), doc,
			`serialize(., map{"method":"xml","omit-xml-declaration":true(),"normalization-form":"NFC"})`)
		require.NoError(t, err)
		require.Equal(t, `<e a="`+composedRef+`"/>`, res.StringValue())
	})

	t.Run("html: element name stays decomposed while text is normalized", func(t *testing.T) {
		// The html writer emits non-ASCII literally (no numeric-reference escaping),
		// so a normalized text node reads as the literal composed é.
		doc := mustParseXML(t, `<x`+decomposed+`>`+decomposed+`</x`+decomposed+`>`)
		res, err := evaluate(t.Context(), doc,
			`serialize(., map{"method":"html","normalization-form":"NFC"})`)
		require.NoError(t, err)
		require.Equal(t, `<x`+decomposed+`>`+composed+`</x`+decomposed+`>`, res.StringValue())
	})

	t.Run("xml: char-map replacement stays un-normalized while surrounding text is normalized", func(t *testing.T) {
		// Text "X" + decomposed é; map X -> decomposed é. The mapped replacement is
		// emitted verbatim (decomposed) while the surrounding decomposed é composes.
		doc := mustParseXML(t, `<e>X`+decomposed+`</e>`)
		res, err := evaluate(t.Context(), doc,
			`serialize(., map{"method":"xml","omit-xml-declaration":true(),"normalization-form":"NFC",`+
				`"use-character-maps":map{"X": codepoints-to-string((101, 769))}})`)
		require.NoError(t, err)
		// Replacement decomposed é (verbatim, U+0301 > U+00FF so not escaped) then
		// the surrounding é composed to &#xE9;.
		require.Equal(t, `<e>`+decomposed+composedRef+`</e>`, res.StringValue())
	})
}

// TestSerialize_CharMapURIAttributeExclusion covers the Serialization 3.1 §7
// character-expansion rule: when URI escaping is applied to a URI attribute value
// (escape-uri-attributes=yes, the default), character mapping is SKIPPED for that
// value; character maps apply to non-URI attributes normally, and to URI
// attributes only when escaping is disabled.
func TestSerialize_CharMapURIAttributeExclusion(t *testing.T) {
	const src = `<?xml version="1.0" encoding="UTF-8"?>` +
		`<html><head/><body><a href="ab" title="a">x</a></body></html>`

	t.Run("URI attr not char-mapped when escaping on (default); non-URI attr is", func(t *testing.T) {
		doc := mustParseXML(t, src)
		res, err := evaluate(t.Context(), doc,
			`serialize(., map{"method":"html","include-content-type":false(),"use-character-maps":map{"a":"Z"}})`)
		require.NoError(t, err)
		out := res.StringValue()
		// href is URI-escaped, so its "a" is NOT mapped to "Z".
		require.Contains(t, out, `href="ab"`, "output:\n%s", out)
		// title is not a URI attribute, so its "a" IS mapped.
		require.Contains(t, out, `title="Z"`, "output:\n%s", out)
	})

	t.Run("URI attr IS char-mapped when escape-uri-attributes=no", func(t *testing.T) {
		doc := mustParseXML(t, src)
		res, err := evaluate(t.Context(), doc,
			`serialize(., map{"method":"html","include-content-type":false(),"escape-uri-attributes":false(),"use-character-maps":map{"a":"Z"}})`)
		require.NoError(t, err)
		out := res.StringValue()
		// URI escaping disabled, so the character map applies to href too.
		require.Contains(t, out, `href="Zb"`, "output:\n%s", out)
		require.Contains(t, out, `title="Z"`, "output:\n%s", out)
	})
}

// TestSerialize_NormalizationCharMapAndEmptyDefaults covers three fixes:
//   - a character-map replacement string is NOT subjected to Unicode
//     Normalization (Serialization 3.1 §11) — a normalization pass must leave
//     replacements verbatim while normalizing the surrounding content;
//   - a map-form option value present with the empty sequence selects that
//     parameter's DEFAULT (present-empty = use default, not an error);
//   - the map-form json-node-output-method accepts a namespaced xs:QName without
//     losing its namespace (an extension → unsupported when a node is serialized).
func TestSerialize_NormalizationCharMapAndEmptyDefaults(t *testing.T) {
	// Finding 2: replacement string not normalized.
	t.Run("char-map replacement is not normalized (text method)", func(t *testing.T) {
		// Input "X" + decomposed "é"; map X -> decomposed "é" replacement; NFC.
		// The decomposed source é composes to U+00E9, but the X replacement stays
		// decomposed (verbatim).
		res, err := evaluate(t.Context(), nil,
			`serialize(codepoints-to-string((88, 101, 769)), `+
				`map{"method":"text","normalization-form":"NFC","use-character-maps":map{"X": codepoints-to-string((101, 769))}})`)
		require.NoError(t, err)
		require.Equal(t, "éé", res.StringValue())
	})
	t.Run("char-map replacement is not normalized (xml method)", func(t *testing.T) {
		doc := mustParseXML(t, "<e>X</e>")
		res, err := evaluate(t.Context(), doc,
			`serialize(., map{"method":"xml","omit-xml-declaration":true(),"normalization-form":"NFC",`+
				`"use-character-maps":map{"X": codepoints-to-string((101, 769))}})`)
		require.NoError(t, err)
		// The X replacement (decomposed "é") is emitted verbatim, un-normalized.
		require.Equal(t, "<e>é</e>", res.StringValue())
	})

	// Finding 3: present-empty map option value applies the default.
	t.Run("indent () applies the default (no indentation)", func(t *testing.T) {
		doc := mustParseXML(t, `<a><b>x</b></a>`)
		res, err := evaluate(t.Context(), doc, `serialize(., map{"method":"xml","indent":()})`)
		require.NoError(t, err)
		require.Equal(t, "<a><b>x</b></a>", res.StringValue())
	})
	t.Run("method () applies the default xml method", func(t *testing.T) {
		doc := mustParseXML(t, `<r>x</r>`)
		res, err := evaluate(t.Context(), doc, `serialize(., map{"method":()})`)
		require.NoError(t, err)
		require.Contains(t, res.StringValue(), "<r>x</r>", "output:\n%s", res.StringValue())
	})
	t.Run("cdata-section-elements () applies the default (none)", func(t *testing.T) {
		doc := mustParseXML(t, `<doc><b>x</b></doc>`)
		res, err := evaluate(t.Context(), doc, `serialize(., map{"method":"xml","cdata-section-elements":()})`)
		require.NoError(t, err)
		out := res.StringValue()
		require.Contains(t, out, "<b>x</b>", "output:\n%s", out)
		require.NotContains(t, out, "CDATA", "output:\n%s", out)
	})
	t.Run("use-character-maps () applies the default (no mapping)", func(t *testing.T) {
		doc := mustParseXML(t, `<e>x</e>`)
		res, err := evaluate(t.Context(), doc, `serialize(., map{"method":"xml","use-character-maps":()})`)
		require.NoError(t, err)
		require.Contains(t, res.StringValue(), "<e>x</e>", "output:\n%s", res.StringValue())
	})
	t.Run("html-version () applies the default (HTML5) with html method", func(t *testing.T) {
		doc := mustParseXML(t, `<?xml version="1.0" encoding="UTF-8"?><html><head/><body/></html>`)
		res, err := evaluate(t.Context(), doc,
			`serialize(., map{"method":"html","include-content-type":false(),"html-version":()})`)
		require.NoError(t, err)
		require.Contains(t, res.StringValue(), "<!DOCTYPE html>", "output:\n%s", res.StringValue())
	})
	t.Run("byte-order-mark () applies the default (no error)", func(t *testing.T) {
		doc := mustParseXML(t, `<r/>`)
		_, err := evaluate(t.Context(), doc, `serialize(., map{"method":"xml","byte-order-mark":()})`)
		require.NoError(t, err)
	})

	// Finding 4: map-form json-node-output-method namespaced QName.
	t.Run("json-node-output-method namespaced QName over a node is unsupported SEPM0016", func(t *testing.T) {
		doc := mustParseXML(t, `<a>x</a>`)
		_, err := evaluate(t.Context(), doc,
			`serialize(., map{"method":"json","json-node-output-method":QName("urn:x","custom")})`)
		require.Error(t, err)
		var xerr *xpath3.XPathError
		require.ErrorAs(t, err, &xerr)
		require.Equal(t, "SEPM0016", xerr.Code)
	})
	// F&O 3.1: an xs:QName json-node-output-method value must have a non-absent
	// namespace, so ANY no-namespace QName (domain token, xml, or otherwise) is an
	// invalid value [XPTY0004]; the built-in values are supplied as strings.
	t.Run("json-node-output-method no-namespace QName (domain token text) is XPTY0004", func(t *testing.T) {
		doc := mustParseXML(t, `<a>x</a>`)
		_, err := evaluate(t.Context(), doc,
			`serialize(., map{"method":"json","json-node-output-method":QName("","text")})`)
		require.Error(t, err)
		var xerr *xpath3.XPathError
		require.ErrorAs(t, err, &xerr)
		require.Equal(t, "XPTY0004", xerr.Code)
	})
	t.Run("json-node-output-method no-namespace QName (non-domain) is XPTY0004", func(t *testing.T) {
		doc := mustParseXML(t, `<a>x</a>`)
		_, err := evaluate(t.Context(), doc,
			`serialize(., map{"method":"json","json-node-output-method":QName("","custom")})`)
		require.Error(t, err)
		var xerr *xpath3.XPathError
		require.ErrorAs(t, err, &xerr)
		require.Equal(t, "XPTY0004", xerr.Code)
	})
	t.Run("json-node-output-method no-namespace QName xml is XPTY0004", func(t *testing.T) {
		doc := mustParseXML(t, `<a>x</a>`)
		_, err := evaluate(t.Context(), doc,
			`serialize(., map{"method":"json","json-node-output-method":QName("","xml")})`)
		require.Error(t, err)
		var xerr *xpath3.XPathError
		require.ErrorAs(t, err, &xerr)
		require.Equal(t, "XPTY0004", xerr.Code)
	})
	t.Run("json-node-output-method string xml (default) over a node is fine", func(t *testing.T) {
		doc := mustParseXML(t, `<a>x</a>`)
		res, err := evaluate(t.Context(), doc,
			`serialize(., map{"method":"json","json-node-output-method":"xml"})`)
		require.NoError(t, err)
		require.Contains(t, res.StringValue(), "<a>x", "output:\n%s", res.StringValue())
	})
}

// TestSerialize_HTMLDoctypeNameFixedHTML verifies the html output method's
// explicit DOCTYPE uses the spec-mandated name "html" (HTML5 / Serialization 3.1
// html-method doctype rule), NOT the source document element's arbitrary case, so
// a mixed-case <HtMl> root still emits `<!DOCTYPE html ...>`.
func TestSerialize_HTMLDoctypeNameFixedHTML(t *testing.T) {
	doc := mustParseXML(t, `<?xml version="1.0" encoding="UTF-8"?><HtMl><head/><body/></HtMl>`)
	res, err := evaluate(t.Context(), doc,
		`serialize(., map{"method":"html","include-content-type":false(),"doctype-system":"about:legacy-compat"})`)
	require.NoError(t, err)
	out := res.StringValue()
	require.Contains(t, out, `<!DOCTYPE html SYSTEM "about:legacy-compat">`, "output:\n%s", out)
	require.NotContains(t, out, "DOCTYPE HtMl", "output:\n%s", out)
}

// TestSerialize_EncodingDeclarationAndDoctypeSystemQuotes covers two spec
// requirements: the XML declaration carries an encoding declaration
// (Serialization 3.1 §5.1.6) driven by the encoding parameter (default UTF-8),
// and a doctype-system value containing BOTH quote characters is an invalid
// value (§3, SEPM0016), not malformed output.
func TestSerialize_EncodingDeclarationAndDoctypeSystemQuotes(t *testing.T) {
	t.Run("encoding param appears in the XML declaration", func(t *testing.T) {
		doc := mustParseXML(t, `<?xml version="1.0" encoding="UTF-8"?><root/>`)
		res, err := evaluate(t.Context(), doc,
			`serialize(., map{"method":"xml","omit-xml-declaration":false(),"encoding":"UTF-16"})`)
		require.NoError(t, err)
		require.Contains(t, res.StringValue(), `encoding="UTF-16"`, "output:\n%s", res.StringValue())
	})

	t.Run("map-form default encoding UTF-8 appears in the declaration", func(t *testing.T) {
		doc := mustParseXML(t, `<r/>`)
		res, err := evaluate(t.Context(), doc, `serialize(., map{"method":"xml","omit-xml-declaration":false()})`)
		require.NoError(t, err)
		require.Contains(t, res.StringValue(), `encoding="UTF-8"`, "output:\n%s", res.StringValue())
	})

	t.Run("doctype-system with both quote characters is SEPM0016", func(t *testing.T) {
		doc := mustParseXML(t, `<root/>`)
		// The XPath string literal doubles the embedded " so the doctype-system
		// value is a"b'c (containing both a quotation mark and an apostrophe).
		_, err := evaluate(t.Context(), doc,
			`serialize(., map{"method":"xml","omit-xml-declaration":false(),"doctype-system":"a""b'c"})`)
		require.Error(t, err)
		var xerr *xpath3.XPathError
		require.ErrorAs(t, err, &xerr)
		require.Equal(t, "SEPM0016", xerr.Code)
	})

	t.Run("doctype-system with only a double quote is accepted", func(t *testing.T) {
		doc := mustParseXML(t, `<?xml version="1.0" encoding="UTF-8"?><root><a>x</a></root>`)
		res, err := evaluate(t.Context(), doc,
			`serialize(., map{"method":"xml","omit-xml-declaration":false(),"doctype-system":"a'b"})`)
		require.NoError(t, err)
		require.Contains(t, res.StringValue(), `<!DOCTYPE root SYSTEM "a'b">`, "output:\n%s", res.StringValue())
	})

	t.Run("html doctype-system containing a double quote is single-quoted", func(t *testing.T) {
		// html output method + a doctype-system value that contains only a double
		// quote (valid: SEPM0016 rejects only BOTH quotes). The SYSTEM literal must
		// be apostrophe-enclosed (SYSTEM 'a"b'), not the malformed SYSTEM "a"b".
		doc := mustParseXML(t, `<html><head/><body/></html>`)
		res, err := evaluate(t.Context(), doc,
			`serialize(., map{"method":"html","doctype-system":"a""b"})`)
		require.NoError(t, err)
		require.Contains(t, res.StringValue(), `<!DOCTYPE html SYSTEM 'a"b'>`, "output:\n%s", res.StringValue())
	})

	t.Run("html doctype-public plus doctype-system with a double quote", func(t *testing.T) {
		// PUBLIC id is double-quoted (a PubidLiteral can never hold a "), while the
		// following SYSTEM literal switches to apostrophes for its embedded quote.
		doc := mustParseXML(t, `<html><head/><body/></html>`)
		res, err := evaluate(t.Context(), doc,
			`serialize(., map{"method":"html","doctype-public":"-//X//EN","doctype-system":"a""b"})`)
		require.NoError(t, err)
		require.Contains(t, res.StringValue(), `<!DOCTYPE html PUBLIC "-//X//EN" 'a"b'>`, "output:\n%s", res.StringValue())
	})
}

// TestSerialize_JSONCharacterMapsAndNormalization covers Serialization 3.1
// §9.1.11 / §9.1.9: use-character-maps AND normalization-form ARE applicable to
// the json output method (matching Saxon). A mapped character is replaced by its
// verbatim replacement instead of being JSON-escaped, the replacement is neither
// re-escaped nor normalized (§11), and normalization applies to the JSON string
// content.
func TestSerialize_JSONCharacterMapsAndNormalization(t *testing.T) {
	t.Run("character map replaces a character in a JSON string value", func(t *testing.T) {
		res, err := evaluate(t.Context(), nil,
			`serialize("x", map{"method":"json","use-character-maps":map{"x":"y"}})`)
		require.NoError(t, err)
		require.Equal(t, `"y"`, res.StringValue())
	})

	t.Run("character map prevents JSON escaping of a slash (Saxon example)", func(t *testing.T) {
		// Default json escaping renders "/" as "\/"; a char map "/"->"/" emits the
		// verbatim replacement instead, so the output is an unescaped "/".
		res, err := evaluate(t.Context(), nil,
			`serialize("a/b", map{"method":"json","use-character-maps":map{"/":"/"}})`)
		require.NoError(t, err)
		require.Equal(t, `"a/b"`, res.StringValue())
	})

	t.Run("character map applies to JSON object keys", func(t *testing.T) {
		res, err := evaluate(t.Context(), nil,
			`serialize(map{"x":1}, map{"method":"json","use-character-maps":map{"x":"y"}})`)
		require.NoError(t, err)
		require.Contains(t, res.StringValue(), `"y"`, "output:\n%s", res.StringValue())
	})

	t.Run("character map replacement in JSON is not re-escaped", func(t *testing.T) {
		// In XPath a string literal does NOT process backslash escapes, so "\n" is
		// the two characters backslash+n; the char map emits it verbatim, NOT
		// re-escaped to a doubled backslash. Output is "\n" (quote, backslash, n,
		// quote).
		res, err := evaluate(t.Context(), nil,
			`serialize("x", map{"method":"json","use-character-maps":map{"x":"\n"}})`)
		require.NoError(t, err)
		require.Equal(t, "\"\\n\"", res.StringValue())
	})

	t.Run("char-map replacement in JSON is not normalized", func(t *testing.T) {
		// Input "X" + decomposed "é"; map X -> decomposed "é"; NFC. The source é
		// composes to U+00E9, but the X replacement stays decomposed "e"+U+0301
		// (verbatim, §11).
		res, err := evaluate(t.Context(), nil,
			`serialize(codepoints-to-string((88, 101, 769)), `+
				`map{"method":"json","normalization-form":"NFC","use-character-maps":map{"X": codepoints-to-string((101, 769))}})`)
		require.NoError(t, err)
		require.Equal(t, "\"e\u0301\u00e9\"", res.StringValue())
	})
}

// TestSerialize_TextMethodSENR0001 verifies that sequence normalization
// (Serialization 3.1 §2 — "It is a serialization error [err:SENR0001] if an item
// … is an attribute node, a namespace node or a function item") applies to the
// text output method too: such items are rejected with SENR0001 rather than
// silently contributing their string value. This routes through the same
// serializeNodeKindError checkpoint the xml/xhtml/html methods use.
func TestSerialize_TextMethodSENR0001(t *testing.T) {
	requireSENR0001 := func(t *testing.T, src, expr string) {
		t.Helper()
		doc := mustParseXML(t, src)
		_, err := evaluate(t.Context(), doc, expr)
		require.Error(t, err)
		var xerr *xpath3.XPathError
		require.ErrorAs(t, err, &xerr)
		require.Equal(t, "SENR0001", xerr.Code)
	}

	t.Run("attribute node", func(t *testing.T) {
		requireSENR0001(t, `<root id="x">text</root>`, `serialize(/root/@id, map{"method":"text"})`)
	})
	t.Run("namespace node", func(t *testing.T) {
		requireSENR0001(t, `<root xmlns:p="urn:p">text</root>`, `serialize(/root/namespace::*[local-name()='p'], map{"method":"text"})`)
	})
	t.Run("function item", func(t *testing.T) {
		requireSENR0001(t, `<root>text</root>`, `serialize(true#0, map{"method":"text"})`)
	})
}

// TestSerialize_HTMLDoctypeNonHTMLRoot verifies Serialization 3.1 §7.4.6: an
// explicitly specified doctype-system/doctype-public makes the html output method
// emit a document type declaration (named "html", not the document element's name)
// immediately before the first element, regardless of the document element's name
// or whether the input is a bare element. With no explicit doctype a
// non-html-rooted node stays a fragment — the default <!DOCTYPE html> is emitted
// only for an <html>-rooted document.
func TestSerialize_HTMLDoctypeNonHTMLRoot(t *testing.T) {
	t.Run("non-html document + doctype-system emits DOCTYPE html", func(t *testing.T) {
		doc := mustParseXML(t, `<page><body><p>Hi</p></body></page>`)
		res, err := evaluate(t.Context(), doc,
			`serialize(., map{"method":"html","doctype-system":"about:legacy-compat"})`)
		require.NoError(t, err)
		out := res.StringValue()
		require.Contains(t, out, `<!DOCTYPE html SYSTEM "about:legacy-compat">`, "output:\n%s", out)
		// The DOCTYPE name is the fixed token "html", never the root element name.
		require.NotContains(t, out, "<!DOCTYPE page", "output:\n%s", out)
	})

	t.Run("bare non-html element + doctype-public/system emits DOCTYPE html", func(t *testing.T) {
		doc := mustParseXML(t, `<page><body><p>Hi</p></body></page>`)
		root := doc.DocumentElement() // a bare element, not a document node
		res, err := evaluate(t.Context(), root,
			`serialize(., map{"method":"html","doctype-public":"-//X//DTD//EN","doctype-system":"x.dtd"})`)
		require.NoError(t, err)
		out := res.StringValue()
		require.Contains(t, out, `<!DOCTYPE html PUBLIC "-//X//DTD//EN" "x.dtd">`, "output:\n%s", out)
		require.Contains(t, out, "<body>", "output:\n%s", out)
	})

	t.Run("non-html root without explicit doctype stays a fragment", func(t *testing.T) {
		doc := mustParseXML(t, `<page><body><p>Hi</p></body></page>`)
		res, err := evaluate(t.Context(), doc, `serialize(., map{"method":"html"})`)
		require.NoError(t, err)
		require.NotContains(t, res.StringValue(), "<!DOCTYPE", "output:\n%s", res.StringValue())
	})
}
