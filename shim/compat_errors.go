package shim

import (
	stdxml "encoding/xml"
	"errors"

	helium "github.com/lestrrat-go/helium"
)

// convertParseError converts a helium parse error into an encoding/xml SyntaxError
// so that callers checking for *xml.SyntaxError get the expected type.
func convertParseError(err error) error {
	if err == nil {
		return nil
	}

	var pe helium.ErrParseError
	if errors.As(err, &pe) {
		return &stdxml.SyntaxError{
			Line: pe.LineNumber,
			Msg:  pe.Err.Error(),
		}
	}

	return err
}
