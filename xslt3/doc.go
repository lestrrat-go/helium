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
package xslt3
