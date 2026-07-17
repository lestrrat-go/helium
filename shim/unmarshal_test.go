package shim_test

import (
	stdxml "encoding/xml"
	"testing"

	"github.com/lestrrat-go/helium/shim"
	"github.com/stretchr/testify/require"
)

// TestUnmarshalNestedPath exercises findPathLeaf / findPathLeafInner for
// multi-segment struct tag paths (e.g. "a>b>c").
func TestUnmarshalNestedPath(t *testing.T) {
	t.Run("leaf", func(t *testing.T) {
		type Doc struct {
			Value string `xml:"a>b>c"`
		}
		var d Doc
		in := []byte(`<Doc><a><b><c>hello</c></b></a></Doc>`)
		if err := shim.Unmarshal(in, &d); err != nil {
			t.Fatalf("Unmarshal error: %v", err)
		}
		if d.Value != "hello" {
			t.Fatalf("expected 'hello', got %q", d.Value)
		}
	})

	// slice exercises the slice branch over findPathLeaf.
	t.Run("slice", func(t *testing.T) {
		type Doc struct {
			Values []string `xml:"a>b"`
		}
		var d Doc
		in := []byte(`<Doc><a><b>one</b><b>two</b></a></Doc>`)
		if err := shim.Unmarshal(in, &d); err != nil {
			t.Fatalf("Unmarshal error: %v", err)
		}
		if len(d.Values) != 2 || d.Values[0] != "one" || d.Values[1] != "two" {
			t.Fatalf("unexpected values: %#v", d.Values)
		}
	})

	// xml-name exercises setXMLName at the leaf of a path.
	t.Run("xml-name", func(t *testing.T) {
		type Doc struct {
			Leaf stdxml.Name `xml:"a>b"`
		}
		var d Doc
		in := []byte(`<Doc><a><b/></a></Doc>`)
		if err := shim.Unmarshal(in, &d); err != nil {
			t.Fatalf("Unmarshal error: %v", err)
		}
		if d.Leaf.Local != "b" {
			t.Fatalf("expected leaf name 'b', got %q", d.Leaf.Local)
		}
	})

	// missing exercises the leaf-not-found branch.
	t.Run("missing", func(t *testing.T) {
		type Doc struct {
			Value string `xml:"a>b>c"`
		}
		var d Doc
		in := []byte(`<Doc><a><b><other>x</other></b></a></Doc>`)
		if err := shim.Unmarshal(in, &d); err != nil {
			t.Fatalf("Unmarshal error: %v", err)
		}
		if d.Value != "" {
			t.Fatalf("expected empty value, got %q", d.Value)
		}
	})
}

func TestUnmarshalDirectChild(t *testing.T) {
	// xml-name exercises the single-segment findPath + setXMLName branch.
	t.Run("xml-name", func(t *testing.T) {
		type Doc struct {
			Child stdxml.Name `xml:"child"`
		}
		var d Doc
		in := []byte(`<Doc><child/></Doc>`)
		if err := shim.Unmarshal(in, &d); err != nil {
			t.Fatalf("Unmarshal error: %v", err)
		}
		if d.Child.Local != "child" {
			t.Fatalf("expected child name 'child', got %q", d.Child.Local)
		}
	})

	// slice exercises the single-segment slice path with repeated matches.
	t.Run("slice", func(t *testing.T) {
		type Doc struct {
			Items []string `xml:"item"`
		}
		var d Doc
		in := []byte(`<Doc><item>a</item><item>b</item><item>c</item></Doc>`)
		if err := shim.Unmarshal(in, &d); err != nil {
			t.Fatalf("Unmarshal error: %v", err)
		}
		if len(d.Items) != 3 {
			t.Fatalf("expected 3 items, got %#v", d.Items)
		}
	})
}

// TestUnmarshalUnsupportedVersionNeedsAReadVersion pins that the shim's
// unsupported-version verdict names only a version the shim itself read out of
// the declaration and rejects. When the raw scan and the parser disagree about
// which version a malformed declaration carries, the parser's error is reported
// instead — it quotes the version actually rejected — so the verdict can never
// name a version nobody declared, nor contradict itself by calling 1.0
// unsupported. Every case here still REJECTS; only the wording is at stake.
func TestUnmarshalUnsupportedVersionNeedsAReadVersion(t *testing.T) {
	type item struct {
		Value string `xml:"value"`
	}

	for _, tc := range []struct {
		name string
		xml  string
	}{
		// Never closed by "?>", so the raw scan reads no version at all and
		// there is none to call unsupported.
		{"unterminated declaration", `<?xml version="2.0"`},
		// Repeats the pseudo-attribute: the raw scan reads the first (1.0)
		// while the parser rejects the second (2.0), so naming the scanned one
		// would report "1.0" as unsupported while claiming 1.0 is supported.
		{"repeated version pseudo-attribute", `<?xml version="1.0" version="2.0"?><item/>`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var out item
			err := shim.Unmarshal([]byte(tc.xml), &out)
			require.Error(t, err, "a malformed declaration must never be accepted")
			require.NotContains(t, err.Error(), "only version 1.0 is supported",
				"no read version supports this verdict")

			// A malformed declaration is a syntax error, the same category
			// encoding/xml reports it under. The wording is the shim's own.
			var syntaxErr *stdxml.SyntaxError
			require.ErrorAs(t, err, &syntaxErr)
			require.NotContains(t, syntaxErr.Msg, `""`,
				"an empty string is not a version anyone declared")
			require.Contains(t, syntaxErr.Msg, "2.0",
				"the error names the version actually rejected")
		})
	}
}

// TestUnmarshalXMLDeclVersionDivergesFromStdlib pins shim's OWN behavior for
// declarations where shim and encoding/xml deliberately disagree, so a later
// change cannot silently alter it. These cases cannot live in
// TestUnmarshalXMLDeclValidationMatchStdlib: that table asserts agreement with
// stdlib, and stdlib accepts the spaced-Eq form shim rejects.
func TestUnmarshalXMLDeclVersionDivergesFromStdlib(t *testing.T) {
	type item struct {
		Value string `xml:"value"`
	}

	cases := []struct {
		name string
		xml  string
	}{
		// XML 1.0 Eq ::= S? '=' S?, so version = "2.0" declares version 2.0 and
		// is rejected. stdlib's version scan does not admit the spaces and reads
		// no version at all, so it accepts. shim follows the spec here.
		{"spaced eq", `<?xml version = "2.0"?><item><value>hello</value></item>`},
		{"unspaced eq", `<?xml version="2.0"?><item><value>hello</value></item>`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var out item
			err := shim.Unmarshal([]byte(tc.xml), &out)
			require.Error(t, err)
			require.Equal(t, `xml: unsupported version "2.0"; only version 1.0 is supported`, err.Error())
		})
	}
}
