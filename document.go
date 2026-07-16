package helium

import (
	"bytes"
	"errors"
	"fmt"
	"strings"

	"github.com/lestrrat-go/helium/enum"
	"github.com/lestrrat-go/helium/internal/lexicon"
	"github.com/lestrrat-go/helium/internal/pool"
)

type DocumentStandaloneType int

const (
	StandaloneInvalidValue = -99
	StandaloneExplicitYes  = 1
	StandaloneExplicitNo   = 0
	StandaloneNoXMLDecl    = -1
	StandaloneImplicitNo   = -2
)

// DocProperties is a bitmask of document properties, mirroring
// libxml2's xmlDocProperties. Properties are set by the parser
// or by user code to record how the document was produced and
// what validations it passed.
type DocProperties int

const (
	DocWellFormed DocProperties = 1 << iota // document is XML well-formed
	DocNSValid                              // document is namespace-valid
	DocOld10                                // parsed with XML 1.0 (4th edition or earlier)
	DocDTDValid                             // DTD validation was successful
	DocXInclude                             // XInclude substitution was done
	DocUserBuilt                            // built via API, not by parsing
	DocInternal                             // built for internal processing
	DocHTML                                 // parsed or built as HTML
)

// Document represents an XML document (libxml2: xmlDoc).
type Document struct {
	docnode
	version    string
	encoding   string
	standalone DocumentStandaloneType
	url        string // document URI for base URI resolution (mirrors libxml2's xmlDoc.URL)
	properties DocProperties

	intSubset *DTD
	extSubset *DTD
	ids       map[string]*Element
	idsSkip   bool

	// standaloneNormAttrs records specified attributes whose value was changed by
	// tokenized-type normalization driven by an external-subset ATTLIST
	// declaration. Populated by the parser (which alone holds the pre-normalization
	// value) and consulted by DTD validation for the VC: Standalone Document
	// Declaration in a standalone="yes" document. See valid.go.
	standaloneNormAttrs []standaloneNormAttr

	// Slab allocators for high-frequency node types.
	// These reduce per-node heap allocation overhead by allocating
	// nodes in chunks and handing them out one at a time.
	// Chunks are obtained from global pools and returned on Free().
	elemSlab []Element
	textSlab []Text
	nsSlab   []Namespace
	attrSlab []Attribute

	// Track allocated chunks for pool return.
	elemChunks []*[slabSize]Element
	textChunks []*[slabSize]Text
	nsChunks   []*[slabSize]Namespace
	attrChunks []*[slabSize]Attribute

	// Slab allocator for parsed text content bytes.
	textContentSlab   []byte
	textContentChunks []*[textContentSlabSize]byte

	// slabEscaped records that at least one slab-backed node owned by this
	// document was linked into another document (an XInclude/XSLT-style
	// cross-document move). Once set, Free must NOT return this document's chunks
	// to the pool: a moved node still points into one of them, and recycling it
	// would let a later parse overwrite the live node. Set by the tree-insertion
	// paths (node.go noteCrossDocumentEscape).
	slabEscaped bool
}

// NewDefaultDocument creates a minimal user-built document with version "1.0",
// no encoding, and implicit-no standalone (libxml2: xmlNewDoc).
func NewDefaultDocument() *Document {
	doc := NewDocument("1.0", "", StandaloneImplicitNo)
	doc.properties |= DocUserBuilt
	return doc
}

// NewHTMLDocument creates a new HTML document (HTMLDocumentNode type).
func NewHTMLDocument() *Document {
	doc := &Document{
		standalone: StandaloneNoXMLDecl,
		properties: DocHTML,
	}
	doc.etype = HTMLDocumentNode
	doc.name = "(document)"
	return doc
}

// NewDocument allocates an empty XML Document with the given XML
// declaration version, character encoding, and standalone status. The
// returned document owns no children yet; use CreateElement and the other
// Create* methods to build a tree, then attach the root via AddChild.
func NewDocument(version, encoding string, standalone DocumentStandaloneType) *Document {
	doc := &Document{
		encoding:   encoding,
		standalone: standalone,
		version:    version,
	}

	doc.etype = DocumentNode
	doc.name = "(document)"
	return doc
}

