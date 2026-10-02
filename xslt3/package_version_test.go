package xslt3_test

import (
	"testing"

	"github.com/lestrrat-go/helium/xslt3"
	"github.com/stretchr/testify/require"
)

// TestPackageVersionPastInt32 verifies that a version component past 2^31-1
// parses as a number on every platform. Parsing it into a 32-bit int fails, which
// would turn the whole version into a name and change how it compares.
func TestPackageVersionPastInt32(t *testing.T) {
	t.Parallel()

	v := xslt3.ParsePackageVersion("1.3000000000")
	require.Equal(t, []int64{1, 3000000000}, v.Numbers)
	require.Empty(t, v.Name)
	require.Equal(t, 1, v.Compare(xslt3.ParsePackageVersion("1.2999999999")))

	c := xslt3.ParseVersionConstraint("1.3000000000.*")
	require.Equal(t, []int64{1, 3000000000}, c.Prefix)
	require.True(t, c.Matches(xslt3.ParsePackageVersion("1.3000000000.2")))
	require.False(t, c.Matches(xslt3.ParsePackageVersion("1.5")))
}
