package xsd

import (
	"fmt"

	"github.com/lestrrat-go/helium/internal/lexicon"
)

// validityError formats a validation error in libxml2 format:
//
//	./test/schemas/{file}.xml:{line}: Schemas validity error : Element '{name}': {msg}\n
func validityError(file string, line int, elemName, msg string) string {
	return fmt.Sprintf("%s:%d: Schemas validity error : Element '%s': %s\n", file, line, elemName, msg)
}

// validityErrorAttr formats a validation error for an attribute in libxml2 format:
//
//	./test/schemas/{file}.xml:{line}: Schemas validity error : Element '{elem}', attribute '{attr}': {msg}\n
func validityErrorAttr(file string, line int, elemName, attrName, msg string) string {
	return fmt.Sprintf("%s:%d: Schemas validity error : Element '%s', attribute '%s': %s\n", file, line, elemName, attrName, msg)
}

// schemaParserError formats a schema compilation error in libxml2 format:
//
//	{file}:{line}: element {elemLocal}: Schemas parser error : Element '{xsdNS}{xsdElem}': {msg}\n
func schemaParserError(file string, line int, elemLocal, xsdElem, msg string) string {
	return fmt.Sprintf("%s:%d: element %s: Schemas parser error : Element '{%s}%s': %s\n", file, line, elemLocal, lexicon.NamespaceXSD, xsdElem, msg)
}

// schemaParserErrorAttr formats a schema compilation error for a specific attribute:
//
//	{file}:{line}: element {elemLocal}: Schemas parser error : Element '{xsdNS}{xsdElem}', attribute '{attr}': {msg}\n
func schemaParserErrorAttr(file string, line int, elemLocal, xsdElem, attr, msg string) string {
	return fmt.Sprintf("%s:%d: element %s: Schemas parser error : Element '{%s}%s', attribute '%s': %s\n", file, line, elemLocal, lexicon.NamespaceXSD, xsdElem, attr, msg)
}

// schemaParserWarning formats a schema compilation warning in libxml2 format:
//
//	{file}:{line}: element {elemLocal}: Schemas parser warning : Element '{xsdNS}{xsdElem}': {msg}\n
func schemaParserWarning(file string, line int, elemLocal, xsdElem, msg string) string {
	return fmt.Sprintf("%s:%d: element %s: Schemas parser warning : Element '{%s}%s': %s\n", file, line, elemLocal, lexicon.NamespaceXSD, xsdElem, msg)
}

// schemaComponentError formats a schema compilation error for a component (e.g. "local complex type"):
//
//	{file}:{line}: element {elemLocal}: Schemas parser error : {component}: {msg}\n
func schemaComponentError(file string, line int, elemLocal, component, msg string) string {
	return fmt.Sprintf("%s:%d: element %s: Schemas parser error : %s: %s\n", file, line, elemLocal, component, msg)
}

// schemaElemDeclError formats a schema compilation error for an element declaration:
//
//	{file}:{line}: element element: Schemas parser error : element decl. '{name}': {msg}\n
func schemaElemDeclError(file string, line int, declName, msg string) string {
	return fmt.Sprintf("%s:%d: element element: Schemas parser error : element decl. '%s': %s\n", file, line, declName, msg)
}

// schemaElemDeclErrorAttr formats a schema compilation error for an element declaration attribute:
//
//	{file}:{line}: element element: Schemas parser error : element decl. '{name}', attribute '{attr}': {msg}\n
func schemaElemDeclErrorAttr(file string, line int, declName, attr, msg string) string {
	return fmt.Sprintf("%s:%d: element element: Schemas parser error : element decl. '%s', attribute '%s': %s\n", file, line, declName, attr, msg)
}
