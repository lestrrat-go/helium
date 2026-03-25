// Package xpath3 implements XPath 3.1 expression parsing and evaluation
// against helium XML document trees.
package xpath3

import (
	"fmt"
	"io"

	"github.com/lestrrat-go/helium"
)

// Expression is a compiled XPath 3.1 expression, reusable across evaluations.
type Expression struct {
	source     string
	ast        Expr
	program    *vmProgram
	prefixPlan prefixValidationPlan
}

// Validate runs static namespace prefix validation using the given bindings.
// This catches undeclared namespace prefixes in function calls, type names, etc.
func (e *Expression) Validate(namespaces map[string]string) error {
	return e.prefixPlan.Validate(namespaces, false, nil)
}

// String returns the original XPath expression string.
func (e *Expression) String() string {
	return e.source
}

// DumpVM writes a textual dump of compiled VM instructions.
func (e *Expression) DumpVM(w io.Writer) error {
	if e == nil || e.program == nil {
		return fmt.Errorf("xpath3: expression has no compiled program")
	}
	return e.program.dumpTo(w)
}

// Result holds the outcome of an XPath 3.1 evaluation.
type Result struct {
	seq Sequence
}

// Copy returns a deep copy of the Result whose backing storage is
// independent of any EvalState. Use this to retain a Result beyond
// the next EvaluateReuse call.
func (r Result) Copy() Result {
	if seqLen(r.seq) == 0 {
		return Result{}
	}
	return Result{seq: cloneSequence(r.seq)}
}

// Sequence returns the raw result sequence.
func (r *Result) Sequence() Sequence {
	return r.seq
}

// IsNodeSet returns true if the result consists entirely of nodes.
func (r *Result) IsNodeSet() bool {
	for item := range seqItems(r.seq) {
		if _, ok := item.(NodeItem); !ok {
			return false
		}
	}
	return true
}

// Nodes extracts all nodes from the result.
// Returns ErrNotNodeSet if any non-node items are present.
func (r *Result) Nodes() ([]helium.Node, error) {
	if seqLen(r.seq) == 0 {
		return nil, nil
	}
	nodes := make([]helium.Node, 0, r.seq.Len())
	for item := range seqItems(r.seq) {
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
	if seqLen(r.seq) != 1 {
		return false
	}
	_, ok := r.seq.Get(0).(AtomicValue)
	return ok
}

// Atomics extracts all atomic values from the result.
func (r *Result) Atomics() ([]AtomicValue, error) {
	var result []AtomicValue
	for item := range seqItems(r.seq) {
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
	if seqLen(r.seq) != 1 {
		return false, false
	}
	av, ok := r.seq.Get(0).(AtomicValue)
	if !ok || av.TypeName != TypeBoolean {
		return false, false
	}
	b, ok := av.Value.(bool)
	return b, ok
}

// IsNumber returns the float64 value and true if the result is a single number.
func (r *Result) IsNumber() (float64, bool) {
	if seqLen(r.seq) != 1 {
		return 0, false
	}
	av, ok := r.seq.Get(0).(AtomicValue)
	if !ok {
		return 0, false
	}
	return av.ToFloat64(), av.TypeName == TypeDouble || isIntegerDerived(av.TypeName) || av.TypeName == TypeDecimal || av.TypeName == TypeFloat
}

// IsString returns the string value and true if the result is a single string.
func (r *Result) IsString() (string, bool) {
	if seqLen(r.seq) != 1 {
		return "", false
	}
	av, ok := r.seq.Get(0).(AtomicValue)
	if !ok || av.TypeName != TypeString {
		return "", false
	}
	s, ok := av.Value.(string)
	return s, ok
}

func (e *Expression) evaluate(ec *evalContext) (Sequence, error) {
	if e.program == nil {
		return nil, fmt.Errorf("xpath3: expression has no compiled program")
	}
	return e.program.execute(ec)
}

func (e *Expression) astExpr() Expr {
	if e.ast != nil {
		return e.ast
	}
	if e.source == "" {
		return nil
	}
	ast, err := Parse(e.source)
	if err != nil {
		return nil
	}
	return ast
}
