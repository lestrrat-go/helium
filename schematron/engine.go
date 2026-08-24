package schematron

import (
	"context"
	"errors"

	helium "github.com/lestrrat-go/helium"
)

// errWrongEngine guards the engine seam: a compiled expression carries the
// query language binding it was compiled with, and only that engine can
// evaluate it. A Schema uses one engine throughout, so this never fires in
// practice.
var errWrongEngine = errors.New("schematron: expression belongs to a different query language binding")

// engine abstracts the query language used by a schema. Each Schematron
// query language binding (see QueryBinding) is backed by one engine, so
// compilation and validation never name a concrete XPath package.
type engine interface {
	// compile compiles a single expression written in the engine's query
	// language.
	compile(src string) (compiledExpr, error)

	// runner returns an evaluator bound to the schema's namespace
	// declarations.
	runner(namespaces map[string]string) runner
}

// compiledExpr is an expression compiled by an engine. Only the engine that
// produced it may evaluate it.
type compiledExpr interface {
	// source returns the original expression text.
	source() string
}

// runner evaluates compiled expressions. Runners are immutable values: bind
// returns a new runner rather than mutating the receiver, so a rule's <let>
// bindings never leak into another rule.
type runner interface {
	// evaluate evaluates expr with node as the context node.
	evaluate(ctx context.Context, expr compiledExpr, node helium.Node) (value, error)

	// bind returns a runner with an additional variable in scope. The value
	// must have been produced by the same engine.
	bind(name string, v value) runner
}

// value is the outcome of one evaluation. It hides the differences between
// the query language bindings: the same Schematron construct converts a
// result to a boolean, a string, or a name differently under XPath 1.0 and
// under XPath 3.1.
type value interface {
	// nodeSet returns the nodes in the result, or nil when the result is not
	// made up entirely of nodes. A rule context that yields anything but
	// nodes selects nothing.
	nodeSet() []helium.Node

	// effectiveBoolean reports the truth value used by <assert> and
	// <report>. XPath 3.1 raises an error for sequences that have no
	// effective boolean value; XPath 1.0 never fails.
	effectiveBoolean() (bool, error)

	// stringValue returns the text <value-of> contributes to a message.
	stringValue() string

	// nodeName returns the name <name> contributes to a message, or the
	// empty string when the result holds no named node.
	nodeName() string
}

// nodeDisplayName returns the name <name> reports for a node. Only element
// and attribute nodes have one (matching libxml2 behavior).
func nodeDisplayName(n helium.Node) string {
	if n.Type() == helium.ElementNode {
		return n.Name()
	}
	// Use type assertion for attributes since Attribute.Type() may not be set correctly.
	if attr, ok := n.(*helium.Attribute); ok {
		return attr.Name()
	}
	return ""
}
