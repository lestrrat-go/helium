package xpointer_test

import (
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpointer"
	"github.com/stretchr/testify/require"
)

const schemeElement = "element"

func TestEvaluateXPath1(t *testing.T) {
	t.Parallel()

	doc, err := helium.NewParser().Parse(t.Context(), []byte("<root><child>text</child></root>"))
	require.NoError(t, err)

	nodes, err := xpointer.Evaluate(t.Context(), doc, "xpath1(/root/child)")
	require.NoError(t, err)
	require.Len(t, nodes, 1)
	require.Equal(t, "child", nodes[0].Name())
}

// A compiled expression must produce the same results as one-shot Evaluate
// across multiple documents, with parsing + XPath compilation done once.
func TestCompile_ReuseAcrossDocuments(t *testing.T) {
	t.Parallel()

	compiled, err := xpointer.Compile("xpath1(/root/child)")
	require.NoError(t, err)

	for _, src := range []string{
		"<root><child>a</child></root>",
		"<root><child>b</child><child>c</child></root>",
		"<root><other/><child>d</child></root>",
	} {
		doc, perr := helium.NewParser().Parse(t.Context(), []byte(src))
		require.NoError(t, perr, src)

		nodes, eerr := compiled.Evaluate(t.Context(), doc)
		require.NoError(t, eerr, src)
		require.NotEmpty(t, nodes, src)
		for _, n := range nodes {
			require.Equal(t, "child", n.Name(), src)
		}
	}
}

// Compile must surface XPath syntax errors at compile time rather than
// deferring them to each Evaluate call.
func TestCompile_ReportsXPathSyntaxErrorEarly(t *testing.T) {
	t.Parallel()
	_, err := xpointer.Compile("xpath1(///)")
	require.Error(t, err, "compile should reject malformed xpath1 body")
}

func TestParseFragmentID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		fragment string
		scheme   string
		body     string
	}{
		{"foo", "", "foo"},
		{"xpointer(//p)", "xpointer", "//p"},
		{"xpath1(/root/child)", "xpath1", "/root/child"},
		{"element(/1/2)", schemeElement, "/1/2"},
	}

	for _, test := range tests {
		scheme, body, err := xpointer.ParseFragmentID(test.fragment)
		require.NoError(t, err, "ParseFragmentID(%q)", test.fragment)
		require.Equal(t, test.scheme, scheme, "ParseFragmentID(%q) scheme", test.fragment)
		require.Equal(t, test.body, body, "ParseFragmentID(%q) body", test.fragment)
	}
}

func TestXmlnsScheme(t *testing.T) {
	t.Parallel()

	t.Run("xmlns with xpath1", func(t *testing.T) {
		t.Parallel()

		// Mirrors libxml2 issue289base test
		doc, err := helium.NewParser().Parse(t.Context(), []byte(`<?xml version="1.0"?>
<rootB xmlns="abc://d/e:f">
</rootB>`))
		require.NoError(t, err)

		nodes, err := xpointer.Evaluate(t.Context(), doc, `xmlns(b=abc://d/e:f) xpath1(/b:rootB)`)
		require.NoError(t, err)
		require.Len(t, nodes, 1)
		require.Equal(t, "rootB", nodes[0].Name())
	})

	t.Run("xmlns with xpointer", func(t *testing.T) {
		t.Parallel()

		doc, err := helium.NewParser().Parse(t.Context(), []byte(`<?xml version="1.0"?>
<root xmlns:ns="urn:test"><ns:child>hello</ns:child></root>`))
		require.NoError(t, err)

		nodes, err := xpointer.Evaluate(t.Context(), doc, `xmlns(n=urn:test) xpointer(/root/n:child)`)
		require.NoError(t, err)
		require.Len(t, nodes, 1)
		require.Equal(t, "child", nodes[0].(*helium.Element).LocalName())
	})

	t.Run("multiple xmlns bindings", func(t *testing.T) {
		t.Parallel()

		doc, err := helium.NewParser().Parse(t.Context(), []byte(`<?xml version="1.0"?>
<root xmlns:a="urn:a" xmlns:b="urn:b"><a:x/><b:y/></root>`))
		require.NoError(t, err)

		nodes, err := xpointer.Evaluate(t.Context(), doc, `xmlns(p=urn:a) xmlns(q=urn:b) xpointer(/root/q:y)`)
		require.NoError(t, err)
		require.Len(t, nodes, 1)
		require.Equal(t, "y", nodes[0].(*helium.Element).LocalName())
	})

	t.Run("invalid xmlns body", func(t *testing.T) {
		t.Parallel()

		doc, err := helium.NewParser().Parse(t.Context(), []byte(`<root/>`))
		require.NoError(t, err)

		_, err = xpointer.Evaluate(t.Context(), doc, `xmlns(noequalssign) xpointer(/root)`)
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid xmlns()")
	})
}