// Free returns pooled slab chunks for reuse by future parse calls.
// This is optional — if not called, GC handles cleanup normally.
// Calling Free on a document that is still in use causes undefined behavior.
//
// If any of this document's slab-backed nodes was moved into another document
// (via AddChild/AddSibling/Replace), Free does NOT recycle its chunks: a moved
// node still references one of them, so returning it to the pool would let a
// later parse overwrite the live node. In that case Free is a no-op and GC
// reclaims the chunks once they are no longer referenced.
func (d *Document) Free() {
	if d.slabEscaped {
		return
	}
	for _, c := range d.elemChunks {
		elemChunkPool.Put(c)
	}
	for _, c := range d.textChunks {
		textChunkPool.Put(c)
	}
	for _, c := range d.nsChunks {
		nsChunkPool.Put(c)
	}
	for _, c := range d.attrChunks {
		attrChunkPool.Put(c)
	}
	for _, c := range d.textContentChunks {
		textContentChunkPool.Put(c)
	}
	d.elemChunks = nil
	d.textChunks = nil
	d.nsChunks = nil
	d.attrChunks = nil
	d.textContentChunks = nil
	d.elemSlab = nil
	d.textSlab = nil
	d.nsSlab = nil
	d.attrSlab = nil
	d.textContentSlab = nil
}

// AddChild appends cur as the last child of the document, detaching it from any
// previous parent first. It returns an error if cur is nil or if the insertion
// would create a cycle.
func (d *Document) AddChild(cur Node) error {
	return addChild(d, cur)
}

// AppendText appends b as a Text child of the document, merging into a trailing
// Text node when possible.
func (d *Document) AppendText(b []byte) error {
	return appendText(d, b)
}

func (d *Document) AddSibling(_ Node) error {
	return errors.New("can't add sibling to a document")
}

func (d *Document) SetTreeDoc(doc *Document) {
	setTreeDoc(d, doc)
}

func (d *Document) Encoding() string {
	// In order to differentiate between a document with explicit
	// encoding in the XML declaration and one without, the XML dump
	// routine must check for d.encoding == "", and not Encoding()
	if enc := d.encoding; enc != "" {
		return d.encoding
	}
	return "utf8"
}

func (d *Document) SetEncoding(enc string) {
	d.encoding = enc
}

// RawEncoding returns the document's encoding exactly as recorded, WITHOUT the
// "utf8" default that Encoding() synthesizes for documents whose XML
// declaration omitted an encoding. An empty result means the source had no
// encoding declaration. This is the value the XML serializer consults (it emits
// encoding="..." only when the raw encoding is non-empty), so a faithful copy
// of a document — e.g. an xsl:strip-space copy — must propagate this rather
// than Encoding(), which would synthesize a spurious encoding="utf8".
func (d *Document) RawEncoding() string {
	return d.encoding
}

func (d *Document) Standalone() DocumentStandaloneType {
	return d.standalone
}

// SkipIDs reports whether ID attribute interning was skipped when this
// document was produced (mirrors the parser's SkipIDs option). When true,
// GetElementByID and fn:id always return no match without an O(n) tree walk.
func (d *Document) SkipIDs() bool {
	return d.idsSkip
}

// SetSkipIDs sets the document's ID-skip state. This is used when producing a
// derived document (e.g. an xsl:strip-space copy) that must mirror the source's
// ID semantics.
//
// The ID-skip state is authoritative: while it is true, GetElementByID (and
// therefore fn:id) resolves NO ids and returns nil — even if the document
// already has a populated ID table from a normal parse. Setting it back to
// false restores normal resolution against the existing table (or the lazy
// tree walk for API-built documents).
func (d *Document) SetSkipIDs(v bool) {
	d.idsSkip = v
}

func (d *Document) Version() string {
	return d.version
}

// SetVersion sets the document's XML declaration version (e.g. "1.0" or "1.1").
// Serialization consults this to decide whether XML 1.1 restricted control
// characters are emitted as character references rather than rejected.
func (d *Document) SetVersion(v string) {
	d.version = v
}

// URL returns the document URI, used as the base for relative URI resolution.
// This mirrors libxml2's xmlDoc.URL field. An empty URL is valid for in-memory
// documents; callers that need a non-empty diagnostic label must provide their
// own fallback.
func (d *Document) URL() string {
	return d.url
}

// SetURL sets the document URI.
func (d *Document) SetURL(url string) {
	d.url = url
}

