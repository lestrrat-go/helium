// Package relaxng implements RELAX NG (XML syntax) schema validation.
package relaxng

import (
	"errors"
	"os"
	"path/filepath"

	helium "github.com/lestrrat-go/helium"
)

// Compile compiles a RELAX NG document into a Grammar.
// (libxml2: xmlRelaxNGNewParserCtxt + xmlRelaxNGParse)
func Compile(doc *helium.Document, opts ...CompileOption) (*Grammar, error) {
	cfg := &compileConfig{}
	for _, o := range opts {
		o(cfg)
	}
	return compileSchema(doc, "", cfg)
}

// CompileFile reads and compiles a RELAX NG file into a Grammar.
func CompileFile(path string, opts ...CompileOption) (*Grammar, error) {
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
		var pe helium.ErrParseError
		if errors.As(err, &pe) {
			filename := cfg.filename
			if filename == "" {
				filename = path
			}
			errs := formatXMLParseError(filename, pe)
			errs += rngParserError("xmlRelaxNGParse: could not load " + filename)
			return &Grammar{compileErrors: errs}, nil
		}
		return nil, err
	}
	doc.SetURL(path)
	baseDir := filepath.Dir(path)
	return compileSchema(doc, baseDir, cfg)
}

// Validate validates a document against a compiled grammar.
// It returns the validation output string in libxml2-compatible format.
// (libxml2: xmlRelaxNGValidateDoc)
func Validate(doc *helium.Document, grammar *Grammar, opts ...ValidateOption) string {
	cfg := &validateConfig{}
	for _, o := range opts {
		o(cfg)
	}
	return validateDocument(doc, grammar, cfg)
}