func TestXPointerEscaping(t *testing.T) {
	t.Parallel()

	// Test XPointer escaping through public APIs with table-driven approach
	tests := []struct {
		name     string
		expr     string
		wantErr  bool
		wantType string // expected node type or "error" for error cases
	}{
		{"simple expression", "element(/1)", false, schemeElement},
		{"xpath expression", "xpath1(//root)", false, schemeElement},
		{"fragment id simple", "testid", false, schemeElement},
		{"xmlns binding", "xmlns(t=test) xpath1(//root)", false, schemeElement},
		{"cascading parts", "element(nonexist)element(/1)", false, schemeElement},
		{"invalid expression", "xpointer(:::invalid)", true, "error"},
	}

	doc, err := helium.NewParser().Parse(t.Context(), []byte(`<root xml:id="testid"><child>test</child></root>`))
	require.NoError(t, err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			nodes, err := xpointer.Evaluate(t.Context(), doc, tt.expr)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			if tt.wantType == schemeElement {
				require.NotNil(t, nodes)
				require.Greater(t, len(nodes), 0)
			}
		})
	}
}

func TestParseFragmentIDTable(t *testing.T) {
	t.Parallel()

	// Test ParseFragmentID with table-driven approach
	tests := []struct {
		name       string
		fragment   string
		wantScheme string
		wantBody   string
		wantErr    bool
	}{
		{"bare name", "foo", "", "foo", false},
		{"xpointer scheme", "xpointer(//p)", "xpointer", "//p", false},
		{"xpath1 scheme", "xpath1(/root/child)", "xpath1", "/root/child", false},
		{"element scheme", "element(/1/2)", schemeElement, "/1/2", false},
		{"xmlns scheme", "xmlns(ns=uri)", "xmlns", "ns=uri", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			scheme, body, err := xpointer.ParseFragmentID(tt.fragment)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.wantScheme, scheme)
			require.Equal(t, tt.wantBody, body)
		})
	}
}

func TestCascadingParts(t *testing.T) {
	t.Parallel()

	// Document with known structure: <book><chapter><image/></chapter></book>
	doc, err := helium.NewParser().Parse(t.Context(), []byte(`<book><chapter><image href="linus.gif"/></chapter></book>`))
	require.NoError(t, err)

	t.Run("first part fails, second succeeds", func(t *testing.T) {
		t.Parallel()

		// element(foo) fails (no ID "foo"), element(/1/1/1) succeeds
		nodes, err := xpointer.Evaluate(t.Context(), doc, "element(foo)element(/1/1/1)")
		require.NoError(t, err)
		require.Len(t, nodes, 1)
		require.Equal(t, "image", nodes[0].(*helium.Element).LocalName())
	})

	t.Run("first part succeeds, second ignored", func(t *testing.T) {
		t.Parallel()

		// element(/1/1/1) succeeds immediately, element(foo) never tried
		nodes, err := xpointer.Evaluate(t.Context(), doc, "element(/1/1/1)element(foo)")
		require.NoError(t, err)
		require.Len(t, nodes, 1)
		require.Equal(t, "image", nodes[0].(*helium.Element).LocalName())
	})

	t.Run("all parts fail", func(t *testing.T) {
		t.Parallel()

		// Both element(foo) and element(bar) fail
		nodes, err := xpointer.Evaluate(t.Context(), doc, "element(foo)element(bar)")
		require.NoError(t, err)
		require.Nil(t, nodes)
	})

	t.Run("xpath1 cascade", func(t *testing.T) {
		t.Parallel()

		// First XPath returns empty, second finds the element
		nodes, err := xpointer.Evaluate(t.Context(), doc, "xpath1(//nonexistent)xpath1(//image)")
		require.NoError(t, err)
		require.Len(t, nodes, 1)
		require.Equal(t, "image", nodes[0].(*helium.Element).LocalName())
	})

	t.Run("syntax error aborts cascade", func(t *testing.T) {
		t.Parallel()

		// xpointer with invalid XPath is a syntax error — cascade aborts
		_, err := xpointer.Evaluate(t.Context(), doc, "xpointer(:::invalid)element(/1/1/1)")
		require.Error(t, err)
	})

	t.Run("unknown scheme continues cascade", func(t *testing.T) {
		t.Parallel()

		// unknown scheme allows fallback to next part
		nodes, err := xpointer.Evaluate(t.Context(), doc, "bogus(data)element(/1/1/1)")
		require.NoError(t, err)
		require.Len(t, nodes, 1)
		require.Equal(t, "image", nodes[0].(*helium.Element).LocalName())
	})
}