// Properties returns the document's property flags.
func (d *Document) Properties() DocProperties {
	return d.properties
}

// SetProperties replaces the document's property flags.
func (d *Document) SetProperties(p DocProperties) {
	d.properties = p
}

// HasProperty reports whether all bits in p are set.
func (d *Document) HasProperty(p DocProperties) bool {
	return d.properties&p == p
}

func (d *Document) IntSubset() *DTD {
	return d.intSubset
}

// RemoveInternalSubset detaches the document's internal-subset DTD (if any) from
// the child list and clears the association, so a subsequent
// CreateInternalSubset can install a fresh one in its place. It is a no-op when
// no internal subset is present.
func (d *Document) RemoveInternalSubset() {
	if d.intSubset == nil {
		return
	}
	unlinkNode(d.intSubset)
	d.intSubset = nil
}

func (d *Document) ExtSubset() *DTD {
	return d.extSubset
}

func (d *Document) Replace(_ ...Node) error {
	return ErrInvalidOperation
}

// DocumentElement returns the root element of the document, or nil if none exists.
//
// The child scan goes through Children, so it applies the owned-boundary rule
// and a per-list seen guard: a corrupt document child list (a cyclic sibling
// pointer before the root) terminates the scan instead of hanging, which would
// otherwise stall DTD/XSD validation before the traversal cycle guards run.
func (d *Document) DocumentElement() *Element {
	for n := range Children(d) {
		if elem, ok := AsNode[*Element](n); ok {
			return elem
		}
	}
	return nil
}

func (d *Document) SetDocumentElement(root MutableNode) error {
	if d == nil {
		return ErrNilNode
	}

	// Reject a nil or typed-nil operand BEFORE any root.Type() dereference so the
	// call returns ErrNilNode instead of panicking.
	if isNilNode(root) {
		return ErrNilNode
	}

	// The document element must be a concrete *Element. Accepting any node kind
	// here let a Text/Comment/DTD/NamespaceDecl node become the root, producing a
	// document that is not well formed. Checking the concrete type (not root.Type())
	// also rejects a non-element node that merely REPORTS ElementNode — e.g. an
	// XIncludeMarker constructed with ElementNode — which DocumentElement() would
	// never return, leaving the document element effectively nil.
	if _, ok := AsNode[*Element](root); !ok {
		return fmt.Errorf("%w: document element must be an element node, got %s", ErrInvalidOperation, root.Type())
	}

	// Do NOT link root to d here. Let AddChild/Replace perform the linking AFTER
	// their cycle/self preflight so a rejected insertion leaves the candidate and
	// the tree untouched. The scan goes through Children so a corrupt document
	// child list (a cyclic sibling pointer before the root) terminates instead of
	// hanging.
	var old Node
	for n := range Children(d) {
		if n.Type() == ElementNode {
			old = n
			break
		}
	}

	if old == nil {
		if err := d.AddChild(root); err != nil {
			return err
		}
	} else {
		if err := old.(MutableNode).Replace(root); err != nil { //nolint:forcetypeassert
			return err
		}
	}
	return nil
}

func (d *Document) CreateReference(name string) (*EntityRef, error) {
	n, err := d.CreateCharRef(name)
	if err != nil {
		return nil, err
	}

	ent, ok := d.GetEntity(n.name)
	if ok {
		n.content = []byte(ent.content)
		// Original code says:
		// The parent pointer in entity is a DTD pointer and thus is NOT
		// updated.  Not sure if this is 100% correct.
		setFirstChild(n, ent)
		setLastChild(n, ent)
	}

	return n, nil
}

func (d *Document) CreateAttribute(name, value string, ns *Namespace) (attr *Attribute, err error) {
	if strings.ContainsRune(name, ':') {
		return nil, fmt.Errorf("attribute name %q contains a colon: use CreateAttribute with a local name and Namespace parameter", name)
	}
	var n Node
	if d != nil {
		attr = d.allocAttribute(name, ns)
	} else {
		attr = newAttribute(name, ns)
	}
	if value != "" {
		n, err = d.stringToNodeList(value)
		if err != nil {
			attr = nil
			return
		}

		setFirstChild(attr, n)
		for n != nil {
			n.baseDocNode().parent = attr
			x := n.NextSibling()
			if x == nil {
				setLastChild(attr, n)
			}
			n = x
		}
	}
	return attr, nil
}

