package html

import (
	"strings"

	"github.com/lestrrat-go/helium"
)

// treeBuilder implements SAXHandler and builds a helium DOM tree.
type treeBuilder struct {
	doc *helium.Document
	cur helium.MutableNode // current insertion point
}

func newTreeBuilder() *treeBuilder {
	doc := helium.NewHTMLDocument()
	return &treeBuilder{
		doc: doc,
		cur: doc,
	}
}

func (t *treeBuilder) SetDocumentLocator(loc DocumentLocator) error {
	return nil
}

func (t *treeBuilder) StartDocument() error {
	return nil
}

func (t *treeBuilder) EndDocument() error {
	return nil
}

func (t *treeBuilder) StartElement(name string, attrs []Attribute) error {
	// The HTML tokenizer can produce tag names that carry a colon (e.g. the
	// MS-Office construct <o:p>). CreateElement rejects a colon, so such a name
	// is built through CreateElementNS with the colon split into a namespace
	// prefix and a local name, preserving the original "prefix:local" name.
	var elem *helium.Element
	var err error
	if prefix, local, found := strings.Cut(name, ":"); found {
		// CreateElementNS rejects a colon in the local part, so a malformed
		// multi-colon name such as <foo:bar:baz> (first-colon split leaves
		// local "bar:baz") would fail and drop the node entirely. Split at the
		// LAST colon instead in that case: prefix "foo:bar", local "baz". The
		// local part is then guaranteed colon-free so element creation never
		// fails and the node is always attached, keeping t.cur consistent with
		// EndElement. Element.Name() reconstructs prefix+":"+local, so the
		// serialized name stays "foo:bar:baz" — byte-identical to how a
		// colon-bearing name was built before the CreateElement colon check.
		if strings.ContainsRune(local, ':') {
			idx := strings.LastIndex(name, ":")
			prefix, local = name[:idx], name[idx+1:]
		}
		// The empty-URI prefix binding is the parse-side representation of a
		// colon-bearing HTML tag name: the prefix is left unbound (href ""),
		// preserving the original "prefix:local" name for the HTML serializer,
		// which emits it verbatim (html/dump.go). The generic XML writer instead
		// REJECTS such an element (ErrWriterUnboundNamespacePrefix): "prefix:local"
		// with no xmlns:prefix is not reparseable. Keep this binding as-is — only
		// the generic writer's emission decision changed.
		// A colon-bearing HTML ATTRIBUTE, by contrast, never reaches the writer
		// at all: SetAttribute/SetBooleanAttribute reject the name and the parser
		// routes that rejection through strictness handling (strict: fatal parse
		// error; tolerant: warning, attribute dropped — the attribute loop below,
		// behavior adjudicated in PR #1254). Output missing a dropped attribute is
		// fully parseable, so the writer's unbound-prefix rejection is not
		// reachable from html.Parse attribute input.
		var ns *helium.Namespace
		ns, err = t.doc.CreateNamespace(prefix, "")
		if err != nil {
			return err
		}
		elem, err = t.doc.CreateElementNS(local, ns)
	} else {
		elem, err = t.doc.CreateElement(name)
	}
	if err != nil {
		return err
	}

	// SetAttribute stores the value verbatim, which is what we want: the HTML
	// parser has already resolved entities in attribute values, so parsing them
	// again (SetParsedAttribute) would re-interpret them as XML entity references
	// and fail on bare '&' characters.
	// Boolean attributes use SetBooleanAttribute (no children) so the
	// serializer can distinguish them from attrs with empty string values.
	//
	// Both setters reject a colon-bearing attribute name (SetAttributeNS is the
	// namespaced entry point). The first such rejection is captured but the loop
	// keeps going so the element and every other attribute are still built, then
	// the error is returned to the parser. The parser routes it through
	// handleSAXErr: in tolerant mode it is downgraded to a warning and the
	// dropped attribute is the only casualty; in strict mode it is surfaced as
	// the fatal parse error. Returning after the element is attached (below)
	// keeps t.cur consistent with EndElement in both modes.
	var attrErr error
	for _, a := range attrs {
		var err error
		if a.Boolean {
			err = elem.SetBooleanAttribute(a.Name)
		} else {
			err = elem.SetAttribute(a.Name, a.Value)
		}
		if err != nil && attrErr == nil {
			attrErr = err
		}
	}

	if err := t.cur.AddChild(elem); err != nil {
		return err
	}
	t.cur = elem
	return attrErr
}

func (t *treeBuilder) EndElement(name string) error {
	if t.cur == nil || t.cur == t.doc {
		return nil
	}
	if parent, ok := t.cur.Parent().(helium.MutableNode); ok && parent != nil {
		t.cur = parent
	} else {
		t.cur = t.doc
	}
	return nil
}

func (t *treeBuilder) Characters(ch []byte) error {
	if t.cur == nil || t.cur == t.doc {
		return nil
	}
	return t.cur.AppendText(ch)
}

func (t *treeBuilder) CDataBlock(value []byte) error {
	if t.cur == nil || t.cur == t.doc {
		return nil
	}
	return t.cur.AppendText(value)
}

func (t *treeBuilder) Comment(value []byte) error {
	comment := t.doc.CreateComment(value)
	return t.cur.AddChild(comment)
}

func (t *treeBuilder) InternalSubset(name, externalID, systemID string) error {
	_, err := t.doc.CreateInternalSubset(name, externalID, systemID)
	return err
}

func (t *treeBuilder) ProcessingInstruction(target, data string) error {
	pi := t.doc.CreatePI(target, data)
	return t.cur.AddChild(pi)
}

func (t *treeBuilder) IgnorableWhitespace(ch []byte) error {
	return t.Characters(ch)
}

func (t *treeBuilder) Error(err error) error {
	return nil
}

func (t *treeBuilder) Warning(err error) error {
	return nil
}
