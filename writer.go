package helium

import (
	"bytes"
	"fmt"
	"io"
	"maps"
	"reflect"
	"slices"
	"strings"

	henc "github.com/lestrrat-go/helium/internal/encoding"
	"github.com/lestrrat-go/helium/internal/lexicon"
	"github.com/lestrrat-go/helium/internal/writerctl"
	"github.com/lestrrat-go/helium/internal/xmlchar"
	"golang.org/x/text/unicode/norm"
)

// Write serializes a node (document or element) to the given writer using
// default settings.
func Write(out io.Writer, node Node) error {
	return NewWriter().WriteTo(out, node)
}

// WriteString serializes a node (document or element) to a string using
// default settings.
func WriteString(node Node) (string, error) {
	var buf strings.Builder
	if err := Write(&buf, node); err != nil {
		return "", err
	}
	return buf.String(), nil
}

const xmlTextNoEnc = "textnoenc"

// Writer serializes an XML document tree (libxml2: xmlSaveCtxt).
//
// It is a value-style wrapper: fluent methods return updated copies and the
// original is never mutated. Mutable runtime state (indent depth, resolved
// escapeNonASCII flag, XHTML detection) lives in a writeSession created
// inside each terminal method.
type Writer struct {
	format             bool
	indentString       string
	indentStringSet    bool // IndentString was called (distinguishes an explicit "" from unset)
	skipDTD            bool
	noEmpty            bool
	noDecl             bool
	noEscapeNonASCII   bool
	allowPrefixUndecl  bool // emit xmlns:prefix="" undeclarations (XML 1.1)
	rejectInvalidChars bool // error (SERE0006) instead of replacing XML-invalid chars
	// charMap substitutes a mapped rune in text and attribute-value content
	// with its literal replacement string (XSLT/XQuery Serialization 3.1 §7
	// character maps). Empty/nil disables the feature.
	charMap map[rune]string
	// cdataElements holds the element names (expanded {uri}local form) whose
	// direct text children are serialized as CDATA sections rather than escaped
	// text (the cdata-section-elements serialization parameter). Matching is by
	// exact expanded name. Empty/nil disables the feature.
	cdataElements map[string]struct{}
	// suppressIndent holds the element names (expanded {uri}local form) whose
	// subtree is serialized without indentation even when Format is enabled (the
	// suppress-indentation serialization parameter). Matching is by exact
	// expanded name. Empty/nil disables the feature.
	suppressIndent map[string]struct{}
	// standalone forces the standalone pseudo-attribute of the XML declaration
	// (the standalone serialization parameter). It has no effect when the XML
	// declaration is omitted. standalonePreserve (the zero value) keeps the
	// document's own standalone status.
	standalone standaloneMode
	// outputVersion overrides the effective output XML version (the version
	// serialization parameter). Empty keeps the document's own version. It drives
	// BOTH the version pseudo-attribute of the XML declaration AND the XML 1.1
	// serialization rules (restricted-character references, namespace
	// undeclarations), so the declaration and escaping stay consistent.
	outputVersion string
	// outputEncoding overrides the encoding pseudo-attribute of the XML
	// declaration (the encoding serialization parameter). Empty keeps the
	// document's own encoding, leaving default output byte-identical.
	outputEncoding string
	// declOnlyEncoding requests declaration-only encoding: the effective encoding
	// labels the XML declaration (and the XHTML <meta charset>) but the
	// transcoding encoder is NOT installed — octets stay UTF-8 with the existing
	// escapeNonASCII char-reference behavior. It is an internal mode (no public
	// setter) used by xpath3 fn:serialize, whose string result treats the encoding
	// parameter as declaration-only per W3C Serialization (a non-representable
	// character becomes a character reference, no octet transcoding). When false
	// (the default, including every WriteTo-to-io.Writer caller) an OutputEncoding
	// override installs a real transcoding encoder.
	declOnlyEncoding bool
	// normalize / normForm request Unicode normalization of text-node and
	// attribute-value character content (the normalization-form serialization
	// parameter, Serialization 3.1 §4 character-expansion phase). Normalization is
	// scoped to text and attribute nodes ONLY — element/attribute names, comments,
	// PIs, the DOCTYPE, and the XML declaration are never normalized. A
	// character-map replacement is assumed to be normalization-inert (fn:serialize
	// substitutes a sentinel rune for a mapped character), so a replacement passes
	// through un-normalized. normalize is false by default, keeping output
	// byte-identical when no normalization is requested.
	normalize bool
	normForm  norm.Form
	// normFormRaw is the exact form string passed to Normalization, retained so
	// WriteTo can reject an unrecognized value (a typo, or a form the writer does
	// not implement) with ErrUnsupportedNormalizationForm rather than silently
	// disabling normalization. Empty (the default) is valid and means "no
	// normalization requested".
	normFormRaw string
	// initialNSScope seeds the serializer's namespace scope with bindings
	// (prefix -> URI; empty prefix = default namespace) treated as already in
	// force on an ancestor that is not itself part of the serialized output. A
	// node using such a prefix is not given a redundant xmlns declaration, so a
	// subtree can be serialized as a self-contained fragment whose in-scope
	// ancestor namespaces are supplied externally. Nil/empty leaves output
	// byte-identical.
	initialNSScope map[string]string
}

// standaloneMode controls how the writer emits the standalone pseudo-attribute
// of the XML declaration.
type standaloneMode int

const (
	// standalonePreserve emits the document's own standalone status.
	standalonePreserve standaloneMode = iota
	// standaloneForceOmit emits no standalone pseudo-attribute.
	standaloneForceOmit
	// standaloneForceYes / standaloneForceNo force the pseudo-attribute value.
	standaloneForceYes
	standaloneForceNo
)

// writeSession holds the mutable state for a single serialization pass.
// It is created inside WriteTo and threaded through the internal helper
// methods so that Writer itself stays immutable.
type writeSession struct {
	Writer
	escapeNonASCII bool
	// asciiOutput is set when an explicit OutputEncoding override names US-ASCII
	// (by any IANA alias — us-ascii, ascii, csASCII, ANSI_X3.4-1968, …, matched
	// via internal/encoding.IsASCII) on the Document serialization path: the
	// writer installs no transcoding encoder but escapes every non-ASCII character
	// in text and attribute-value content as a hex character reference, so the
	// emitted octets are pure US-ASCII and agree with the encoding declaration.
	// Independent of escapeNonASCII (which only covers Latin-1) so BMP/astral
	// characters are escaped too.
	//
	// On the real octet-producing WriteTo path (asciiReject, i.e. asciiOutput &&
	// !declOnlyEncoding) any non-ASCII byte that CANNOT be char-referenced has no
	// faithful US-ASCII serialization and fails with ErrUnsupportedOutputEncoding:
	// per-site guards give early, labelled errors for names, comments, CDATA,
	// PI target/data, namespace prefixes, DTD-internal names, and character-map
	// replacements, and an exhaustive output-writer net (asciiRejectWriter,
	// installed in writeDoc) is the backstop that rejects any surviving byte
	// >= 0x80 from any raw-write site. fn:serialize's declaration-only mode also
	// sets asciiOutput but not asciiReject: it returns a UTF-8 string, so
	// non-ASCII text/attr values still char-reference while reference-less content
	// and character-map replacements stay raw.
	//
	// asciiOutput is NEVER set on the bare element/fragment path — OutputEncoding
	// affects the Document path only.
	asciiOutput bool
	xml11       bool // true when the document declares XML 1.1: restricted control chars serialize as decimal character references
	isXHTML     bool
	encoding    string // document encoding, used for XHTML meta injection
	indent      int    // current indent depth (used when format is true)
	err         error  // sticky write error; once set, further writes are skipped
	// cdataText reports that the element currently being serialized is a
	// cdata-section-element, so its direct text children are emitted as CDATA
	// sections rather than escaped text.
	cdataText bool
	// suppressDepth > 0 means the current subtree descends from a
	// suppress-indentation element, so indentation is disabled for it even when
	// format is enabled.
	suppressDepth int
	// nsScope maps a namespace prefix to the URI currently in force in the
	// serialized OUTPUT — the union of the xmlns declarations emitted on the
	// ancestor path. reconcileNamespaces consults it so a prefixed element or
	// attribute whose namespace was declared on an ancestor outside the
	// serialized subtree still gets a declaration. It is nil until the first
	// namespaced element, so a plain-XML dump allocates nothing.
	nsScope map[string]string
}