func (d *Document) CreateNamespace(prefix, uri string) (*Namespace, error) {
	var ns *Namespace
	if d != nil {
		ns = d.allocNamespace()
	} else {
		ns = &Namespace{}
	}
	ns.prefix = prefix
	ns.href = uri
	ns.etype = NamespaceNode
	ns.context = d
	return ns, nil
}

func (d *Document) allocNamespace() *Namespace {
	if len(d.nsSlab) == 0 {
		chunk := nsChunkPool.Get()
		d.nsChunks = append(d.nsChunks, chunk)
		d.nsSlab = chunk[:]
	}
	ns := &d.nsSlab[0]
	*ns = Namespace{}
	d.nsSlab = d.nsSlab[1:]
	return ns
}

func (d *Document) allocAttribute(name string, ns *Namespace) *Attribute {
	if len(d.attrSlab) == 0 {
		chunk := attrChunkPool.Get()
		d.attrChunks = append(d.attrChunks, chunk)
		d.attrSlab = chunk[:]
	}
	attr := &d.attrSlab[0]
	*attr = Attribute{}
	d.attrSlab = d.attrSlab[1:]
	attr.etype = AttributeNode
	attr.name = name
	attr.ns = ns
	return attr
}

func (d *Document) createLiteralAttribute(name, value string, ns *Namespace) *Attribute {
	var attr *Attribute
	if d != nil {
		attr = d.allocAttribute(name, ns)
	} else {
		attr = newAttribute(name, ns)
	}

	if value == "" {
		return attr
	}

	t := d.CreateText([]byte(value))
	setFirstChild(attr, t)
	setLastChild(attr, t)
	t.parent = attr
	return attr
}

func (d *Document) CreatePI(target, data string) *ProcessingInstruction {
	pi := &ProcessingInstruction{
		target: target,
		data:   data,
	}
	pi.doc = d
	return pi
}

func (d *Document) CreateDTD() (*DTD, error) {
	dtd := newDTD()
	dtd.doc = d
	return dtd, nil
}

func (d *Document) InternalSubset() (*DTD, error) {
	// equiv: xmlGetIntSubset (tree.c)
	if d.intSubset == nil {
		return nil, errors.New("no internal subset is associated with this document")
	}
	return d.intSubset, nil
}

func (d *Document) CreateInternalSubset(name, externalID, systemID string) (*DTD, error) {
	// equiv: xmlCreateIntSubset (tree.c)
	if _, err := d.InternalSubset(); err == nil {
		return nil, errors.New("document " + d.name + " already has an internal subset")
	}

	cur, err := d.CreateDTD()
	if err != nil {
		return nil, err
	}

	cur.name = name
	cur.externalID = externalID
	cur.systemID = systemID

	if d == nil {
		return cur, nil
	}

	d.intSubset = cur
	cur.parent = d
	cur.doc = d

	// Insert before the root element (matching libxml2's xmlCreateIntSubset).
	// If no children exist yet, just append. The scan goes through Children so a
	// corrupt document child list terminates instead of hanging.
	var root Node
	for c := range Children(d) {
		if c.Type() == ElementNode {
			root = c
			break
		}
	}
	if root == nil {
		if err := d.AddChild(cur); err != nil {
			return nil, err
		}
	} else {
		// Insert cur before root.
		cur.next = root
		if prev := root.PrevSibling(); prev != nil {
			prev.baseDocNode().next = cur
			cur.prev = prev
		} else {
			setFirstChild(d, cur)
		}
		root.baseDocNode().prev = cur
	}

	return cur, nil
}

const slabSize = 256
const textContentSlabSize = 64 * 1024

var (
	elemChunkPool        = pool.New(func() *[slabSize]Element { return new([slabSize]Element) }, nil)
	textChunkPool        = pool.New(func() *[slabSize]Text { return new([slabSize]Text) }, nil)
	nsChunkPool          = pool.New(func() *[slabSize]Namespace { return new([slabSize]Namespace) }, nil)
	attrChunkPool        = pool.New(func() *[slabSize]Attribute { return new([slabSize]Attribute) }, nil)
	textContentChunkPool = pool.New(func() *[textContentSlabSize]byte { return new([textContentSlabSize]byte) }, nil)
)

