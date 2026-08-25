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

// TestUnmarshalClaimOrderCharacterization pins which child each field claims
// and in what order, across decodeElementInto's four child-scan paths
// (single-segment element bindings, the any-element pass, multi-segment path
// bindings, and their interaction through the shared consumed set). A
// resuming cursor must claim exactly the same children in exactly the same
// order as a scan that restarts from the beginning every time, so this is the
// baseline that a cursor-based rewrite must reproduce byte for byte.
func TestUnmarshalClaimOrderCharacterization(t *testing.T) {
	// Two slice fields draining interleaved <a>/<b> children: each field
	// must claim only its own tag, in document order, regardless of the
	// other tag's children sitting between its matches.
	t.Run("interleaved slices", func(t *testing.T) {
		type Doc struct {
			As []string `xml:"a"`
			Bs []string `xml:"b"`
		}
		var d Doc
		in := []byte(`<Doc><a>a1</a><b>b1</b><a>a2</a><b>b2</b><a>a3</a></Doc>`)
		require.NoError(t, shim.Unmarshal(in, &d))
		require.Equal(t, []string{"a1", "a2", "a3"}, d.As)
		require.Equal(t, []string{"b1", "b2"}, d.Bs)
	})

	// A slice field followed by a `,any` slice field: the any field must
	// receive exactly the children the first field left behind, in document
	// order.
	t.Run("slice then any leftovers", func(t *testing.T) {
		type Doc struct {
			As   []string `xml:"a"`
			Rest []string `xml:",any"`
		}
		var d Doc
		in := []byte(`<Doc><a>a1</a><b>b1</b><a>a2</a><c>c1</c></Doc>`)
		require.NoError(t, shim.Unmarshal(in, &d))
		require.Equal(t, []string{"a1", "a2"}, d.As)
		require.Equal(t, []string{"b1", "c1"}, d.Rest)
	})

	// A scalar field with several matches: last one wins, matching stdlib.
	t.Run("scalar last wins", func(t *testing.T) {
		type Doc struct {
			Value string `xml:"v"`
		}
		var d Doc
		in := []byte(`<Doc><v>v1</v><v>v2</v><v>v3</v></Doc>`)
		require.NoError(t, shim.Unmarshal(in, &d))
		require.Equal(t, "v3", d.Value)
	})

	// A `w>leaf` path slice where one wrapper holds several leaves and
	// several wrappers hold one: the claim order must follow the flattened
	// depth-first document order, not group by wrapper.
	t.Run("nested path DFS order", func(t *testing.T) {
		type Doc struct {
			Leaves []string `xml:"w>leaf"`
		}
		var d Doc
		in := []byte(`<Doc>` +
			`<w><leaf>l1</leaf><leaf>l2</leaf></w>` +
			`<w><leaf>l3</leaf></w>` +
			`<w><leaf>l4</leaf></w>` +
			`</Doc>`)
		require.NoError(t, shim.Unmarshal(in, &d))
		require.Equal(t, []string{"l1", "l2", "l3", "l4"}, d.Leaves)
	})

	// Two path bindings sharing one wrapper, each namespace-qualified to a
	// different namespace (an unqualified path sharing the exact same local
	// path as a qualified one is rejected by validateTagPathConflicts, as it
	// is by encoding/xml, so this is the sharpest legal variant): the second
	// binding must still see the leaves the first skipped over for failing
	// its own namespace check.
	t.Run("shared wrapper namespace skip", func(t *testing.T) {
		type Doc struct {
			NSFirst  []string `xml:"http://example.com/ns1 w>leaf"`
			NSSecond []string `xml:"http://example.com/ns2 w>leaf"`
		}
		var d Doc
		in := []byte(`<Doc xmlns:n1="http://example.com/ns1" xmlns:n2="http://example.com/ns2">` +
			`<w><n1:leaf>a1</n1:leaf><n2:leaf>b1</n2:leaf><n1:leaf>a2</n1:leaf></w>` +
			`</Doc>`)
		require.NoError(t, shim.Unmarshal(in, &d))
		require.Equal(t, []string{"a1", "a2"}, d.NSFirst)
		require.Equal(t, []string{"b1"}, d.NSSecond)
	})

	// A path binding followed by an any-element pass: once the path binding
	// claims a leaf under its wrapper, the wrapper must stay marked consumed
	// so the any pass does not also claim the wrapper element itself.
	t.Run("path consumes wrapper for any pass", func(t *testing.T) {
		type Doc struct {
			Path []string `xml:"w>leaf"`
			Rest []string `xml:",any"`
		}
		var d Doc
		in := []byte(`<Doc><w><leaf>l1</leaf></w><other>o1</other></Doc>`)
		require.NoError(t, shim.Unmarshal(in, &d))
		require.Equal(t, []string{"l1"}, d.Path)
		require.Equal(t, []string{"o1"}, d.Rest)
	})
}