// nsSaved records a prefix's prior binding in nsScope so it can be restored
// after an element's subtree is serialized.
type nsSaved struct {
	prefix string
	href   string
	had    bool
}

// writeString writes str to out, recording the first error encountered into
// s.err. Once s.err is set, subsequent writes are skipped so the sticky error
// is preserved and propagated by the terminal serialization methods.
func (s *writeSession) writeString(out io.Writer, str string) {
	if s.err != nil {
		return
	}
	n, err := io.WriteString(out, str)
	if err != nil {
		s.err = err
		return
	}
	if n != len(str) {
		s.err = io.ErrShortWrite
	}
}

// writeBytes writes b to out, recording the first error encountered into s.err.
func (s *writeSession) writeBytes(out io.Writer, b []byte) {
	if s.err != nil {
		return
	}
	n, err := out.Write(b)
	if err != nil {
		s.err = err
		return
	}
	if n != len(b) {
		s.err = io.ErrShortWrite
	}
}

// check records err into the sticky s.err (keeping the first one) so callers
// that obtain an error from a leaf helper can funnel it through the session.
func (s *writeSession) check(err error) {
	if s.err == nil && err != nil {
		s.err = err
	}
}

// hasNonASCII reports whether s contains a byte >= 0x80, i.e. any non-ASCII
// character in its UTF-8 encoding. Works over both string and []byte.
func hasNonASCII[T ~string | ~[]byte](s T) bool {
	for i := range len(s) {
		if s[i] >= 0x80 {
			return true
		}
	}
	return false
}

// unsupportedASCIIErr builds an ErrUnsupportedOutputEncoding for a non-ASCII
// character appearing in a context (what) that cannot hold a character
// reference under a US-ASCII output encoding.
func unsupportedASCIIErr(what string) error {
	return fmt.Errorf("cannot serialize non-ASCII %s under US-ASCII output encoding: %w", what, ErrUnsupportedOutputEncoding)
}

// asciiReject reports whether a raw non-ASCII byte must be rejected on the
// current path: an explicit US-ASCII OutputEncoding override on the real
// octet-producing WriteTo path. It excludes declaration-only mode
// (fn:serialize), where asciiOutput is also set but the string result keeps
// octets as UTF-8 with char-reference escaping and preserves raw character-map
// replacements — rejecting there would break fn:serialize semantics. Every
// reference-less guard and the character-map reject key on this, not on
// asciiOutput alone.
func (s *writeSession) asciiReject() bool {
	return s.asciiOutput && !s.declOnlyEncoding
}

// rejectNonASCIIStr records a sticky ErrUnsupportedOutputEncoding when
// asciiReject is on and s carries a non-ASCII byte. It guards contexts that
// cannot hold a character reference — element/attribute names, namespace
// prefixes, PI target/data, notation names — where a US-ASCII output encoding
// that cannot carry the character literally has no faithful serialization.
// Returns true when it recorded the error.
func (s *writeSession) rejectNonASCIIStr(what, str string) bool {
	if !s.asciiReject() || !hasNonASCII(str) {
		return false
	}
	s.check(unsupportedASCIIErr(what))
	return true
}

// rejectNonASCIIBytes is the []byte counterpart of rejectNonASCIIStr, for the
// content of comments and CDATA sections (which likewise cannot hold a
// character reference). The offending content is not embedded in the error —
// it may be large and attacker-controlled.
func (s *writeSession) rejectNonASCIIBytes(what string, b []byte) bool {
	if !s.asciiReject() || !hasNonASCII(b) {
		return false
	}
	s.check(unsupportedASCIIErr(what))
	return true
}

// asciiRejectWriter wraps the output writer under an explicit US-ASCII
// OutputEncoding on the real octet-producing WriteTo path (never in
// declaration-only fn:serialize mode). It is the exhaustive backstop for the
// asciiOutput invariant: by the time any byte reaches it, text and attribute
// values have already been char-referenced to pure ASCII and every
// reference-less context has been guarded, so any surviving byte >= 0x80 is a
// raw-write leak from an unguarded site (a character-map replacement, an
// already-encoded textnoenc node, a DTD external/system/public-ID literal, an
// entity or notation value/name). On the first such byte Write returns
// ErrUnsupportedOutputEncoding without forwarding it, so no raw non-ASCII octet
// is emitted; a pure-ASCII buffer forwards unchanged. The error is sticky so a
// later write cannot slip a raw octet past it.
type asciiRejectWriter struct {
	w   io.Writer
	err error
}

func (a *asciiRejectWriter) Write(p []byte) (int, error) {
	if a.err != nil {
		return 0, a.err
	}
	if hasNonASCII(p) {
		a.err = unsupportedASCIIErr("output")
		return 0, a.err
	}
	return a.w.Write(p)
}

// hasXmlnsPrefix reports whether name carries the reserved "xmlns:" QName
// prefix. Namespaces-in-XML forbids using "xmlns" as an element/attribute
// prefix; such a name (e.g. "xmlns:root") would be serialized as a forbidden
// prefixed name. Note this does NOT match the bare name "xmlns": that name is a
// valid element name (<xmlns/> is well-formed) and is only reserved as an
// attribute name, which checkAttributeName handles separately.
func hasXmlnsPrefix(name string) bool {
	return strings.HasPrefix(name, "xmlns:")
}

// checkElementName validates an element name about to be emitted verbatim. An
// unvalidated name (e.g. from CreateElement) can carry whitespace, quotes, or
// '>' that inject raw markup into the output. On failure it records a sticky
// error (preserving any earlier one) and returns false. Shared by both the
// generic and XHTML serialization paths so they cannot diverge.
//
// An element whose QName prefix is the reserved "xmlns" prefix is rejected:
// IsValidQName only checks QName grammar, but Namespaces-in-XML forbids using
// "xmlns" as a prefix. With an active namespace (which bypasses dumpNs) such a
// name (e.g. "xmlns:root") could otherwise be serialized as <xmlns:root/>. The
// bare name "xmlns" is NOT rejected: it is a valid element name (<xmlns/> is
// well-formed XML); "xmlns" is reserved only as an attribute name.
func (s *writeSession) checkElementName(name string) bool {
	if hasXmlnsPrefix(name) {
		s.check(fmt.Errorf("helium: reserved element name %q: namespace declarations must use DeclareNamespace: %w", name, ErrWriterReservedElementName))
		return false
	}
	if !xmlchar.IsValidQName(name) {
		s.check(fmt.Errorf("helium: invalid element name %q: %w", name, ErrWriterInvalidElementName))
		return false
	}
	// An element name cannot hold a character reference, so a non-ASCII name has
	// no faithful US-ASCII serialization.
	if s.rejectNonASCIIStr("element name", name) {
		return false
	}
	return true
}

