package schematron

import "fmt"

// schematronError formats a validation error in libxml2 format:
//
//	{file}:{line}: element {elemName}: schematron error : {nodePath} line {line}: {message}\n
func schematronError(file string, line int, elemName, nodePath, message string) string {
	return fmt.Sprintf("%s:%d: element %s: schematron error : %s line %d: %s\n", file, line, elemName, nodePath, line, message)
}
