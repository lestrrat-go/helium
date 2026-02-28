package html

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOptionsNoImplied(t *testing.T) {
	doc, err := Parse([]byte(`<p>hello</p>`), WithNoImplied())
	require.NoError(t, err)

	// Without NoImplied, the parser would insert <html><body> around <p>.
	// With NoImplied, the first child should be <p> directly.
	first := doc.FirstChild()
	require.NotNil(t, first, "document should have a child")
	require.Equal(t, "p", first.Name())
}

func TestOptionsNoDefaultDTD(t *testing.T) {
	// Parse a document without any DOCTYPE
	doc, err := Parse([]byte(`<html><body><p>hi</p></body></html>`))
	require.NoError(t, err)

	// Without NoDefaultDTD, serialization adds a default DOCTYPE
	var withDTD bytes.Buffer
	require.NoError(t, DumpDoc(&withDTD, doc))
	require.True(t, strings.Contains(withDTD.String(), "<!DOCTYPE"), "default should include DOCTYPE")

	// With NoDefaultDTD, no DOCTYPE in output
	var noDTD bytes.Buffer
	require.NoError(t, DumpDoc(&noDTD, doc, WithNoDefaultDTD()))
	require.False(t, strings.Contains(noDTD.String(), "<!DOCTYPE"), "WithNoDefaultDTD should suppress DOCTYPE")
}

func TestOptionsNoBlanks(t *testing.T) {
	input := `<html> <body> <p>text</p> </body> </html>`
	doc, err := Parse([]byte(input), WithNoBlanks())
	require.NoError(t, err)

	var buf bytes.Buffer
	require.NoError(t, DumpDoc(&buf, doc, WithNoDefaultDTD()))
	output := buf.String()

	// The output should not contain whitespace between tags
	// (the original spaces " " between <html> and <body>, etc. should be stripped)
	require.False(t, strings.Contains(output, "<html> <body>"),
		"whitespace-only text nodes should be stripped")
	require.True(t, strings.Contains(output, "text"),
		"non-whitespace text should be preserved")
}

func TestOptionsNoError(t *testing.T) {
	var errorCalled bool
	sax := &SAXCallbacks{
		ErrorHandler: ErrorFunc(func(msg string, args ...interface{}) error {
			errorCalled = true
			return nil
		}),
	}

	// Parse malformed HTML that would normally trigger errors
	// (e.g., unexpected end tag)
	input := `<html><body></nonexistent></body></html>`
	err := ParseWithSAX([]byte(input), sax, WithNoError())
	require.NoError(t, err)
	require.False(t, errorCalled, "error handler should not be called with WithNoError")
}

func TestOptionsNoErrorDefault(t *testing.T) {
	var errorCalled bool
	sax := &SAXCallbacks{
		ErrorHandler: ErrorFunc(func(msg string, args ...interface{}) error {
			errorCalled = true
			return nil
		}),
	}

	// Without NoError, the error handler should be called
	input := `<html><body></nonexistent></body></html>`
	err := ParseWithSAX([]byte(input), sax)
	require.NoError(t, err)
	require.True(t, errorCalled, "error handler should be called without WithNoError")
}

func TestOptionsNoWarning(t *testing.T) {
	var warningCalled bool
	sax := &SAXCallbacks{
		WarningHandler: WarningFunc(func(msg string, args ...interface{}) error {
			warningCalled = true
			return nil
		}),
	}

	// Parse valid HTML with WithNoWarning
	input := `<html><body><p>hello</p></body></html>`
	err := ParseWithSAX([]byte(input), sax, WithNoWarning())
	require.NoError(t, err)
	require.False(t, warningCalled, "warning handler should not be called with WithNoWarning")
}

func TestOptionsPushParserCarriesOptions(t *testing.T) {
	pp := NewPushParser(WithNoImplied())
	require.NoError(t, pp.Push([]byte(`<p>hello</p>`)))
	doc, err := pp.Close()
	require.NoError(t, err)

	first := doc.FirstChild()
	require.NotNil(t, first)
	require.Equal(t, "p", first.Name())
}