func TestBareNameChildSequence(t *testing.T) {
	t.Parallel()

	// Parse with NewParser so xml:id registers in the ID table.
	p := helium.NewParser()
	doc, err := p.Parse(t.Context(), []byte(`<?xml version="1.0"?>
<root xml:id="r"><a><b>found</b></a></root>`))
	require.NoError(t, err)

	t.Run("name/1/1 navigates from ID", func(t *testing.T) {
		t.Parallel()

		// "r/1/1" = look up ID "r", then 1st child (a), then 1st child (b)
		nodes, err := xpointer.Evaluate(t.Context(), doc, "r/1/1")
		require.NoError(t, err)
		require.Len(t, nodes, 1)
		require.Equal(t, "b", nodes[0].(*helium.Element).LocalName())
	})

	t.Run("name/1 navigates one level", func(t *testing.T) {
		t.Parallel()

		nodes, err := xpointer.Evaluate(t.Context(), doc, "r/1")
		require.NoError(t, err)
		require.Len(t, nodes, 1)
		require.Equal(t, "a", nodes[0].(*helium.Element).LocalName())
	})

	t.Run("unknown ID returns nil", func(t *testing.T) {
		t.Parallel()

		nodes, err := xpointer.Evaluate(t.Context(), doc, "nosuchid/1")
		require.NoError(t, err)
		require.Nil(t, nodes)
	})
}

func TestElementChildIndexBounds(t *testing.T) {
	t.Parallel()

	doc, err := helium.NewParser().Parse(t.Context(), []byte(`<?xml version="1.0"?>
<root xml:id="r"><child>text</child></root>`))
	require.NoError(t, err)

	tests := []struct {
		name string
		expr string
	}{
		{"zero index", "element(/0)"},
		{"negative index", "element(/-1)"},
		{"zero index mid-sequence", "element(/1/0)"},
		{"zero index from id", "element(r/0)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// A child-sequence index < 1 is malformed: it must be an error,
			// not a silent empty result (which would unlink an XInclude node).
			nodes, err := xpointer.Evaluate(t.Context(), doc, tt.expr)
			require.Error(t, err, "expr %q must be rejected", tt.expr)
			require.Nil(t, nodes)
		})
	}
}

func TestElementChildIndexOverflow(t *testing.T) {
	t.Parallel()

	doc, err := helium.NewParser().Parse(t.Context(), []byte(`<?xml version="1.0"?>
<root xml:id="r"><child>text</child></root>`))
	require.NoError(t, err)

	tests := []struct {
		name string
		expr string
	}{
		// 18446744073709551617 == math.MaxUint64+2. Naive base-10 accumulation
		// into an int wraps this to 1 and would silently select the first child.
		{"absolute overflow", "element(/18446744073709551617)"},
		{"overflow then index", "element(/18446744073709551617/1)"},
		{"index then overflow", "element(/1/18446744073709551617)"},
		{"overflow from id", "element(r/99999999999999999999)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// An index that exceeds the int range must be reported as a syntax
			// error, never wrapped to a small in-range value (which would select
			// the wrong node) or coerced to a silent empty result.
			nodes, err := xpointer.Evaluate(t.Context(), doc, tt.expr)
			require.Error(t, err, "expr %q must be rejected as out of range", tt.expr)
			require.Nil(t, nodes)
		})
	}
}