// checkAttributeName validates an attribute name about to be emitted verbatim.
// An unvalidated name can inject raw markup (extra attributes, '>') into the
// start tag. On failure it records a sticky error and returns false.
//
// The reserved "xmlns" name is also rejected: a normal attribute named
// "xmlns" (or one whose QName prefix is "xmlns", e.g. "xmlns:foo") would be
// emitted as a namespace declaration even though it never went through
// DeclareNamespace. Namespace declarations are stored as separate Namespace
// nodes (nsDefs) and serialized by dumpNs; the serializer's own correct
// xmlns output never reaches this function, so rejecting here only blocks
// user-supplied misuse.
func (s *writeSession) checkAttributeName(name string) bool {
	if name == "xmlns" || hasXmlnsPrefix(name) {
		s.check(fmt.Errorf("helium: reserved attribute name %q: namespace declarations must use DeclareNamespace: %w", name, ErrWriterReservedAttributeName))
		return false
	}
	if !xmlchar.IsValidQName(name) {
		s.check(fmt.Errorf("helium: invalid attribute name %q: %w", name, ErrWriterInvalidAttributeName))
		return false
	}
	// An attribute name cannot hold a character reference, so a non-ASCII name
	// has no faithful US-ASCII serialization.
	if s.rejectNonASCIIStr("attribute name", name) {
		return false
	}
	return true
}

// checkNamespacePrefix validates a namespace declaration prefix about to be
// emitted as "xmlns:"+prefix. An unvalidated prefix (e.g. from
// DeclareNamespace) can carry whitespace, quotes, or '>' that inject raw markup
// into the start tag. The empty prefix (default namespace, xmlns="...") is
// allowed; any non-empty prefix must be a valid NCName (no colon). The
// reserved "xmlns" prefix is rejected: Namespaces-in-XML forbids declaring it,
// so dumpNs must not emit xmlns:xmlns="...". The "xml" prefix is handled by
// dumpNs before this function is called. On failure it records a sticky error
// (preserving any earlier one) and returns false. Shared by both the generic
// and XHTML serialization paths so they cannot diverge.
func (s *writeSession) checkNamespacePrefix(prefix string) bool {
	if prefix == "xmlns" {
		s.check(fmt.Errorf("helium: reserved namespace prefix %q must not be declared: %w", prefix, ErrWriterReservedNamespacePrefix))
		return false
	}
	if prefix != "" && !xmlchar.IsValidNCName(prefix) {
		s.check(fmt.Errorf("helium: invalid namespace prefix %q: %w", prefix, ErrWriterInvalidNamespacePrefix))
		return false
	}
	// A namespace prefix cannot hold a character reference, so a non-ASCII prefix
	// has no faithful US-ASCII serialization. (The namespace URI is an attribute
	// value and stays char-referenced.)
	if s.rejectNonASCIIStr("namespace prefix", prefix) {
		return false
	}
	return true
}

// NewWriter creates a new Writer with default settings.
func NewWriter() Writer {
	return Writer{}
}

// Format controls whether indented (pretty-printed) output is emitted.
func (w Writer) Format(v bool) Writer {
	w.format = v
	return w
}

// IndentString sets the string used for each indent level (honored only when
// Format is enabled). An explicit empty string requests formatted output with
// newlines but no per-level indentation; leaving IndentString unset uses the
// two-space default.
func (w Writer) IndentString(s string) Writer {
	w.indentString = s
	w.indentStringSet = true
	return w
}

// SelfCloseEmptyElements controls whether empty elements are serialized as
// self-closing tags (for example, <br/>). When false, they are emitted as
// explicit open+close pairs (for example, <br></br>).
func (w Writer) SelfCloseEmptyElements(v bool) Writer {
	w.noEmpty = !v
	return w
}

// XMLDeclaration controls whether the XML declaration is emitted.
func (w Writer) XMLDeclaration(v bool) Writer {
	w.noDecl = !v
	return w
}

// IncludeDTD controls whether DTD nodes are emitted.
func (w Writer) IncludeDTD(v bool) Writer {
	w.skipDTD = !v
	return w
}

// EscapeNonASCII controls whether non-ASCII characters are escaped as numeric
// character references when serializing UTF-8 output.
func (w Writer) EscapeNonASCII(v bool) Writer {
	w.noEscapeNonASCII = !v
	return w
}

// AllowPrefixUndeclarations controls whether xmlns:prefix="" undeclarations
// may be emitted.
func (w Writer) AllowPrefixUndeclarations(v bool) Writer {
	w.allowPrefixUndecl = v
	return w
}

// RejectInvalidChars controls how the writer handles a character that is not
// valid in the target XML version (e.g. a C0/C1 control character in XML 1.0
// output). When false (the default) such a character is replaced with U+FFFD;
// when true the write fails with ErrInvalidXMLChar (the XSLT/XQuery
// serialization error SERE0006). This detection is folded into the existing
// text/attribute escaping pass, so it adds no extra traversal.
func (w Writer) RejectInvalidChars(v bool) Writer {
	w.rejectInvalidChars = v
	return w
}

// CharacterMap installs a character map: each mapped rune appearing in text or
// attribute-value content is replaced by its literal replacement string,
// emitted verbatim (not re-escaped), per XSLT/XQuery Serialization 3.1 §7. A nil
// or empty map disables the feature.
//
// The map is copied, so later mutation of the caller's map does not affect the
// Writer (the value-style contract above).
func (w Writer) CharacterMap(m map[rune]string) Writer {
	w.charMap = maps.Clone(m)
	return w
}

// CDATASectionElements names the elements (each as an expanded {uri}local name)
// whose direct text children are serialized as CDATA sections instead of escaped
// text (the cdata-section-elements serialization parameter). Matching is by exact
// expanded name. A nil or empty map disables the feature.
//
// The map is copied, so later mutation of the caller's map does not affect the
// Writer (the value-style contract above).
func (w Writer) CDATASectionElements(m map[string]struct{}) Writer {
	w.cdataElements = maps.Clone(m)
	return w
}

// SuppressIndentElements names the elements (each as an expanded {uri}local name)
// whose subtree is serialized without indentation even when Format is enabled
// (the suppress-indentation serialization parameter). Matching is by exact
// expanded name. A nil or empty map disables the feature.
//
// The map is copied, so later mutation of the caller's map does not affect the
// Writer (the value-style contract above).
func (w Writer) SuppressIndentElements(m map[string]struct{}) Writer {
	w.suppressIndent = maps.Clone(m)
	return w
}

// Standalone forces the standalone pseudo-attribute of the XML declaration:
// v=true emits standalone="yes", v=false emits standalone="no". It overrides the
// document's own standalone status and has no effect when the XML declaration is
// omitted. When neither Standalone nor OmitStandalone is called, the document's
// own standalone status is used.
func (w Writer) Standalone(v bool) Writer {
	if v {
		w.standalone = standaloneForceYes
	} else {
		w.standalone = standaloneForceNo
	}
	return w
}

// OmitStandalone forces the XML declaration to carry no standalone
// pseudo-attribute, overriding the document's own standalone status (the
// standalone="omit" serialization parameter value). It has no effect when the
// XML declaration is omitted.
func (w Writer) OmitStandalone() Writer {
	w.standalone = standaloneForceOmit
	return w
}

