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
		refs := compile(t, "$kind = 'x'").StaticReferences(nil)
		require.Equal(t, []string{"kind"}, refs.FreeVariables)
	})

	t.Run("bound variable not reported", func(t *testing.T) {
		refs := compile(t, "for $x in (1,2,3) return $x + 1").StaticReferences(nil)
		require.Empty(t, refs.FreeVariables)
	})

	t.Run("quantified bound variable not reported, free one is", func(t *testing.T) {
		refs := compile(t, "some $x in (1,2) satisfies $x = $y").StaticReferences(nil)
		require.Equal(t, []string{"y"}, refs.FreeVariables)
	})

	t.Run("no variables", func(t *testing.T) {
		refs := compile(t, "@a = 'x'").StaticReferences(nil)
		require.Empty(t, refs.FreeVariables)
	})

	t.Run("instance of and cast type names reported", func(t *testing.T) {
		refs := compile(t, "(@a cast as xs:int) instance of foo:bar").StaticReferences(nil)
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

	collectFuncs := func(refs xpath3.StaticReferences) map[string]bool {
		m := map[string]bool{}
		for _, fn := range refs.FunctionNames {
			key := fn.Name
			if fn.Prefix != "" {
				key = fn.Prefix + ":" + fn.Name
			}
			m[key] = true
		}
		return m
	}

	funcURIs := func(refs xpath3.StaticReferences) map[string]bool {
		m := map[string]bool{}
		for _, fn := range refs.FunctionNames {
			m[fn.URI] = true
		}
		return m
	}

	typeURIs := func(refs xpath3.StaticReferences) map[string]bool {
		m := map[string]bool{}
		for _, tn := range refs.TypeNames {
			m[tn.URI] = true
		}
		return m
	}

	const (
		nsXSD = "http://www.w3.org/2001/XMLSchema"
		nsFn  = "http://www.w3.org/2005/xpath-functions"
	)

	t.Run("nested type name in array() reported", func(t *testing.T) {
		refs := compile(t, "1 instance of array(t:smallInt)").StaticReferences(nil)
		require.Equal(t, "t", collectNames(refs)["smallInt"])
	})

	t.Run("nested type names in map() reported", func(t *testing.T) {
		refs := compile(t, "1 instance of map(xs:string, t:foo)").StaticReferences(nil)
		names := collectNames(refs)
		require.Equal(t, "xs", names["string"])
		require.Equal(t, "t", names["foo"])
	})

	t.Run("nested type names in function() reported", func(t *testing.T) {
		refs := compile(t, "1 treat as function(t:arg) as t:ret").StaticReferences(nil)
		names := collectNames(refs)
		require.Equal(t, "t", names["arg"])
		require.Equal(t, "t", names["ret"])
	})

	t.Run("deeply nested type name reported", func(t *testing.T) {
		refs := compile(t, "1 instance of array(map(xs:string, t:deep))").StaticReferences(nil)
		require.Equal(t, "t", collectNames(refs)["deep"])
	})

	t.Run("prefixed bound variable not reported", func(t *testing.T) {
		refs := compile(t, "for $p:x in (1) return $p:x").StaticReferences(nil)
		require.Empty(t, refs.FreeVariables)
	})

	t.Run("prefixed free variable reported with prefix", func(t *testing.T) {
		refs := compile(t, "$p:x = 'a'").StaticReferences(nil)
		require.Equal(t, []string{"p:x"}, refs.FreeVariables)
	})

	t.Run("prefixed quantified bound variable not reported", func(t *testing.T) {
		refs := compile(t, "some $p:x in (1,2) satisfies $p:x = 1").StaticReferences(nil)
		require.Empty(t, refs.FreeVariables)
	})

	t.Run("inner shadowing binding does not unbind outer variable", func(t *testing.T) {
		// The trailing $x is bound by the OUTER for; the inner for shadows $x and on
		// exit must restore (not delete) the outer binding.
		refs := compile(t, "for $x in 1 return ((for $x in 2 return $x), $x)").StaticReferences(nil)
		require.Empty(t, refs.FreeVariables)
	})

	t.Run("prefixed inner shadowing binding does not unbind outer variable", func(t *testing.T) {
		refs := compile(t, "for $p:x in 1 return ((for $p:x in 2 return $p:x), $p:x)").StaticReferences(nil)
		require.Empty(t, refs.FreeVariables)
	})

	t.Run("quantified inner shadowing binding does not unbind outer variable", func(t *testing.T) {
		refs := compile(t, "for $x in 1 return ((some $x in 2 satisfies $x = 1), $x)").StaticReferences(nil)
		require.Empty(t, refs.FreeVariables)
	})

	t.Run("inline-function param shadowing does not unbind outer variable", func(t *testing.T) {
		refs := compile(t, "for $x in 1 return ((function($x) { $x }), $x)").StaticReferences(nil)
		require.Empty(t, refs.FreeVariables)
	})

	t.Run("inline function literal param and return types reported", func(t *testing.T) {
		refs := compile(t, "exists(function($v as t:arg) as t:ret { true() })").StaticReferences(nil)
		names := collectNames(refs)
		require.Equal(t, "t", names["arg"])
		require.Equal(t, "t", names["ret"])
	})

	t.Run("inline function literal nested type in param type reported", func(t *testing.T) {
		refs := compile(t, "exists(function($v as array(t:deep)) as xs:boolean { true() })").StaticReferences(nil)
		require.Equal(t, "t", collectNames(refs)["deep"])
	})

	t.Run("catch error variables not reported free", func(t *testing.T) {
		refs := compile(t, "try { xs:integer('x') } catch * { ($err:code, $err:description, $err:value, $err:module, $err:line-number, $err:column-number) }").StaticReferences(nil)
		require.Empty(t, refs.FreeVariables)
	})

	t.Run("non-error free variable in catch still reported", func(t *testing.T) {
		refs := compile(t, "try { 1 } catch * { ($err:code, $foo) }").StaticReferences(nil)
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
		refs := e.StaticReferences(nil)
		require.Equal(t, []string{"v"}, refs.FreeVariables)
		require.Equal(t, "t", collectNames(refs)["T"])
	})

	t.Run("type name in element() kind test reported", func(t *testing.T) {
		refs := compile(t, ". instance of element(*, t:T)").StaticReferences(nil)
		require.Equal(t, "t", collectNames(refs)["T"])
	})

	t.Run("type name in attribute() kind test reported", func(t *testing.T) {
		refs := compile(t, ". instance of attribute(*, t:T)").StaticReferences(nil)
		require.Equal(t, "t", collectNames(refs)["T"])
	})

	t.Run("type name in document-node(element()) kind test reported", func(t *testing.T) {
		refs := compile(t, ". instance of document-node(element(*, t:T))").StaticReferences(nil)
		require.Equal(t, "t", collectNames(refs)["T"])
	})

	t.Run("type name in path-step kind test reported", func(t *testing.T) {
		refs := compile(t, "self::element(*, t:T)").StaticReferences(nil)
		require.Equal(t, "t", collectNames(refs)["T"])
	})

	t.Run("prefixed function call callee reported", func(t *testing.T) {
		// A user-type constructor is a function call carrying a user-namespace callee.
		refs := compile(t, "t:smallInt(@kind) = 1").StaticReferences(nil)
		require.True(t, collectFuncs(refs)["t:smallInt"])
	})

	t.Run("unprefixed function call callee reported", func(t *testing.T) {
		refs := compile(t, "string(@kind)").StaticReferences(nil)
		require.True(t, collectFuncs(refs)["string"])
	})

	t.Run("named function ref callee reported", func(t *testing.T) {
		refs := compile(t, "t:f#1").StaticReferences(nil)
		require.True(t, collectFuncs(refs)["t:f"])
	})

	// --- resolved URIs across all name forms (the convergent dimension) ---

	ns := map[string]string{"t": "urn:t", "xs": nsXSD}

	t.Run("prefixed type name resolves via in-scope namespace", func(t *testing.T) {
		refs := compile(t, "1 instance of t:T").StaticReferences(ns)
		require.True(t, typeURIs(refs)["urn:t"])
	})

	t.Run("predeclared xs type name resolves to XSD even when not declared", func(t *testing.T) {
		refs := compile(t, "1 instance of xs:integer").StaticReferences(nil)
		require.True(t, typeURIs(refs)[nsXSD])
	})

	t.Run("braced-uri type name resolves to its uri", func(t *testing.T) {
		refs := compile(t, "1 instance of Q{urn:t}smallInt").StaticReferences(nil)
		require.True(t, typeURIs(refs)["urn:t"])
	})

	t.Run("braced-uri xs type name resolves to XSD", func(t *testing.T) {
		refs := compile(t, "1 instance of Q{"+nsXSD+"}integer").StaticReferences(nil)
		require.True(t, typeURIs(refs)[nsXSD])
	})

	t.Run("braced-uri cast type name resolves to its uri", func(t *testing.T) {
		refs := compile(t, "@x cast as Q{urn:t}smallInt").StaticReferences(nil)
		require.True(t, typeURIs(refs)["urn:t"])
	})

	t.Run("prefixed function call resolves via in-scope namespace", func(t *testing.T) {
		refs := compile(t, "t:smallInt(@kind)").StaticReferences(ns)
		require.True(t, funcURIs(refs)["urn:t"])
	})

	t.Run("braced-uri function call resolves to its uri", func(t *testing.T) {
		refs := compile(t, "Q{urn:t}smallInt(@kind)").StaticReferences(nil)
		require.True(t, funcURIs(refs)["urn:t"])
	})

	t.Run("braced-uri named function ref resolves to its uri", func(t *testing.T) {
		refs := compile(t, "Q{urn:t}f#1").StaticReferences(nil)
		require.True(t, funcURIs(refs)["urn:t"])
	})

	t.Run("braced-uri arrow target resolves to its uri", func(t *testing.T) {
		refs := compile(t, "1 => Q{urn:t}f()").StaticReferences(nil)
		require.True(t, funcURIs(refs)["urn:t"])
	})

	t.Run("unprefixed function resolves to fn namespace", func(t *testing.T) {
		refs := compile(t, "string(@kind)").StaticReferences(nil)
		require.True(t, funcURIs(refs)[nsFn])
	})

	t.Run("prefixed arrow target resolves via predeclared fn-family namespace", func(t *testing.T) {
		refs := compile(t, "1 => fn:abs()").StaticReferences(nil)
		require.True(t, funcURIs(refs)[nsFn])
	})
}
