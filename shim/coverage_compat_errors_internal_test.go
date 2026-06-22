package shim

import (
	stdxml "encoding/xml"
	"errors"
	"testing"

	helium "github.com/lestrrat-go/helium"
)

const wantExpectedElementName = "expected element name after <"

func TestConvertParseErrorNil(t *testing.T) {
	if got := convertParseError(nil); got != nil {
		t.Fatalf("expected nil, got %v", got)
	}
}

func TestConvertParseErrorPassthrough(t *testing.T) {
	in := errors.New("some other error")
	if got := convertParseError(in); got != in {
		t.Fatalf("expected passthrough error, got %v", got)
	}
}

func TestConvertParseErrorMapsToSyntaxError(t *testing.T) {
	pe := helium.ErrParseError{
		LineNumber: 7,
		Line:       "</foo>",
		Err:        errors.New("invalid name start char"),
	}
	got := convertParseError(pe)
	se, ok := got.(*stdxml.SyntaxError)
	if !ok {
		t.Fatalf("expected *xml.SyntaxError, got %T", got)
	}
	if se.Line != 7 {
		t.Fatalf("expected line 7, got %d", se.Line)
	}
	if se.Msg != "unexpected end element </foo>" {
		t.Fatalf("unexpected msg: %q", se.Msg)
	}
}

func TestMapErrorMessage(t *testing.T) {
	tests := []struct {
		name string
		line string
		raw  string
		want string
	}{
		{"name-start-no-end", "<1bad>", "invalid name start char", wantExpectedElementName},
		{"name-start-prefix", "<1bad>", "invalid name start char: x", wantExpectedElementName},
		{"qname", "", "failed to parse QName: bad", "expected attribute name in element"},
		{"start-tag", "", "start tag expected, '<' not found", "start tag expected, '<' not found"},
		{"char-data", "", "invalid char data", "invalid char data"},
		{"semicolon-with-entity", "text &amp", "';' is required", "invalid character entity &amp (no semicolon)"},
		{"semicolon-no-entity", "no amp here", "';' is required", "';' is required"},
		{"name-required-bad-entity", "x &￾; y", "name is required", "invalid character entity &￾;"},
		{"name-required-default", "no amp", "name is required", "invalid character entity & (no semicolon)"},
		{"local-empty-qname", "", "local name empty! failed to parse QName", wantExpectedElementName},
		{"local-empty-plain", "", "local name empty! something", "local name empty! something"},
		{"unknown-passthrough", "", "totally unknown error", "totally unknown error"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pe := helium.ErrParseError{Line: tt.line, Err: errors.New(tt.raw)}
			if got := mapErrorMessage(pe); got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestMapErrorMessageEndTagMismatch(t *testing.T) {
	// "expected end tag 'prefix:local'" with a differing close tag in context.
	pe := helium.ErrParseError{
		Line: "<a:foo></b:foo>",
		Err:  errors.New("expected end tag 'a:foo'"),
	}
	got := mapErrorMessage(pe)
	want := `element <foo> in space a closed by </foo> in space b`
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestMapErrorMessageNamespaceNotFound(t *testing.T) {
	pe := helium.ErrParseError{
		Line: "<x:foo></y:foo>",
		Err:  errors.New("namespace 'x' not found"),
	}
	got := mapErrorMessage(pe)
	want := `element <foo> in space x closed by </foo> in space y`
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestConvertEndTagMismatch(t *testing.T) {
	tests := []struct {
		name        string
		expectedTag string
		contextLine string
		want        string
	}{
		{"differing-local", "a:foo", "<a:foo></a:bar>", "element <a:foo> closed by </a:bar>"},
		{"prefix-vs-noprefix", "a:foo", "<a:foo></foo>", `element <foo> in space a closed by </foo> in space ""`},
		{"differing-prefix", "a:foo", "<a:foo></b:foo>", "element <foo> in space a closed by </foo> in space b"},
		{"no-close-with-prefix", "a:foo", "no close tag here", `element <foo> in space a closed by </foo> in space ""`},
		{"no-close-no-prefix", "foo", "no close tag here", "expected end tag 'foo'"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := convertEndTagMismatch(tt.expectedTag, tt.contextLine); got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestConvertNamespaceError(t *testing.T) {
	tests := []struct {
		name        string
		ns          string
		contextLine string
		want        string
	}{
		{"matched", "x", "<x:foo></y:foo>", "element <foo> in space x closed by </foo> in space y"},
		{"no-open-tag", "x", "nothing here", "namespace 'x' not found"},
		{"open-no-close", "x", "<x:foo bar", "namespace 'x' not found"},
		{"local-mismatch", "x", "<x:foo></y:bar>", "namespace 'x' not found"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := convertNamespaceError(tt.ns, tt.contextLine); got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExtractEndElement(t *testing.T) {
	tests := []struct {
		line string
		want string
	}{
		{"<a></foo>", "foo"},
		{"no end here", ""},
		{"</foo bar", "foo"},
		{"</foo", "foo"},
		{"</", ""},
		{"</>", ""},
	}
	for _, tt := range tests {
		if got := extractEndElement(tt.line); got != tt.want {
			t.Fatalf("extractEndElement(%q)=%q, want %q", tt.line, got, tt.want)
		}
	}
}

func TestExtractEntityName(t *testing.T) {
	tests := []struct {
		line string
		want string
	}{
		{"text &amp more", "amp"},
		{"no ampersand", ""},
		{"trailing &", ""},
		{"&abc", "abc"},
	}
	for _, tt := range tests {
		if got := extractEntityName(tt.line); got != tt.want {
			t.Fatalf("extractEntityName(%q)=%q, want %q", tt.line, got, tt.want)
		}
	}
}

func TestExtractBadEntity(t *testing.T) {
	tests := []struct {
		line string
		want string
	}{
		{"x &￾; y", "￾"},
		{"no ampersand", ""},
		{"trailing &", ""},
		{"&abc;", ""}, // first char is a name char
		{"&￾", ""},    // no following semicolon
	}
	for _, tt := range tests {
		if got := extractBadEntity(tt.line); got != tt.want {
			t.Fatalf("extractBadEntity(%q)=%q, want %q", tt.line, got, tt.want)
		}
	}
}

func TestIsXMLNameChar(t *testing.T) {
	valid := []rune{'a', 'Z', '5', '_', '-', '.', ':', 0xC0, 0xD8, 0xF8, 0x300, 0xB7, 0x370, 0x37F}
	for _, r := range valid {
		if !isXMLNameChar(r) {
			t.Fatalf("expected %#U to be a name char", r)
		}
	}
	invalid := []rune{' ', '\t', '<', '>', '&', 0xD7, 0xF7, 0x2000, 0x37E}
	for _, r := range invalid {
		if isXMLNameChar(r) {
			t.Fatalf("expected %#U to not be a name char", r)
		}
	}
}
