// Package xsd implements XML Schema (XSD) compilation and validation.
//
// # Version
//
// The compiler targets XSD 1.0 by default. XSD 1.1 is opt-in via
// [Compiler.Version]:
//
//	schema, err := xsd.NewCompiler().
//	    Version(xsd.Version11).
//	    Compile(ctx, doc)
//
// When the version is not set explicitly, a vc:minVersion="1.1" attribute on the
// root <xs:schema> auto-selects 1.1. In 1.1 mode the 1.1-only lexical forms (e.g.
// "+INF" for xs:double/xs:float, year "0000" on the date types) and the 1.1
// built-in datatypes (xs:dateTimeStamp, xs:dayTimeDuration, xs:yearMonthDuration,
// xs:anyAtomicType, xs:error) are recognized. XSD 1.1 support includes
// assertions and assertion facets, conditional type assignment, open content
// (including schema-level xs:defaultOpenContent), schema default attributes,
// broader xs:all support, xs:override, temporal datatype facets such as
// explicitTimezone, relaxed wildcard/UPA behavior, document-wide
// xs:ID/xs:IDREF/xs:ENTITY value-space validation, identity-constraint scoping,
// and compile-time validation of element default/fixed constraints.
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