// OutputVersion overrides the effective output XML version (the version
// serialization parameter, e.g. "1.0" or "1.1"), driving BOTH the version
// pseudo-attribute of the XML declaration AND the XML 1.1 serialization rules
// (restricted-character references and namespace undeclarations). An empty
// string keeps the document's own version, leaving default output byte-identical.
//
// The effective version must be a valid XML VersionNum (`'1.' [0-9]+`, e.g. "1.0"
// or "1.1"); a value that is malformed or carries an illegal character (which
// would inject markup into the declaration) fails serialization with
// ErrInvalidOutputVersion and emits nothing.
func (w Writer) OutputVersion(v string) Writer {
	w.outputVersion = v
	return w
}

// isValidXMLVersion reports whether v is a valid XML VersionNum (XML §2.8):
//
//	VersionNum ::= '1.' [0-9]+
//
// The version is emitted raw into the XML declaration's version pseudo-attribute
// between double quotes, so this rejects a value carrying a quote or other
// illegal character (which would inject markup) or an otherwise-malformed version
// (which would produce an unparseable declaration). The '1.' prefix is the spec
// grammar for an XML 1.0/1.1 processor; it is ASCII-only and allocation-free.
func isValidXMLVersion(v string) bool {
	rest, ok := strings.CutPrefix(v, "1.")
	if !ok || rest == "" {
		return false
	}
	for i := range len(rest) {
		if rest[i] < '0' || rest[i] > '9' {
			return false
		}
	}
	return true
}

// effectiveVersion returns the version driving serialization: the OutputVersion
// override when set, otherwise the document's own version (defaulting to "1.0").
func (d Writer) effectiveVersion(doc *Document) string {
	if d.outputVersion != "" {
		return d.outputVersion
	}
	if doc != nil && doc.version != "" {
		return doc.version
	}
	return "1.0"
}

// OutputEncoding overrides the encoding pseudo-attribute of the XML declaration
// (the encoding serialization parameter). An empty string keeps the document's
// own encoding, leaving default output byte-identical. It affects the Document
// serialization path only; a bare element/fragment is byte-identical to output
// without an override.
//
// The effective encoding must be a valid XML EncName (`[A-Za-z] ([A-Za-z0-9._] |
// '-')*`); a malformed label (one carrying a quote or other illegal character,
// which would inject markup into the declaration) fails serialization with
// ErrUnsupportedOutputEncoding before any output byte is written. An empty
// effective encoding simply omits the encoding pseudo-attribute.
//
// When set on a WriteTo-to-io.Writer path the emitted octets are re-encoded to
// the named encoding so the bytes agree with the declaration. US-ASCII (by any
// alias) installs no encoder but escapes every non-ASCII character as a numeric
// character reference; a non-ASCII character in a context that cannot hold a
// reference (comment, CDATA, PI, a name), or an encoding the writer cannot
// otherwise emit, fails with ErrUnsupportedOutputEncoding.
func (w Writer) OutputEncoding(v string) Writer {
	w.outputEncoding = v
	return w
}

// Normalization requests Unicode normalization of text-node and attribute-value
// character content (the normalization-form serialization parameter). form is one
// of "NFC", "NFD", "NFKC", "NFKD" to enable it, or "" / "none" to disable it
// (leaving output byte-identical). Any other value is an error: WriteTo fails with
// ErrUnsupportedNormalizationForm before emitting any output byte, so a typo or an
// unsupported form (e.g. "fully-normalized") is observable rather than silently
// swallowed. Normalization is scoped to text and attribute nodes (Serialization
// 3.1 §4 character-expansion phase) — element/attribute names, comments, PIs, the
// DOCTYPE, and the XML declaration are never normalized. A character map
// (CharacterMap) is expected to carry normalization-inert replacements
// (fn:serialize substitutes sentinel runes), so a mapped character's replacement
// is not normalized.
func (w Writer) Normalization(form string) Writer {
	w.normFormRaw = form
	w.normForm, w.normalize = xmlNormalizationForm(form)
	return w
}

// declarationOnlyEncoding returns a copy of the writer with declaration-only
// encoding enabled (see the declOnlyEncoding field). It has no public setter;
// it is reached only through the internal/writerctl hook, registered below.
func (w Writer) declarationOnlyEncoding() Writer {
	w.declOnlyEncoding = true
	return w
}

func init() {
	writerctl.EnableDeclarationOnlyEncoding = writerDeclarationOnlyEncoding
}

// writerDeclarationOnlyEncoding adapts declarationOnlyEncoding to the untyped
// internal/writerctl hook (any in, any out) so a sibling package can enable the
// mode without a public method or an import cycle.
func writerDeclarationOnlyEncoding(w any) any {
	ww, ok := w.(Writer)
	if !ok {
		return w
	}
	return ww.declarationOnlyEncoding()
}

// InheritedNamespaces seeds the serializer's namespace scope with bindings
// (prefix -> URI; the empty prefix is the default namespace) treated as already
// in force on an ancestor outside the serialized output. A node using such a
// prefix is not given a redundant xmlns re-declaration, so a subtree can be
// serialized as a self-contained fragment whose in-scope ancestor namespaces are
// supplied externally (for example, capturing an element's inner XML while its
// ancestors are not serialized). A nil or empty map leaves output byte-identical.
//
// The map is copied, so later mutation of the caller's map does not affect the
// Writer (the value-style contract above).
func (w Writer) InheritedNamespaces(bindings map[string]string) Writer {
	w.initialNSScope = maps.Clone(bindings)
	return w
}

// effectiveEncoding returns the encoding driving the XML-declaration
// pseudo-attribute: the OutputEncoding override when set, otherwise the
// document's own encoding. The override is whitespace-trimmed so the encoder
// lookup (which trims) and the emitted label agree and the declaration carries a
// valid EncName rather than a padded, unparseable one.
func (d Writer) effectiveEncoding(doc *Document) string {
	if d.outputEncoding != "" {
		return strings.TrimSpace(d.outputEncoding)
	}
	if doc != nil {
		return doc.encoding
	}
	return ""
}

// writeCDATASplit emits c as one or more CDATA sections, splitting on any "]]>"
// sequence so the output stays well-formed (the "]]" is kept in one section and
// the ">" starts the next). Empty content emits an empty CDATA section. Used for
// both explicit CDATA-section nodes and the text children of a
// cdata-section-element.
func (s *writeSession) writeCDATASplit(out io.Writer, c []byte) {
	if len(c) == 0 {
		s.writeString(out, "<![CDATA[]]>")
		return
	}
	start := 0
	for i := 0; i+2 < len(c); i++ {
		if c[i] == ']' && c[i+1] == ']' && c[i+2] == '>' {
			end := i + 2
			s.writeString(out, "<![CDATA[")
			s.writeBytes(out, c[start:end])
			s.writeString(out, "]]>")
			start = end
		}
	}
	if start < len(c) {
		s.writeString(out, "<![CDATA[")
		s.writeBytes(out, c[start:])
		s.writeString(out, "]]>")
	}
}

func (s *writeSession) indentStr() string {
	// An explicit IndentString("") means "no per-level indentation" (newlines
	// only); only an unset IndentString falls back to the two-space default.
	if !s.indentStringSet {
		return "  "
	}
	return s.indentString
}

