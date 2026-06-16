package helium_test

import (
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

// TestMalformedCharRefNoPanic guards against the index-out-of-range panic in
// parseStringCharRef when malformed character references appear inside entity
// declarations. Each malformed input must yield a non-nil error rather than
// panic.
func TestMalformedCharRefNoPanic(t *testing.T) {
	malformed := []string{
		`<!DOCTYPE root [<!ENTITY e "&#">]><root>&e;</root>`,
		`<!DOCTYPE root [<!ENTITY e "&#x">]><root>&e;</root>`,
		`<!DOCTYPE root [<!ENTITY e "&#1">]><root>&e;</root>`,
		`<!DOCTYPE root [<!ENTITY e "&#xZ">]><root>&e;</root>`,
		`<!DOCTYPE root [<!ENTITY e "&#;">]><root>&e;</root>`,
	}

	for _, input := range malformed {
		t.Run(input, func(t *testing.T) {
			require.NotPanics(t, func() {
				_, err := helium.NewParser().SubstituteEntities(true).Parse(t.Context(), []byte(input))
				require.Error(t, err, "malformed char ref must return an error")
			})
		})
	}
}

// TestValidCharRefStillWorks guards against over-rejection: a valid decimal
// character reference entity must still parse and expand correctly.
func TestValidCharRefStillWorks(t *testing.T) {
	input := `<!DOCTYPE root [<!ENTITY e "&#65;">]><root>&e;</root>`
	doc, err := helium.NewParser().SubstituteEntities(true).Parse(t.Context(), []byte(input))
	require.NoError(t, err, "valid char ref must parse")
	require.NotNil(t, doc)

	root := doc.DocumentElement()
	require.NotNil(t, root)
	child := root.FirstChild()
	require.NotNil(t, child, "expanded entity must produce a text child")
	require.Equal(t, "A", string(child.Content()), "&#65; must expand to 'A'")
}