func TestShorthandAfterSchemeRejected(t *testing.T) {
	t.Parallel()

	// Parse with NewParser so xml:id registers in the ID table.
	doc, err := helium.NewParser().Parse(t.Context(), []byte(`<?xml version="1.0"?>
<root><target xml:id="fallback">found</target></root>`))
	require.NoError(t, err)

	// A bare shorthand appended after a scheme-based part is invalid; it must
	// not select the "fallback" element. The whole pointer must fail to parse.
	nodes, err := xpointer.Evaluate(t.Context(), doc, "xpointer(//missing)fallback")
	require.Error(t, err)
	require.Nil(t, nodes)

	// Sanity check: the same shorthand on its own still resolves the ID.
	nodes, err = xpointer.Evaluate(t.Context(), doc, "fallback")
	require.NoError(t, err)
	require.Len(t, nodes, 1)
	require.Equal(t, "target", nodes[0].(*helium.Element).LocalName())
}

func TestTrailingChildSequenceAfterSchemeRejected(t *testing.T) {
	t.Parallel()

	doc, err := helium.NewParser().Parse(t.Context(), []byte(`<?xml version="1.0"?>
<root><target xml:id="fallback">found</target></root>`))
	require.NoError(t, err)

	// A trailing bare child-sequence appended after a scheme-based part is
	// malformed: it must be rejected, not silently ignored. Silently ignoring
	// it would let XInclude unlink the include node instead of reporting the
	// malformed pointer.
	for _, expr := range []string{
		"xpointer(//missing)r/1",
		"xpointer(//missing)/1",
	} {
		nodes, err := xpointer.Evaluate(t.Context(), doc, expr)
		require.Error(t, err, "expr %q must be rejected", expr)
		require.Nil(t, nodes, "expr %q", expr)
	}

	// The tolerated compatibility exception: a lone unbalanced ")" left over
	// from a scheme body (libxml2 parity / xinclude coalesce.xml golden test).
	nodes, err := xpointer.Evaluate(t.Context(), doc, "xpointer(//missing))")
	require.NoError(t, err, "lone trailing ) must remain tolerated")
	require.Nil(t, nodes)
}

func TestInvalidSchemeNameRejected(t *testing.T) {
	t.Parallel()

	doc, err := helium.NewParser().Parse(t.Context(), []byte(`<?xml version="1.0"?>
<root><target xml:id="x">found</target></root>`))
	require.NoError(t, err)

	// A scheme name is a QName per the XPointer framework. A malformed scheme
	// name must be a syntax error, NOT an unknown-scheme cascade — otherwise a
	// later well-formed part (element(/1)) could succeed and silently bypass
	// the malformed leading part. libxml2 rejects these.
	for _, expr := range []string{
		"1bad(/x)",
		"xpointer(//missing)1bad(/x)element(/1)",
		"-bad(/x)",
		"bad name(/x)",
		":bad(/x)",
	} {
		nodes, err := xpointer.Evaluate(t.Context(), doc, expr)
		require.Error(t, err, "expr %q must be rejected as a syntax error", expr)
		require.Nil(t, nodes, "expr %q", expr)
	}

	// A syntactically valid but unsupported scheme name is a QName, so it must
	// continue to cascade as an unknown scheme rather than abort parsing. Here
	// the unknown "foo" scheme yields no nodes and the following element() part
	// resolves the target.
	nodes, err := xpointer.Evaluate(t.Context(), doc, "foo(/x)element(x)")
	require.NoError(t, err)
	require.Len(t, nodes, 1)
	require.Equal(t, "target", nodes[0].(*helium.Element).LocalName())
}

