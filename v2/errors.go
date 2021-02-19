package helium

import "errors"

func errInvalidOperation() error {
	return errors.New("operation cannot be performed")
}
