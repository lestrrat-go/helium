package examples_test

import (
	"context"
	"errors"
	"fmt"

	"github.com/lestrrat-go/helium"
)

func Example_helium_parse_error_sentinel() {
	// ErrParseError wraps a sentinel error identifying the specific
	// parse failure. Use errors.Is to match against known sentinels
	// through Unwrap.

	_, err := helium.NewParser().Parse(context.Background(), []byte(`<root>`))

	fmt.Println("is ErrLtSlashRequired:", errors.Is(err, helium.ErrLtSlashRequired))
	// Output:
	// is ErrLtSlashRequired: true
}
