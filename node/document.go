package node

// Document represents the root document node
type Document struct {
	treeNode
	version    string
	encoding   string
	standalone DocumentStandaloneType

	intSubset *DTD
	extSubset *DTD
}

func NewDocument() *Document {
	doc := &Document{}
	doc.treeNode = treeNode{
		doc: doc,
	}
	doc.version = "1.0"
	doc.encoding = "utf-8"
	doc.standalone = StandaloneImplicitNo
	return doc
}

func NewDocumentWithOptions(version, encoding string, standalone DocumentStandaloneType) *Document {
	doc := &Document{
		version:    version,
		encoding:   encoding,
		standalone: standalone,
	}
	doc.treeNode = treeNode{
		doc: doc,
	}
	return doc
}

func (d *Document) CreateElement(name string) *Element {
	e := NewElement(name)
	_ = e.SetOwnerDocument(d)
	return e
}

func (d *Document) CreateComment(content []byte) *Comment {
	c := NewComment(content)
	_ = c.SetOwnerDocument(d)
	return c
}

func (d *Document) CreateText(content []byte) *Text {
	t := NewText(content)
	_ = t.SetOwnerDocument(d)
	return t
}

func (d *Document) CreateAttribute(name, value string) *Attribute {
	attr := newAttribute(name, nil)
	_ = attr.SetOwnerDocument(d)
	if value != "" {
		text := NewText([]byte(value))
		_ = text.SetOwnerDocument(d)
		_ = attr.AddChild(text)
	}
	return attr
}

func (d *Document) CreateEntity(name string, typ EntityType, publicID, systemID, content, orig string) *Entity {
	e := newEntity(name, typ, publicID, systemID, content, orig)
	_ = e.SetOwnerDocument(d)
	return e
}

func (d *Document) CreateElementContent(name, prefix string, ctype ElementContentType, occur ElementContentOccur) *ElementContent {
	return newElementContent(name, prefix, ctype, occur)
}

func (d *Document) Encoding() string {
	if enc := d.encoding; enc != "" {
		return d.encoding
	}
	return "utf8"
}

func (d *Document) Standalone() DocumentStandaloneType {
	return d.standalone
}

func (d *Document) SetStandalone(standalone DocumentStandaloneType) {
	d.standalone = standalone
}

func (d *Document) SetFirstChild(child Node) {
	d.firstChild = child
}

func (d *Document) SetLastChild(child Node) {
	d.lastChild = child
}


func (d *Document) Version() string {
	return d.version
}

func (d *Document) IntSubset() *DTD {
	return d.intSubset
}

func (d *Document) ExtSubset() *DTD {
	return d.extSubset
}

func (d *Document) Type() NodeType {
	return DocumentNodeType
}

func (d *Document) LocalName() string {
	return "#document"
}

func (d *Document) AddChild(cur Node) error {
	return addChild(d, cur)
}

func (d *Document) AddContent(b []byte) error {
	return addContent(d, b)
}

func (d *Document) AddSibling(n Node) error {
	return ErrInvalidOperation
}

func (d *Document) Replace(n Node) error {
	return ErrInvalidOperation
}

func (d *Document) SetNextSibling(sibling Node) error {
	return ErrInvalidOperation
}

func (d *Document) SetPrevSibling(sibling Node) error {
	return ErrInvalidOperation
}

func (d *Document) SetDocumentElement(root Node) error {
	if d == nil {
		return nil
	}

	if root == nil || root.Type() == NamespaceDeclNodeType {
		return nil
	}

	_ = root.SetParent(d)
	var old Node
	for old = d.firstChild; old != nil; old = old.NextSibling() {
		if old.Type() == ElementNodeType {
			break
		}
	}

	if old == nil {
		if err := d.AddChild(root); err != nil {
			return err
		}
	} else {
		_ = old.Replace(root)
	}
	return nil
}

func (d *Document) Content(dst []byte) ([]byte, error) {
	result := dst
	for e := d.firstChild; e != nil; e = e.NextSibling() {
		var err error
		result, err = e.Content(result)
		if err != nil {
			return result, err
		}
	}
	return result, nil
}

// IsMixedElement checks if an element allows mixed content
func (d *Document) IsMixedElement(name string) (bool, error) {
	// Simple implementation - assume all elements can have mixed content
	// TODO: Implement proper DTD-based mixed content checking
	return true, nil
}

// CreatePI creates a processing instruction
func (d *Document) CreatePI(target, data string) (*ProcessingInstructionNode, error) {
	pi := &ProcessingInstructionNode{
		target: target,
		data:   data,
	}
	_ = pi.SetOwnerDocument(d)
	return pi, nil
}

// InternalSubset creates or returns the internal DTD subset
func (d *Document) InternalSubset() (*DTD, error) {
	if d.intSubset == nil {
		d.intSubset = &DTD{}
		_ = d.intSubset.SetOwnerDocument(d)
	}
	return d.intSubset, nil
}

// CreateInternalSubset creates the internal DTD subset
func (d *Document) CreateInternalSubset(name, eid, uri string) (*DTD, error) {
	if d.intSubset == nil {
		d.intSubset = &DTD{
			externalID: eid,
			systemID:   uri,
		}
		d.intSubset.name = name
		_ = d.intSubset.SetOwnerDocument(d)
	}
	return d.intSubset, nil
}

// GetEntity retrieves an entity by name
func (d *Document) GetEntity(name string) *Entity {
	if d.intSubset != nil {
		if entity, exists := d.intSubset.entities[name]; exists {
			return entity
		}
	}
	if d.extSubset != nil {
		if entity, exists := d.extSubset.entities[name]; exists {
			return entity
		}
	}
	return nil
}

// GetParameterEntity retrieves a parameter entity by name
func (d *Document) GetParameterEntity(name string) *Entity {
	if d.intSubset != nil {
		if entity, exists := d.intSubset.pentities[name]; exists {
			return entity
		}
	}
	if d.extSubset != nil {
		if entity, exists := d.extSubset.pentities[name]; exists {
			return entity
		}
	}
	return nil
}

// CreateCharRef creates a character reference (stub)
func (d *Document) CreateCharRef(name string) *EntityRef {
	// TODO: Implement proper character reference creation
	return &EntityRef{}
}

// CreateReference creates an entity reference (stub)
func (d *Document) CreateReference(name string) *EntityRef {
	// TODO: Implement proper entity reference creation
	return &EntityRef{}
}
