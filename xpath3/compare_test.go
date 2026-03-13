package xpath3_test

import (
	"testing"

	"github.com/lestrrat-go/helium/xpath3"
	"github.com/stretchr/testify/require"
)

func TestGeneralCompareShortCircuitsAtomization(t *testing.T) {
	t.Run("empty left skips right atomization", func(t *testing.T) {
		ok, err := xpath3.GeneralCompare(xpath3.TokenEquals, nil, xpath3.Sequence{xpath3.NewMap(nil)})
		require.NoError(t, err)
		require.False(t, ok)
	})

	t.Run("early match skips later atomization error", func(t *testing.T) {
		right := xpath3.Sequence{
			xpath3.AtomicValue{TypeName: xpath3.TypeString, Value: "match"},
			xpath3.NewMap(nil),
		}
		ok, err := xpath3.GeneralCompare(xpath3.TokenEquals, xpath3.SingleString("match"), right)
		require.NoError(t, err)
		require.True(t, ok)
	})
}
