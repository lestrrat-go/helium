package relaxng_test

import (
	"testing"

	"github.com/lestrrat-go/helium/relaxng"
	"github.com/stretchr/testify/require"
)

func TestZeroCompilerFluent(t *testing.T) {
	var c relaxng.Compiler
	require.NotPanics(t, func() {
		c2 := c.SchemaFilename("test.rng")
		_ = c2
	})
}

func TestZeroValidatorFluent(t *testing.T) {
	var v relaxng.Validator
	require.NotPanics(t, func() {
		v2 := v.Filename("test.xml")
		_ = v2
	})
}