func TestElementBodyValidatedBeforeLookup(t *testing.T) {
	t.Parallel()

	doc, err := helium.NewParser().Parse(t.Context(), []byte(`<?xml version="1.0"?>
<root xml:id="r"><child>text</child></root>`))
	require.NoError(t, err)

	// A malformed element() body must be reported as a syntax error regardless
	// of whether the leading ID exists. element(0) has an invalid leading token
	// (a bare "0" is neither an NCName nor an absolute child sequence), and
	// element(missing/0) has an out-of-range index whose ID does not exist.
	for _, expr := range []string{
		"element(0)",
		"element(missing/0)",
	} {
		nodes, err := xpointer.Evaluate(t.Context(), doc, expr)
		require.Error(t, err, "expr %q must be rejected", expr)
		require.Nil(t, nodes, "expr %q", expr)
	}
}

func TestElementGrammarStrictness(t *testing.T) {
	t.Parallel()

	// Document with an NCName id "id" and nested structure for the valid cases.
	doc, err := helium.NewParser().Parse(t.Context(), []byte(`<?xml version="1.0"?>
<root xml:id="id"><a><b>found</b></a></root>`))
	require.NoError(t, err)

	rejected := []struct {
		name string
		expr string
	}{
		// Empty child-sequence segments (trailing slash, doubled slash).
		{"trailing slash absolute", "element(/)"},
		{"doubled slash absolute", "element(/1//2)"},
		{"trailing slash from id", "element(r/)"},
		// Index lexeme grammar [1-9][0-9]*.
		{"zero absolute", "element(/0)"},
		{"bare zero token", "element(0)"},
		{"plus-signed index", "element(+1)"},
		{"leading-zero index", "element(01)"},
		{"negative index", "element(/-1)"},
		// Missing/out-of-range index whose ID does not exist.
		{"missing id with zero index", "element(missing/0)"},
		{"id with trailing slash", "element(id/)"},
	}
	for _, tt := range rejected {
		t.Run("reject/"+tt.name, func(t *testing.T) {
			t.Parallel()

			nodes, err := xpointer.Evaluate(t.Context(), doc, tt.expr)
			require.Error(t, err, "expr %q must be rejected", tt.expr)
			require.Nil(t, nodes, "expr %q", tt.expr)
		})
	}

	valid := []struct {
		name     string
		expr     string
		wantName string
	}{
		{"id plus sequence", "element(id/1/1)", "b"},
		{"absolute sequence", "element(/1/1)", "a"},
		{"absolute single", "element(/1)", "root"},
		{"id alone", "element(id)", "root"},
	}
	for _, tt := range valid {
		t.Run("accept/"+tt.name, func(t *testing.T) {
			t.Parallel()

			nodes, err := xpointer.Evaluate(t.Context(), doc, tt.expr)
			require.NoError(t, err, "expr %q must be accepted", tt.expr)
			require.Len(t, nodes, 1, "expr %q", tt.expr)
			require.Equal(t, tt.wantName, nodes[0].(*helium.Element).LocalName())
		})
	}
}

func TestShorthandNCNameStrictness(t *testing.T) {
	t.Parallel()

	doc, err := helium.NewParser().Parse(t.Context(), []byte(`<?xml version="1.0"?>
<root xml:id="good">found</root>`))
	require.NoError(t, err)

	t.Run("valid NCName resolves", func(t *testing.T) {
		t.Parallel()

		nodes, err := xpointer.Evaluate(t.Context(), doc, "good")
		require.NoError(t, err)
		require.Len(t, nodes, 1)
		require.Equal(t, "root", nodes[0].(*helium.Element).LocalName())
	})

	// A shorthand pointer that is not a valid NCName must be a syntax error,
	// not a silent empty result. Includes a colon (NCNames are non-colonized),
	// a leading digit, and invalid UTF-8.
	for _, expr := range []string{"1bad", "a:b", "bad name", "\xff\xfe"} {
		t.Run("reject/"+expr, func(t *testing.T) {
			t.Parallel()

			nodes, err := xpointer.Evaluate(t.Context(), doc, expr)
			require.Error(t, err, "expr %q must be rejected", expr)
			require.Nil(t, nodes, "expr %q", expr)
		})
	}
}

