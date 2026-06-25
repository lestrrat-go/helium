package shim_test

import (
	stdxml "encoding/xml"
	"testing"

	"github.com/lestrrat-go/helium/shim"
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
