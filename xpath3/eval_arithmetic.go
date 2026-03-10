package xpath3

import (
	"fmt"
	"math"
	"math/big"
)

func evalArithmetic(ec *evalContext, e BinaryExpr) (Sequence, error) {
	left, err := eval(ec, e.Left)
	if err != nil {
		return nil, err
	}
	right, err := eval(ec, e.Right)
	if err != nil {
		return nil, err
	}
	if len(left) == 0 || len(right) == 0 {
		return nil, nil // empty sequence
	}
	if len(left) > 1 {
		return nil, &XPathError{Code: "XPTY0004", Message: "arithmetic operand must be a single item"}
	}
	if len(right) > 1 {
		return nil, &XPathError{Code: "XPTY0004", Message: "arithmetic operand must be a single item"}
	}
	la, err := AtomizeItem(left[0])
	if err != nil {
		return nil, err
	}
	ra, err := AtomizeItem(right[0])
	if err != nil {
		return nil, err
	}

	// Duration/date/time arithmetic — handle before numeric promotion.
	// Contract: evalDateTimeArithmetic returns handled==false only when both
	// operands are non-duration/non-datetime, in which case err is always nil.
	if result, handled, err := evalDateTimeArithmetic(e.Op, la, ra); err != nil || handled {
		return result, err
	}

	// Promote xs:untypedAtomic to xs:double for arithmetic
	if la.TypeName == TypeUntypedAtomic {
		castVal, err := CastAtomic(la, TypeDouble)
		if err != nil {
			return nil, err
		}
		la = castVal
	}
	if ra.TypeName == TypeUntypedAtomic {
		castVal, err := CastAtomic(ra, TypeDouble)
		if err != nil {
			return nil, err
		}
		ra = castVal
	}

	// Tier 1: both integer → big.Int arithmetic
	if isIntegerDerived(la.TypeName) && isIntegerDerived(ra.TypeName) {
		return integerArith(e.Op, la.BigInt(), ra.BigInt())
	}
	// Tier 2: either decimal (or integer promoted) → big.Rat arithmetic
	if needsDecimalArith(la.TypeName, ra.TypeName) {
		return decimalArith(e.Op, toRat(la), toRat(ra))
	}
	// Tier 3: float/double → float64 arithmetic
	return floatArith(e.Op, la, ra)
}

func integerArith(op TokenType, a, b *big.Int) (Sequence, error) {
	result := new(big.Int)
	switch op {
	case TokenPlus:
		result.Add(a, b)
	case TokenMinus:
		result.Sub(a, b)
	case TokenStar:
		result.Mul(a, b)
	case TokenDiv:
		// integer / integer → decimal
		if b.Sign() == 0 {
			return nil, &XPathError{Code: "FOAR0002", Message: "division by zero"}
		}
		r := new(big.Rat).SetFrac(new(big.Int).Set(a), new(big.Int).Set(b))
		return SingleDecimal(r), nil
	case TokenIdiv:
		if b.Sign() == 0 {
			return nil, &XPathError{Code: "FOAR0002", Message: "integer division by zero"}
		}
		result.Quo(a, b) // truncates toward zero
	case TokenMod:
		if b.Sign() == 0 {
			return nil, &XPathError{Code: "FOAR0002", Message: "modulo by zero"}
		}
		result.Rem(a, b)
	}
	return SingleIntegerBig(result), nil
}

func decimalArith(op TokenType, a, b *big.Rat) (Sequence, error) {
	result := new(big.Rat)
	switch op {
	case TokenPlus:
		result.Add(a, b)
	case TokenMinus:
		result.Sub(a, b)
	case TokenStar:
		result.Mul(a, b)
	case TokenDiv:
		if b.Sign() == 0 {
			return nil, &XPathError{Code: "FOAR0002", Message: "division by zero"}
		}
		result.Quo(a, b)
	case TokenIdiv:
		if b.Sign() == 0 {
			return nil, &XPathError{Code: "FOAR0002", Message: "integer division by zero"}
		}
		// Truncate quotient toward zero
		q := new(big.Rat).Quo(a, b)
		// Extract integer part
		num := q.Num()
		den := q.Denom()
		intPart := new(big.Int).Quo(num, den)
		return SingleIntegerBig(intPart), nil
	case TokenMod:
		if b.Sign() == 0 {
			return nil, &XPathError{Code: "FOAR0002", Message: "modulo by zero"}
		}
		// a mod b = a - (a idiv b) * b
		q := new(big.Rat).Quo(a, b)
		intPart := new(big.Int).Quo(q.Num(), q.Denom())
		qr := new(big.Rat).SetInt(intPart)
		result.Sub(a, new(big.Rat).Mul(qr, b))
	}
	return SingleDecimal(result), nil
}