func TestMultiSchemeExpressionsTable(t *testing.T) {
	t.Parallel()

	// Test expressions with multiple schemes using table-driven approach
	tests := []struct {
		name      string
		xml       string
		expr      string
		wantCount int
		wantName  string // expected element name if wantCount > 0
		wantErr   bool
	}{
		{
			name:      "single xpointer scheme",
			xml:       `<root><child>content</child></root>`,
			expr:      "xpointer(/root/child)",
			wantCount: 1,
			wantName:  "child",
		},
		{
			name:      "xmlns plus xpath1",
			xml:       `<root xmlns:b="urn:test"><b:item>test</b:item></root>`,
			expr:      "xmlns(b=urn:test) xpath1(/root/b:item)",
			wantCount: 1,
			wantName:  "item",
		},
		{
			name:      "bare name with ID",
			xml:       `<root><item xml:id="myid">content</item></root>`,
			expr:      "myid",
			wantCount: 1,
			wantName:  "item",
		},
		{
			name:      "cascading schemes - first fails",
			xml:       `<root><child>test</child></root>`,
			expr:      "element(nonexistent)xpath1(//child)",
			wantCount: 1,
			wantName:  "child",
		},
		{
			name:      "all schemes fail",
			xml:       `<root><child>test</child></root>`,
			expr:      "element(foo)element(bar)",
			wantCount: 0,
		},
		{
			name:    "syntax error in scheme",
			xml:     `<root/>`,
			expr:    "xpointer(:::invalid)",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			doc, err := helium.NewParser().Parse(t.Context(), []byte(tt.xml))
			require.NoError(t, err)

			nodes, err := xpointer.Evaluate(t.Context(), doc, tt.expr)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			if tt.wantCount == 0 {
				require.Nil(t, nodes)
			} else {
				require.Len(t, nodes, tt.wantCount)
				if tt.wantName != "" {
					require.Equal(t, tt.wantName, nodes[0].(*helium.Element).LocalName())
				}
			}
		})
	}
}

// A nil compiled expression or a nil document reaching evaluation must
// return an error, not panic. This covers the nil *Expression receiver and
// every scheme path (shorthand, element(), xpointer()) that dereferences the
// document during evaluation.
func TestEvaluate_NilGuards(t *testing.T) {
	t.Parallel()

	t.Run("nil expression receiver", func(t *testing.T) {
		t.Parallel()

		doc, err := helium.NewParser().Parse(t.Context(), []byte(`<r><a/></r>`))
		require.NoError(t, err)

		var e *xpointer.Expression
		_, err = e.Evaluate(t.Context(), doc)
		require.Error(t, err)
	})

	docExprs := []struct {
		name string
		expr string
	}{
		{name: "shorthand", expr: "shorthandID"},
		{name: "element", expr: "element(/1)"},
		{name: "xpointer", expr: "xpointer(/r)"},
	}
	for _, tt := range docExprs {
		t.Run("nil document "+tt.name, func(t *testing.T) {
			t.Parallel()

			e, err := xpointer.Compile(tt.expr)
			require.NoError(t, err)

			_, err = e.Evaluate(t.Context(), nil)
			require.Error(t, err)
		})
	}
}

// The xmlns() scheme must reject prefixes that are not valid NCNames as well
// as the reserved prefixes "xml" and "xmlns", which may not be (re)bound.
func TestEvaluate_XmlnsPrefixValidation(t *testing.T) {
	t.Parallel()

	bad := []struct {
		name string
		expr string
	}{
		{name: "reserved xmlns", expr: "xmlns(xmlns=http://example.com/ns)xpointer(/r)"},
		{name: "reserved xml", expr: "xmlns(xml=http://example.com/ns)xpointer(/r)"},
		{name: "non-NCName prefix", expr: "xmlns(1bad=http://example.com/ns)xpointer(/r)"},
		{name: "prefix with colon", expr: "xmlns(a:b=http://example.com/ns)xpointer(/r)"},
	}
	for _, tt := range bad {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := xpointer.Compile(tt.expr)
			require.Error(t, err)
		})
	}
}
