// Package stream implements a streaming XML writer that produces well-formed
// XML incrementally via method calls, writing directly to an [io.Writer]
// without building an in-memory DOM tree.
//
// Use [NewWriter] to create a writer, optionally configure indentation and
// quote style, then call Start/End/Write methods to emit XML:
//
//	w := stream.NewWriter(os.Stdout).Indent("  ")
//	w.StartDocument()
//	w.StartElement("root")
//	w.WriteAttribute("id", "1")
//	w.WriteString("content")
//	w.EndElement()
//	w.EndDocument()
//
// The writer tracks open elements and namespace scopes, and uses sticky
// error handling — check [Writer.Error] after a sequence of calls.
//
// # Examples
//
// Example code for this package lives in the examples/ directory at the
// repository root (files prefixed with stream_). Because examples are in
// a separate test module they do not appear in the generated documentation.
package stream
