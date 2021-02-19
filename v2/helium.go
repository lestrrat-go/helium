//go:generate go run internal/cmd/gennodes/main.go
//go:generate stringer -type ElementType -output element_type_gen.go

package helium

import (
	"context"

	"github.com/pkg/errors"
)

// Parse parses the given []byte buffer and creates a Document object.
func Parse(ctx context.Context, data []byte, options ...ParseOption) (*Document, error) {
	return parse(ctx, data, options...)
}

func accumulateDecimalCharRef(val int32, c rune) (int32, error) {
	if c >= '0' && c <= '9' {
		val = val*10 + (rune(c) - '0')
	} else {
		return 0, errors.New("invalid decimal CharRef")
	}
	return val, nil
}

func accumulateHexCharRef(val int32, c rune) (int32, error) {
	if c >= '0' && c <= '9' {
		val = val*16 + (rune(c) - '0')
	} else if c >= 'a' && c <= 'f' {
		val = val*16 + (rune(c) - 'a') + 10
	} else if c >= 'A' && c <= 'F' {
		val = val*16 + (rune(c) - 'A') + 10
	} else {
		return 0, errors.New("invalid hex CharRef")
	}
	return val, nil
}
