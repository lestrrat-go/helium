package xpath3

import (
	"fmt"
	"math"
	"math/big"
)

func castToDouble(v AtomicValue) (AtomicValue, error) {
	switch v.TypeName {
	case TypeDouble:
		return v, nil
	case TypeFloat:
		return AtomicValue{TypeName: TypeDouble, Value: v.DoubleVal()}, nil
	case TypeInteger:
		f, _ := new(big.Float).SetInt(v.BigInt()).Float64()
		return AtomicValue{TypeName: TypeDouble, Value: f}, nil
	case TypeDecimal:
		f, _ := v.BigRat().Float64()
		return AtomicValue{TypeName: TypeDouble, Value: f}, nil
	case TypeBoolean:
		if v.BooleanVal() {
			return AtomicValue{TypeName: TypeDouble, Value: float64(1)}, nil
		}
		return AtomicValue{TypeName: TypeDouble, Value: float64(0)}, nil
	case TypeString, TypeUntypedAtomic:
		return CastFromString(v.StringVal(), TypeDouble)
	}
	return AtomicValue{}, &XPathError{Code: "XPTY0004", Message: fmt.Sprintf("cannot cast %s to xs:double", v.TypeName)}
}

func castToFloat(v AtomicValue) (AtomicValue, error) {
	result, err := castToDouble(v)
	if err != nil {
		return AtomicValue{}, err
	}
	// Narrow to single-precision semantics before changing the type
	result.Value = float64(float32(result.DoubleVal()))
	result.TypeName = TypeFloat
	return result, nil
}

func castToInteger(v AtomicValue) (AtomicValue, error) {
	switch v.TypeName {
	case TypeInteger:
		return v, nil
	case TypeDouble, TypeFloat:
		f := v.DoubleVal()
		if math.IsNaN(f) || math.IsInf(f, 0) {
			return AtomicValue{}, &XPathError{Code: "FOCA0002", Message: "cannot cast NaN/INF to xs:integer"}
		}
		f = math.Trunc(f)
		bi, _ := new(big.Float).SetFloat64(f).Int(nil)
		return AtomicValue{TypeName: TypeInteger, Value: bi}, nil
	case TypeDecimal:
		// Truncate rational toward zero
		r := v.BigRat()
		q := new(big.Int).Quo(r.Num(), r.Denom())
		return AtomicValue{TypeName: TypeInteger, Value: q}, nil
	case TypeBoolean:
		if v.BooleanVal() {
			return AtomicValue{TypeName: TypeInteger, Value: big.NewInt(1)}, nil
		}
		return AtomicValue{TypeName: TypeInteger, Value: big.NewInt(0)}, nil
	case TypeString, TypeUntypedAtomic:
		return CastFromString(v.StringVal(), TypeInteger)
	}
	return AtomicValue{}, &XPathError{Code: "XPTY0004", Message: fmt.Sprintf("cannot cast %s to xs:integer", v.TypeName)}
}

func castToDecimal(v AtomicValue) (AtomicValue, error) {
	switch v.TypeName {
	case TypeInteger:
		r := new(big.Rat).SetInt(v.BigInt())
		return AtomicValue{TypeName: TypeDecimal, Value: r}, nil
	case TypeDouble, TypeFloat:
		f := v.DoubleVal()
		if math.IsNaN(f) || math.IsInf(f, 0) {
			return AtomicValue{}, &XPathError{Code: "FOCA0002", Message: "cannot cast NaN/INF to xs:decimal"}
		}
		r := new(big.Rat).SetFloat64(f)
		return AtomicValue{TypeName: TypeDecimal, Value: r}, nil
	case TypeBoolean:
		if v.BooleanVal() {
			return AtomicValue{TypeName: TypeDecimal, Value: big.NewRat(1, 1)}, nil
		}
		return AtomicValue{TypeName: TypeDecimal, Value: big.NewRat(0, 1)}, nil
	case TypeString, TypeUntypedAtomic:
		return CastFromString(v.StringVal(), TypeDecimal)
	}
	return AtomicValue{}, &XPathError{Code: "XPTY0004", Message: fmt.Sprintf("cannot cast %s to xs:decimal", v.TypeName)}
}

func castToBoolean(v AtomicValue) (AtomicValue, error) {
	switch v.TypeName {
	case TypeBoolean:
		return v, nil
	case TypeInteger:
		return AtomicValue{TypeName: TypeBoolean, Value: v.BigInt().Sign() != 0}, nil
	case TypeDouble, TypeFloat:
		f := v.DoubleVal()
		return AtomicValue{TypeName: TypeBoolean, Value: f != 0 && !math.IsNaN(f)}, nil
	case TypeDecimal:
		return AtomicValue{TypeName: TypeBoolean, Value: v.BigRat().Sign() != 0}, nil
	case TypeString, TypeUntypedAtomic:
		return CastFromString(v.StringVal(), TypeBoolean)
	}
	return AtomicValue{}, &XPathError{Code: "XPTY0004", Message: fmt.Sprintf("cannot cast %s to xs:boolean", v.TypeName)}
}

func castToBase64Binary(v AtomicValue) (AtomicValue, error) {
	switch v.TypeName {
	case TypeHexBinary:
		return AtomicValue{TypeName: TypeBase64Binary, Value: v.BytesVal()}, nil
	case TypeString, TypeUntypedAtomic:
		return CastFromString(v.StringVal(), TypeBase64Binary)
	}
	return AtomicValue{}, &XPathError{Code: "XPTY0004", Message: fmt.Sprintf("cannot cast %s to xs:base64Binary", v.TypeName)}
}

func castToHexBinary(v AtomicValue) (AtomicValue, error) {
	switch v.TypeName {
	case TypeBase64Binary:
		return AtomicValue{TypeName: TypeHexBinary, Value: v.BytesVal()}, nil
	case TypeString, TypeUntypedAtomic:
		return CastFromString(v.StringVal(), TypeHexBinary)
	}
	return AtomicValue{}, &XPathError{Code: "XPTY0004", Message: fmt.Sprintf("cannot cast %s to xs:hexBinary", v.TypeName)}
}
