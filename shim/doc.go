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
//   - HTMLAutoClose is not supported.
package shim
