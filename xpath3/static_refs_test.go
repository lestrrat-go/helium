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

	collectNames := func(refs xpath3.StaticReferences) map[string]string {
		names := map[string]string{}
		for _, tn := range refs.TypeNames {
			names[tn.Name] = tn.Prefix
		}
		return names
	}

	t.Run("nested type name in array() reported", func(t *testing.T) {
		refs := compile(t, "1 instance of array(t:smallInt)").StaticReferences()
		require.Equal(t, "t", collectNames(refs)["smallInt"])
	})

	t.Run("nested type names in map() reported", func(t *testing.T) {
		refs := compile(t, "1 instance of map(xs:string, t:foo)").StaticReferences()
		names := collectNames(refs)
		require.Equal(t, "xs", names["string"])
		require.Equal(t, "t", names["foo"])
	})

	t.Run("nested type names in function() reported", func(t *testing.T) {
		refs := compile(t, "1 treat as function(t:arg) as t:ret").StaticReferences()
		names := collectNames(refs)
		require.Equal(t, "t", names["arg"])
		require.Equal(t, "t", names["ret"])
	})

	t.Run("deeply nested type name reported", func(t *testing.T) {
		refs := compile(t, "1 instance of array(map(xs:string, t:deep))").StaticReferences()
		require.Equal(t, "t", collectNames(refs)["deep"])
	})

	t.Run("prefixed bound variable not reported", func(t *testing.T) {
		refs := compile(t, "for $p:x in (1) return $p:x").StaticReferences()
		require.Empty(t, refs.FreeVariables)
	})

	t.Run("prefixed free variable reported with prefix", func(t *testing.T) {
		refs := compile(t, "$p:x = 'a'").StaticReferences()
		require.Equal(t, []string{"p:x"}, refs.FreeVariables)
	})

	t.Run("prefixed quantified bound variable not reported", func(t *testing.T) {
		refs := compile(t, "some $p:x in (1,2) satisfies $p:x = 1").StaticReferences()
		require.Empty(t, refs.FreeVariables)
	})

	t.Run("inner shadowing binding does not unbind outer variable", func(t *testing.T) {
		// The trailing $x is bound by the OUTER for; the inner for shadows $x and on
		// exit must restore (not delete) the outer binding.
		refs := compile(t, "for $x in 1 return ((for $x in 2 return $x), $x)").StaticReferences()
		require.Empty(t, refs.FreeVariables)
	})

	t.Run("prefixed inner shadowing binding does not unbind outer variable", func(t *testing.T) {
		refs := compile(t, "for $p:x in 1 return ((for $p:x in 2 return $p:x), $p:x)").StaticReferences()
		require.Empty(t, refs.FreeVariables)
	})

	t.Run("quantified inner shadowing binding does not unbind outer variable", func(t *testing.T) {
		refs := compile(t, "for $x in 1 return ((some $x in 2 satisfies $x = 1), $x)").StaticReferences()
		require.Empty(t, refs.FreeVariables)
	})

	t.Run("inline-function param shadowing does not unbind outer variable", func(t *testing.T) {
		refs := compile(t, "for $x in 1 return ((function($x) { $x }), $x)").StaticReferences()
		require.Empty(t, refs.FreeVariables)
	})

	t.Run("inline function literal param and return types reported", func(t *testing.T) {
		refs := compile(t, "exists(function($v as t:arg) as t:ret { true() })").StaticReferences()
		names := collectNames(refs)
		require.Equal(t, "t", names["arg"])
		require.Equal(t, "t", names["ret"])
	})

	t.Run("inline function literal nested type in param type reported", func(t *testing.T) {
		refs := compile(t, "exists(function($v as array(t:deep)) as xs:boolean { true() })").StaticReferences()
		require.Equal(t, "t", collectNames(refs)["deep"])
	})

	t.Run("catch error variables not reported free", func(t *testing.T) {
		refs := compile(t, "try { xs:integer('x') } catch * { ($err:code, $err:description, $err:value, $err:module, $err:line-number, $err:column-number) }").StaticReferences()
		require.Empty(t, refs.FreeVariables)
	})

	t.Run("non-error free variable in catch still reported", func(t *testing.T) {
		refs := compile(t, "try { 1 } catch * { ($err:code, $foo) }").StaticReferences()
		require.Equal(t, []string{testFoo}, refs.FreeVariables)
	})

	t.Run("pointer-form AST nodes are walked", func(t *testing.T) {
		// CompileExpr accepts caller-built ASTs with POINTER node forms (the VM lowerer
		// dereferences them); StaticReferences must walk them consistently.
		ast := &xpath3.CastExpr{
			Expr: &xpath3.VariableExpr{Name: "v"},
			Type: xpath3.AtomicTypeName{Prefix: "t", Name: "T"},
		}
		e, err := xpath3.NewCompiler().CompileExpr(ast)
		require.NoError(t, err)
		refs := e.StaticReferences()
		require.Equal(t, []string{"v"}, refs.FreeVariables)
		require.Equal(t, "t", collectNames(refs)["T"])
	})
}