// CreateElement allocates a new Element named name and owned by this
// document, but does not attach it to the tree; the caller must insert it
// via AddChild or a sibling/parent method. When d is non-nil the element is
// drawn from the document's slab allocator, so the element must not outlive a
// call to Document.Free. A nil receiver allocates a standalone element with no
// owning document.
func (d *Document) CreateElement(name string) *Element {
	var e *Element
	if d != nil {
		e = d.allocElement()
	} else {
		e = newElement(name)
	}
	e.name = name
	e.etype = ElementNode
	e.doc = d
	return e
}

func (d *Document) allocElement() *Element {
	if len(d.elemSlab) == 0 {
		chunk := elemChunkPool.Get()
		d.elemChunks = append(d.elemChunks, chunk)
		d.elemSlab = chunk[:]
	}
	e := &d.elemSlab[0]
	*e = Element{}
	d.elemSlab = d.elemSlab[1:]
	return e
}

// CreateText allocates a new Text node holding a copy of value, owned by this
// document, without attaching it to the tree. When d is non-nil both the node
// and its content bytes come from the document's slab allocators and must not
// outlive a call to Document.Free. A nil receiver allocates a standalone text
// node.
func (d *Document) CreateText(value []byte) *Text {
	var e *Text
	if d != nil {
		e = d.allocText()
	} else {
		e = &Text{}
	}
	e.etype = TextNode
	if d != nil {
		e.content = d.allocTextContent(len(value))
		copy(e.content, value)
	} else {
		e.content = make([]byte, len(value))
		copy(e.content, value)
	}
	e.name = textNodeName
	e.doc = d
	return e
}

func (d *Document) allocTextContent(size int) []byte {
	if size <= 0 {
		return nil
	}
	if size > textContentSlabSize {
		return make([]byte, size)
	}
	if len(d.textContentSlab) < size {
		chunk := textContentChunkPool.Get()
		d.textContentChunks = append(d.textContentChunks, chunk)
		d.textContentSlab = chunk[:]
	}
	buf := d.textContentSlab[:size:size]
	d.textContentSlab = d.textContentSlab[size:]
	return buf
}

func (d *Document) growOwnedTextContent(cur []byte, extra int) []byte {
	if extra <= 0 {
		return cur
	}

	need := len(cur) + extra
	if cap(cur) >= need {
		return cur
	}

	newCap := max(cap(cur)*2, need, 64)

	next := d.allocTextContent(newCap)
	next = next[:len(cur)]
	copy(next, cur)
	return next
}

func (d *Document) allocText() *Text {
	if len(d.textSlab) == 0 {
		chunk := textChunkPool.Get()
		d.textChunks = append(d.textChunks, chunk)
		d.textSlab = chunk[:]
	}
	t := &d.textSlab[0]
	*t = Text{}
	d.textSlab = d.textSlab[1:]
	return t
}

// CreateComment allocates a new Comment node holding value, owned by this
// document, without attaching it to the tree.
func (d *Document) CreateComment(value []byte) *Comment {
	e := newComment(value)
	e.doc = d
	return e
}

// CreateCDATASection mirrors xmlNewCDataBlock in libxml2's tree.c.
func (d *Document) CreateCDATASection(value []byte) *CDATASection {
	e := newCDATASection(value)
	e.doc = d
	return e
}

// CreateElementContent builds an ElementContent leaf node of the given type for
// use in a DTD content model: an element-reference leaf ([ElementContentElement],
// which requires a name) or a #PCDATA leaf ([ElementContentPCDATA], which
// requires an empty name). To build a sequence (,) or choice (|) node use
// [Document.CreateElementContentSeq] / [Document.CreateElementContentChoice],
// which attach and validate the two children; a bare sequence/choice leaf built
// here has nil children and cannot be serialized. The node defaults to the
// "once" occurrence; use [ElementContent.SetOccurrence] to change it.
func (d *Document) CreateElementContent(name string, etype ElementContentType) (*ElementContent, error) {
	e, err := newElementContent(name, etype)
	if err != nil {
		return nil, err
	}
	return e, nil
}

// CreateElementContentSeq builds a sequence (,) content node with the two given
// children and occurrence indicator. Both children must be non-nil and
// structurally complete; otherwise it returns an error. This is the safe way to
// compose a multi-particle content model — the resulting node can be serialized
// and matched without a nil dereference.
func (d *Document) CreateElementContentSeq(c1, c2 *ElementContent, occur ElementContentOccur) (*ElementContent, error) {
	return newElementContentBinary(ElementContentSeq, c1, c2, occur)
}

