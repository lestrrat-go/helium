package xpath3

import (
	"fmt"
	"strings"

	ixpath "github.com/lestrrat-go/helium/internal/xpath"
)

// evalGeneralComparison implements general comparison (= != < <= > >=).
// Per XPath 3.1 Section 3.7.1: atomize both operands, then existentially
// quantify — true if ANY pair satisfies the value comparison.
func evalGeneralComparison(ec *evalContext, e BinaryExpr) (Sequence, error) {
	left, err := eval(ec, e.Left)
	if err != nil {
		return nil, err
	}
	right, err := eval(ec, e.Right)
	if err != nil {
		return nil, err
	}
	result, err := GeneralCompare(e.Op, left, right)
	if err != nil {
		return nil, err
	}
	return SingleBoolean(result), nil
}

// evalValueComparison implements value comparison (eq ne lt le gt ge).
// Per XPath 3.1 Section 3.7.2: both operands must be single atomic values.
func evalValueComparison(ec *evalContext, e BinaryExpr) (Sequence, error) {
	left, err := eval(ec, e.Left)
	if err != nil {
		return nil, err
	}
	right, err := eval(ec, e.Right)
	if err != nil {
		return nil, err
	}
	// Empty sequence yields empty sequence
	if len(left) == 0 || len(right) == 0 {
		return nil, nil
	}
	if len(left) > 1 || len(right) > 1 {
		return nil, &XPathError{Code: "XPTY0004", Message: "value comparison requires singletons"}
	}
	la, err := AtomizeItem(left[0])
	if err != nil {
		return nil, err
	}
	ra, err := AtomizeItem(right[0])
	if err != nil {
		return nil, err
	}
	result, err := ValueCompare(e.Op, la, ra)
	if err != nil {
		return nil, err
	}
	return SingleBoolean(result), nil
}

func evalNodeComparison(ec *evalContext, e BinaryExpr) (Sequence, error) {
	left, err := eval(ec, e.Left)
	if err != nil {
		return nil, err
	}
	right, err := eval(ec, e.Right)
	if err != nil {
		return nil, err
	}
	// Empty sequence yields empty sequence
	if len(left) == 0 || len(right) == 0 {
		return nil, nil
	}
	if len(left) > 1 || len(right) > 1 {
		return nil, &XPathError{Code: "XPTY0004", Message: "node comparison requires singletons"}
	}
	ln, ok := left[0].(NodeItem)
	if !ok {
		return nil, &XPathError{Code: "XPTY0004", Message: "node comparison requires node operands"}
	}
	rn, ok := right[0].(NodeItem)
	if !ok {
		return nil, &XPathError{Code: "XPTY0004", Message: "node comparison requires node operands"}
	}
	switch e.Op {
	case TokenIs:
		return SingleBoolean(ln.Node == rn.Node), nil
	case TokenNodePre:
		lp := ec.docOrder.Position(ln.Node)
		rp := ec.docOrder.Position(rn.Node)
		if lp < 0 || rp < 0 {
			ec.docOrder.BuildFrom(ixpath.DocumentRoot(ln.Node))
			lp = ec.docOrder.Position(ln.Node)
			rp = ec.docOrder.Position(rn.Node)
		}
		return SingleBoolean(lp < rp), nil
	case TokenNodeFol:
		lp := ec.docOrder.Position(ln.Node)
		rp := ec.docOrder.Position(rn.Node)
		if lp < 0 || rp < 0 {
			ec.docOrder.BuildFrom(ixpath.DocumentRoot(ln.Node))
			lp = ec.docOrder.Position(ln.Node)
			rp = ec.docOrder.Position(rn.Node)
		}
		return SingleBoolean(lp > rp), nil
	}
	return nil, fmt.Errorf("%w: %s", ErrUnsupportedBinaryOp, e.Op)
}

// GeneralCompare performs a general comparison between two sequences.
// Atomizes both sides and returns true if any pair of atomic values
// satisfies the operator.
func GeneralCompare(op TokenType, left, right Sequence) (bool, error) {
	leftAtoms, err := AtomizeSequence(left)
	if err != nil {
		return false, err
	}
	rightAtoms, err := AtomizeSequence(right)
	if err != nil {
		return false, err
	}
	for _, la := range leftAtoms {
		for _, ra := range rightAtoms {
			pa, pb := promoteForComparison(la, ra)
			match, err := compareAtomic(op, pa, pb)
			if err != nil {
				return false, err
			}
			if match {
				return true, nil
			}
		}
	}
	return false, nil
}

// ValueCompare performs a value comparison between two atomic values.
func ValueCompare(op TokenType, a, b AtomicValue) (bool, error) {
	pa, pb := promoteForComparison(a, b)
	return compareAtomic(op, pa, pb)
}

