package xpath3

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFnAvgLexicalDecimal(t *testing.T) {
	seq, err := fnAvg(t.Context(), []Sequence{{
		AtomicValue{TypeName: TypeDecimal, Value: "1.5"},
		AtomicValue{TypeName: TypeDecimal, Value: "2.25"},
	}})
	require.NoError(t, err)
	require.Len(t, seq, 1)

	av := seq[0].(AtomicValue)
	require.Equal(t, TypeDecimal, av.TypeName)
	require.Equal(t, "1.875", DecimalToString(av.BigRat()))
}