// CreateElementContentChoice builds a choice (|) content node with the two given
// children and occurrence indicator. Both children must be non-nil and
// structurally complete; otherwise it returns an error. This is the safe way to
// compose a multi-particle content model — the resulting node can be serialized
// and matched without a nil dereference.
func (d *Document) CreateElementContentChoice(c1, c2 *ElementContent, occur ElementContentOccur) (*ElementContent, error) {
	return newElementContentBinary(ElementContentOr, c1, c2, occur)
}

// GetEntity looks up a general entity declaration by name, searching the
// internal subset first and then the external subset (the latter is skipped
// for standalone="yes" documents). found reports whether a declaration was
// located.
func (d *Document) GetEntity(name string) (ent *Entity, found bool) {
	if ints := d.intSubset; ints != nil {
		ent, found = ints.LookupEntity(name)
		if found {
			return
		}
	}

	if d.standalone != StandaloneExplicitYes {
		if exts := d.extSubset; exts != nil {
			ent, found = exts.LookupEntity(name)
			return
		}
	}

	return
}

// GetParameterEntity looks up a parameter entity declaration by name,
// searching the internal subset first and then the external subset (the
// latter is skipped for standalone="yes" documents). The boolean result
// reports whether a declaration was located.
func (d *Document) GetParameterEntity(name string) (*Entity, bool) {
	if ints := d.intSubset; ints != nil {
		if ent, ok := ints.LookupParameterEntity(name); ok {
			return ent, true
		}
	}

	if d.standalone != StandaloneExplicitYes {
		if exts := d.extSubset; exts != nil {
			return exts.LookupParameterEntity(name)
		}
	}

	return nil, false
}

var errElementDeclNotFound = errors.New("element declaration not found")

// IsMixedElement reports whether the element named name is declared with mixed
// (or EMPTY/ANY) content in the document's internal subset. It returns an error
// if no element declaration is found for name.
func (d *Document) IsMixedElement(name string) (bool, error) {
	if d.intSubset == nil {
		return false, errElementDeclNotFound
	}

	edecl, ok := d.intSubset.GetElementDesc(name)
	if !ok {
		return false, errElementDeclNotFound
	}

	switch edecl.decltype {
	case enum.UndefinedElementType:
		return false, errElementDeclNotFound
	case enum.ElementElementType:
		return false, nil
	case enum.EmptyElementType, enum.AnyElementType, enum.MixedElementType:
		/*
		 * return 1 for EMPTY since we want VC error to pop up
		 * on <empty>     </empty> for example
		 */
		return true, nil
	}
	return true, nil
}

// elementDeclType returns the declared content-model type of the element named
// name and reports whether a declaration was found. It searches the internal
// subset first and consults the external subset only when the internal subset
// has no declaration for name (mirroring libxml2 areBlanks' two-subset lookup,
// where doc->extSubset is checked only if the doc->intSubset lookup is NULL). An
// internal-subset placeholder declaration (UndefinedElementType) counts as found
// and stops the search, exactly as a non-NULL elemDecl does in libxml2.
//
// Unlike IsMixedElement (which collapses EMPTY/ANY/MIXED into a single "mixed"
// bool for VC-error callers, mirroring xmlIsMixedElement), this returns the raw
// content-model type so whitespace classification can apply libxml2 areBlanks'
// own decl switch, in which EMPTY and UNDEFINED fall through to the heuristic
// rather than being treated as mixed.
func (d *Document) elementDeclType(name string) (enum.ElementType, bool) {
	for _, dtd := range []*DTD{d.intSubset, d.extSubset} {
		if dtd == nil {
			continue
		}
		if edecl, ok := dtd.GetElementDesc(name); ok {
			return edecl.decltype, true
		}
	}
	return enum.UndefinedElementType, false
}

/*
 * @doc:  the document
 * @value:  the value of the attribute
 *
 * Parse the value string and build the node list associated. Should
 * produce a flat tree with only TEXTs and ENTITY_REFs.
 * Returns a pointer to the first child
 */
