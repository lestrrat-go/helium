package xslt3_test

import (
	"strconv"
	"testing"

	"github.com/lestrrat-go/helium/xslt3"
	"github.com/stretchr/testify/require"
)

// TestPackageVersionPastInt32 verifies that a version component past 2^31-1
// never wraps. Where int is 64 bits it is a number. Where int is 32 bits it
// does not fit and is handled like any non-numeric component: a version
// becomes all name and no numbers, and a "N.*" prefix keeps the numeric
// components before it.
func TestPackageVersionPastInt32(t *testing.T) {
	t.Parallel()

	v := xslt3.ParsePackageVersion("1.3000000000")
	c := xslt3.ParseVersionConstraint("1.3000000000.*")
	if strconv.IntSize == 32 {
		require.Nil(t, v.Numbers)
		require.Equal(t, "1.3000000000", v.Name)
		require.Equal(t, []int{1}, c.Prefix)
		return
	}
	// A variable, so the conversion compiles where int is 32 bits.
	threeBillion := int64(3000000000)
	require.Equal(t, []int{1, int(threeBillion)}, v.Numbers)
	require.Empty(t, v.Name)
	require.Equal(t, []int{1, int(threeBillion)}, c.Prefix)
	require.True(t, c.Matches(xslt3.ParsePackageVersion("1.3000000000.2")))
	require.False(t, c.Matches(xslt3.ParsePackageVersion("1.5")))
}
