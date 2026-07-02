package xpath3

import (
	"context"
	"errors"
	"math"
	"time"

	ixpath "github.com/lestrrat-go/helium/internal/xpath"
)

// This file implements XPath 1.0 compatibility mode, the evaluation semantics
// XSLT selects for backwards-compatible processing (an effective [xsl:]version
// below 2.0). Every rule here is reached ONLY when evalContext.xpath10Compat is
// true, so ordinary XPath 3.1 / XSLT 3.0 / XSD evaluation is byte-identical.
//
// The three effect sites (XPath 2.0/3.0 language spec) are:
//   - Function conversion rules (§3.1.5.2): coerceToSequenceTypeE
//   - Arithmetic (§3.4): evalArithmetic / evalUnaryExpr
//   - General comparisons (§3.5.2): evalGeneralComparison

// doubleNaNAtom returns an xs:double NaN, the fn:number result for an empty or
// lexically-invalid value.
func doubleNaNAtom() AtomicValue {
	return AtomicValue{TypeName: TypeDouble, Value: NewDouble(math.NaN())}
}

// atomToCompatDouble converts a single atom to xs:double with fn:number
// semantics: a lexically-invalid value yields NaN rather than an error.
func atomToCompatDouble(a AtomicValue) AtomicValue {
	d, err := CastAtomic(a, TypeDouble)
	if err != nil {
		return doubleNaNAtom()
	}
	return d
}

// xpath10CompatDoubleFromAtoms applies the arithmetic operand rule: discard all
// atoms after the first, then convert to xs:double (empty → NaN).
func xpath10CompatDoubleFromAtoms(atoms []AtomicValue) AtomicValue {
	if len(atoms) == 0 {
		return doubleNaNAtom()
	}
	return atomToCompatDouble(atoms[0])
}

// xpath10CompatStringItem implements the xs:string(?) function-conversion rule:
// the value is replaced by fn:string applied to its first item (empty → "").
func xpath10CompatStringItem(seq Sequence) (AtomicValue, error) {
	if seqLen(seq) == 0 {
		return AtomicValue{TypeName: TypeString, Value: ""}, nil
	}
	item := seq.Get(0)
	switch item.(type) {
	case FunctionItem, MapItem, ArrayItem:
		return AtomicValue{}, &XPathError{Code: errCodeFOTY0014, Message: "xpath 1.0 compatibility: cannot take string value of a function item"}
	}
	if ni, ok := item.(NodeItem); ok {
		return AtomicValue{TypeName: TypeString, Value: ixpath.StringValue(ni.Node)}, nil
	}
	a, err := AtomizeItem(item)
	if err != nil {
		return AtomicValue{}, err
	}
	s, _ := atomicToString(a)
	return AtomicValue{TypeName: TypeString, Value: s}, nil
}

// xpath10CompatNumberItem implements the xs:double(?)/xs:numeric(?)
// function-conversion rule: fn:number of the first item (empty or invalid → NaN).
func xpath10CompatNumberItem(seq Sequence) (AtomicValue, error) {
	if seqLen(seq) == 0 {
		return doubleNaNAtom(), nil
	}
	a, err := AtomizeItem(seq.Get(0))
	if err != nil {
		// FOTY0013 (atomizing function items) propagates per XPath 3.1 §2.7.2.
		var xpErr *XPathError
		if errors.As(err, &xpErr) && xpErr.Code == errCodeFOTY0013 {
			return AtomicValue{}, err
		}
		return doubleNaNAtom(), nil
	}
	return atomToCompatDouble(a), nil
}

// atomizeAllCompat fully atomizes a sequence into a slice. Compat-mode operands
// are small (XSLT 1.0 stylesheets), so materializing is acceptable here.
func atomizeAllCompat(seq Sequence) ([]AtomicValue, error) {
	var out []AtomicValue
	err := atomizeStream(seq, func(av AtomicValue) (bool, error) {
		out = append(out, av)
		return true, nil
	})
	return out, err
}

