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
// [Compiler.Compile] and [Compiler.CompileFile] return [ErrCompileFailed]
// (with a nil schema) on fatal schema errors.
//
// # Validation
//
// Use [NewValidator] to validate a document against a compiled schema:
//
//	err := schematron.NewValidator(schema).
//	    Label("input.xml").
//	    Validate(ctx, doc)
//
// [Validator.Validate] returns [ErrNoSchema] when the Validator has no (or a
// zero) compiled schema, and [ErrValidationFailed] when assertions fail.
// Individual errors are delivered as [*ValidationError] values to the
// configured [helium.ErrorHandler] (structured fields: Filename, Line,
// Element, Path, Message).
//
// # Examples
//
// Example code for this package lives in the examples/ directory at the
// repository root (files prefixed with schematron_). Because examples are
// in a separate test module they do not appear in the generated
// documentation.
package schematron
