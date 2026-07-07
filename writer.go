package helium

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"

	henc "github.com/lestrrat-go/helium/internal/encoding"
	"github.com/lestrrat-go/helium/internal/lexicon"
	"github.com/lestrrat-go/helium/internal/xmlchar"
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
	xml11          bool // true when the document declares XML 1.1: restricted control chars serialize as decimal character references
	isXHTML        bool
	encoding       string // document encoding, used for XHTML meta injection
	indent         int    // current indent depth (used when format is true)
	err            error  // sticky write error; once set, further writes are skipped
	// cdataText reports that the element currently being serialized is a
	// cdata-section-element, so its direct text children are emitted as CDATA
	// sections rather than escaped text.
	cdataText bool
	// suppressDepth > 0 means the current subtree descends from a
	// suppress-indentation element, so indentation is disabled for it even when
	// format is enabled.
	suppressDepth int
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
		s.check(fmt.Errorf("helium: reserved element name %q: namespace declarations must use DeclareNamespace", name))
		return false
	}
	if xmlchar.IsValidQName(name) {
		return true
	}
	s.check(fmt.Errorf("helium: invalid element name %q", name))
	return false
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
		s.check(fmt.Errorf("helium: reserved attribute name %q: namespace declarations must use DeclareNamespace", name))
		return false
	}
	if xmlchar.IsValidQName(name) {
		return true
	}
	s.check(fmt.Errorf("helium: invalid attribute name %q", name))
	return false
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
		s.check(fmt.Errorf("helium: reserved namespace prefix %q must not be declared", prefix))
		return false
	}
	if prefix == "" || xmlchar.IsValidNCName(prefix) {
		return true
	}
	s.check(fmt.Errorf("helium: invalid namespace prefix %q", prefix))
	return false
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

// IndentString sets the string used for each indent level.
func (w Writer) IndentString(s string) Writer {
	w.indentString = s
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
func (w Writer) CharacterMap(m map[rune]string) Writer {
	w.charMap = m
	return w
}

// CDATASectionElements names the elements (each as an expanded {uri}local name)
// whose direct text children are serialized as CDATA sections instead of escaped
// text (the cdata-section-elements serialization parameter). Matching is by exact
// expanded name. A nil or empty map disables the feature.
func (w Writer) CDATASectionElements(m map[string]struct{}) Writer {
	w.cdataElements = m
	return w
}

// SuppressIndentElements names the elements (each as an expanded {uri}local name)
// whose subtree is serialized without indentation even when Format is enabled
// (the suppress-indentation serialization parameter). Matching is by exact
// expanded name. A nil or empty map disables the feature.
func (w Writer) SuppressIndentElements(m map[string]struct{}) Writer {
	w.suppressIndent = m
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
func (w Writer) OutputVersion(v string) Writer {
	w.outputVersion = v
	return w
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
	if s.indentString == "" {
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
	if doc, ok := node.(*Document); ok {
		return d.writeDoc(out, doc)
	}
	s := writeSession{Writer: d, escapeNonASCII: !d.noEscapeNonASCII}
	// A bare element carries no document version, so only an explicit
	// OutputVersion("1.1") override enables XML 1.1 serialization here; without
	// it, output stays byte-identical to the prior behavior.
	s.xml11 = d.outputVersion == "1.1"
	return s.writeNode(out, node)
}

func (d Writer) writeDoc(out io.Writer, doc *Document) error {
	s := writeSession{Writer: d}
	// An XML 1.1 document (or an OutputVersion("1.1") override) may carry
	// restricted control characters; serialize them as decimal character
	// references. XML 1.0 output is unaffected.
	s.xml11 = d.effectiveVersion(doc) == "1.1"

	// Mirrors libxml2's xmlSaveWriteText: when output encoding is UTF-8
	// (no encoder), escape non-ASCII chars 0x80-0xDF as numeric refs.
	// When an encoder is present, pass them through for re-encoding.
	s.escapeNonASCII = !d.noEscapeNonASCII
	if enc := doc.encoding; enc != "" {
		lower := strings.ToLower(enc)
		if lower != "utf-8" && lower != encUTF8 && lower != "us-ascii" && lower != "ascii" {
			if e := henc.Load(enc); e != nil {
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
	s.encoding = doc.encoding
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
	d.writeString(out, `<?xml version="`)
	d.writeString(out, d.effectiveVersion(doc)+`"`)

	if encoding := doc.encoding; encoding != "" {
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
			d.check(errors.New("helium: comment content must not contain \"--\" or end with \"-\""))
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
				d.check(errors.New("helium: invalid PI target"))
				return d.err
			}
			// PI data must not contain "?>", which would terminate the PI early.
			if strings.Contains(pi.data, "?>") {
				// check() keeps the first sticky error, so an earlier I/O failure
				// is not clobbered by this validation error.
				d.check(errors.New("helium: PI content must not contain \"?>\""))
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
			// one or more CDATA sections instead of escaping it.
			d.writeCDATASplit(out, c)
		} else {
			if err := escapeText(out, c, false, d.escapeNonASCII, d.rejectInvalidChars, d.xml11, d.charMap); err != nil {
				return err
			}
		}
		return d.err // no recursing down
	case CDATASectionNode:
		// Mirrors xmlsave.c XML_CDATA_SECTION_NODE handling.
		// Splits content on "]]>" sequences so the output is well-formed.
		// Read-only serialization: use the internal slice without a copy.
		d.writeCDATASplit(out, rawContent(n))
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
	if nser, ok := n.(Namespacer); ok {
		if prefix := nser.Prefix(); prefix != "" {
			name = prefix + ":" + nser.LocalName()
		} else {
			name = nser.LocalName()
		}
		nslist = nser.Namespaces()
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
					if err := escapeAttrValue(out, rawContent(achld), d.escapeNonASCII, d.rejectInvalidChars, d.xml11, d.charMap); err != nil {
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
	elemSuppressed := d.suppressDepth > 0 || matchesNameSet(d.suppressIndent, n)
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
