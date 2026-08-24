// Package schematron implements Schematron schema compilation and validation.
//
// It supports a subset of Schematron matching libxml2's implementation:
// schema, pattern, rule, assert, report, let, name, value-of.
//
// # Query language bindings
//
// The queryBinding attribute of the <schema> element names the query language
// the schema's expressions are written in. Two families are implemented:
// XPath 1.0 ("xslt", "xslt1", "xpath", "xpath1", and the absent-attribute
// default) and XPath 3.1 ("xslt3", "xpath3", "xpath31"). Any other value,
// including the XSLT 2.0 and XPath 2.0 bindings, fails compilation with
// [ErrUnsupportedQueryBinding]. Use [Compiler.QueryBinding] to force a
// binding regardless of the attribute, and [Compiler.DefaultQueryBinding] to
// choose the binding for a schema that names none.
//
// Under the XPath 3.1 binding a schema cannot reach the filesystem or the
// network: fn:doc, fn:collection and fn:unparsed-text need a URI resolver or
// an HTTP client, and the evaluator is given neither.
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