func (d *Document) stringToNodeList(value string) (ret Node, err error) {
	// Fast path: no entity references — create a single text node directly.
	if strings.IndexByte(value, '&') < 0 {
		return d.CreateText([]byte(value)), nil
	}

	rdr := strings.NewReader(value)
	buf := bytes.Buffer{}
	var last Node
	var charval int32
	var r rune
	var r2 rune
	for rdr.Len() > 0 {
		r, _, err = rdr.ReadRune()
		if err != nil {
			return
		}

		// if this is not any sort of an entity , just go
		if r != '&' {
			_, _ = buf.WriteRune(r)
			continue
		}

		// well, at least the first rune sure looks like an entity, see what
		// else we have.
		r, _, err = rdr.ReadRune()
		if err != nil {
			return
		}

		if r == '#' {
			r2, _, err = rdr.ReadRune()
			if err != nil {
				return
			}

			var accumulator func(int32, rune) (int32, error)
			if r2 == 'x' {
				accumulator = accumulateHexCharRef
			} else {
				if err2 := rdr.UnreadRune(); err2 != nil {
					err = err2
					return
				}
				accumulator = accumulateDecimalCharRef
			}
			for {
				r, _, err = rdr.ReadRune()
				if err != nil {
					return
				}
				if r == ';' {
					break
				}
				charval, err = accumulator(charval, r)
				if err != nil {
					return
				}
			}
		} else {
			if err2 := rdr.UnreadRune(); err2 != nil {
				err = err2
				return
			}
			entbuf := bytes.Buffer{}
			for rdr.Len() > 0 {
				r, _, err = rdr.ReadRune()
				if err != nil {
					return
				}
				if r == ';' {
					break
				}
				_, _ = entbuf.WriteRune(r)
			}

			if r != ';' {
				err = errors.New("entity was unterminated (could not find terminating semicolon)")
				return
			}

			val := entbuf.String()

			// Predefined entities (amp, lt, gt, apos, quot) are not
			// document-dependent, so resolve them up front. This ensures
			// they inline correctly even when the document (and thus its
			// internal/external subset) is nil.
			ent, err2 := resolvePredefinedEntity(val)
			ok := err2 == nil
			if !ok {
				ent, ok = d.GetEntity(val)
			}

			// Predefined entities are inlined; all others (resolved or not)
			// become entity reference nodes. This matches libxml2's
			// xmlNodeParseAttValue behavior in tree.c.
			if ok && ent.EntityType() == enum.InternalPredefinedEntity {
				_, _ = buf.Write(ent.Content())
			} else {
				// flush the buffer so far
				if buf.Len() > 0 {
					node := d.CreateText(buf.Bytes())
					buf.Reset()

					if last == nil {
						last = node
						ret = node
					} else {
						if err2 := last.(MutableNode).AddSibling(node); err2 != nil { //nolint:forcetypeassert
							err = err2
							return
						}
						last = node
					}
				}

				// create a new REFERENCE_REF node
				var node Node
				node, err = d.CreateReference(val)
				if err != nil {
					return
				}

				// Parse entity content to build children, mirroring
				// xmlNodeParseAttValue in libxml2 tree.c.
				// Use the expanding flag to prevent infinite recursion
				// when entities reference each other.
				if ok && ent.FirstChild() == nil && !ent.expanding {
					ent.expanding = true
					var refchildren Node
					refchildren, err = d.stringToNodeList(string(ent.Content()))
					ent.expanding = false
					if err != nil {
						return
					}
					setFirstChild(ent, refchildren)
					for n := refchildren; n != nil; {
						n.baseDocNode().parent = ent
						if x := n.NextSibling(); x != nil {
							n = x
						} else {
							n = nil
						}
					}
				}

				if last == nil {
					last = node
					ret = node
				} else {
					if err2 := last.(MutableNode).AddSibling(node); err2 != nil { //nolint:forcetypeassert
						err = err2
						return
					}
					last = node
				}
			}
		}

		if charval != 0 {
			_, _ = buf.WriteRune(charval)
			charval = 0
		}
	}

	if buf.Len() > 0 {
		n := d.CreateText(buf.Bytes())

		if last == nil {
			ret = n
		} else {
			if err := last.(MutableNode).AddSibling(n); err != nil { //nolint:forcetypeassert
				return nil, err
			}
		}
	}

	return
}