func (s *writeSession) writeIndent(out io.Writer) {
	if !s.format || s.indent <= 0 {
		return
	}
	str := s.indentStr()
	for range s.indent {
		s.writeString(out, str)
	}
}

// hasOnlyTextChildren returns true when every child is a text or entity-ref node.
func hasOnlyTextChildren(n Node) bool {
	for c := range Children(n) {
		switch c.Type() {
		case TextNode, EntityRefNode, CDATASectionNode:
			// ok
		default:
			return false
		}
	}
	return true
}

// hasTextlikeChild returns true when any DIRECT child of n is a text, CDATA, or
// entity-reference node. This mirrors libxml2's xmlNodeDumpOutputInternal
// (xmlsave.c): an element with any such child is mixed content, so it is marked
// suppressed and formatting is disabled for its ENTIRE subtree (via suppressDepth)
// until it closes — the children, and their descendants, are serialized inline
// rather than having indentation whitespace injected. Formatting a mixed element's
// subtree would alter the text content and not be idempotent.
func hasTextlikeChild(n Node) bool {
	for c := range Children(n) {
		switch c.Type() {
		case TextNode, CDATASectionNode, EntityRefNode:
			return true
		}
	}
	return false
}

// isNilNode reports whether node is nil, covering both a literal nil interface
// and a typed-nil concrete pointer wrapped in a non-nil Node interface
// (Go's interface nil trap).
func isNilNode(node Node) bool {
	if node == nil {
		return true
	}
	v := reflect.ValueOf(node)
	return v.Kind() == reflect.Pointer && v.IsNil()
}

// WriteTo serializes a node (document or element) to the given writer.
// When the node is a Document, document-level setup (encoding, XHTML
// detection, DTD filtering) is applied automatically.
func (d Writer) WriteTo(out io.Writer, node Node) error {
	// Guard against a nil node — both a literal nil interface and a typed-nil
	// concrete pointer (e.g. a (*Element)(nil) stored in a Node) — so callers
	// get ErrNilNode instead of a panic from method calls on the nil node.
	if isNilNode(node) {
		return ErrNilNode
	}
	// Reject an unrecognized normalization-form (a typo, or a form the writer does
	// not implement) before any output byte, on both the Document and bare-element
	// paths: Normalization stores the raw value and defers the check here so the
	// failure is observable rather than a silent no-op. "" (the default) and "none"
	// are valid and disable normalization.
	if !validNormalizationForm(d.normFormRaw) {
		return fmt.Errorf("helium: unsupported normalization form %q: %w", d.normFormRaw, ErrUnsupportedNormalizationForm)
	}
	// Validate a non-empty OutputVersion override once, ahead of the node-type
	// branch, so a bare element/fragment is rejected exactly like a Document: the
	// override drives the XML 1.1 escaping rules here and (on the Document path)
	// the declaration's version pseudo-attribute, so a malformed value must fail
	// on every path rather than being silently treated as XML 1.0. An empty
	// override keeps the document's own version and is validated on the Document
	// path (writeDoc) against the document's version.
	if d.outputVersion != "" && !isValidXMLVersion(d.outputVersion) {
		return fmt.Errorf("helium: invalid output XML version %q: %w", d.outputVersion, ErrInvalidOutputVersion)
	}
	if doc, ok := node.(*Document); ok {
		return d.writeDoc(out, doc)
	}
	s := writeSession{Writer: d, escapeNonASCII: !d.noEscapeNonASCII}
	// A bare element carries no document version, so only an explicit
	// OutputVersion("1.1") override enables XML 1.1 serialization here; without
	// it, output stays byte-identical to the prior behavior.
	s.xml11 = d.outputVersion == "1.1"
	// OutputEncoding affects the Document path only. A bare element/fragment has
	// no XML declaration to disagree with, so asciiOutput stays off here and the
	// serialized bytes are byte-identical to output without an OutputEncoding
	// override (an out-of-range character stays escaped by escapeNonASCII exactly
	// as before).
	s.seedNSScope()
	return s.writeNode(out, node)
}

func (d Writer) writeDoc(out io.Writer, doc *Document) error {
	s := writeSession{Writer: d}

	// Validate the effective version and encoding BEFORE any output byte is
	// produced: both are caller-controlled (OutputVersion/SetVersion,
	// OutputEncoding/SetEncoding) and are emitted raw between the declaration's
	// pseudo-attribute quotes, so an illegal character would inject markup and a
	// malformed value would produce an unparseable declaration. This runs ahead of
	// the transcoding-encoder setup below because that encoder's deferred Close
	// flushes a BOM at EOF — installing it before validation would leak that BOM to
	// the caller even when the declaration itself is never written. Nothing has been
	// written yet, so return the error directly. This is a separate, earlier check
	// than the US-ASCII transcoding reject (asciiReject) — that one rejects a
	// legal-but-unrepresentable label; this one rejects a label that is not a
	// well-formed VersionNum/EncName.
	version := d.effectiveVersion(doc)
	if !isValidXMLVersion(version) {
		return fmt.Errorf("helium: invalid output XML version %q: %w", version, ErrInvalidOutputVersion)
	}
	if enc := d.effectiveEncoding(doc); enc != "" && !xmlchar.IsValidEncName(enc) {
		return fmt.Errorf("helium: invalid output encoding name %q: %w", enc, ErrUnsupportedOutputEncoding)
	}

	// An XML 1.1 document (or an OutputVersion("1.1") override) may carry
	// restricted control characters; serialize them as decimal character
	// references. XML 1.0 output is unaffected.
	s.xml11 = version == "1.1"

	// Mirrors libxml2's xmlSaveWriteText: when output encoding is UTF-8
	// (no encoder), escape non-ASCII chars 0x80-0xDF as numeric refs.
	// When an encoder is present, pass them through for re-encoding.
	//
	// The encoder keys off the EFFECTIVE encoding (the OutputEncoding override
	// when set, else the document's own encoding) — the same value the XML
	// declaration and the XHTML <meta> use — so the emitted bytes agree with the
	// declaration. Declaration-only mode (fn:serialize) skips the transcoding
	// encoder entirely: the label is set from the effective encoding but octets stay
	// UTF-8 with char-reference escaping. US-ASCII output still forces non-ASCII
	// characters to references even in declaration-only mode, since a US-ASCII string
	// cannot hold them literally either.
	s.escapeNonASCII = !d.noEscapeNonASCII
	if enc := d.effectiveEncoding(doc); enc != "" {
		lower := strings.ToLower(enc)
		switch {
		case lower == "utf-8" || lower == encUTF8:
			// UTF-8 represents every character; no encoder and no extra escaping.
		case henc.IsASCII(enc):
			// US-ASCII by any IANA alias (us-ascii, ascii, csASCII, ANSI_X3.4-1968,
			// iso-ir-6, …). No transcoding encoder is installed either way.
			if d.outputEncoding != "" {
				// OVERRIDE: the declaration cannot carry a non-ASCII character
				// literally, so escape every one as a character reference — the
				// octets are pure US-ASCII and match the declaration.
				s.asciiOutput = true
				// On the real octet-producing path (not declaration-only
				// fn:serialize) install the exhaustive ASCII-reject net: the
				// per-site guards and the escapers' char-referencing already keep
				// every reachable value pure ASCII, so any byte >= 0x80 reaching
				// this wrapper is a raw-write leak, caught before it is emitted.
				if !d.declOnlyEncoding {
					out = &asciiRejectWriter{w: out}
				}
			} else if henc.IsASCIIRawUTF8Alias(enc) {
				// NO-OVERRIDE, one of the two aliases (ANSI_X3.4-1968, csASCII) the
				// document serializer emits as raw UTF-8 via a UTF-8 passthrough:
				// disable non-ASCII escaping so the bytes stay byte-identical. Every
				// other US-ASCII alias keeps default Latin-1 character-reference
				// escaping on the no-override path.
				s.escapeNonASCII = false
			}
		case !d.declOnlyEncoding:
			e := henc.Load(enc)
			// An explicitly-set OutputEncoding the writer cannot emit is a hard
			// error: emitting UTF-8 under that declaration would make the bytes
			// disagree with it. Without an override the effective encoding is the
			// document's own (parsed) encoding, which stays declaration-only when
			// unloadable — keeping default output byte-identical.
			if e == nil && d.outputEncoding != "" {
				return fmt.Errorf("cannot serialize with output encoding %q: %w", enc, ErrUnsupportedOutputEncoding)
			}
			if e != nil {
				s.escapeNonASCII = false
				w := e.NewEncoder().Writer(out)
				if closer, ok := w.(io.Closer); ok {
					defer func() { _ = closer.Close() }()
				}
				out = w
			}
		}
	}

	// Detect XHTML. Mirrors xmlSaveDocInternal in xmlsave.c.
	s.isXHTML = false
	s.encoding = d.effectiveEncoding(doc)
	if dtd := doc.intSubset; dtd != nil {
		s.isXHTML = isXHTMLDTD(dtd)
	}

	if err := s.writeNode(out, doc); err != nil {
		return err
	}

	for e := range Children(doc) {
		if s.skipDTD && e.Type() == DTDNode {
			continue
		}
		if s.isXHTML && e.Type() == ElementNode {
			if err := s.dumpXHTMLNode(out, e); err != nil {
				return err
			}
		} else {
			if err := s.writeNode(out, e); err != nil {
				return err
			}
		}
		s.writeString(out, "\n")
	}
	return s.err
}

