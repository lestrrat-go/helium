// Package schematron implements Schematron validation.
//
// It supports a subset of Schematron matching libxml2's implementation:
// schema, pattern, rule, assert, report, let, name, value-of.
package schematron

import (
	"fmt"
	"os"

	helium "github.com/lestrrat-go/helium"
)

// Compile compiles a Schematron document into a Schema.
// (libxml2: xmlSchematronNewParserCtxt + xmlSchematronParse)
func Compile(doc *helium.Document, opts ...CompileOption) (*Schema, error) {
	cfg := &compileConfig{}
	for _, o := range opts {
		o(cfg)
	}
	return compileSchema(doc, cfg)
}

// CompileFile reads and compiles a Schematron file into a Schema.
func CompileFile(path string, opts ...CompileOption) (*Schema, error) {
	cfg := &compileConfig{}
	for _, o := range opts {
		o(cfg)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("schematron: read file: %w", err)
	}
	doc, err := helium.Parse(data)
	if err != nil {
		return nil, fmt.Errorf("schematron: parse document: %w", err)
	}
	return compileSchema(doc, cfg)
}

// Validate validates a document against a compiled schema.
// It returns the validation output string in libxml2-compatible format.
// (libxml2: xmlSchematronValidateDoc)
func Validate(doc *helium.Document, schema *Schema, opts ...ValidateOption) string {
	cfg := &validateConfig{}
	for _, o := range opts {
		o(cfg)
	}
	return validateDocument(doc, schema, cfg)
}
