package xpointer

import (
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

func TestEvaluateXPath1(t *testing.T) {
	doc, err := helium.Parse([]byte("<root><child>text</child></root>"))
	require.NoError(t, err)

	nodes, err := Evaluate(doc, "xpath1(/root/child)")
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
		scheme, body, err := ParseFragmentID(test.fragment)
		require.NoError(t, err, "ParseFragmentID(%q)", test.fragment)
		require.Equal(t, test.scheme, scheme, "ParseFragmentID(%q) scheme", test.fragment)
		require.Equal(t, test.body, body, "ParseFragmentID(%q) body", test.fragment)
	}
}

func TestXmlnsScheme(t *testing.T) {
	t.Run("xmlns with xpath1", func(t *testing.T) {
		// Mirrors libxml2 issue289base test
		doc, err := helium.Parse([]byte(`<?xml version="1.0"?>
<rootB xmlns="abc://d/e:f">
</rootB>`))
		require.NoError(t, err)

		nodes, err := Evaluate(doc, `xmlns(b=abc://d/e:f) xpath1(/b:rootB)`)
		require.NoError(t, err)
		require.Len(t, nodes, 1)
		require.Equal(t, "rootB", nodes[0].Name())
	})

	t.Run("xmlns with xpointer", func(t *testing.T) {
		doc, err := helium.Parse([]byte(`<?xml version="1.0"?>
<root xmlns:ns="urn:test"><ns:child>hello</ns:child></root>`))
		require.NoError(t, err)

		nodes, err := Evaluate(doc, `xmlns(n=urn:test) xpointer(/root/n:child)`)
		require.NoError(t, err)
		require.Len(t, nodes, 1)
		require.Equal(t, "child", nodes[0].(*helium.Element).LocalName())
	})

	t.Run("multiple xmlns bindings", func(t *testing.T) {
		doc, err := helium.Parse([]byte(`<?xml version="1.0"?>
<root xmlns:a="urn:a" xmlns:b="urn:b"><a:x/><b:y/></root>`))
		require.NoError(t, err)

		nodes, err := Evaluate(doc, `xmlns(p=urn:a) xmlns(q=urn:b) xpointer(/root/q:y)`)
		require.NoError(t, err)
		require.Len(t, nodes, 1)
		require.Equal(t, "y", nodes[0].(*helium.Element).LocalName())
	})

	t.Run("invalid xmlns body", func(t *testing.T) {
		doc, err := helium.Parse([]byte(`<root/>`))
		require.NoError(t, err)

		_, err = Evaluate(doc, `xmlns(noequalssign) xpointer(/root)`)
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid xmlns()")
	})
}

func TestUnescapeXPointer(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"no escape", "hello", "hello"},
		{"escaped paren close", "a^)b", "a)b"},
		{"escaped paren open", "a^(b", "a(b"},
		{"escaped circumflex", "a^^b", "a^b"},
		{"multiple escapes", "^(^)^^", "()^"},
		{"empty", "", ""},
		{"trailing circumflex", "a^", "a^"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, unescapeXPointer(tt.in))
		})
	}
}

func TestParseSchemeCircumflexEscape(t *testing.T) {
	t.Run("escaped close paren", func(t *testing.T) {
		scheme, body, remaining, err := parseScheme("xpointer(a^)b)")
		require.NoError(t, err)
		require.Equal(t, "xpointer", scheme)
		require.Equal(t, "a)b", body)
		require.Equal(t, "", remaining)
	})

	t.Run("escaped open paren", func(t *testing.T) {
		scheme, body, remaining, err := parseScheme("xpointer(a^(b)")
		require.NoError(t, err)
		require.Equal(t, "xpointer", scheme)
		require.Equal(t, "a(b", body)
		require.Equal(t, "", remaining)
	})

	t.Run("escaped circumflex", func(t *testing.T) {
		scheme, body, remaining, err := parseScheme("xpointer(a^^b)")
		require.NoError(t, err)
		require.Equal(t, "xpointer", scheme)
		require.Equal(t, "a^b", body)
		require.Equal(t, "", remaining)
	})

	t.Run("cascade after escaped body", func(t *testing.T) {
		scheme, body, remaining, err := parseScheme("xpointer(a^)b)element(/1)")
		require.NoError(t, err)
		require.Equal(t, "xpointer", scheme)
		require.Equal(t, "a)b", body)
		require.Equal(t, "element(/1)", remaining)
	})
}

func TestParseParts(t *testing.T) {
	t.Run("single scheme", func(t *testing.T) {
		parts, err := parseParts("xpointer(/root)")
		require.NoError(t, err)
		require.Len(t, parts, 1)
		require.Equal(t, "xpointer", parts[0].scheme)
		require.Equal(t, "/root", parts[0].body)
	})

	t.Run("xmlns plus xpath1", func(t *testing.T) {
		parts, err := parseParts("xmlns(b=urn:test) xpath1(/b:root)")
		require.NoError(t, err)
		require.Len(t, parts, 2)
		require.Equal(t, "xmlns", parts[0].scheme)
		require.Equal(t, "b=urn:test", parts[0].body)
		require.Equal(t, "xpath1", parts[1].scheme)
		require.Equal(t, "/b:root", parts[1].body)
	})

	t.Run("bare name", func(t *testing.T) {
		parts, err := parseParts("myid")
		require.NoError(t, err)
		require.Len(t, parts, 1)
		require.Equal(t, "", parts[0].scheme)
		require.Equal(t, "myid", parts[0].body)
	})
}
