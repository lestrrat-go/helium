// Package shim provides a drop-in replacement for [encoding/xml] backed by
// the helium XML parser.
//
// The API mirrors [encoding/xml] so that switching between the two requires
// only changing the import path. The underlying parser is helium's SAX-based
// parser, which provides stricter XML compliance and better performance for
// large documents.
//
// # Known Differences from encoding/xml
//
// The following behaviors differ from [encoding/xml] and are not expected to
// change:
//
//   - InnerXML serialization of empty elements: when unmarshaling a field
//     tagged with ",innerxml", empty elements such as <T1></T1> are
//     serialized as self-closed <T1/>. The helium DOM does not preserve the
//     original serialization form of empty elements.
//   - Non-strict mode (Decoder.Strict = false) is not supported. The shim
//     always parses in strict XML mode.
//   - The [HTMLAutoClose] variable and the Decoder.AutoClose field are not
//     supported. The HTMLAutoClose variable is omitted entirely. The
//     AutoClose field is present for signature compatibility but is a no-op.
//   - The deprecated [encoding/xml.Escape] function is omitted. Use
//     [EscapeText] instead.
//   - Namespace strictness: undeclared namespace prefixes are rejected.
//     [encoding/xml] silently accepts undeclared prefixes and places the
//     raw prefix string in Name.Space.
//   - Declaration strictness: an XML declaration that does not conform to the
//     XMLDecl grammar (XML 1.0 §2.8) is rejected. The grammar requires a
//     version and admits only version, encoding and standalone, in that order
//     and at most once each. Rejected forms include a "charset="
//     pseudo-attribute, a missing or empty version, an empty encoding, a
//     standalone that is not "yes" or "no", and pseudo-attributes out of
//     order (<?xml encoding="UTF-8" version="1.0"?>). [encoding/xml] accepts
//     all of them. This shim is backed by a spec-conforming parser and does
//     not accept XML the specification does not permit. [Unmarshal] and
//     [Decoder] agree: both reject every such declaration.
//   - Declaration placement: an XML declaration is admitted only as the very
//     first thing in the document (prolog ::= XMLDecl? Misc* ...), with only
//     whitespace allowed ahead of it. A "<?xml" appearing after an earlier
//     declaration, a comment, a processing instruction or a doctype is
//     rejected. [encoding/xml] accepts it, reporting it as an ordinary
//     ProcInst. [Unmarshal] and [Decoder] agree in rejecting it.
//   - Reserved target: the target "xml" is reserved in ANY casing —
//     PITarget ::= Name - (('X'|'x')('M'|'m')('L'|'l')) (XML 1.0 §2.6) — so
//     <?XML ...?>, <?Xml ...?> and <?xMl ...?> are illegal wherever they appear.
//     Only the lowercase "xml" introduces a declaration; any other casing is
//     rejected as an illegal target. A target that merely BEGINS with "xml" is
//     unaffected, because the reserved name is subtracted only when it stands
//     alone: <?xmlversion ="2.0"?> and <?xml-stylesheet ...?> are well-formed
//     ordinary PIs, declare no version, and are accepted anywhere a PI may go.
//     [Unmarshal] and [Decoder] agree on every one of these, including a
//     [Decoder] driven by a TokenReader.
//   - Version strictness: a declaration carrying whitespace around the
//     version pseudo-attribute's "=" (<?xml version = "2.0"?>) is rejected
//     as an unsupported version. [encoding/xml] accepts it — it searches
//     for the literal "version=", so a space before the "=" makes the scan
//     miss the pseudo-attribute and read the document as declaring no
//     version at all. XML 1.0 permits the whitespace (Eq ::= S? '=' S?),
//     so the shim reads such a declaration and applies the same
//     unsupported-version rule it applies without the spaces.
//   - Attribute ordering: xmlns namespace declarations are emitted before
//     regular attributes. Source-document attribute order is not preserved
//     because the SAX parser delivers namespaces and attributes as
//     separate slices.
//   - [Decoder.InputOffset] returns an approximate byte offset estimated
//     from the serialized size of each token, not an exact count of bytes
//     consumed from the input. It may diverge from [encoding/xml] for
//     namespace-prefixed names, entity references, CDATA sections, and
//     self-closing elements.
//   - [Decoder.InputPos] is based on a SAX locator snapshot taken at event
//     time. Column numbers may differ from [encoding/xml]. During prolog
//     token emission the reported position is (1, 1).
//
// # Examples
//
// Example code for this package lives in the examples/ directory at the
// repository root (files prefixed with shim_). Because examples are in
// a separate test module they do not appear in the generated documentation.
package shim
