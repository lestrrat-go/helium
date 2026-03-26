// Package xsd implements XML Schema (XSD) 1.0 compilation and validation.
//
// # Compilation
//
// Use [NewCompiler] to compile an XSD document or file into a [*Schema]:
//
//	schema, err := xsd.NewCompiler().
//	    SchemaFilename("schema.xsd").
//	    CompileFile(ctx, "schema.xsd")
//
// # Validation
//
// Use [NewValidator] to validate a document against a compiled schema:
//
//	err := xsd.NewValidator(schema).
//	    Filename("input.xml").
//	    Validate(ctx, doc)
//
// On failure, the returned error is [ErrValidationFailed]. Individual
// validation errors are delivered to the [helium.ErrorHandler] configured
// via [Validator.ErrorHandler].
//
// # Error Handling
//
// Both [Compiler] and [Validator] accept an [helium.ErrorHandler] via the
// ErrorHandler builder method. When set, individual errors are delivered to
// the handler as they occur during compilation or validation.
package xsd
