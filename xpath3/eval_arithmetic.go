package xpath3

import (
	"context"
	"fmt"
	"math"
	"math/big"
	"time"
)

func evalArithmetic(evalFn exprEvaluator, goCtx context.Context, ec *evalContext, e BinaryExpr) (Sequence, error) {
	left, err := evalFn(goCtx, ec, e.Left)
	if err != nil {
		return nil, err
	}
	right, err := evalFn(goCtx, ec, e.Right)
	if err != nil {
		return nil, err
	}
	if seqLen(left) == 0 || seqLen(right) == 0 {
		return nil, nil //nolint:nilnil // empty sequence
	}
	if left.Len() > 1 {
		return nil, &XPathError{Code: errCodeXPTY0004, Message: "arithmetic operand must be a single item"}
	}
	if right.Len() > 1 {
		return nil, &XPathError{Code: errCodeXPTY0004, Message: "arithmetic operand must be a single item"}
	}
	la, err := AtomizeItem(left.Get(0))
	if err != nil {
		return nil, err
	}
	ra, err := AtomizeItem(right.Get(0))
	if err != nil {
		return nil, err
	}

	// Duration/date/time arithmetic — handle before numeric promotion.
	// Contract: evalDateTimeArithmetic returns handled==false only when both
	// operands are non-duration/non-datetime, in which case err is always nil.
	if result, handled, err := evalDateTimeArithmetic(ec, e.Op, la, ra); err != nil || handled {
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

	// Both operands must be numeric after promotion
	if !la.IsNumeric() {
		return nil, &XPathError{Code: errCodeXPTY0004, Message: "arithmetic operand must be numeric, got " + la.TypeName}
	}
	if !ra.IsNumeric() {
		return nil, &XPathError{Code: errCodeXPTY0004, Message: "arithmetic operand must be numeric, got " + ra.TypeName}
	}

	// Promote user-defined schema types to their built-in numeric base.
	la = promoteSchemaNumeric(la)
	ra = promoteSchemaNumeric(ra)

	// Tier 1: both integer → try int64 fast path, fall back to big.Int
	if isIntegerDerived(la.TypeName) && isIntegerDerived(ra.TypeName) {
		lv, lok := la.Int64Val()
		rv, rok := ra.Int64Val()
		if lok && rok {
			return integerArithInt64(e.Op, lv, rv)
		}
		return integerArith(e.Op, la.BigInt(), ra.BigInt())
	}
	// Tier 2: either decimal (or integer promoted) → big.Rat arithmetic
	if needsDecimalArith(la.TypeName, ra.TypeName) {
		return decimalArith(e.Op, toRat(la), toRat(ra))
	}
	// Tier 3: float/double → float64 arithmetic
	return floatArith(e.Op, la, ra)
}

// promoteSchemaNumeric promotes a user-defined numeric type to its built-in base.
func promoteSchemaNumeric(a AtomicValue) AtomicValue {
	return PromoteSchemaType(a)
}

// PromoteSchemaType promotes a user-defined schema type to its built-in base
// type, based on the underlying Go value type. If the type is already a known
// XSD type, the value is returned unchanged.
func PromoteSchemaType(a AtomicValue) AtomicValue {
	if IsKnownXSDType(a.TypeName) {
		return a
	}
	// If the value carries a known built-in base type, use it directly.
	if a.BaseType != "" && IsKnownXSDType(a.BaseType) {
		return AtomicValue{TypeName: a.BaseType, Value: a.Value}
	}
	switch v := a.Value.(type) {
	case int64:
		return AtomicValue{TypeName: TypeInteger, Value: v}
	case *big.Int:
		return AtomicValue{TypeName: TypeInteger, Value: v}
	case *big.Rat:
		return AtomicValue{TypeName: TypeDecimal, Value: v}
	case *FloatValue:
		return AtomicValue{TypeName: TypeDouble, Value: v}
	case float64:
		return AtomicValue{TypeName: TypeDouble, Value: v}
	case bool:
		return AtomicValue{TypeName: TypeBoolean, Value: v}
	case string:
		return AtomicValue{TypeName: TypeString, Value: v}
	case time.Time:
		// Default to xs:dateTime — callers can narrow further
		return AtomicValue{TypeName: TypeDateTime, Value: v}
	case Duration:
		return AtomicValue{TypeName: TypeDuration, Value: v}
	case []byte:
		return AtomicValue{TypeName: TypeBase64Binary, Value: v}
	case QNameValue:
		return AtomicValue{TypeName: TypeQName, Value: v}
	}
	return a
}

// integerArithInt64 performs integer arithmetic using int64 values.
// Falls back to big.Int on overflow.
func integerArithInt64(op TokenType, a, b int64) (Sequence, error) {
	switch op { //nolint:exhaustive
	case TokenPlus:
		r, ok := addInt64(a, b)
		if ok {
			return SingleInteger(r), nil
		}
		return integerArith(op, big.NewInt(a), big.NewInt(b))
	case TokenMinus:
		r, ok := subInt64(a, b)
		if ok {
			return SingleInteger(r), nil
		}
		return integerArith(op, big.NewInt(a), big.NewInt(b))
	case TokenStar:
		r, ok := mulInt64(a, b)
		if ok {
			return SingleInteger(r), nil
		}
		return integerArith(op, big.NewInt(a), big.NewInt(b))
	case TokenDiv:
		if b == 0 {
			return nil, &XPathError{Code: errCodeFOAR0001, Message: "division by zero"}
		}
		r := new(big.Rat).SetFrac64(a, b)
		return SingleDecimal(r), nil
	case TokenIdiv:
		if b == 0 {
			return nil, &XPathError{Code: errCodeFOAR0001, Message: "integer division by zero"}
		}
		if a == math.MinInt64 && b == -1 {
			return integerArith(op, big.NewInt(a), big.NewInt(b))
		}
		return SingleInteger(a / b), nil
	case TokenMod:
		if b == 0 {
			return nil, &XPathError{Code: errCodeFOAR0001, Message: "modulo by zero"}
		}
		return SingleInteger(a % b), nil
	default:
		return nil, &XPathError{Code: errCodeXPTY0004, Message: "unsupported integer arithmetic operator"}
	}
}

// addInt64 returns a+b and true if no overflow, or (0, false) on overflow.
func addInt64(a, b int64) (int64, bool) {
	r := a + b
	if (b > 0 && r < a) || (b < 0 && r > a) {
		return 0, false
	}
	return r, true
}

// subInt64 returns a-b and true if no overflow, or (0, false) on overflow.
func subInt64(a, b int64) (int64, bool) {
	r := a - b
	if (b > 0 && r > a) || (b < 0 && r < a) {
		return 0, false
	}
	return r, true
}

// mulInt64 returns a*b and true if no overflow, or (0, false) on overflow.
func mulInt64(a, b int64) (int64, bool) {
	if a == 0 || b == 0 {
		return 0, true
	}
	r := a * b
	if r/a != b {
		return 0, false
	}
	return r, true
}

func integerArith(op TokenType, a, b *big.Int) (Sequence, error) {
	result := new(big.Int)
	switch op { //nolint:exhaustive // only arithmetic operators are valid here; default handles the rest
	case TokenPlus:
		result.Add(a, b)
	case TokenMinus:
		result.Sub(a, b)
	case TokenStar:
		result.Mul(a, b)
	case TokenDiv:
		// integer / integer → decimal
		if b.Sign() == 0 {
			return nil, &XPathError{Code: errCodeFOAR0001, Message: "division by zero"}
		}
		r := new(big.Rat).SetFrac(new(big.Int).Set(a), new(big.Int).Set(b))
		return SingleDecimal(r), nil
	case TokenIdiv:
		if b.Sign() == 0 {
			return nil, &XPathError{Code: errCodeFOAR0001, Message: "integer division by zero"}
		}
		result.Quo(a, b) // truncates toward zero
	case TokenMod:
		if b.Sign() == 0 {
			return nil, &XPathError{Code: errCodeFOAR0001, Message: "modulo by zero"}
		}
		result.Rem(a, b)
	default:
		return nil, &XPathError{Code: errCodeXPTY0004, Message: "unsupported integer arithmetic operator"}
	}
	return SingleIntegerBig(result), nil
}

func decimalArith(op TokenType, a, b *big.Rat) (Sequence, error) {
	result := new(big.Rat)
	switch op { //nolint:exhaustive // only arithmetic operators are valid here; default handles the rest
	case TokenPlus:
		result.Add(a, b)
	case TokenMinus:
		result.Sub(a, b)
	case TokenStar:
		result.Mul(a, b)
	case TokenDiv:
		if b.Sign() == 0 {
			return nil, &XPathError{Code: errCodeFOAR0001, Message: "division by zero"}
		}
		result.Quo(a, b)
	case TokenIdiv:
		if b.Sign() == 0 {
			return nil, &XPathError{Code: errCodeFOAR0001, Message: "integer division by zero"}
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
			return nil, &XPathError{Code: errCodeFOAR0001, Message: "modulo by zero"}
		}
		// a mod b = a - (a idiv b) * b
		q := new(big.Rat).Quo(a, b)
		intPart := new(big.Int).Quo(q.Num(), q.Denom())
		qr := new(big.Rat).SetInt(intPart)
		result.Sub(a, new(big.Rat).Mul(qr, b))
	default:
		return nil, &XPathError{Code: errCodeXPTY0004, Message: "unsupported decimal arithmetic operator"}
	}
	return SingleDecimal(result), nil
}

func floatArith(op TokenType, la, ra AtomicValue) (Sequence, error) {
	// Guard: floatArith should only be called with float/double operands.
	// Integer and decimal operands are handled by integerArith/decimalArith.
	if !isFloatOrDouble(la.TypeName) && !isFloatOrDouble(ra.TypeName) {
		return nil, &XPathError{Code: errCodeXPTY0004, Message: fmt.Sprintf("unexpected types in float arithmetic: %s, %s", la.TypeName, ra.TypeName)}
	}
	ln := la.ToFloat64()
	rn := ra.ToFloat64()
	resultType := TypeDouble
	if la.TypeName != TypeDouble && ra.TypeName != TypeDouble {
		resultType = TypeFloat
	}

	var result float64
	switch op { //nolint:exhaustive // only arithmetic operators are valid here; default handles the rest
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
			return nil, &XPathError{Code: errCodeFOAR0002, Message: "idiv with NaN"}
		}
		if math.IsInf(ln, 0) {
			return nil, &XPathError{Code: errCodeFOAR0002, Message: "idiv with infinite dividend"}
		}
		if rn == 0 {
			return nil, &XPathError{Code: errCodeFOAR0001, Message: "integer division by zero"}
		}
		// Per F&O §6.2.4: finite idiv ±INF = 0 (math.Trunc(finite/Inf) = 0).
		truncated := math.Trunc(ln / rn)
		bi, _ := new(big.Float).SetFloat64(truncated).Int(nil)
		return SingleIntegerBig(bi), nil
	case TokenMod:
		result = math.Mod(ln, rn)
	default:
		return nil, &XPathError{Code: errCodeXPTY0004, Message: "unsupported float arithmetic operator"}
	}

	if resultType == TypeFloat {
		return SingleAtomic(AtomicValue{TypeName: TypeFloat, Value: NewFloat(result)}), nil
	}
	return SingleAtomic(AtomicValue{TypeName: TypeDouble, Value: NewDouble(result)}), nil
}

