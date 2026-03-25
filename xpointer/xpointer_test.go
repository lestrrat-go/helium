package xpointer_test

import (
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpointer"
	"github.com/stretchr/testify/require"
)

func TestEvaluateXPath1(t *testing.T) {
	doc, err := helium.NewParser().Parse(t.Context(), []byte("<root><child>text</child></root>"))
	require.NoError(t, err)

	nodes, err := xpointer.Evaluate(t.Context(), doc, "xpath1(/root/child)")
	require.NoError(t, err)
	require.Len(t, nodes, 1)
	require.Equal(t, "child", nodes[0].Name())
}

func TestParseFragmentID(t *testing.T) {
	tests := []struct {
		fragment string
		scheme   string
		body     string
	}{
		{"foo", "", "foo"},
		{"xpointer(//p)", "xpointer", "//p"},
		{"xpath1(/root/child)", "xpath1", "/root/child"},
		{"element(/1/2)", "element", "/1/2"},
	}

	for _, test := range tests {
		scheme, body, err := xpointer.ParseFragmentID(test.fragment)
		require.NoError(t, err, "ParseFragmentID(%q)", test.fragment)
		require.Equal(t, test.scheme, scheme, "ParseFragmentID(%q) scheme", test.fragment)
		require.Equal(t, test.body, body, "ParseFragmentID(%q) body", test.fragment)
	}
}

func TestXmlnsScheme(t *testing.T) {
	t.Run("xmlns with xpath1", func(t *testing.T) {
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
		doc, err := helium.NewParser().Parse(t.Context(), []byte(`<?xml version="1.0"?>
<root xmlns:ns="urn:test"><ns:child>hello</ns:child></root>`))
		require.NoError(t, err)

		nodes, err := xpointer.Evaluate(t.Context(), doc, `xmlns(n=urn:test) xpointer(/root/n:child)`)
		require.NoError(t, err)
		require.Len(t, nodes, 1)
		require.Equal(t, "child", nodes[0].(*helium.Element).LocalName())
	})

	t.Run("multiple xmlns bindings", func(t *testing.T) {
		doc, err := helium.NewParser().Parse(t.Context(), []byte(`<?xml version="1.0"?>
<root xmlns:a="urn:a" xmlns:b="urn:b"><a:x/><b:y/></root>`))
		require.NoError(t, err)

		nodes, err := xpointer.Evaluate(t.Context(), doc, `xmlns(p=urn:a) xmlns(q=urn:b) xpointer(/root/q:y)`)
		require.NoError(t, err)
		require.Len(t, nodes, 1)
		require.Equal(t, "y", nodes[0].(*helium.Element).LocalName())
	})

	t.Run("invalid xmlns body", func(t *testing.T) {
		doc, err := helium.NewParser().Parse(t.Context(), []byte(`<root/>`))
		require.NoError(t, err)

		_, err = xpointer.Evaluate(t.Context(), doc, `xmlns(noequalssign) xpointer(/root)`)
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid xmlns()")
	})
}

func TestXPointerEscaping(t *testing.T) {
	// Test XPointer escaping through public APIs with table-driven approach
	tests := []struct {
		name     string
		expr     string
		wantErr  bool
		wantType string // expected node type or "error" for error cases
	}{
		{"simple expression", "element(/1)", false, "element"},
		{"xpath expression", "xpath1(//root)", false, "element"},
		{"fragment id simple", "testid", false, "element"},
		{"xmlns binding", "xmlns(t=test) xpath1(//root)", false, "element"},
		{"cascading parts", "element(nonexist)element(/1)", false, "element"},
		{"invalid expression", "xpointer(:::invalid)", true, "error"},
	}

	doc, err := helium.NewParser().Parse(t.Context(), []byte(`<root xml:id="testid"><child>test</child></root>`))
	require.NoError(t, err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodes, err := xpointer.Evaluate(t.Context(), doc, tt.expr)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			if tt.wantType == "element" {
				require.NotNil(t, nodes)
				require.Greater(t, len(nodes), 0)
			}
		})
	}
}

