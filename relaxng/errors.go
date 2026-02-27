package relaxng

import "fmt"

// validityError formats a validation error in libxml2 format:
//
//	{file}:{line}: element {name}: Relax-NG validity error : {msg}\n
func validityError(file string, line int, elemName, msg string) string {
	return fmt.Sprintf("%s:%d: element %s: Relax-NG validity error : %s\n", file, line, elemName, msg)
}

// bareValidityError formats a validation error without file/line/element prefix:
//
//	Relax-NG validity error : {msg}\n
func bareValidityError(msg string) string {
	return fmt.Sprintf("Relax-NG validity error : %s\n", msg)
}

// rngParserError formats a schema compilation error in libxml2 format:
//
//	Relax-NG parser error : {msg}\n
func rngParserError(msg string) string {
	return fmt.Sprintf("Relax-NG parser error : %s\n", msg)
}

// rngParserErrorAt formats a schema compilation error with file/line context:
//
//	{file}:{line}: element {elemName}: Relax-NG parser error : {msg}\n
func rngParserErrorAt(file string, line int, elemName, msg string) string {
	return fmt.Sprintf("%s:%d: element %s: Relax-NG parser error : %s\n", file, line, elemName, msg)
}
