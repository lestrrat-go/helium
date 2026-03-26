package helium

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/lestrrat-go/helium/enum"
	"github.com/lestrrat-go/helium/internal/lexicon"
	"github.com/lestrrat-go/helium/internal/pool"
	"github.com/lestrrat-go/pdebug"
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
func (d *Document) Free() {
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
	d.elemChunks = nil
	d.textChunks = nil
	d.nsChunks = nil
	d.attrChunks = nil
	d.elemSlab = nil
	d.textSlab = nil
	d.nsSlab = nil
	d.attrSlab = nil
}

func (d Document) XMLString(writers ...Writer) (string, error) {
	out := bytes.Buffer{}
	if err := d.XML(&out, writers...); err != nil {
		return "", err
	}
	return out.String(), nil
}

func (d *Document) XML(out io.Writer, writers ...Writer) error {
	writer := NewWriter()
	if len(writers) > 0 {
		writer = writers[0]
	}
	return writer.WriteDoc(out, d)
}

func (d *Document) AddChild(cur Node) error {
	return addChild(d, cur)
}

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

func (d *Document) Standalone() DocumentStandaloneType {
	return d.standalone
}

func (d *Document) Version() string {
	return d.version
}

// URL returns the document URI, used as the base for relative URI resolution.
// This mirrors libxml2's xmlDoc.URL field.
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

func (d *Document) ExtSubset() *DTD {
	return d.extSubset
}

func (d *Document) Replace(_ Node) error {
	return ErrInvalidOperation
}

// DocumentElement returns the root element of the document, or nil if none exists.
func (d *Document) DocumentElement() *Element {
	for n := d.firstChild; n != nil; n = n.NextSibling() {
		if n.Type() == ElementNode {
			return n.(*Element)
		}
	}
	return nil
}

func (d *Document) SetDocumentElement(root MutableNode) error {
	if d == nil {
		// what are you trying to do?
		return nil
	}

	if root == nil || root.Type() == NamespaceDeclNode {
		return nil
	}

	root.SetParent(d)
	var old Node
	for old = d.firstChild; old != nil; old = old.NextSibling() {
		if old.Type() == ElementNode {
			break
		}
	}

	if old == nil {
		if err := d.AddChild(root); err != nil {
			return err
		}
	} else {
		if err := old.(MutableNode).Replace(root); err != nil {
			return err
		}
	}
	return nil
}

func (d *Document) CreateReference(name string) (*EntityRef, error) {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START document.CreateReference '%s'", name)
		defer g.IRelease("END document.CreateReference")
	}
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
	if pdebug.Enabled {
		g := pdebug.IPrintf("START document.CreateAttribute '%s' (%s)", name, value)
		defer func() {
			g.IRelease("END document.CreateAttribute (attr.Value = '%s')", attr.Value())
		}()
	}
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