func floatArith(op TokenType, la, ra AtomicValue) (Sequence, error) {
	// Guard: floatArith should only be called with float/double operands.
	// Integer and decimal operands are handled by integerArith/decimalArith.
	if !isFloatOrDouble(la.TypeName) && !isFloatOrDouble(ra.TypeName) {
		return nil, &XPathError{Code: "XPTY0004", Message: fmt.Sprintf("unexpected types in float arithmetic: %s, %s", la.TypeName, ra.TypeName)}
	}
	ln := la.ToFloat64()
	rn := ra.ToFloat64()
	resultType := TypeDouble
	if la.TypeName != TypeDouble && ra.TypeName != TypeDouble {
		resultType = TypeFloat
	}

	var result float64
	switch op {
	case TokenPlus:
		result = ln + rn
	case TokenMinus:
		result = ln - rn
	case TokenStar:
		result = ln * rn
	case TokenDiv:
		result = ln / rn
	case TokenIdiv:
		if math.IsNaN(ln) || math.IsNaN(rn) {
			return nil, &XPathError{Code: "FOAR0002", Message: "idiv with NaN"}
		}
		if math.IsInf(ln, 0) {
			return nil, &XPathError{Code: "FOAR0002", Message: "idiv with infinite dividend"}
		}
		if rn == 0 {
			return nil, &XPathError{Code: "FOAR0002", Message: "integer division by zero"}
		}
		// Per F&O §6.2.4: finite idiv ±INF = 0 (math.Trunc(finite/Inf) = 0).
		truncated := math.Trunc(ln / rn)
		bi, _ := new(big.Float).SetFloat64(truncated).Int(nil)
		return SingleIntegerBig(bi), nil
	case TokenMod:
		result = math.Mod(ln, rn)
	}

	return SingleAtomic(AtomicValue{TypeName: resultType, Value: result}), nil
}

func evalUnaryExpr(ec *evalContext, e UnaryExpr) (Sequence, error) {
	r, err := eval(ec, e.Operand)
	if err != nil {
		return nil, err
	}
	if len(r) == 0 {
		return nil, nil
	}
	if len(r) > 1 {
		return nil, &XPathError{Code: "XPTY0004", Message: "unary minus operand must be a single item"}
	}
	a, err := AtomizeItem(r[0])
	if err != nil {
		return nil, err
	}
	if isIntegerDerived(a.TypeName) {
		return SingleIntegerBig(new(big.Int).Neg(a.BigInt())), nil
	}
	if a.TypeName == TypeDecimal {
		return SingleDecimal(new(big.Rat).Neg(a.BigRat())), nil
	}
	n := a.ToFloat64()
	if a.TypeName == TypeFloat {
		return SingleAtomic(AtomicValue{TypeName: TypeFloat, Value: -n}), nil
	}
	return SingleDouble(-n), nil
}

// needsDecimalArith returns true if arithmetic should use big.Rat (decimal tier).
func needsDecimalArith(lt, rt string) bool {
	lDec := lt == TypeDecimal || isIntegerDerived(lt)
	rDec := rt == TypeDecimal || isIntegerDerived(rt)
	if !lDec || !rDec {
		return false
	}
	// At least one must be decimal (not both integer — that's tier 1)
	return lt == TypeDecimal || rt == TypeDecimal
}

// toRat converts an AtomicValue (integer or decimal) to *big.Rat.
func toRat(a AtomicValue) *big.Rat {
	if isIntegerDerived(a.TypeName) {
		return new(big.Rat).SetInt(a.BigInt())
	}
	return a.BigRat()
}
