package xmldsig1

import (
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

// payloadFragment is the same-document fragment URI used by the internal test
// suite to reference the element carrying Id="payload".
const payloadFragment = "#payload"

// mustParse parses xml into a document, failing the test on error. Shared by the
// internal test suite.
func mustParse(t *testing.T, xml string) *helium.Document {
	t.Helper()
	doc, err := helium.NewParser().Parse(t.Context(), []byte(xml))
	require.NoError(t, err)
	return doc
}
