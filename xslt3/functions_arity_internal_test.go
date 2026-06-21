package xslt3

import (
	"testing"

	"github.com/lestrrat-go/helium/xpath3"
	"github.com/stretchr/testify/require"
)

// TestXSLTFunctionAritiesMatchRegistry guards xsltFunctionArities (the static
// table consulted by compile-time pattern validation) against drift from the
// runtime registries. Every fn:-namespace XSLT function registered at runtime
// must appear in the static table with identical min/max arity, and every
// static entry must be backed by a runtime registration.
func TestXSLTFunctionAritiesMatchRegistry(t *testing.T) {
	ec := &execContext{stylesheet: &Stylesheet{}}

	// Collect the runtime fn:-namespace XSLT functions from both registries:
	// the local-name map (xsltFunctions) lives in the fn: namespace, and the
	// QName map (xsltFunctionsNS) registers fn:-namespace entries explicitly.
	runtime := map[string][2]int{}
	for name, fn := range ec.xsltFunctions() {
		runtime[name] = [2]int{fn.MinArity(), fn.MaxArity()}
	}
	for qn, fn := range ec.xsltFunctionsNS() {
		if qn.URI != xpath3.NSFn {
			continue // skip schema constructors and other namespaces
		}
		// function-lookup is an XPath built-in (only specially registered in a
		// package context); it is not an XSLT-defined function for patterns.
		if qn.Name == "function-lookup" {
			continue
		}
		runtime[qn.Name] = [2]int{fn.MinArity(), fn.MaxArity()}
	}

	for name, bounds := range runtime {
		got, ok := xsltFunctionArities[name]
		require.Truef(t, ok, "xsltFunctionArities missing runtime fn:%s", name)
		require.Equalf(t, bounds, got, "arity mismatch for fn:%s", name)
	}
	for name := range xsltFunctionArities {
		_, ok := runtime[name]
		require.Truef(t, ok, "xsltFunctionArities has fn:%s with no runtime registration", name)
	}
}