func (d *writeSession) dumpDocContent(out io.Writer, n Node) error {
	doc, ok := AsNode[*Document](n)
	if !ok {
		return nil
	}

	// The effective version and encoding are validated at the writeDoc entry —
	// before any output byte (including the transcoding encoder's BOM) — so both are
	// well-formed here: version is a valid VersionNum and, when non-empty, encoding
	// is a valid EncName. They are emitted raw between the declaration's
	// pseudo-attribute quotes.
	version := d.effectiveVersion(doc)
	encoding := d.effectiveEncoding(doc)

	d.writeString(out, `<?xml version="`)
	d.writeString(out, version+`"`)

	if encoding != "" {
		d.writeString(out, ` encoding="`+encoding+`"`)
	}

	// A forced standalone (the serialization parameter) overrides the document's
	// own standalone status; standaloneForceOmit emits no pseudo-attribute.
	switch d.standalone {
	case standaloneForceOmit:
		// emit nothing
	case standaloneForceYes:
		d.writeString(out, ` standalone="`+lexicon.ValueYes+`"`)
	case standaloneForceNo:
		d.writeString(out, ` standalone="`+lexicon.ValueNo+`"`)
	case standalonePreserve:
		switch doc.Standalone() {
		case StandaloneExplicitNo:
			d.writeString(out, ` standalone="`+lexicon.ValueNo+`"`)
		case StandaloneExplicitYes:
			d.writeString(out, ` standalone="`+lexicon.ValueYes+`"`)
		}
	}
	d.writeString(out, "?>\n")
	return d.err
}