func evalUnaryExpr(evalFn exprEvaluator, goCtx context.Context, ec *evalContext, e UnaryExpr) (Sequence, error) {
	r, err := evalFn(goCtx, ec, e.Operand)
	if err != nil {
		return nil, err
	}
	if seqLen(r) == 0 {
		return nil, nil //nolint:nilnil
	}
	if r.Len() > 1 {
		return nil, &XPathError{Code: errCodeXPTY0004, Message: "unary minus operand must be a single item"}
	}
	a, err := AtomizeItem(r.Get(0))
	if err != nil {
		return nil, err
	}
	// Promote xs:untypedAtomic to xs:double per XPath 3.1 spec
	if a.TypeName == TypeUntypedAtomic {
		castVal, err := CastAtomic(a, TypeDouble)
		if err != nil {
			return nil, err
		}
		a = castVal
	}
	if !e.Negate {
		// Unary plus: type check only, return value unchanged
		if !isSubtypeOf(a.TypeName, TypeDecimal) && a.TypeName != TypeFloat && a.TypeName != TypeDouble {
			return nil, &XPathError{Code: errCodeXPTY0004, Message: "unary operator requires numeric type, got " + a.TypeName}
		}
		return SingleAtomic(a), nil
	}
	if isIntegerDerived(a.TypeName) {
		if v, ok := a.Value.(int64); ok {
			if v != math.MinInt64 {
				return SingleInteger(-v), nil
			}
			// MinInt64 overflows on negation; fall through to big.Int
		}
		return SingleIntegerBig(new(big.Int).Neg(a.BigInt())), nil
	}
	if a.TypeName == TypeDecimal {
		return SingleDecimal(new(big.Rat).Neg(a.BigRat())), nil
	}
	if a.TypeName == TypeFloat {
		return SingleAtomic(AtomicValue{TypeName: TypeFloat, Value: a.FloatVal().Neg()}), nil
	}
	if a.TypeName == TypeDouble {
		return SingleAtomic(AtomicValue{TypeName: TypeDouble, Value: a.FloatVal().Neg()}), nil
	}
	return nil, &XPathError{Code: errCodeXPTY0004, Message: "unary operator requires numeric type, got " + a.TypeName}
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
		if v, ok := a.Value.(int64); ok {
			return new(big.Rat).SetInt64(v)
		}
		return new(big.Rat).SetInt(a.BigInt())
	}
	return a.BigRat()
}
