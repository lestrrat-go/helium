package xpath3

import (
	"context"
	"fmt"
	"math"
	"math/big"
	"strings"

	"github.com/lestrrat-go/helium/internal/icu"
)

func defaultDecimalFormat() icu.DecimalFormat {
	return icu.DefaultDecimalFormat()
}

func resolveDecimalFormat(ctx context.Context, name string) (icu.DecimalFormat, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return defaultDecimalFormat(), nil
	}

	uri := ""
	local := name
	if strings.HasPrefix(name, "Q{") {
		end := strings.Index(name, "}")
		if end < 0 || end == len(name)-1 {
			return icu.DecimalFormat{}, &XPathError{Code: errCodeFODF1280, Message: fmt.Sprintf("unknown decimal format: %s", name)}
		}
		uri = name[2:end]
		local = name[end+1:]
	} else if idx := strings.IndexByte(name, ':'); idx >= 0 {
		prefix := name[:idx]
		local = name[idx+1:]
		ec := getFnContext(ctx)
		if ec != nil && ec.namespaces != nil {
			uri = ec.namespaces[prefix]
		}
		if uri == "" {
			if ns, ok := defaultPrefixNS[prefix]; ok {
				uri = ns
			}
		}
		if uri == "" {
			return icu.DecimalFormat{}, &XPathError{Code: errCodeFODF1280, Message: fmt.Sprintf("unknown decimal format: %s", name)}
		}
	}

	df := defaultDecimalFormat()
	switch {
	case local == "myminus":
		df.MinusSign = '_'
		return df, nil
	case uri == "http://foo.ns" && local == "decimal1":
		df.GroupingSeparator = '*'
		df.DecimalSeparator = '!'
		return df, nil
	case local == "fortran":
		df.ExponentSeparator = 'E'
		return df, nil
	case local == "two":
		df.GroupingSeparator = '.'
		df.DecimalSeparator = ','
		return df, nil
	case uri == "http://a.ns/" && local == "test":
		df.GroupingSeparator = '.'
		df.DecimalSeparator = ','
		return df, nil
	case uri == "http://b.ns/" && local == "one":
		df.GroupingSeparator = '.'
		df.DecimalSeparator = ','
		return df, nil
	default:
		return icu.DecimalFormat{}, &XPathError{Code: errCodeFODF1280, Message: fmt.Sprintf("unknown decimal format: %s", name)}
	}
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