// writeNode is the internal implementation for node serialization.
func (d *writeSession) writeNode(out io.Writer, n Node) error {
	var err error
	switch n.Type() {
	case DocumentNode:
		if !d.noDecl {
			if err = d.dumpDocContent(out, n); err != nil {
				return err
			}
		}
		return nil
	case DTDNode:
		if err = d.dumpDTD(out, n); err != nil {
			return err
		}
		return nil
	case CommentNode:
		// A comment must not contain "--" or end with "-" (that would form
		// "--->" with the closing delimiter), else the output is not well-formed.
		// Validate the byte slice directly: a string() copy here would double the
		// peak memory for a large (attacker-controlled) comment before the same
		// bytes are written below.
		// rawContent avoids the defensive copy Content() makes: this path is
		// read-only and a copy here would double the peak memory for a large
		// (attacker-controlled) comment before the same bytes are written below.
		content := rawContent(n)
		if bytes.Contains(content, []byte("--")) || (len(content) > 0 && content[len(content)-1] == '-') {
			// check() keeps the first sticky error, so an earlier I/O failure is
			// not clobbered by this validation error.
			d.check(fmt.Errorf("helium: comment content must not contain \"--\" or end with \"-\": %w", ErrWriterInvalidComment))
			return d.err
		}
		// Comment text cannot hold a character reference, so a non-ASCII comment
		// has no faithful US-ASCII serialization.
		if d.rejectNonASCIIBytes("comment content", content) {
			return d.err
		}
		d.writeString(out, "<!--")
		d.writeBytes(out, content)
		d.writeString(out, "-->")
		return d.err
	case ProcessingInstructionNode:
		// Mirrors xmlsave.c XML_PI_NODE handling.
		if pi, ok := AsNode[*ProcessingInstruction](n); ok {
			// The PI target must be a valid XML Name (and not the reserved
			// "xml"); otherwise an invalid/crafted target injects raw markup
			// into the output (it is emitted verbatim below).
			if !xmlchar.IsValidPITarget(pi.target) {
				// check() keeps the first sticky error, so an earlier I/O failure
				// is not clobbered by this validation error.
				d.check(fmt.Errorf("helium: invalid PI target: %w", ErrWriterInvalidPITarget))
				return d.err
			}
			// PI data must not contain "?>", which would terminate the PI early.
			if strings.Contains(pi.data, "?>") {
				// check() keeps the first sticky error, so an earlier I/O failure
				// is not clobbered by this validation error.
				d.check(fmt.Errorf("helium: PI content must not contain \"?>\": %w", ErrWriterInvalidPIContent))
				return d.err
			}
			// Neither the PI target nor its data can hold a character reference, so
			// non-ASCII in either has no faithful US-ASCII serialization.
			if d.rejectNonASCIIStr("PI target", pi.target) || d.rejectNonASCIIStr("PI data", pi.data) {
				return d.err
			}
			d.writeString(out, "<?")
			d.writeString(out, pi.target)
			if pi.data != "" {
				d.writeString(out, " ")
				d.writeString(out, pi.data)
			}
			d.writeString(out, "?>")
		}
		return d.err
	case EntityRefNode:
		// An entity-reference name is emitted verbatim between "&" and ";", so it
		// cannot hold a character reference: a non-ASCII name has no faithful
		// US-ASCII serialization. Guard before the first write so no raw octet
		// leaks ahead of the sticky error.
		if d.rejectNonASCIIStr("entity reference name", n.Name()) {
			return d.err
		}
		d.writeString(out, "&")
		d.writeString(out, n.Name())
		d.writeString(out, ";")
		return d.err
	case TextNode:
		// Read-only serialization: use the internal slice without a copy.
		c := rawContent(n)
		if n.Name() == xmlTextNoEnc {
			// xmlTextNoEnc is a libxml2 marker (set on the node's name, not
			// its content) indicating the text should be emitted without
			// XML-escaping.  This is used during entity expansion
			// serialization where the replacement text is already encoded.
			if _, err := out.Write(c); err != nil {
				return err
			}
		} else if d.cdataText {
			// The parent element is a cdata-section-element: emit the text as
			// one or more CDATA sections instead of escaping it. A CDATA section
			// serializes a text node, so its content is normalized too (character
			// maps are not applied inside a CDATA section).
			//
			// A CDATA section cannot hold a character reference, so non-ASCII
			// content has no faithful US-ASCII serialization.
			if d.rejectNonASCIIBytes("CDATA section", c) {
				return d.err
			}
			if d.normalize {
				c = d.normForm.Bytes(c)
			}
			d.writeCDATASplit(out, c)
		} else {
			cm := d.charMap
			if d.normalize {
				c = d.normalizeContent(c)
				cm = nil
			}
			if err := escapeText(out, c, false, d.escapeNonASCII, d.asciiOutput, d.asciiReject(), d.rejectInvalidChars, d.xml11, cm); err != nil {
				return err
			}
		}
		return d.err // no recursing down
	case CDATASectionNode:
		// Mirrors xmlsave.c XML_CDATA_SECTION_NODE handling.
		// Splits content on "]]>" sequences so the output is well-formed.
		// Read-only serialization: use the internal slice without a copy. A CDATA
		// section serializes a text node, so its content is normalized when
		// requested (the delimiters are markup and stay verbatim).
		cdata := rawContent(n)
		// A CDATA section cannot hold a character reference, so non-ASCII content
		// has no faithful US-ASCII serialization.
		if d.rejectNonASCIIBytes("CDATA section", cdata) {
			return d.err
		}
		if d.normalize {
			cdata = d.normForm.Bytes(cdata)
		}
		d.writeCDATASplit(out, cdata)
		return d.err
	case ElementDeclNode:
		if edecl, ok := AsNode[*ElementDecl](n); ok {
			if err = d.dumpElementDecl(out, edecl); err != nil {
				return err
			}
		}
		return nil
	case AttributeDeclNode:
		if adecl, ok := AsNode[*AttributeDecl](n); ok {
			if err = d.dumpAttributeDecl(out, adecl); err != nil {
				return err
			}
		}
		return nil
	case EntityNode:
		if ent, ok := AsNode[*Entity](n); ok {
			if err = d.dumpEntityDecl(out, ent); err != nil {
				return err
			}
		}
		return nil
	case NotationNode:
		if nota, ok := AsNode[*Notation](n); ok {
			if err = d.dumpNotationDecl(out, nota); err != nil {
				return err
			}
		}
		return nil
	}

	// if it got here it's some sort of an element
	var name string
	var nslist []*Namespace
	nser, isNser := n.(Namespacer)
	if isNser {
		if prefix := nser.Prefix(); prefix != "" {
			name = prefix + ":" + nser.LocalName()
		} else {
			name = nser.LocalName()
		}
		nslist = nser.Namespaces()
		// When the element's active namespace is a default (empty prefix) whose
		// URI differs from a default declaration in its own nsDefs, the active
		// binding is authoritative — it qualifies the element — so drop the
		// conflicting declared default. Emitting both would put two xmlns
		// attributes on the start tag, which is not reparseable. reconcileOne
		// then synthesizes the single active default. A matching (equal-URI)
		// declared default is kept, so a normally parsed <e xmlns="u"/> still
		// declares its default exactly once.
		if active := nser.Namespace(); active != nil && active.prefix == "" && active.href != "" {
			nslist = dropConflictingDefaultNS(nslist, active.href)
		}
	} else {
		name = n.Name()
	}

	// The element name is emitted verbatim below. checkElementName rejects
	// names that are not well-formed XML QNames (whitespace, quotes, '>') and
	// records a sticky error without clobbering an earlier I/O failure.
	if !d.checkElementName(name) {
		return d.err
	}

	d.writeString(out, "<")
	d.writeString(out, name)

	if d.err != nil {
		return d.err
	}

	if len(nslist) > 0 {
		if err := d.dumpNsList(out, nslist); err != nil {
			return err
		}
	}

	// Keep the serialized fragment self-contained: declare any namespace this
	// element or its attributes use whose prefix was bound on an ancestor that
	// lies OUTSIDE the output (e.g. nodes an XSLT result tree grafts in from a
	// source document). Without this the emitted prefix is unbound and the
	// result cannot be reparsed. Reconciliation is purely additive — when the
	// prefix is already declared in scope with the same URI it emits nothing —
	// so a normal full-document dump is byte-identical to before.
	if isNser {
		if saved := d.reconcileNamespaces(out, n, nser, nslist); saved != nil {
			defer d.nsScopeRestore(saved)
		}
		if d.err != nil {
			return d.err
		}
	}

	if e, ok := n.(*Element); ok {
		// A per-list seen guard bounds a corrupt attribute chain: a cyclic
		// SetNextSibling, or a non-*Attribute successor (which would otherwise
		// leave attr unchanged and spin), terminates the walk. A normal
		// properties list is short and acyclic, so this never triggers there.
		seenAttrs := make(map[*docnode]struct{})
		for attr := e.properties; attr != nil; {
			akey := attr.baseDocNode()
			if _, dup := seenAttrs[akey]; dup {
				break
			}
			seenAttrs[akey] = struct{}{}
			// The attribute name is emitted verbatim. checkAttributeName
			// rejects names that would inject raw markup into the start tag.
			if !d.checkAttributeName(attr.Name()) {
				return d.err
			}
			d.writeString(out, " "+attr.Name()+`="`)
			if d.err != nil {
				return d.err
			}
			count := 0
			for achld := range Children(attr) {
				count++
				if achld.Type() == TextNode {
					if err := d.writeAttrValueContent(out, rawContent(achld)); err != nil {
						return err
					}
				} else {
					if err := d.writeNode(out, achld); err != nil {
						return err
					}
				}
			}
			d.writeString(out, `"`)
			a := attr.NextSibling()
			if a == nil {
				break
			}
			if at, ok := AsNode[*Attribute](a); ok {
				attr = at
			}
		}

		if child := e.FirstChild(); child == nil {
			if d.noEmpty {
				d.writeString(out, "></")
				d.writeString(out, name)
				d.writeString(out, ">")
			} else {
				d.writeString(out, "/>")
			}
			return d.err
		}
	}

	d.writeString(out, ">")

	// suppress-indentation: an element named in the suppress set (and its whole
	// subtree) is serialized without indentation even when format is on. cdata-
	// section-elements: an element named in the cdata set has its direct text
	// children emitted as CDATA sections. Both flags are saved/restored around
	// the children so sibling and ancestor state is unaffected.
	elemSuppressed := d.suppressDepth > 0 || matchesNameSet(d.suppressIndent, n) ||
		(d.format && hasTextlikeChild(n))
	effFormat := d.format && !elemSuppressed
	savedCDATA := d.cdataText
	d.cdataText = matchesNameSet(d.cdataElements, n)
	if elemSuppressed {
		d.suppressDepth++
	}

	if n.FirstChild() != nil {
		textOnly := effFormat && hasOnlyTextChildren(n)
		if effFormat && !textOnly {
			d.writeString(out, "\n")
			d.indent++
		}
		// Children applies the owned-boundary rule and a per-list seen guard, so
		// a corrupt (cyclic) child list terminates the descent instead of
		// spinning; this matches the doc-level and attribute loops above.
		for child := range Children(n) {
			if effFormat && !textOnly {
				d.writeIndent(out)
			}
			if err := d.writeNode(out, child); err != nil {
				d.cdataText = savedCDATA
				if elemSuppressed {
					d.suppressDepth--
				}
				return err
			}
			if effFormat && !textOnly {
				d.writeString(out, "\n")
			}
		}
		if effFormat && !textOnly {
			d.indent--
			d.writeIndent(out)
		}
	}

	d.cdataText = savedCDATA
	if elemSuppressed {
		d.suppressDepth--
	}

	d.writeString(out, "</")
	d.writeString(out, name)
	d.writeString(out, ">")

	return d.err
}

