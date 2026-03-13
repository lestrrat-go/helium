package xpath3

import (
	"math"
	"math/big"
)

// IEEE 754 mantissa precision constants.
const (
	PrecisionFloat  = 24 // xs:float (IEEE 754 single)
	PrecisionDouble = 53 // xs:double (IEEE 754 double)
)

// floatSpecial represents special IEEE 754 values that big.Float cannot represent.
type floatSpecial byte

const (
	floatNormal floatSpecial = iota
	floatNaN
	floatPosInf
	floatNegInf
	floatNegZero
)

// FloatValue wraps *big.Float with support for NaN, ±Inf, and -0.
// Precision is 24 for xs:float and 53 for xs:double.
type FloatValue struct {
	bf      *big.Float
	special floatSpecial
}

// NewFloat creates an xs:float (24-bit precision) FloatValue from a float64.
// The value is round-tripped through float32 so that values outside the
// float32 exponent range correctly overflow to ±Inf or underflow to ±0.
func NewFloat(f float64) *FloatValue {
	f32 := float32(f)
	return newFloatValue(float64(f32), PrecisionFloat)
}

// NewDouble creates an xs:double (53-bit precision) FloatValue from a float64.
func NewDouble(f float64) *FloatValue {
	return newFloatValue(f, PrecisionDouble)
}

func newFloatValue(f float64, prec uint) *FloatValue {
	if math.IsNaN(f) {
		return &FloatValue{special: floatNaN, bf: new(big.Float).SetPrec(prec)}
	}
	if math.IsInf(f, 1) {
		return &FloatValue{special: floatPosInf, bf: new(big.Float).SetPrec(prec)}
	}
	if math.IsInf(f, -1) {
		return &FloatValue{special: floatNegInf, bf: new(big.Float).SetPrec(prec)}
	}
	if f == 0 && math.Signbit(f) {
		return &FloatValue{special: floatNegZero, bf: new(big.Float).SetPrec(prec)}
	}
	return &FloatValue{bf: new(big.Float).SetPrec(prec).SetFloat64(f)}
}

// Float64 returns the float64 representation.
func (fv *FloatValue) Float64() float64 {
	switch fv.special {
	case floatNormal:
		f, _ := fv.bf.Float64()
		return f
	case floatNaN:
		return math.NaN()
	case floatPosInf:
		return math.Inf(1)
	case floatNegInf:
		return math.Inf(-1)
	case floatNegZero:
		return math.Copysign(0, -1)
	}
	panic("unreachable")
}

// Precision returns the mantissa bit precision.
func (fv *FloatValue) Precision() uint {
	return fv.bf.Prec()
}

// IsNaN returns true if this value is NaN.
func (fv *FloatValue) IsNaN() bool {
	return fv.special == floatNaN
}

// IsInf reports whether the value is infinity.
// sign > 0 tests +Inf, sign < 0 tests -Inf, sign == 0 tests either.
func (fv *FloatValue) IsInf(sign int) bool {
	if sign > 0 {
		return fv.special == floatPosInf
	}
	if sign < 0 {
		return fv.special == floatNegInf
	}
	return fv.special == floatPosInf || fv.special == floatNegInf
}

// IsZero reports whether the value is +0 or -0.
func (fv *FloatValue) IsZero() bool {
	if fv.special == floatNegZero {
		return true
	}
	if fv.special != floatNormal {
		return false
	}
	return fv.bf.Sign() == 0
}

// Signbit reports whether the value is negative (including -0 and -Inf).
func (fv *FloatValue) Signbit() bool {
	switch fv.special {
	case floatNormal:
		return fv.bf.Signbit()
	case floatNegInf, floatNegZero:
		return true
	case floatNaN, floatPosInf:
		return false
	}
	panic("unreachable")
}

// IsSpecial returns true for NaN, ±Inf, or -0.
func (fv *FloatValue) IsSpecial() bool {
	return fv.special != floatNormal
}

// Neg returns the negation of this value with the same precision.
func (fv *FloatValue) Neg() *FloatValue {
	prec := fv.bf.Prec()
	switch fv.special {
	case floatNormal:
		if fv.bf.Sign() == 0 {
			return &FloatValue{special: floatNegZero, bf: new(big.Float).SetPrec(prec)}
		}
		return &FloatValue{bf: new(big.Float).SetPrec(prec).Neg(fv.bf)}
	case floatNaN:
		return &FloatValue{special: floatNaN, bf: new(big.Float).SetPrec(prec)}
	case floatPosInf:
		return &FloatValue{special: floatNegInf, bf: new(big.Float).SetPrec(prec)}
	case floatNegInf:
		return &FloatValue{special: floatPosInf, bf: new(big.Float).SetPrec(prec)}
	case floatNegZero:
		return &FloatValue{bf: new(big.Float).SetPrec(prec)}
	}
	panic("unreachable")
}

// WithPrecision returns a new FloatValue with the given precision.
// For normal values, this re-rounds the underlying big.Float.
func (fv *FloatValue) WithPrecision(prec uint) *FloatValue {
	if fv.special != floatNormal {
		return &FloatValue{special: fv.special, bf: new(big.Float).SetPrec(prec)}
	}
	return &FloatValue{bf: new(big.Float).SetPrec(prec).Set(fv.bf)}
}

// makeFloatResult creates an AtomicValue with the appropriate FloatValue
// for the given type name (TypeFloat or TypeDouble).
func makeFloatResult(typeName string, f float64) AtomicValue {
	if typeName == TypeFloat {
		return AtomicValue{TypeName: TypeFloat, Value: NewFloat(f)}
	}
	return AtomicValue{TypeName: TypeDouble, Value: NewDouble(f)}
}