// TestUnmarshalUnsupportedVersionNeedsAReadVersion pins that the shim's
// unsupported-version verdict, which comes from helium's parse, names a version
// only when helium actually read and rejected one. A malformed declaration
// helium fails on before judging the version names no version, so the verdict
// can never quote a version nobody declared. The old "only version 1.0 is
// supported" wording is gone entirely. Every case here still REJECTS.
func TestUnmarshalUnsupportedVersionNeedsAReadVersion(t *testing.T) {
	type item struct {
		Value string `xml:"value"`
	}

	for _, tc := range []struct {
		name string
		xml  string
		// namesRejectedVersion is true when the verdict is about the version
		// itself, so the message must quote the one actually rejected.
		namesRejectedVersion bool
	}{
		// helium reads and rejects the version before reaching the missing "?>",
		// so its error names it.
		{"unterminated declaration", `<?xml version="2.0"`, true},
		// Repeating a pseudo-attribute does not conform to the XMLDecl grammar
		// (XML 1.0 §2.8), so helium rejects it as a malformed declaration before
		// any version verdict is reached. The verdict is not about a version, so
		// it names none.
		{"repeated version pseudo-attribute", `<?xml version="1.0" version="2.0"?><item/>`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var out item
			err := shim.Unmarshal([]byte(tc.xml), &out)
			require.Error(t, err, "a malformed declaration must never be accepted")
			require.NotContains(t, err.Error(), "only version 1.0 is supported",
				"the old 1.0-only wording is gone")

			// A malformed declaration is a syntax error, the same category
			// encoding/xml reports it under. The wording is the shim's own.
			var syntaxErr *stdxml.SyntaxError
			require.ErrorAs(t, err, &syntaxErr)
			require.NotContains(t, syntaxErr.Msg, `""`,
				"an empty string is not a version anyone declared")
			if !tc.namesRejectedVersion {
				return
			}
			require.Contains(t, syntaxErr.Msg, "2.0",
				"the error names the version actually rejected")
		})
	}
}

// TestUnmarshalMalformedXMLDeclDivergesFromStdlib pins shim's OWN behavior for
// declarations that do not conform to the XMLDecl grammar (XML 1.0 §2.8):
//
//	XMLDecl      ::= '<?xml' VersionInfo EncodingDecl? SDDecl? S? '?>'
//	VersionInfo  ::= S 'version' Eq ("'" VersionNum "'" | '"' VersionNum '"')
//	VersionNum   ::= '1.' [0-9]+
//	EncodingDecl ::= S 'encoding' Eq ('"' EncName '"' | "'" EncName "'")
//	EncName      ::= [A-Za-z] ([A-Za-z0-9._] | '-')*
//
// The version is mandatory, the three pseudo-attributes are the only ones the
// grammar admits, and it fixes their order. shim rejects every form below;
// encoding/xml accepts them all. These cases cannot live in
// TestUnmarshalXMLDeclValidationMatchStdlib: that table asserts agreement with
// stdlib, and this is a deliberate divergence — shim is backed by a
// spec-conforming parser and does not accept XML the spec does not permit.
func TestUnmarshalMalformedXMLDeclDivergesFromStdlib(t *testing.T) {
	type item struct {
		Value string `xml:"value"`
	}

	for _, tc := range []struct {
		name string
		// why states what disqualifies the declaration under the grammar above.
		why string
		xml string
	}{
		{
			name: "charset pseudo-attribute",
			why:  "charset is not one of the three admitted pseudo-attributes",
			xml:  `<?xml version="1.0" charset="UTF-8"?><item><value>hello</value></item>`,
		},
		{
			name: "empty decl",
			why:  "VersionInfo is mandatory, and this declares nothing at all",
			xml:  `<?xml?><item><value>hello</value></item>`,
		},
		{
			name: "no version",
			why:  "VersionInfo is mandatory; an encoding alone does not satisfy it",
			xml:  `<?xml encoding="UTF-8"?><item><value>hello</value></item>`,
		},
		{
			name: "version empty string",
			why:  `VersionNum ::= '1.' [0-9]+ — the empty string is not a VersionNum`,
			xml:  `<?xml version=""?><item><value>hello</value></item>`,
		},
		{
			name: "encoding empty string",
			why:  "EncName must begin with a letter, so it is never empty",
			xml:  `<?xml version="1.0" encoding=""?><item><value>hello</value></item>`,
		},
		{
			name: "pseudo-attributes out of order",
			why:  "XMLDecl fixes the order: version, then encoding, then standalone",
			xml:  `<?xml encoding="UTF-8" version="1.0"?><item><value>hello</value></item>`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// stdlib accepts each of these; the divergence is the point of the test.
			var stdOut item
			require.NoError(t, stdxml.Unmarshal([]byte(tc.xml), &stdOut),
				"the encoding/xml side of the divergence")

			var out item
			err := shim.Unmarshal([]byte(tc.xml), &out)
			require.Error(t, err, tc.why)

			// A malformed declaration is a syntax error, the same category
			// encoding/xml reports its own parse failures under.
			var syntaxErr *stdxml.SyntaxError
			require.ErrorAs(t, err, &syntaxErr)
		})
	}
}

// TestUnmarshalXMLDeclVersionDivergesFromStdlib pins shim's OWN behavior for an
// unsupported version, where shim and encoding/xml disagree, so a later change
// cannot silently alter it. These cases cannot live in
// TestUnmarshalXMLDeclValidationMatchStdlib: that table asserts agreement with
// stdlib. stdlib accepts the spaced-Eq form shim rejects, and for the unspaced
// form both reject but shim reports helium's wording, not stdlib's.
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
			// The verdict is helium's: it names the version outside the 1.x family
			// it rejected, reported as a syntax error.
			require.ErrorContains(t, err, `unsupported XML version "2.0"`)
			var syntaxErr *stdxml.SyntaxError
			require.ErrorAs(t, err, &syntaxErr)
		})
	}
}
