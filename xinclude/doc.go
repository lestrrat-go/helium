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
// The processor is secure by default: with no resolver configured it denies
// all filesystem access, so untrusted input cannot disclose local files via an
// xi:include (mirroring the deny-all default of [helium.NewParser]). Included
// documents are parsed with their own inner parser, confined to the resolver's
// filesystem (see [Processor.Resolver]). To grant access, supply a resolver via
// [Processor.Resolver] / [NewFSResolver] backed by a confined [fs.FS] (e.g.
// [os.Root.FS]); to restore the historical behavior of opening any OS path,
// pass NewFSResolver([helium.PermissiveFS]()).
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
