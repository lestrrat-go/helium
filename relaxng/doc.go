// Package relaxng implements RELAX NG (XML syntax) schema compilation and
// validation.
//
// # Compilation
//
// Use [NewCompiler] to compile a RELAX NG document or file into a [*Grammar]:
//
//	grammar, err := relaxng.NewCompiler().
//	    CompileFile(ctx, "schema.rng")
//
// # Validation
//
// Use [NewValidator] to validate a document against a compiled grammar:
//
//	err := relaxng.NewValidator(grammar).
//	    Label("input.xml").
//	    Validate(ctx, doc)
//
// On failure, the returned error is [ErrValidationFailed]. Individual
// validation errors are delivered to the configured [helium.ErrorHandler].
//
// # Error Handling
//
// Both [Compiler] and [Validator] accept an [helium.ErrorHandler] via the
// ErrorHandler builder method. Individual errors are delivered to the handler
// as they occur.
//
// # Examples
//
// Example code for this package lives in the examples/ directory at the
// repository root (files prefixed with relaxng_). Because examples are in
// a separate test module they do not appear in the generated documentation.
package relaxng