func TestParseFragmentIDTable(t *testing.T) {
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
		{"element scheme", "element(/1/2)", "element", "/1/2", false},
		{"xmlns scheme", "xmlns(ns=uri)", "xmlns", "ns=uri", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
	// Document with known structure: <book><chapter><image/></chapter></book>
	doc, err := helium.NewParser().Parse(t.Context(), []byte(`<book><chapter><image href="linus.gif"/></chapter></book>`))
	require.NoError(t, err)

	t.Run("first part fails, second succeeds", func(t *testing.T) {
		// element(foo) fails (no ID "foo"), element(/1/1/1) succeeds
		nodes, err := xpointer.Evaluate(t.Context(), doc, "element(foo)element(/1/1/1)")
		require.NoError(t, err)
		require.Len(t, nodes, 1)
		require.Equal(t, "image", nodes[0].(*helium.Element).LocalName())
	})

	t.Run("first part succeeds, second ignored", func(t *testing.T) {
		// element(/1/1/1) succeeds immediately, element(foo) never tried
		nodes, err := xpointer.Evaluate(t.Context(), doc, "element(/1/1/1)element(foo)")
		require.NoError(t, err)
		require.Len(t, nodes, 1)
		require.Equal(t, "image", nodes[0].(*helium.Element).LocalName())
	})

	t.Run("all parts fail", func(t *testing.T) {
		// Both element(foo) and element(bar) fail
		nodes, err := xpointer.Evaluate(t.Context(), doc, "element(foo)element(bar)")
		require.NoError(t, err)
		require.Nil(t, nodes)
	})

	t.Run("xpath1 cascade", func(t *testing.T) {
		// First XPath returns empty, second finds the element
		nodes, err := xpointer.Evaluate(t.Context(), doc, "xpath1(//nonexistent)xpath1(//image)")
		require.NoError(t, err)
		require.Len(t, nodes, 1)
		require.Equal(t, "image", nodes[0].(*helium.Element).LocalName())
	})

	t.Run("syntax error aborts cascade", func(t *testing.T) {
		// xpointer with invalid XPath is a syntax error — cascade aborts
		_, err := xpointer.Evaluate(t.Context(), doc, "xpointer(:::invalid)element(/1/1/1)")
		require.Error(t, err)
	})

	t.Run("unknown scheme continues cascade", func(t *testing.T) {
		// unknown scheme allows fallback to next part
		nodes, err := xpointer.Evaluate(t.Context(), doc, "bogus(data)element(/1/1/1)")
		require.NoError(t, err)
		require.Len(t, nodes, 1)
		require.Equal(t, "image", nodes[0].(*helium.Element).LocalName())
	})
}

func TestBareNameChildSequence(t *testing.T) {
	// Parse with NewParser so xml:id registers in the ID table.
	p := helium.NewParser()
	doc, err := p.Parse(t.Context(), []byte(`<?xml version="1.0"?>
<root xml:id="r"><a><b>found</b></a></root>`))
	require.NoError(t, err)

	t.Run("name/1/1 navigates from ID", func(t *testing.T) {
		// "r/1/1" = look up ID "r", then 1st child (a), then 1st child (b)
		nodes, err := xpointer.Evaluate(t.Context(), doc, "r/1/1")
		require.NoError(t, err)
		require.Len(t, nodes, 1)
		require.Equal(t, "b", nodes[0].(*helium.Element).LocalName())
	})

	t.Run("name/1 navigates one level", func(t *testing.T) {
		nodes, err := xpointer.Evaluate(t.Context(), doc, "r/1")
		require.NoError(t, err)
		require.Len(t, nodes, 1)
		require.Equal(t, "a", nodes[0].(*helium.Element).LocalName())
	})

	t.Run("unknown ID returns nil", func(t *testing.T) {
		nodes, err := xpointer.Evaluate(t.Context(), doc, "nosuchid/1")
		require.NoError(t, err)
		require.Nil(t, nodes)
	})
}

func TestMultiSchemeExpressionsTable(t *testing.T) {
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
