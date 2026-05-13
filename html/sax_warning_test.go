package html_test

import (
	"errors"
	"testing"

	"github.com/lestrrat-go/helium/html"
	"github.com/stretchr/testify/require"
)

// In the default (non-strict) Parse path, a tree-builder objection to a
// duplicate DOCTYPE must NOT abort parsing — HTML's libxml2-compatible
// tolerance is preserved. Regression for the doc3.htm-style fixture.
func TestSAXErrors_DuplicateDoctype_DefaultParseStillSucceeds(t *testing.T) {
	input := `<!DOCTYPE a><!DOCTYPE b><html><body>hi</body></html>`
	doc, err := html.NewParser().Parse(t.Context(), []byte(input))
	require.NoError(t, err)
	require.NotNil(t, doc)
}

// A caller-supplied handler whose callback returns a non-ErrHandlerUnspecified
// error must have that error routed through OnWarning, not silently dropped.
func TestSAXErrors_NonUnspecifiedRoutedToOnWarning(t *testing.T) {
	customErr := errors.New("caller-supplied: cannot accept this event")

	var warnings []string
	sax := &html.SAXCallbacks{}
	sax.SetOnWarning(html.WarningFunc(func(err error) error {
		warnings = append(warnings, err.Error())
		return nil
	}))
	sax.SetOnInternalSubset(html.InternalSubsetFunc(func(_, _, _ string) error {
		return customErr
	}))

	input := `<!DOCTYPE html><html><body/></html>`
	err := html.NewParser().ParseWithSAX(t.Context(), []byte(input), sax)
	require.NoError(t, err, "non-strict parse must keep going past handler errors")
	require.NotEmpty(t, warnings, "OnWarning must observe the handler's error")
	require.Contains(t, warnings[0], "cannot accept this event")
}

// With Strict(true), a tree-builder objection from the default Parse path
// MUST surface as a fatal parse error. Mirror-image of the regression test
// above: the same input that succeeds non-strict must fail strict.
func TestSAXErrors_StrictPromotesTreeBuilderObjectionToFatal(t *testing.T) {
	input := `<!DOCTYPE a><!DOCTYPE b><html><body>hi</body></html>`
	_, err := html.NewParser().Strict(true).Parse(t.Context(), []byte(input))
	require.Error(t, err, "Strict(true) must surface tree-builder objections")
	require.Contains(t, err.Error(), "internal subset",
		"fatal error message should reference the offending invariant")
}

// Strict(true) must NOT trip on ErrHandlerUnspecified — that sentinel is
// always the "no handler is registered for this event" plumbing signal,
// not a real error. Well-formed HTML parsed in strict mode succeeds.
func TestSAXErrors_StrictIgnoresErrHandlerUnspecified(t *testing.T) {
	input := `<html><body><p>well-formed</p></body></html>`
	doc, err := html.NewParser().Strict(true).Parse(t.Context(), []byte(input))
	require.NoError(t, err)
	require.NotNil(t, doc)
}
