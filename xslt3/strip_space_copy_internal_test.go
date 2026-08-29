package xslt3

import (
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

func TestCopyAndStrip(t *testing.T) {
	t.Run("entity replacement whitespace", func(t *testing.T) {
		t.Parallel()

		const source = `<!DOCTYPE root [<!ENTITY payload "<a>   </a>">]><root>&payload;&payload;</root>`
		src, err := helium.NewParser().Parse(t.Context(), []byte(source))
		require.NoError(t, err)

		cp, _, err := copyAndStrip(src, []nameTest{{Local: "*"}}, nil, false, nil)
		require.NoError(t, err)
		cpEntity, found := cp.IntSubset().LookupEntity("payload")
		require.True(t, found)
		require.NotSame(t, src.IntSubset(), cpEntity.Parent())

		first := cp.DocumentElement().FirstChild()
		second := first.NextSibling()
		require.Same(t, cpEntity, first.FirstChild())
		require.Same(t, cpEntity, second.FirstChild())
		require.Nil(t, cpEntity.FirstChild().FirstChild())
	})

	t.Run("whitespace-only entity uses reference parent", func(t *testing.T) {
		t.Parallel()

		const source = `<!DOCTYPE root [<!ENTITY ws "   ">]><root>&ws;</root>`
		src, err := helium.NewParser().Parse(t.Context(), []byte(source))
		require.NoError(t, err)

		cp, _, err := copyAndStrip(src, []nameTest{{Local: "*"}}, nil, false, nil)
		require.NoError(t, err)
		require.Nil(t, cp.DocumentElement().FirstChild())
	})

	t.Run("xml space preserves entity replacement whitespace", func(t *testing.T) {
		t.Parallel()

		const source = `<!DOCTYPE root [<!ENTITY payload "<a xml:space='preserve'>   </a>">]><root>&payload;</root>`
		src, err := helium.NewParser().Parse(t.Context(), []byte(source))
		require.NoError(t, err)

		cp, _, err := copyAndStrip(src, []nameTest{{Local: "*"}}, nil, false, nil)
		require.NoError(t, err)
		cpEntity, found := cp.IntSubset().LookupEntity("payload")
		require.True(t, found)
		require.Equal(t, "   ", string(cpEntity.FirstChild().FirstChild().Content()))
	})
}
