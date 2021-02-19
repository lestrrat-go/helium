package helium

import (
	"bytes"
	"strings"

	"github.com/lestrrat-go/pdebug/v3"
	"github.com/pkg/errors"
)

func CreateDocument(options ...DocumentOption) *Document {
	var doc Document
	doc.version = "1.0"
	doc.standalone = StandaloneImplicitNo

	for _, option := range options {
		switch option.Ident() {
		case identDocumentEncoding{}:
			doc.encoding = option.Value().(string)
		case identDocumentVersion{}:
			doc.version = option.Value().(string)
		case identDocumentStandalone{}:
			doc.standalone = option.Value().(DocumentStandaloneType)
		}
	}
	return &doc
}

func (d *Document) CreateText(value []byte) (*Text, error) {
	e := newText(value)
	e.doc = d
	return e, nil
}

func (d *Document) CreateAttribute(name, value string, ns *Namespace) (*Attribute, error) {
	attr := newAttribute(name, ns)
	if value == "" {
		return attr, nil
	}

	n, err := d.stringToNodeList(value)
	if err != nil {
		return nil, errors.Wrap(err, `failed to parse attribute value`)
	}

	attr.setFirstChild(n)
	for n != nil {
		n.SetParent(attr)
		x := n.NextSibling()
		if x == nil {
			attr.setLastChild(n)
		}
		n = x
	}

	return attr, nil
}

func (d *Document) CreateCharRef(name string) (*EntityRef, error) {
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
		n.setFirstChild(ent)
		n.setLastChild(ent)
	}

	return n, nil
}

func (d *Document) GetEntity(name string) (ent *Entity, found bool) {
	if ints := d.intSubset; ints != nil {
		if pdebug.Enabled {
			pdebug.Printf("Looking into internal subset...")
		}
		return ints.LookupEntity(name)
	}

	if exts := d.extSubset; exts != nil {
		return exts.LookupEntity(name)
	}

	return nil, false
}

func (d *Document) stringToNodeList(value string) (Node, error) {
	rdr := strings.NewReader(value)
	var buf bytes.Buffer
	var accumulator func(int32, rune) (int32, error)

	var ret Node
	var last Node
	for rdr.Len() > 0 {
		r, _, err := rdr.ReadRune()
		if err != nil {
			return nil, errors.Wrap(err, `failed to read rune`)
		}

		if r != '&' { // not an entity, just append and Go
			buf.WriteRune(r)
			continue
		}

		// Handle character references
		if r == '#' {
			r, _, err = rdr.ReadRune()
			if err != nil {
				return nil, errors.Wrap(err, `failed to read rune after '&#'`)
			}
			if r == 'x' {
				accumulator = accumulateHexCharRef
			} else {
				accumulator = accumulateDecimalCharRef
			}
			var charval int32
			for {
				r, _, err = rdr.ReadRune()
				if err != nil {
					return nil, errors.Wrap(err, `failed to read rune in char ref`)
				}
				if r == ';' {
					break
				}
				charval, err := accumulator(charval, r)
				if err != nil {
					return nil, errors.Wrap(err, `failed to process char ref`)
				}
			}
			buf.WriteRune(rune(charval))
			continue
		}

		// Otherwise it's an entity reference
		var entbuilder strings.Builder
		entbuilder.WriteRune(r)
		for rdr.Len() > 0 {
			r, _, err = rdr.ReadRune()
			if err != nil {
				return nil, errors.Wrap(err, `failed to read rune in entity ref`)
			}
			if r == ';' {
				break
			}
			entbuilder.WriteRune(r)
		}

		if r != ';' {
			return nil, errors.New("entity was unterminated (could not find terminating semicolon)")
		}

		val := entbuilder.String()
		ent, ok := d.GetEntity(val)
		if ok && ent.EntityType() == InternalPredefinedEntity {
			buf.Write(ent.Content())
		} else {
			// flush the buffer so far
			var last Node
			var ret Node
			if buf.Len() > 0 {
				if pdebug.Enabled {
					pdebug.Printf("Flushing content so far... '%s'", buf.Bytes())
				}
				var node Node
				node, err = d.CreateText(buf.Bytes())
				if err != nil {
					return nil, errors.Wrap(err, `failed to create text node`)
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
			var node Node
			node, err = d.CreateReference(val)
			if err != nil {
				return nil, errors.Wrap(err, `failed to create reference node`)
			}

			// no children
			if ok && ent.FirstChild() == nil {
				// XXX WTF am I doing here...?
				var refchildren Node
				refchildren, err = d.stringToNodeList(string(node.Content()))
				if err != nil {
					return nil, errors.Wrap(err, `failed to parse content`)
				}
				ent.setFirstChild(refchildren)
				for n := refchildren; n != nil; {
					n.SetParent(ent)
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

	if buf.Len() > 0 {
		var n Node
		n, err := d.CreateText(buf.Bytes())
		if err != nil {
			return nil, errors.Wrap(err, `failed to create text node`)
		}

		if last == nil {
			ret = n
		} else {
			last.AddSibling(n)
		}
	}

	return ret, nil
}
