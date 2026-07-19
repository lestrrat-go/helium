package html_test

import (
	"testing"

	"github.com/lestrrat-go/helium/html"
	"github.com/stretchr/testify/require"
)

// A colon-bearing attribute name is rejected by Element.SetAttribute. Under
// Strict(true) that rejection must surface as a fatal parse error naming the
// offending attribute, rather than being silently dropped.
func TestAttrColon_StrictSurfacesSetAttributeError(t *testing.T) {
	const input = `<p x:y="1" a="2">text</p>`

	_, err := html.NewParser().Strict(true).Parse(t.Context(), []byte(input))
	require.Error(t, err, "Strict(true) must surface the colon-attribute rejection")
	require.ErrorContains(t, err, "x:y",
		"fatal error must name the offending attribute")
}

// A colon-bearing boolean attribute name is rejected by
// Element.SetBooleanAttribute. Under Strict(true) that rejection must surface
// identically to the valued case.
func TestAttrColon_StrictSurfacesSetBooleanAttributeError(t *testing.T) {
	const input = `<p x:y a="2">text</p>`

	_, err := html.NewParser().Strict(true).Parse(t.Context(), []byte(input))
	require.Error(t, err, "Strict(true) must surface the colon boolean-attribute rejection")
	require.ErrorContains(t, err, "x:y",
		"fatal error must name the offending attribute")
}

// In the default (non-strict) path the colon-attribute rejection is downgraded
// to a warning: the element and every other attribute survive, only the
// colon-named attribute is dropped. This pins the tolerant, libxml2-compatible
// behavior.
func TestAttrColon_TolerantDropsAttributeKeepsRest(t *testing.T) {
	const input = `<p x:y="1" a="2">text</p>`

	doc, err := html.NewParser().Parse(t.Context(), []byte(input))
	require.NoError(t, err, "non-strict parse must keep going past the rejection")

	p := findElement(doc, "p")
	require.NotNil(t, p, "the element survives the dropped attribute")

	require.False(t, p.HasAttribute("x:y"), "the colon-named attribute is dropped")

	v, ok := p.GetAttribute("a")
	require.True(t, ok, "a sibling attribute after the rejected one survives")
	require.Equal(t, "2", v)

	require.Equal(t, "text", string(p.Content()), "element content is intact")
}