// reconcileNamespaces runs after an element's own xmlns declarations (nslist)
// are emitted. It records those declarations in the output namespace scope,
// then for the element's active namespace and every namespaced attribute emits
// an xmlns declaration for any prefix missing from scope (or bound there to a
// different URI). This keeps a serialized fragment parseable when it uses a
// prefix bound only on an ancestor that is not itself part of the output — the
// case an XSLT result tree creates by grafting in source-document nodes. It
// returns every binding it added to the scope so the caller can restore the
// scope after the element's children are serialized, or nil when the element
// neither declares nor uses a namespace (the plain-XML path stays alloc-free).
// dropConflictingDefaultNS returns nslist without any default-namespace
// declaration (empty prefix) whose URI differs from activeHref. The element's
// active default namespace is authoritative, so a conflicting declared default
// must not also be emitted — two xmlns attributes on one start tag are not
// reparseable. When no declaration conflicts, nslist is returned unchanged so the
// common path allocates nothing. Callers pass a non-empty activeHref.
func dropConflictingDefaultNS(nslist []*Namespace, activeHref string) []*Namespace {
	conflict := false
	for _, ns := range nslist {
		if ns.prefix == "" && ns.href != activeHref {
			conflict = true
			break
		}
	}
	if !conflict {
		return nslist
	}
	out := make([]*Namespace, 0, len(nslist))
	for _, ns := range nslist {
		if ns.prefix == "" && ns.href != activeHref {
			continue
		}
		out = append(out, ns)
	}
	return out
}

func (d *writeSession) reconcileNamespaces(out io.Writer, n Node, nser Namespacer, nslist []*Namespace) []nsSaved {
	var saved []nsSaved

	// The element's own declarations are already emitted; record them so a
	// prefix redeclared locally is not synthesized again below.
	for _, ns := range nslist {
		if ns.prefix == lexicon.PrefixXML || ns.prefix == lexicon.PrefixXMLNS {
			continue
		}
		saved = d.nsScopePush(ns.prefix, ns.href, saved)
	}

	// The element's active namespace. The empty prefix is allowed here so an
	// element whose active DEFAULT namespace was set (SetActiveNamespace("", uri))
	// without a matching declaration gets an xmlns="uri" synthesized.
	if ns := nser.Namespace(); ns != nil {
		saved = d.reconcileOne(out, ns.prefix, ns.href, true, saved)
	}

	// Namespaced attributes. The per-list seen guard bounds a corrupt attribute
	// chain, mirroring the emission loop in writeNode.
	if e, ok := n.(*Element); ok {
		seen := make(map[*docnode]struct{})
		for attr := e.properties; attr != nil; attr = attr.NextAttribute() {
			key := attr.baseDocNode()
			if _, dup := seen[key]; dup {
				break
			}
			seen[key] = struct{}{}
			if ans := attr.ns; ans != nil {
				saved = d.reconcileOne(out, ans.prefix, ans.href, false, saved)
			}
		}
	}
	return saved
}

// reconcileOne emits an xmlns declaration for a namespace used by the current
// element or one of its attributes when that prefix is not already in the
// output scope with the same URI, and records the new binding. The reserved
// xml/xmlns prefixes are never synthesized (xml is implicitly bound). The empty
// prefix (default namespace) is synthesized only for the element's own active
// namespace (isElement) as xmlns="href"; an attribute's empty prefix is skipped
// because an unprefixed attribute is never in a namespace. Appends the recorded
// binding to saved and returns it.
func (d *writeSession) reconcileOne(out io.Writer, prefix, href string, isElement bool, saved []nsSaved) []nsSaved {
	if prefix == lexicon.PrefixXML || prefix == lexicon.PrefixXMLNS || href == "" {
		return saved
	}
	if prefix == "" && !isElement {
		return saved
	}
	if d.nsScope != nil {
		if cur, ok := d.nsScope[prefix]; ok && cur == href {
			return saved
		}
	}
	if prefix == "" {
		d.writeString(out, " xmlns")
	} else {
		// The prefix is emitted verbatim; reject one that is not a valid NCName so
		// a crafted prefix cannot inject raw markup into the start tag.
		if !d.checkNamespacePrefix(prefix) {
			return saved
		}
		d.writeString(out, " xmlns:")
		d.writeString(out, prefix)
	}
	d.writeString(out, `="`)
	if d.err == nil {
		if err := escapeAttrValue(out, []byte(href), d.escapeNonASCII, d.asciiOutput, d.asciiReject(), d.rejectInvalidChars, d.xml11, nil); err != nil {
			d.err = err
		}
	}
	d.writeString(out, `"`)
	return d.nsScopePush(prefix, href, saved)
}

// seedNSScope initializes the output namespace scope from InheritedNamespaces,
// copying the caller's map so it is never mutated during serialization. It is a
// no-op (and allocates nothing) when no inherited bindings were supplied.
func (d *writeSession) seedNSScope() {
	if len(d.initialNSScope) == 0 {
		return
	}
	d.nsScope = make(map[string]string, len(d.initialNSScope))
	maps.Copy(d.nsScope, d.initialNSScope)
}

// nsScopePush binds prefix to href in the output namespace scope, appending the
// prior binding to saved so nsScopeRestore can revert it.
func (d *writeSession) nsScopePush(prefix, href string, saved []nsSaved) []nsSaved {
	if d.nsScope == nil {
		d.nsScope = make(map[string]string)
	}
	old, had := d.nsScope[prefix]
	saved = append(saved, nsSaved{prefix: prefix, href: old, had: had})
	d.nsScope[prefix] = href
	return saved
}

// nsScopeRestore reverts the bindings recorded in saved. It must run in reverse:
// a single element can push the same prefix twice (a local redeclaration plus an
// attribute masking it with a different URI), and only last-in-first-out restore
// yields the prior binding.
func (d *writeSession) nsScopeRestore(saved []nsSaved) {
	for _, s := range slices.Backward(saved) {
		if s.had {
			d.nsScope[s.prefix] = s.href
			continue
		}
		delete(d.nsScope, s.prefix)
	}
}

// nodeExpandedName returns the expanded {uri}local name of a node (Clark
// notation, with an explicit empty namespace as "{}local"), used to match
// against the cdata-section-elements and suppress-indentation name sets. Matching
// is by exact expanded name — a no-namespace element must not match a
// namespaced one with the same local name.
func nodeExpandedName(n Node) string {
	type uriLocal interface {
		URI() string
		LocalName() string
	}
	ul, ok := n.(uriLocal)
	if !ok {
		return ClarkName("", n.Name())
	}
	return ClarkName(ul.URI(), ul.LocalName())
}

// matchesNameSet reports whether n's exact expanded name is present in set. An
// empty set never matches.
func matchesNameSet(set map[string]struct{}, n Node) bool {
	if len(set) == 0 {
		return false
	}
	_, ok := set[nodeExpandedName(n)]
	return ok
}
