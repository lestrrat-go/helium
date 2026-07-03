// Package xslt3 implements an XSLT 3.0 processor targeting Basic XSLT 3.0
// conformance (W3C spec Section 27).
//
// The processing pipeline has two phases: compilation and transformation.
//
// # Compilation
//
// Use [NewCompiler] to build a [Compiler], configure it with base URI,
// URI resolvers, static parameters, and imported schemas, then call
// [Compiler.Compile] to produce a [*Stylesheet]:
//
//	ss, err := xslt3.NewCompiler().
//	    BaseURI(baseDir).
//	    Compile(ctx, stylesheetDoc)
//
// For simple cases, [CompileStylesheet] is a convenience wrapper.
//
// # Transformation
//
// A compiled [Stylesheet] offers four entry points, each returning an
// [Invocation] that can be configured with parameters, handlers, and modes
// via fluent builder methods:
//
//   - [Stylesheet.Transform] — apply templates to a source document
//   - [Stylesheet.ApplyTemplates] — explicit mode and selection
//   - [Stylesheet.CallTemplate] — invoke a named template
//   - [Stylesheet.CallFunction] — invoke a named function
//
// Terminal methods on [Invocation] execute the transformation:
//
//	result, err := ss.Transform(sourceDoc).
//	    SetParameter("param", value).
//	    Do(ctx)
//
// For simple transforms, [Transform], [TransformString], and
// [TransformToWriter] are convenience wrappers.
//
// # Concurrency
//
// A [*Stylesheet] returned by [Compiler.Compile] / [CompileStylesheet] is
// immutable after compilation and is safe for concurrent use by multiple
// goroutines. A single compiled stylesheet may be transformed from many
// goroutines at once (via [Transform], [TransformString], [TransformToWriter],
// [Stylesheet.Transform], [Stylesheet.ApplyTemplates], [Stylesheet.CallTemplate],
// or [Stylesheet.CallFunction]) without external synchronization. Every mutable
// per-transform state (global-variable values, key tables, caches, accumulator
// state, the result tree, and serialization overrides) lives in a per-call
// execution context that is never shared between transforms, and a transform
// never mutates the compiled stylesheet.
//
// The caller's source document is treated as READ-ONLY in ALL cases —
// schema-aware and non-schema-aware alike. Whitespace stripping (xsl:strip-space
// and the schema-aware whitespace rules) runs against a private copy, and
// source-schema validation (which inserts default/fixed attributes into the tree
// it validates) also runs against a private copy, so neither mutates the caller's
// tree. A single source document may therefore be shared, read-only, across
// concurrent transforms — schema-aware or not — as long as the caller does not
// mutate it concurrently. A schema-aware transform is one where the stylesheet
// has an xsl:import-schema, [Invocation.SourceSchemas] supplies schemas, or the
// source carries an xsi:schemaLocation.
//
// Compilation itself is not concurrent with transformation of the same
// stylesheet: finish compiling before sharing the result. [Invocation] and
// [Parameters] values are per-call configuration and are not intended for
// concurrent mutation.
//
// # Examples
//
// Example code for this package lives in the examples/ directory at the
// repository root (files prefixed with xslt3_). Because examples are in
// a separate test module they do not appear in the generated documentation.
package xslt3
