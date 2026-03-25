package schematron_test

import (
	"testing"

	"github.com/lestrrat-go/helium/schematron"
	"github.com/stretchr/testify/require"
)

func TestZeroCompilerFluent(t *testing.T) {
	var c schematron.Compiler
	require.NotPanics(t, func() {
		c2 := c.SchemaFilename("test.sch")
		_ = c2
	})
}

func TestZeroValidatorFluent(t *testing.T) {
	var v schematron.Validator
	require.NotPanics(t, func() {
		v2 := v.Filename("test.xml")
		_ = v2
	})
}
