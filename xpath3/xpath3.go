// Package xpath3 implements XPath 3.1 expression parsing and evaluation
// against helium XML document trees.
package xpath3

import (
	"context"
	"fmt"

	"github.com/lestrrat-go/helium"
)

// Expression is a compiled XPath 3.1 expression, reusable across evaluations.
type Expression struct {
	source     string
	ast        Expr
	prefixPlan prefixValidationPlan
}

// Compile parses an XPath 3.1 expression string into a reusable Expression.
func Compile(expr string) (*Expression, error) {
	ast, err := Parse(expr)
	if err != nil {
		return nil, err
	}
	return &Expression{
		source:     expr,
		ast:        ast,
		prefixPlan: buildPrefixValidationPlan(ast),
	}, nil
}

// MustCompile is like Compile but panics on error.
func MustCompile(expr string) *Expression {
	e, err := Compile(expr)
	if err != nil {
		panic("xpath3: Compile(" + expr + "): " + err.Error())
	}
	return e
}

// Evaluate evaluates the compiled expression against the given context node.
// The context.Context may carry an xpath3.Context (created via NewContext)
// with namespace bindings, variable bindings, and custom functions.
func (e *Expression) Evaluate(ctx context.Context, node helium.Node) (*Result, error) {
	ec := newEvalContext(ctx, node)

	// Static analysis: validate all namespace prefixes in the AST
	if err := e.prefixPlan.Validate(ec.namespaces); err != nil {
		return nil, err
	}

	seq, err := eval(ec, e.ast)
	if err != nil {
		return nil, err
	}
	return &Result{seq: seq}, nil
}

// String returns the original XPath expression string.
func (e *Expression) String() string {
	return e.source
}

// Result holds the outcome of an XPath 3.1 evaluation.
type Result struct {
	seq Sequence
}

// Sequence returns the raw result sequence.
func (r *Result) Sequence() Sequence {
	return r.seq
}

// IsNodeSet returns true if the result consists entirely of nodes.
func (r *Result) IsNodeSet() bool {
	if len(r.seq) == 0 {
		return false
	}
	for _, item := range r.seq {
		if _, ok := item.(NodeItem); !ok {
			return false
		}
	}
	return true
}

// Nodes extracts all nodes from the result.
// Returns ErrNotNodeSet if any non-node items are present.
func (r *Result) Nodes() ([]helium.Node, error) {
	if len(r.seq) == 0 {
		return nil, nil
	}
	nodes := make([]helium.Node, 0, len(r.seq))
	for _, item := range r.seq {
		ni, ok := item.(NodeItem)
		if !ok {
			return nil, ErrNotNodeSet
		}
		nodes = append(nodes, ni.Node)
	}
	return nodes, nil
}

// IsAtomic returns true if the result is a single atomic value.
func (r *Result) IsAtomic() bool {
	if len(r.seq) != 1 {
		return false
	}
	_, ok := r.seq[0].(AtomicValue)
	return ok
}

// Atomics extracts all atomic values from the result.
func (r *Result) Atomics() ([]AtomicValue, error) {
	var result []AtomicValue
	for _, item := range r.seq {
		av, ok := item.(AtomicValue)
		if !ok {
			return nil, fmt.Errorf("%w: item is %T, not AtomicValue", ErrTypeMismatch, item)
		}
		result = append(result, av)
	}
	return result, nil
}

// IsBoolean returns the boolean value and true if the result is a single boolean.
func (r *Result) IsBoolean() (bool, bool) {
	if len(r.seq) != 1 {
		return false, false
	}
	av, ok := r.seq[0].(AtomicValue)
	if !ok || av.TypeName != TypeBoolean {
		return false, false
	}
	b, ok := av.Value.(bool)
	return b, ok
}

// IsNumber returns the float64 value and true if the result is a single number.
func (r *Result) IsNumber() (float64, bool) {
	if len(r.seq) != 1 {
		return 0, false
	}
	av, ok := r.seq[0].(AtomicValue)
	if !ok {
		return 0, false
	}
	return av.ToFloat64(), av.TypeName == TypeDouble || isIntegerDerived(av.TypeName) || av.TypeName == TypeDecimal || av.TypeName == TypeFloat
}

// IsString returns the string value and true if the result is a single string.
func (r *Result) IsString() (string, bool) {
	if len(r.seq) != 1 {
		return "", false
	}
	av, ok := r.seq[0].(AtomicValue)
	if !ok || av.TypeName != TypeString {
		return "", false
	}
	s, ok := av.Value.(string)
	return s, ok
}

// Find is a convenience function: compile + evaluate, returning a node-set.
// Returns an error if the expression does not evaluate to a node-set.
func Find(ctx context.Context, node helium.Node, expr string) ([]helium.Node, error) {
	compiled, err := Compile(expr)
	if err != nil {
		return nil, err
	}
	r, err := compiled.Evaluate(ctx, node)
	if err != nil {
		return nil, err
	}
	return r.Nodes()
}

// Evaluate is a convenience function: compile + evaluate in one call.
func Evaluate(ctx context.Context, node helium.Node, expr string) (*Result, error) {
	compiled, err := Compile(expr)
	if err != nil {
		return nil, err
	}
	return compiled.Evaluate(ctx, node)
}
