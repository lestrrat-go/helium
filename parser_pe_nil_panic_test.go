package helium

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestUndefinedParameterEntityNoPanic guards against a nil-entity dereference in
// decodeEntitiesInternal's parameter-entity ('%') branch.
//
// For an undeclared parameter entity in a context with an external subset (or
// after a prior PE reference), parseStringPEReference deliberately returns a nil
// entity with NO error after clearing pctx.valid — the libxml2-faithful "PE not
// declared, validity error, keep going" convention that the nilPEHandler models.
// The '&' (general-entity) branch and expandEntityValueForRefCheck both guard
// that nil; the '%' branch did not and dereferenced ent.Content(), panicking the
// whole parse. This test drives the full Parse path with a SAX handler that uses
// the (nil, nil) convention and asserts no panic.
func TestUndefinedParameterEntityNoPanic(t *testing.T) {
	cases := []string{
		`<!DOCTYPE r SYSTEM "x" [<!ENTITY e "%missing;">]><r/>`,
		`<!DOCTYPE r SYSTEM "x" [<!ENTITY % p "%missing;">]><r/>`,
		`<!DOCTYPE r SYSTEM "x" [<!ENTITY e "a%missing;b">]><r/>`,
	}

	for _, input := range cases {
		t.Run(input, func(t *testing.T) {
			handler := nilPEHandler{TreeBuilder: NewTreeBuilder()}
			require.NotPanics(t, func() {
				_, _ = NewParser().SAXHandler(handler).Parse(t.Context(), []byte(input))
			})
		})
	}
}

// TestDecodeEntitiesUndeclaredPEReturnsEmpty exercises the decode branch
// directly: an undeclared parameter entity resolved through the (nil, nil)
// convention must expand to nothing (consistent with expandEntityValueForRefCheck)
// rather than panic, and must not surface an error.
func TestDecodeEntitiesUndeclaredPEReturnsEmpty(t *testing.T) {
	pctx := &parserCtx{}
	require.NoError(t, pctx.init(nil, bytes.NewReader(nil)))
	doc := NewDocument("1.0", "", StandaloneImplicitNo)
	tb := NewTreeBuilder()
	pctx.doc = doc
	pctx.sax = nilPEHandler{TreeBuilder: tb}
	pctx.treeBuilder = tb
	pctx.hasExternalSubset = true

	ctx := withParserCtx(t.Context(), pctx)

	var out string
	var err error
	require.NotPanics(t, func() {
		out, err = pctx.decodeEntities(ctx, []byte("a%missing;b"), SubstitutePERef)
	})
	require.NoError(t, err, "an undeclared PE in an external-subset context is a validity error, not fatal")
	require.Equal(t, "ab", out, "the undeclared PE must expand to nothing")
}
