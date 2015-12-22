package helium

import "fmt"

func (e ErrParseError) Error() string {
	return fmt.Sprintf(
		"%s at line %d, column %d\n -> '%s' <-- around here",
		e.Err,
		e.LineNumber,
		e.Column,
		e.Line,
	)
}

func (e ErrUnimplemented) Error() string {
	return "unimplemented method: '" + e.target + "'"
}

func (e ErrDTDDupToken) Error() string {
	return "standlone: attribute enumeration value token " + e.Name + " duplicated"
}
