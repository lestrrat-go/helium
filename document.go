package helium

import (
	"bytes"
	"errors"
	"io"
	"strings"

	"github.com/lestrrat/helium/internal/debug"
)

func CreateDocument() *Document {
	return NewDocument("1.0", "", StandaloneImplicitNo)
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

func (d Document) XMLString() (string, error) {
	out := bytes.Buffer{}
	if err := d.XML(&out); err != nil {
		return "", err
	}
	return out.String(), nil
}

func (d *Document) XML(out io.Writer) error {
	return (&Dumper{}).DumpDoc(out, d)
}

func (d *Document) AddChild(cur Node) error {
	return addChild(d, cur)
}

func (d *Document) AddContent(b []byte) error {
	return addContent(d, b)
}

func (d *Document) AddSibling(n Node) error {
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

func (d *Document) Standalone() DocumentStandaloneType {
	return d.standalone
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

func (d *Document) Replace(n Node) {
	panic("d.Replace does not make sense")
}

func (d *Document) SetDocumentElement(root Node) error {
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
		d.AddChild(root)
	} else {
		old.Replace(root)
	}
	return nil
}

func (d *Document) CreateReference(name string) (*EntityRef, error) {
	if debug.Enabled {
		g := debug.IPrintf("START document.CreateReference '%s'", name)
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
		n.setFirstChild(ent)
		n.setLastChild(ent)
	}

	return n, nil
}

func (d *Document) CreateAttribute(name, value string, ns *Namespace) (attr *Attribute, err error) {
	if debug.Enabled {
		g := debug.IPrintf("START document.CreateAttribute '%s' (%s)", name, value)
		defer func() {
			g.IRelease("END document.CreateAttribute (attr.Value = '%s')", attr.Value())
		}()
	}
	var n Node
	attr = newAttribute(name, ns)
	if value != "" {
		n, err = d.stringToNodeList(value)
		if err != nil {
			attr = nil
			return
		}

		attr.setFirstChild(n)
		for n != nil {
			n.SetParent(attr)
			x := n.NextSibling()
			if x == nil {
				n.setLastChild(x)
			}
			n = x
		}
	}
	return attr, nil
}

func (d *Document) CreateNamespace(prefix, uri string) (*Namespace, error) {
	ns := newNamespace(prefix, uri)
	ns.context = d
	return ns, nil
}

func (d *Document) CreatePI(target, data string) (*ProcessingInstruction, error) {
	return &ProcessingInstruction{
		target: target,
		data:   data,
	}, nil
}

func (d *Document) CreateDTD() (*DTD, error) {
	dtd := newDTD()
	dtd.doc = d
	return dtd, nil
}

func (d *Document) CreateElement(name string) (*Element, error) {
	e := newElement(name)
	e.doc = d
	return e, nil
}

func (d *Document) CreateText(value []byte) (*Text, error) {
	e := newText(value)
	e.doc = d
	return e, nil
}

func (d *Document) CreateComment(value []byte) (*Comment, error) {
	e := newComment(value)
	e.doc = d
	return e, nil
}

func (d *Document) CreateElementContent(name string, etype ElementContentType) (*ElementContent, error) {
	e, err := newElementContent(name, etype)
	if err != nil {
		return nil, err
	}
	return e, nil
}

func (d *Document) GetEntity(name string) (*Entity, bool) {
	if ints := d.intSubset; ints != nil {
		return ints.LookupEntity(name)
	}

	if exts := d.extSubset; exts != nil {
		return exts.LookupEntity(name)
	}

	return nil, false
}

func (d *Document) GetParameterEntity(name string) (*Entity, bool) {
	if ints := d.intSubset; ints != nil {
		return ints.LookupParameterEntity(name)
	}

	if exts := d.extSubset; exts != nil {
		return exts.LookupParameterEntity(name)
	}

	return nil, false
}

func (d *Document) IsMixedElement(name string) (bool, error) {
	if d.intSubset == nil {
		return false, errors.New("element declaration not found")
	}

	edecl, ok := d.intSubset.GetElementDesc(name)
	if !ok {
		return false, errors.New("element declaration not found")
	}

	switch edecl.decltype {
	case UndefinedElementType:
		return false, errors.New("element declaration not found")
	case ElementElementType:
		return false, nil
	case EmptyElementType, AnyElementType, MixedElementType:
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
func (d *Document) stringToNodeList(value string) (Node, error) {
	if debug.Enabled {
		g := debug.IPrintf("START document.stringToNodeList '%s'", value)
		defer g.IRelease("END document.stringToNodeList")
	}
	rdr := strings.NewReader(value)
	buf := bytes.Buffer{}
	var ret Node
	var last Node
	var charval int32
	for rdr.Len() > 0 {
		r, _, err := rdr.ReadRune()
		if err != nil {
			return nil, err
		}

		// if this is not any sort of an entity , just go go go
		if r != '&' {
			buf.WriteRune(r)
			continue
		}

		// well, at least the first rune sure looks like an entity, see what
		// else we have.
		r, _, err = rdr.ReadRune()
		if err != nil {
			return nil, err
		}

		if r == '#' {
			r2, _, err := rdr.ReadRune()
			if err != nil {
				return nil, err
			}

			var accumulator func(int32, rune) (int32, error)
			if r2 == 'x' {
				accumulator = accumulateHexCharRef
			} else {
				rdr.UnreadRune()
				accumulator = accumulateDecimalCharRef
			}
			for {
				r, _, err = rdr.ReadRune()
				if err != nil {
					return nil, err
				}
				if r == ';' {
					break
				}
				charval, err = accumulator(charval, r)
				if err != nil {
					return nil, err
				}
			}
		} else {
			rdr.UnreadRune()
			entbuf := bytes.Buffer{}
			for rdr.Len() > 0 {
				r, _, err = rdr.ReadRune()
				if err != nil {
					return nil, err
				}
				if r == ';' {
					break
				}
				entbuf.WriteRune(r)
			}

			if r != ';' {
				return nil, errors.New("entity was unterminated (could not find terminating semicolon)")
			}

			val := entbuf.String()
			ent, ok := d.GetEntity(val)

			// XXX I *believe* libxml2 SKIPS entities that it can't resolve
			// at this point?
			if ok && ent.EntityType() == int(InternalPredefinedEntity) {
				buf.Write(ent.Content())
			} else {
				// flush the buffer so far
				if buf.Len() > 0 {
					node, err := d.CreateText(buf.Bytes())
					if err != nil {
						return nil, err
					}
					buf.Reset()

					if last == nil {
						last = node
						ret = node
					} else {
						last.AddSibling(node)
						last = node
					}
				}

				// create a new REFERENCE_REF node
				node, err := d.CreateReference(val)
				if err != nil {
					return nil, err
				}

				// no children
				if ok && ent.FirstChild() == nil {
					// XXX WTF am I doing here...?
					refchildren, err := d.stringToNodeList(string(node.Content()))
					if err != nil {
						return nil, err
					}
					ent.setFirstChild(refchildren)
					for n := refchildren; n != nil; {
						ent.AddChild(n)
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
					last.AddSibling(node)
					last = node
				}
			}
		}

		if charval != 0 {
			buf.WriteRune(rune(charval))
			charval = 0
		}
	}

	if buf.Len() > 0 {
		n, err := d.CreateText(buf.Bytes())
		if err != nil {
			return nil, err
		}

		if last == nil {
			ret = n
		} else {
			last.AddSibling(n)
		}
	}

	if debug.Enabled {
		for n := last; n != nil; n = n.PrevSibling() {
			debug.Printf("---> %s (%s)", n.Name(), n.Content())
		}
	}

	return ret, nil
}

func (d *Document) CreateCharRef(name string) (*EntityRef, error) {
	if debug.Enabled {
		g := debug.IPrintf("START document.CreateCharRef '%s'", name)
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
