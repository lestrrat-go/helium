// Package xsd implements XML Schema (XSD) 1.0 compilation and validation.
//
// # Compilation
//
// Use [NewCompiler] to compile an XSD document or file into a [*Schema]:
//
//	schema, err := xsd.NewCompiler().
//	    Label("schema.xsd").
//	    CompileFile(ctx, "schema.xsd")
//
// # Validation
//
// Use [NewValidator] to validate a document against a compiled schema:
//
//	err := xsd.NewValidator(schema).
//	    Label("input.xml").
//	    Validate(ctx, doc)
//
// When the document is invalid, the returned error is [ErrValidationFailed].
// Validate also returns [ErrNilSchema] when the Validator has no compiled
// schema and [ErrNilDocument] when the document is nil. A nil ctx is
// normalized to context.Background(). Individual validation errors are
// delivered to the [helium.ErrorHandler] configured via
// [Validator.ErrorHandler].
//
// # Error Handling
//
// Both [Compiler] and [Validator] accept an [helium.ErrorHandler] via the
// ErrorHandler builder method. When set, individual errors are delivered to
// the handler as they occur during compilation or validation.
//
// # Examples
//
// Example code for this package lives in the examples/ directory at the
// repository root (files prefixed with xsd_). Because examples are in
// a separate test module they do not appear in the generated documentation.
package xsd
