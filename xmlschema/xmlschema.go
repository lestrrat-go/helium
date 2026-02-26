// Package xmlschema implements XML Schema (XSD) validation.
//
// Phase 1 covers structural validation: content model matching for
// sequence, choice, and all compositors with element particles.
package xmlschema

import (
	"os"
	"path/filepath"

	helium "github.com/lestrrat-go/helium"
)

// Compile compiles an XSD document into a Schema.
func Compile(doc *helium.Document, opts ...CompileOption) (*Schema, error) {
	cfg := &compileConfig{}
	for _, o := range opts {
		o(cfg)
	}
	return compileSchema(doc, "", cfg)
}

// CompileFile reads and compiles an XSD file into a Schema.
func CompileFile(path string, opts ...CompileOption) (*Schema, error) {
	cfg := &compileConfig{}
	for _, o := range opts {
		o(cfg)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	doc, err := helium.Parse(data)
	if err != nil {
		return nil, err
	}
	baseDir := filepath.Dir(path)
	return compileSchema(doc, baseDir, cfg)
}

// Validate validates a document against a compiled schema.
// It returns the validation output string in libxml2-compatible format.
func Validate(doc *helium.Document, schema *Schema, opts ...ValidateOption) string {
	cfg := &validateConfig{}
	for _, o := range opts {
		o(cfg)
	}
	return validateDocument(doc, schema, cfg)
}
