package xslt3_test

import (
	"testing"

	"github.com/lestrrat-go/helium/xpath3"
	"github.com/lestrrat-go/helium/xslt3"
	"github.com/stretchr/testify/require"
)

func TestNewParameters(t *testing.T) {
	p := xslt3.NewParameters()
	require.NotNil(t, p)
	require.Equal(t, 0, p.Len())
}

func TestParametersSet(t *testing.T) {
	p := xslt3.NewParameters()
	p.Set("x", xpath3.SingleString("hello"))
	require.Equal(t, 1, p.Len())

	seq, ok := p.Get("x")
	require.True(t, ok)
	require.Equal(t, 1, seq.Len())
}

func TestParametersSetString(t *testing.T) {
	p := xslt3.NewParameters()
	p.SetString("name", "value")

	seq, ok := p.Get("name")
	require.True(t, ok)
	require.Equal(t, 1, seq.Len())
}

func TestParametersSetAtomic(t *testing.T) {
	p := xslt3.NewParameters()
	p.SetAtomic("num", xpath3.AtomicValue{
		TypeName: xpath3.TypeInteger,
		Value:    42,
	})

	seq, ok := p.Get("num")
	require.True(t, ok)
	require.Equal(t, 1, seq.Len())
}

func TestParametersGetMissing(t *testing.T) {
	p := xslt3.NewParameters()
	_, ok := p.Get("missing")
	require.False(t, ok)
}

func TestParametersDelete(t *testing.T) {
	p := xslt3.NewParameters()
	p.SetString("a", "1")
	p.SetString("b", "2")
	require.Equal(t, 2, p.Len())

	p.Delete("a")
	require.Equal(t, 1, p.Len())

	_, ok := p.Get("a")
	require.False(t, ok)

	_, ok = p.Get("b")
	require.True(t, ok)
}

func TestParametersClear(t *testing.T) {
	p := xslt3.NewParameters()
	p.SetString("a", "1")
	p.SetString("b", "2")
	require.Equal(t, 2, p.Len())

	p.Clear()
	require.Equal(t, 0, p.Len())
}

func TestParametersClone(t *testing.T) {
	p := xslt3.NewParameters()
	p.SetString("x", "original")

	clone := p.Clone()
	require.Equal(t, 1, clone.Len())

	// Mutating the clone does not affect the original.
	clone.SetString("x", "mutated")
	clone.SetString("y", "new")

	require.Equal(t, 1, p.Len())
	seq, ok := p.Get("x")
	require.True(t, ok)
	require.Equal(t, 1, seq.Len())
}

func TestParametersCloneNil(t *testing.T) {
	var p *xslt3.Parameters
	clone := p.Clone()
	require.Nil(t, clone)
}

func TestParametersCloneEmpty(t *testing.T) {
	p := xslt3.NewParameters()
	clone := p.Clone()
	require.NotNil(t, clone)
	require.Equal(t, 0, clone.Len())
}

func TestParametersOverwrite(t *testing.T) {
	p := xslt3.NewParameters()
	p.SetString("x", "first")
	p.SetString("x", "second")
	require.Equal(t, 1, p.Len())

	seq, ok := p.Get("x")
	require.True(t, ok)
	require.Equal(t, 1, seq.Len())
}