// coerceXPath10Compat applies the XPath 1.0 function-conversion rules to an
// argument being coerced to expected type st. Returns (result, done, err):
//   - done=true  → the compat rule produced the final coerced value (or error);
//   - done=false → the rule only applied the first-item truncation, and the
//     returned sequence should continue through the ordinary coercion path.
func coerceXPath10Compat(seq Sequence, st SequenceType, ec *evalContext) (Sequence, bool, error) {
	single := st.Occurrence == OccurrenceExactlyOne || st.Occurrence == OccurrenceZeroOrOne
	if at, ok := st.ItemTest.(AtomicOrUnionType); ok && single {
		switch resolveAtomicTypeName(AtomicTypeName(at), ec) {
		case TypeString:
			s, err := xpath10CompatStringItem(seq)
			if err != nil {
				return seq, true, err
			}
			return ItemSlice{s}, true, nil
		case TypeDouble, TypeNumeric:
			d, err := xpath10CompatNumberItem(seq)
			if err != nil {
				return seq, true, err
			}
			return ItemSlice{d}, true, nil
		}
	}
	// First-item rule: any other single-item expected type keeps just the first
	// item; the ordinary coercion then atomizes/casts that singleton.
	if single && seqLen(seq) > 1 {
		return ItemSlice{seq.Get(0)}, false, nil
	}
	return seq, false, nil
}

// generalCompareXPath10Compat evaluates a general comparison (= != < <= > >=)
// under the XPath 1.0 conversion rules (§3.5.2):
//   - if either operand is a single xs:boolean, the other is converted to its
//     effective boolean value and the two booleans are compared;
//   - otherwise, if either operand is numeric, every pair is compared as xs:double;
//   - otherwise every pair is compared as xs:string.
//
// The numeric/string cases stay existential (any satisfying pair wins), matching
// the 1.0 node-set comparison semantics.
func generalCompareXPath10Compat(ctx context.Context, op TokenType, left, right Sequence, coll *collationImpl, ec *evalContext) (bool, error) {
	la, err := atomizeAllCompat(left)
	if err != nil {
		return false, err
	}
	ra, err := atomizeAllCompat(right)
	if err != nil {
		return false, err
	}

	var implTZ *time.Location
	if ec != nil {
		implTZ = ec.getImplicitTimezone()
	}

	leftBool := len(la) == 1 && la[0].TypeName == TypeBoolean
	rightBool := len(ra) == 1 && ra[0].TypeName == TypeBoolean
	if leftBool || rightBool {
		lb, err := EBV(left)
		if err != nil {
			return false, err
		}
		rb, err := EBV(right)
		if err != nil {
			return false, err
		}
		return compareAtomicCollation(op,
			AtomicValue{TypeName: TypeBoolean, Value: lb},
			AtomicValue{TypeName: TypeBoolean, Value: rb},
			implTZ, coll)
	}

	// The relational operators (<, <=, >, >=) always convert both operands to
	// xs:double in XPath 1.0; only the equality operators (=, !=) use the
	// numeric-else-string cascade.
	relational := op == TokenLess || op == TokenLessEq || op == TokenGreater || op == TokenGreaterEq
	for _, a := range la {
		for _, b := range ra {
			if err := fnCountOp(ctx, ec); err != nil {
				return false, err
			}
			pa, pb := promoteXPath10CompatPair(a, b, relational)
			match, err := compareAtomicCollation(op, pa, pb, implTZ, coll)
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

// promoteXPath10CompatPair promotes one atom pair for a 1.0 general comparison.
// For a relational operator both operands become xs:double (fn:number). For an
// equality operator, numeric-vs-anything compares as xs:double, otherwise both
// compare as xs:string (xs:untypedAtomic alone is NOT numeric, so node-vs-node
// stays a string compare, matching XSLT 1.0 string-value comparison).
func promoteXPath10CompatPair(a, b AtomicValue, relational bool) (AtomicValue, AtomicValue) {
	if relational || a.IsNumeric() || b.IsNumeric() {
		return atomToCompatDouble(a), atomToCompatDouble(b)
	}
	return AtomicValue{TypeName: TypeString, Value: stringFromAtomic(a)},
		AtomicValue{TypeName: TypeString, Value: stringFromAtomic(b)}
}
