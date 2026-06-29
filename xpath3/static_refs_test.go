package xpath3_test

import (
	"testing"

	"github.com/lestrrat-go/helium/xpath3"
	"github.com/stretchr/testify/require"
)

func TestStaticReferences(t *testing.T) {
	compile := func(t *testing.T, expr string) *xpath3.Expression {
		t.Helper()
		e, err := xpath3.NewCompiler().Compile(expr)
		require.NoError(t, err)
		return e
	}

	t.Run("free variable reported", func(t *testing.T) {
		refs := compile(t, "$kind = 'x'").StaticReferences()
		require.Equal(t, []string{"kind"}, refs.FreeVariables)
	})

	t.Run("bound variable not reported", func(t *testing.T) {
		refs := compile(t, "for $x in (1,2,3) return $x + 1").StaticReferences()
		require.Empty(t, refs.FreeVariables)
	})

	t.Run("quantified bound variable not reported, free one is", func(t *testing.T) {
		refs := compile(t, "some $x in (1,2) satisfies $x = $y").StaticReferences()
		require.Equal(t, []string{"y"}, refs.FreeVariables)
	})

	t.Run("no variables", func(t *testing.T) {
		refs := compile(t, "@a = 'x'").StaticReferences()
		require.Empty(t, refs.FreeVariables)
	})

	t.Run("instance of and cast type names reported", func(t *testing.T) {
		refs := compile(t, "(@a cast as xs:int) instance of foo:bar").StaticReferences()
		names := map[string]string{}
		for _, tn := range refs.TypeNames {
			names[tn.Name] = tn.Prefix
		}
		require.Equal(t, "xs", names["int"])
		require.Equal(t, "foo", names["bar"])
	})
}
