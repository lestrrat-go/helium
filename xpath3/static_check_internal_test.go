package xpath3

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCompileBuildsPrefixValidationPlan(t *testing.T) {
	expr, err := Compile(`p:noop()`)
	require.NoError(t, err)
	require.NotEmpty(t, expr.prefixPlan.prefixes)
	require.NoError(t, expr.prefixPlan.Validate(map[string]string{
		"p": "urn:test",
	}))
}
