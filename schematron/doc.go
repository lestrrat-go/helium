// Package schematron implements Schematron schema compilation and validation.
//
// It supports a subset of Schematron matching libxml2's implementation:
// schema, pattern, rule, assert, report, let, name, value-of.
//
// # Compilation
//
// Use [NewCompiler] to compile a Schematron document or file into a [*Schema]:
//
//	schema, err := schematron.NewCompiler().
//	    CompileFile(ctx, "rules.sch")
//
// # Validation
//
// Use [NewValidator] to validate a document against a compiled schema:
//
//	err := schematron.NewValidator(schema).
//	    Filename("input.xml").
//	    Validate(ctx, doc)
//
// On failure, the returned error is a [*ValidateError]. When an
// [helium.ErrorHandler] is set via [Validator.ErrorHandler], individual
// errors are delivered as [*ValidationError] values with structured fields
// (Filename, Line, Element, Path, Message).
package schematron
