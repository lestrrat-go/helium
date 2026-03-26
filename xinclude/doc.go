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
// # Builder Design
//
// Boolean toggles like [Processor.NoXIncludeMarkers] and
// [Processor.NoBaseFixup] are parameterless methods because the builder
// starts from defaults and callers only need to opt into non-default
// behavior. The method name is self-documenting: calling NoBaseFixup means
// "disable base URI fixup."
package xinclude