// promoteForComparison applies type promotion rules per XPath 3.1 Section 3.7.
func promoteForComparison(a, b AtomicValue) (AtomicValue, AtomicValue) {
	// untypedAtomic vs untypedAtomic → compare as string
	if a.TypeName == TypeUntypedAtomic && b.TypeName == TypeUntypedAtomic {
		return AtomicValue{TypeName: TypeString, Value: stringFromAtomic(a)},
			AtomicValue{TypeName: TypeString, Value: stringFromAtomic(b)}
	}

	// untypedAtomic vs numeric → cast to double
	if a.TypeName == TypeUntypedAtomic && b.IsNumeric() {
		return AtomicValue{TypeName: TypeDouble, Value: promoteToDouble(a)}, b
	}
	if b.TypeName == TypeUntypedAtomic && a.IsNumeric() {
		return a, AtomicValue{TypeName: TypeDouble, Value: promoteToDouble(b)}
	}

	// untypedAtomic vs string → compare as string
	if a.TypeName == TypeUntypedAtomic && b.TypeName == TypeString {
		return AtomicValue{TypeName: TypeString, Value: stringFromAtomic(a)}, b
	}
	if b.TypeName == TypeUntypedAtomic && a.TypeName == TypeString {
		return a, AtomicValue{TypeName: TypeString, Value: stringFromAtomic(b)}
	}

	// untypedAtomic vs other → cast to other's type
	if a.TypeName == TypeUntypedAtomic {
		return AtomicValue{TypeName: TypeString, Value: stringFromAtomic(a)},
			AtomicValue{TypeName: TypeString, Value: stringFromAtomic(b)}
	}
	if b.TypeName == TypeUntypedAtomic {
		return AtomicValue{TypeName: TypeString, Value: stringFromAtomic(a)},
			AtomicValue{TypeName: TypeString, Value: stringFromAtomic(b)}
	}

	return a, b
}

// stringFromAtomic extracts a string from an atomic value.
func stringFromAtomic(a AtomicValue) string {
	if s, ok := a.Value.(string); ok {
		return s
	}
	s, _ := atomicToString(a)
	return s
}

// compareAtomic compares two (already promoted) atomic values.
func compareAtomic(op TokenType, a, b AtomicValue) (bool, error) {
	// Map general comparison operators to value comparison operators
	op = normalizeCompareOp(op)

	// String comparison
	if a.TypeName == TypeString && b.TypeName == TypeString {
		sa := a.Value.(string)
		sb := b.Value.(string)
		cmp := strings.Compare(sa, sb)
		return applyCompare(op, cmp), nil
	}

	// Boolean comparison
	if a.TypeName == TypeBoolean && b.TypeName == TypeBoolean {
		ba := a.Value.(bool)
		bb := b.Value.(bool)
		return compareBooleans(op, ba, bb), nil
	}

	// Numeric comparison
	if a.IsNumeric() && b.IsNumeric() {
		fa := promoteToDouble(a)
		fb := promoteToDouble(b)
		return compareFloats(op, fa, fb), nil
	}

	// Mixed numeric/string — promote to double
	if a.IsNumeric() || b.IsNumeric() {
		fa := promoteToDouble(a)
		fb := promoteToDouble(b)
		return compareFloats(op, fa, fb), nil
	}

	// Fallback: string comparison
	sa, _ := atomicToString(a)
	sb, _ := atomicToString(b)
	cmp := strings.Compare(sa, sb)
	return applyCompare(op, cmp), nil
}

// normalizeCompareOp maps general comparison operators to value comparison operators.
func normalizeCompareOp(op TokenType) TokenType {
	switch op {
	case TokenEquals:
		return TokenEq
	case TokenNotEquals:
		return TokenNe
	case TokenLess:
		return TokenLt
	case TokenLessEq:
		return TokenLe
	case TokenGreater:
		return TokenGt
	case TokenGreaterEq:
		return TokenGe
	}
	return op
}

func applyCompare(op TokenType, cmp int) bool {
	switch op {
	case TokenEq:
		return cmp == 0
	case TokenNe:
		return cmp != 0
	case TokenLt:
		return cmp < 0
	case TokenLe:
		return cmp <= 0
	case TokenGt:
		return cmp > 0
	case TokenGe:
		return cmp >= 0
	}
	return false
}

func compareFloats(op TokenType, a, b float64) bool {
	switch op {
	case TokenEq:
		return a == b
	case TokenNe:
		return a != b
	case TokenLt:
		return a < b
	case TokenLe:
		return a <= b
	case TokenGt:
		return a > b
	case TokenGe:
		return a >= b
	}
	return false
}

func compareBooleans(op TokenType, a, b bool) bool {
	ai, bi := 0, 0
	if a {
		ai = 1
	}
	if b {
		bi = 1
	}
	switch op {
	case TokenEq:
		return ai == bi
	case TokenNe:
		return ai != bi
	case TokenLt:
		return ai < bi
	case TokenLe:
		return ai <= bi
	case TokenGt:
		return ai > bi
	case TokenGe:
		return ai >= bi
	}
	return false
}
