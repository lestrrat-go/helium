// Package schematron implements Schematron schema compilation and validation.
//
// It supports a subset of Schematron matching libxml2's implementation:
// schema, pattern, rule, assert, report, let, name, value-of.
//
// # Query language bindings
//
// The queryBinding attribute of the <schema> element names the query language
// the schema's expressions are written in. ISO/IEC 19757-3 requires an
// implementation that does not support the named binding to fail with an
// error, so only the values the standard defines and this package implements
// are accepted:
//
//   - an absent attribute or "xslt" selects XPath 1.0 as extended by XSLT 1.0
//     (Annex C, the default binding);
//   - "xslt3" selects the XSLT 3.0 binding (2020 Annex J), whose query
//     language is XPath 3.1;
//   - "xpath3" selects the XPath 3.0 binding (2020 Annex K), which is
//     evaluated with XPath 3.1, a superset of XPath 3.0.
//
// Values are trimmed and compared without regard to case, which the standard
// states for each binding it defines. Every other value fails compilation
// with [ErrUnsupportedQueryBinding], including the XSLT 2.0 and XPath 2.0
// bindings, which the standard defines but this package does not implement,
// and "xpath", "xquery", "exslt" and "stx", which the standard reserves
// without defining.
//
// Use [Compiler.QueryBinding] to force a binding regardless of the attribute,
// and [Compiler.DefaultQueryBinding] to choose the binding for a schema that
// names none.
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
