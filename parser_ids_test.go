package helium_test

import (
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

func TestGetElementByIDAfterParse(t *testing.T) {
	const input = `<root xml:id="root-id"><child xml:id="child-id"/></root>`

	doc, err := helium.NewParser().Parse(t.Context(), []byte(input))
	require.NoError(t, err)
	require.NotNil(t, doc.GetElementByID("root-id"))
	require.NotNil(t, doc.GetElementByID("child-id"))
}

func TestGetElementByIDAfterParseWithSkipIDs(t *testing.T) {
	const input = `<root xml:id="root-id"><child xml:id="child-id"/></root>`

	doc, err := helium.NewParser().SkipIDs(true).Parse(t.Context(), []byte(input))
	require.NoError(t, err)
	require.Nil(t, doc.GetElementByID("root-id"))
	require.Nil(t, doc.GetElementByID("child-id"))
}
