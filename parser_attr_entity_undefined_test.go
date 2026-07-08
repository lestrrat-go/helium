package helium_test

import (
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

// An internal general entity whose replacement text references an undefined
// entity, referenced from an attribute value, must be reported as a
// well-formedness error — not crash the parser. getEntity returns a typed-nil
// *Entity for the undefined inner entity, which a naive interface nil-check
// misses. (W3C not-wf-sa-077.)
func TestAttributeValueUndefinedNestedEntityDoesNotPanic(t *testing.T) {
	t.Parallel()

	const doc = `<!DOCTYPE doc [
<!ENTITY foo "&bar;">
]>
<doc a="&foo;"></doc>`

	require.NotPanics(t, func() {
		_, err := helium.NewParser().Parse(t.Context(), []byte(doc))
		require.Error(t, err, "a reference to an undefined entity must be a well-formedness error")
	})
}
