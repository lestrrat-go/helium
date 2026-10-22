package relaxng_test

import (
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/relaxng"
	"github.com/stretchr/testify/require"
)

func TestUnexpandedEntityReferenceFailsValidation(t *testing.T) {
	t.Parallel()

	textGrammar := compileGrammar(t, `<element name="root" xmlns="http://relaxng.org/ns/structure/1.0"><text/></element>`)
	textSource := `<!DOCTYPE root [<!ENTITY e "content">]><root>&e;</root>`
	textDoc, err := helium.NewParser().Parse(t.Context(), []byte(textSource))
	require.NoError(t, err)
	require.NoError(t, relaxng.NewValidator(textGrammar).Validate(t.Context(), textDoc))

	expandedTextDoc, err := helium.NewParser().SubstituteEntities(true).Parse(t.Context(), []byte(textSource))
	require.NoError(t, err)
	require.NoError(t, relaxng.NewValidator(textGrammar).Validate(t.Context(), expandedTextDoc))

	emptyGrammar := compileGrammar(t, `<element name="root" xmlns="http://relaxng.org/ns/structure/1.0"><empty/></element>`)
	elementSource := `<!DOCTYPE root [<!ENTITY e "<forbidden/>">]><root>&e;</root>`
	elementDoc, err := helium.NewParser().Parse(t.Context(), []byte(elementSource))
	require.NoError(t, err)
	require.ErrorIs(t, relaxng.NewValidator(emptyGrammar).Validate(t.Context(), elementDoc), relaxng.ErrValidationFailed)

	expandedElementDoc, err := helium.NewParser().SubstituteEntities(true).Parse(t.Context(), []byte(elementSource))
	require.NoError(t, err)
	require.ErrorIs(t, relaxng.NewValidator(emptyGrammar).Validate(t.Context(), expandedElementDoc), relaxng.ErrValidationFailed)
}
