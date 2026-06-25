// Package xinclude implements XML Inclusion (XInclude) 1.0 processing.
//
// Use [NewProcessor] to create a processor, configure it with fluent builder
// methods, and call [Processor.Process] to resolve xi:include elements:
//
//	n, err := xinclude.NewProcessor().
//	    NoBaseFixup().
//	    Process(ctx, doc)
//
// The returned count indicates how many inclusions were performed.
//
// # Security
//
// Included documents are parsed with their own inner parser. By default that
// parser inherits helium's safe limits; use [Processor.MaxDepth],
// [Processor.MaxNameLength], [Processor.MaxEntityAmplification], and
// [Processor.MaxContentModelDepth] to raise them for legitimately large
// included documents, or to tighten them for untrusted input. The default resolver opens any OS path
// ([NewFSResolver](nil)); when processing untrusted input, supply a confined
// resolver via [Processor.Resolver] / [NewFSResolver] (e.g. backed by
// [os.Root.FS]).
//
// # Builder Design
//
// Boolean toggles like [Processor.NoXIncludeMarkers] and
// [Processor.NoBaseFixup] are parameterless methods because the builder
// starts from defaults and callers only need to opt into non-default
// behavior. The method name is self-documenting: calling NoBaseFixup means
// "disable base URI fixup."
//
// # Examples
//
// Example code for this package lives in the examples/ directory at the
// repository root (files prefixed with xinclude_). Because examples are
// in a separate test module they do not appear in the generated
// documentation.
package xinclude