func (d *Document) CreatePI(target, data string) *ProcessingInstruction {
	return &ProcessingInstruction{
		target: target,
		data:   data,
	}
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
	// If no children exist yet, just append.
	var root Node
	for c := d.firstChild; c != nil; c = c.NextSibling() {
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

var (
	elemChunkPool = pool.New(func() *[slabSize]Element { return new([slabSize]Element) }, nil)
	textChunkPool = pool.New(func() *[slabSize]Text { return new([slabSize]Text) }, nil)
	nsChunkPool   = pool.New(func() *[slabSize]Namespace { return new([slabSize]Namespace) }, nil)
	attrChunkPool = pool.New(func() *[slabSize]Attribute { return new([slabSize]Attribute) }, nil)
)

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

func (d *Document) CreateText(value []byte) *Text {
	var e *Text
	if d != nil {
		e = d.allocText()
	} else {
		e = &Text{}
	}
	e.etype = TextNode
	e.content = make([]byte, len(value))
	copy(e.content, value)
	e.name = textNodeName
	e.doc = d
	return e
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

func (d *Document) CreateElementContent(name string, etype ElementContentType) (*ElementContent, error) {
	e, err := newElementContent(name, etype)
	if err != nil {
		return nil, err
	}
	return e, nil
}

func (d *Document) GetEntity(name string) (ent *Entity, found bool) {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START document.GetEntity '%s'", name)
		defer func() {
			if found {
				g.IRelease("END document.GetEntity found = %t '%#x', (%p)", found, ent.Content(), ent)
			} else {
				g.IRelease("END document.GetEntity found = false")
			}
		}()
	}
	if ints := d.intSubset; ints != nil {
		if pdebug.Enabled {
			pdebug.Printf("Looking into internal subset...")
		}
		ent, found = ints.LookupEntity(name)
		if found {
			return
		}
	}

	if d.standalone != StandaloneExplicitYes {
		if exts := d.extSubset; exts != nil {
			if pdebug.Enabled {
				pdebug.Printf("Looking into external subset...")
			}
			ent, found = exts.LookupEntity(name)
			return
		}
	}

	return
}

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

/*
 * @doc:  the document
 * @value:  the value of the attribute
 *
 * Parse the value string and build the node list associated. Should
 * produce a flat tree with only TEXTs and ENTITY_REFs.
 * Returns a pointer to the first child
 */
func (d *Document) stringToNodeList(value string) (ret Node, err error) {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START document.stringToNodeList '%s'", value)
		defer func() {
			var content []byte
			if ret == nil {
				content = []byte("(nil)")
			} else {
				content = ret.Content()
			}
			g.IRelease("END document.stringToNodeList '%s'", content)
		}()
	}

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

		// if this is not any sort of an entity , just go go go
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
			ent, ok := d.GetEntity(val)

			// Predefined entities are inlined; all others (resolved or not)
			// become entity reference nodes. This matches libxml2's
			// xmlNodeParseAttValue behavior in tree.c.
			if ok && ent.EntityType() == enum.InternalPredefinedEntity {
				_, _ = buf.Write(ent.Content())
			} else {
				// flush the buffer so far
				if buf.Len() > 0 {
					if pdebug.Enabled {
						pdebug.Printf("Flushing content so far... '%s'", buf.Bytes())
					}
					node := d.CreateText(buf.Bytes())
					buf.Reset()

					if last == nil {
						last = node
						ret = node
					} else {
						if err2 := last.(MutableNode).AddSibling(node); err2 != nil {
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
					if err2 := last.(MutableNode).AddSibling(node); err2 != nil {
						err = err2
						return
					}
					last = node
				}
			}
		}

		if charval != 0 {
			_, _ = buf.WriteRune(rune(charval))
			charval = 0
		}
	}

	if buf.Len() > 0 {
		n := d.CreateText(buf.Bytes())

		if last == nil {
			ret = n
		} else {
			if err := last.(MutableNode).AddSibling(n); err != nil {
				return nil, err
			}
		}
	}

	return
}

func (d *Document) CreateCharRef(name string) (*EntityRef, error) {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START document.CreateCharRef '%s'", name)
		defer g.IRelease("END document.CreateCharRef")
	}

	if name == "" {
		return nil, errors.New("empty name")
	}

	n := newEntityRef()
	n.doc = d
	if name[0] != '&' {
		n.name = name
	} else {
		// the name should be everything but '&' and ';'
		if name[len(name)-1] == ';' {
			n.name = name[1 : len(name)-1]
		} else {
			n.name = name[1:]
		}
	}
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

// GetElementByID returns the first element in the document whose ID matches
// the given value. If the document's ID table has been populated (during
// parsing), it performs an O(1) hash lookup. Otherwise it falls back to an
// O(n) tree walk checking xml:id and DTD-declared ID attributes.
func (d *Document) GetElementByID(id string) *Element {
	if d.ids != nil {
		return d.ids[id]
	}

	// Fallback: O(n) tree walk for documents not built via parser.
	var found *Element
	_ = Walk(d, NodeWalkerFunc(func(n Node) error {
		if n.Type() != ElementNode {
			return nil
		}
		elem := n.(*Element)
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
			for _, adecl := range dtd.AttributesForElement(elem.LocalName()) {
				if adecl.AType() != enum.AttrID {
					continue
				}
				for _, a := range elem.Attributes() {
					if a.LocalName() == adecl.LocalName() && a.Value() == id {
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
