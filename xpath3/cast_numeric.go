package xpath3

import (
	"encoding/base64"
	"fmt"
	"math"
	"math/big"
	"strconv"
	"strings"
)

func castToDouble(v AtomicValue) (AtomicValue, error) {
	switch v.TypeName {
	case TypeDouble:
		return v, nil
	case TypeFloat:
		// Promote float to double precision (preserving the float32-precision value)
		return AtomicValue{TypeName: TypeDouble, Value: NewDouble(v.DoubleVal())}, nil
	case TypeInteger:
		f, _ := new(big.Float).SetInt(v.BigInt()).Float64()
		return AtomicValue{TypeName: TypeDouble, Value: NewDouble(f)}, nil
	case TypeDecimal:
		f, _ := v.BigRat().Float64()
		return AtomicValue{TypeName: TypeDouble, Value: NewDouble(f)}, nil
	case TypeBoolean:
		if v.BooleanVal() {
			return AtomicValue{TypeName: TypeDouble, Value: NewDouble(1)}, nil
		}
		return AtomicValue{TypeName: TypeDouble, Value: NewDouble(0)}, nil
	case TypeString, TypeUntypedAtomic:
		return CastFromString(v.StringVal(), TypeDouble)
	}
	return AtomicValue{}, &XPathError{Code: "XPTY0004", Message: fmt.Sprintf("cannot cast %s to xs:double", v.TypeName)}
}

func castToFloat(v AtomicValue) (AtomicValue, error) {
	// When casting from string, use the xs:float lexical rules directly
	// (XSD 1.0 accepts "INF", "-INF", "NaN" but NOT "+INF")
	if v.TypeName == TypeString || v.TypeName == TypeUntypedAtomic {
		return castStringToFloat(strings.TrimSpace(v.StringVal()))
	}
	// For non-string sources, promote through double then narrow
	dbl, err := castToDouble(v)
	if err != nil {
		return AtomicValue{}, err
	}
	f := dbl.DoubleVal()
	return AtomicValue{TypeName: TypeFloat, Value: NewFloat(f)}, nil
}

// castStringToFloat parses s as an xs:float using XSD 1.0 lexical rules.
// Valid special values: "INF", "-INF", "NaN". All other values must be
// numeric literals that fit in float32 range.
func castStringToFloat(s string) (AtomicValue, error) {
	switch s {
	case "INF":
		return AtomicValue{TypeName: TypeFloat, Value: NewFloat(math.Inf(1))}, nil
	case "-INF":
		return AtomicValue{TypeName: TypeFloat, Value: NewFloat(math.Inf(-1))}, nil
	case "NaN":
		return AtomicValue{TypeName: TypeFloat, Value: NewFloat(math.NaN())}, nil
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return AtomicValue{}, castError(s, TypeFloat)
	}
	// Reject infinity from ParseFloat — only whitelisted "INF"/"-INF" above are valid
	if math.IsInf(f, 0) {
		return AtomicValue{}, castError(s, TypeFloat)
	}
	// Reject finite values that overflow float32 range
	f32 := float32(f)
	if math.IsInf(float64(f32), 0) {
		return AtomicValue{}, castError(s, TypeFloat)
	}
	return AtomicValue{TypeName: TypeFloat, Value: NewFloat(f)}, nil
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

// xsdBase64Encoding is base64.StdEncoding with Strict() to reject non-zero
// trailing bits before padding (per XSD base64Binary lexical rules).
var xsdBase64Encoding = base64.StdEncoding.Strict()

// decodeXSDBase64 decodes a base64 string per XSD lexical rules:
// - whitespace (space, tab, CR, LF) is stripped before decoding
// - trailing non-zero bits before padding are rejected (strict mode)
func decodeXSDBase64(s string) ([]byte, error) {
	// Strip whitespace per XSD base64Binary lexical space (collapse)
	var cleaned strings.Builder
	for _, c := range s {
		if c != ' ' && c != '\t' && c != '\n' && c != '\r' {
			cleaned.WriteRune(c)
		}
	}
	return xsdBase64Encoding.DecodeString(cleaned.String())
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