func (d *Document) CreateCharRef(name string) (*EntityRef, error) {
	if name == "" {
		return nil, errors.New("empty name")
	}

	var decoded string
	if name[0] != '&' {
		decoded = name
	} else {
		// the name should be everything but '&' and ';'
		if name[len(name)-1] == ';' {
			decoded = name[1 : len(name)-1]
		} else {
			decoded = name[1:]
		}
	}
	if decoded == "" {
		return nil, fmt.Errorf("char ref %q has an empty name", name)
	}

	n := newEntityRef()
	n.doc = d
	n.name = decoded
	return n, nil
}

// RegisterID associates an ID value with an element in the document's
// ID table. This is called during parsing to build an O(1) lookup table
// for GetElementByID, mirroring libxml2's xmlAddID.
func (d *Document) RegisterID(id string, elem *Element) {
	if d.ids == nil {
		d.ids = make(map[string]*Element)
	}
	// Normalize the ID value: xs:ID is derived from xs:NCName which
	// collapses whitespace. Strip leading/trailing whitespace so that
	// xml:id="id3 " is findable as "id3".
	id = strings.TrimSpace(id)
	d.ids[id] = elem
}

// IDTable returns the document's ID->element table populated during parsing
// (mirroring libxml2's xmlAddID). The returned map is the document's own, not a
// copy, and is nil for documents built without an interned ID table (e.g. via the
// API rather than the parser); callers must not mutate it. It lets a derived
// document (e.g. an xsl:strip-space copy) rebuild an equivalent table by
// translating each entry's element through the original->copy correspondence,
// faithfully preserving id()/GetElementByID semantics — including for prefixed
// elements whose qualified DTD ATTLIST the lazy GetElementByID fallback would
// otherwise miss.
func (d *Document) IDTable() map[string]*Element {
	return d.ids
}

// GetElementByID returns the first element in the document whose ID matches
// the given value. If the document's ID table has been populated (during
// parsing), it performs an O(1) hash lookup. Otherwise it falls back to an
// O(n) tree walk checking xml:id and DTD-declared ID attributes.
//
// The ID-skip state (see SetSkipIDs) is authoritative and is checked FIRST: a
// document with SkipIDs() == true resolves NO ids — it returns nil without
// consulting the ID table or performing the lazy walk. This keeps GetElementByID
// and fn:id consistent with the SkipIDs contract even on a document that already
// has a populated ID table (e.g. one parsed normally and then SetSkipIDs(true)).
func (d *Document) GetElementByID(id string) *Element {
	if d.idsSkip {
		return nil
	}
	if d.ids != nil {
		return d.ids[id]
	}

	// Fallback: O(n) tree walk for documents not built via parser. The walk
	// error is a best-effort signal only: the walker returns a sentinel error
	// to stop early once the id is found, and a Walk-detected tree cycle
	// (ErrWalkCycle) simply ends the search over the traversable portion — a
	// lookup has no error channel, so it degrades to returning any match found
	// before the cycle (nil if none), never spinning.
	var found *Element
	_ = Walk(d, NodeWalkerFunc(func(n Node) error {
		elem, ok := AsNode[*Element](n)
		if !ok {
			return nil
		}
		for _, a := range elem.Attributes() {
			// Check xml:id (normalize value — xs:ID collapses whitespace)
			if a.Name() == lexicon.QNameXMLID && strings.TrimSpace(a.Value()) == id {
				found = elem
				return errors.New("found")
			}
		}
		// Check DTD-declared ID attributes (internal and external subsets)
		for _, dtd := range []*DTD{d.intSubset, d.extSubset} {
			if dtd == nil {
				continue
			}
			for _, adecl := range dtd.AttributesForElement(elem.Name()) {
				if adecl.AType() != enum.AttrID {
					continue
				}
				for _, a := range elem.Attributes() {
					if a.LocalName() == adecl.LocalName() && a.Prefix() == adecl.prefix && a.Value() == id {
						found = elem
						return errors.New("found")
					}
				}
			}
		}
		return nil
	}))
	return found
}

func (d *Document) AddEntity(name string, typ enum.EntityType, externalID, systemID, content string) (*Entity, error) {
	if d.intSubset == nil {
		return nil, errors.New("document without internal subset")
	}

	return d.intSubset.AddEntity(name, typ, externalID, systemID, content)
}
