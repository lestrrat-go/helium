package xpointer

import (
	"testing"

	helium "github.com/lestrrat-go/helium"
)

func TestEvaluateXPath1(t *testing.T) {
	doc, err := helium.Parse([]byte("<root><child>text</child></root>"))
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}

	nodes, err := Evaluate(doc, "xpath1(/root/child)")
	if err != nil {
		t.Fatalf("Evaluate(xpath1) failed: %v", err)
	}

	if len(nodes) != 1 {
		t.Errorf("expected 1 node, got %d", len(nodes))
	} else if nodes[0].Name() != "child" {
		t.Errorf("expected node 'child', got %q", nodes[0].Name())
	}
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
		scheme, body, err := ParseFragmentID(test.fragment)
		if err != nil {
			t.Errorf("ParseFragmentID(%q) failed: %v", test.fragment, err)
			continue
		}
		if scheme != test.scheme {
			t.Errorf("ParseFragmentID(%q) scheme = %q, want %q", test.fragment, scheme, test.scheme)
		}
		if body != test.body {
			t.Errorf("ParseFragmentID(%q) body = %q, want %q", test.fragment, body, test.body)
		}
	}
}
