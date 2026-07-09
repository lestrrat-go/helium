package helium_test

import (
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

// An internal general entity whose replacement text straddles element
// boundaries — here it closes an element opened outside the entity and opens a
// fresh one closed outside — is not well balanced. Referencing it in element
// content is a fatal well-formedness error (W3C xmlconf not-wf-sa-074; WFC:
// parsed entities must be well-formed, XML §4.3.2).
func TestInternalEntityUnbalancedNestingRejected(t *testing.T) {
	t.Parallel()

	const src = `<!DOCTYPE doc [
<!ENTITY e "</foo><foo>">
]>
<doc>
<foo>&e;</foo>
</doc>`
	_, err := helium.NewParser().SubstituteEntities(true).Parse(t.Context(), []byte(src))
	require.Error(t, err, "an entity that breaks element nesting must be rejected")
	require.Contains(t, err.Error(), "not well balanced")
}

// An internal entity that closes, mid-content, an element opened OUTSIDE it is
// equally unbalanced (a trailing stray end-tag, not a leading one).
func TestInternalEntityTrailingEndTagRejected(t *testing.T) {
	t.Parallel()

	const src = `<!DOCTYPE doc [
<!ENTITY e "text</foo>">
]>
<doc><foo>&e;more</foo></doc>`
	_, err := helium.NewParser().SubstituteEntities(true).Parse(t.Context(), []byte(src))
	require.Error(t, err, "an entity closing an outer element must be rejected")
	require.Contains(t, err.Error(), "not well balanced")
}

// A well-balanced internal entity — a complete element subtree — is accepted and
// its content is spliced into the referencing element.
func TestInternalEntityBalancedElementAccepted(t *testing.T) {
	t.Parallel()

	const src = `<!DOCTYPE doc [
<!ENTITY e "<b>x</b>">
]>
<doc>&e;</doc>`
	doc, err := helium.NewParser().SubstituteEntities(true).Parse(t.Context(), []byte(src))
	require.NoError(t, err, "a balanced element subtree entity must be accepted")
	require.NotNil(t, doc)
}

// A text-only internal entity remains accepted.
func TestInternalEntityTextAccepted(t *testing.T) {
	t.Parallel()

	const src = `<!DOCTYPE doc [
<!ENTITY e "plain text">
]>
<doc>&e;</doc>`
	doc, err := helium.NewParser().SubstituteEntities(true).Parse(t.Context(), []byte(src))
	require.NoError(t, err, "a text entity must be accepted")
	require.NotNil(t, doc)
	require.Equal(t, "plain text", string(doc.DocumentElement().Content()))
}
