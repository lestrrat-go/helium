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
		firstEntity, ok := helium.AsNode[*helium.Entity](first.FirstChild())
		require.True(t, ok)
		require.NotSame(t, cpEntity, firstEntity)
		require.Same(t, firstEntity, second.FirstChild())
		require.Nil(t, firstEntity.FirstChild().FirstChild())
		require.NotNil(t, cpEntity.FirstChild().FirstChild(),
			"the shared DTD declaration must stay byte-faithful")
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
