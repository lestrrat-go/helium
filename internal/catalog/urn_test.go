package catalog_test

import (
	"testing"

	"github.com/lestrrat-go/helium/internal/catalog"
	"github.com/stretchr/testify/require"
)

func TestUnwrapURN(t *testing.T) {
	t.Run("case-insensitive scheme and namespace", func(t *testing.T) {
		want := catalog.UnwrapURN("urn:publicid:foo")
		require.Equal(t, "foo", want)

		// URN scheme + namespace identifier are case-insensitive per
		// RFC 8141 / OASIS catalog spec; uppercase prefix must also match.
		require.Equal(t, want, catalog.UnwrapURN("URN:PUBLICID:foo"))
		require.Equal(t, want, catalog.UnwrapURN("Urn:PublicId:foo"))
	})

	t.Run("payload case preserved", func(t *testing.T) {
		require.Equal(t, "FooBar", catalog.UnwrapURN("URN:PUBLICID:FooBar"))
	})

	t.Run("not a publicid urn", func(t *testing.T) {
		require.Equal(t, "", catalog.UnwrapURN("urn:isbn:0451450523"))
		require.Equal(t, "", catalog.UnwrapURN("http://example.com/foo"))
	})
}
