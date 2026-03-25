package xsd_test

import (
	"testing"

	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

func TestZeroCompilerFluent(t *testing.T) {
	var c xsd.Compiler
	require.NotPanics(t, func() {
		c2 := c.SchemaFilename("test.xsd").BaseDir("/tmp")
		_ = c2
	})
}

func TestZeroValidatorFluent(t *testing.T) {
	var v xsd.Validator
	require.NotPanics(t, func() {
		v2 := v.Filename("test.xml")
		_ = v2
	})
}
