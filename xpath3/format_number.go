package xpath3

import (
	"fmt"
	"math"
	"math/big"

	"github.com/lestrrat-go/helium/internal/icu"
)

func defaultDecimalFormat() icu.DecimalFormat {
	return icu.DefaultDecimalFormat()
}

func formatNumber(a AtomicValue, picture string, df icu.DecimalFormat) (string, error) {
	f := a.ToFloat64()

	isNaN := math.IsNaN(f)
	isPosInf := math.IsInf(f, 1)
	isNegInf := math.IsInf(f, -1)
	negative := f < 0 || (f == 0 && math.Signbit(f))

	var precise *big.Rat
	if isIntegerDerived(a.TypeName) {
		precise = new(big.Rat).SetInt(a.BigInt())
	} else if a.TypeName == TypeDecimal {
		precise = new(big.Rat).Set(a.BigRat())
	}

	if negative {
		f = math.Abs(f)
	}

	result, err := icu.FormatNumber(f, isNaN, isPosInf, isNegInf, negative, precise, picture, df)
	if err != nil {
		return "", &XPathError{Code: errCodeFODF1310, Message: fmt.Sprintf("invalid picture: %q", picture)}
	}
	return result, nil
}
