package xpath3

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/enum"
	"github.com/lestrrat-go/helium/internal/domutil"
	"github.com/lestrrat-go/helium/internal/lexicon"
	"github.com/lestrrat-go/helium/internal/unparsedtext"
	"github.com/lestrrat-go/helium/internal/xmlchar"
	ixpath "github.com/lestrrat-go/helium/internal/xpath"
)

func init() {
	registerFn("node-name", 0, 1, fnNodeName)
	registerFn("nilled", 0, 1, fnNilled)
	registerFn("data", 0, 1, fnData)
	registerFn("base-uri", 0, 1, fnBaseURI)
	registerFn("document-uri", 0, 1, fnDocumentURI)
	registerFn("root", 0, 1, fnRoot)
	registerFn("path", 0, 1, fnPath)
	registerFn("has-children", 0, 1, fnHasChildren)
	registerFn("innermost", 1, 1, fnInnermost)
	registerFn("outermost", 1, 1, fnOutermost)
	registerFn("lang", 1, 2, fnLang)
	registerFn("local-name", 0, 1, fnLocalName)
	registerFn("name", 0, 1, fnName)
	registerFn("namespace-uri", 0, 1, fnNamespaceURI)
	registerFn("number", 0, 1, fnNumber)
	registerFn("generate-id", 0, 1, fnGenerateID)
	registerFn("parse-xml", 1, 1, fnParseXML)
	registerFn("parse-xml-fragment", 1, 1, fnParseXMLFragment)
	registerFn("doc", 1, 1, fnDoc)
	registerFn("doc-available", 1, 1, fnDocAvailable)
	registerFn("id", 1, 2, fnID)
	registerFn("idref", 1, 2, fnIDRef)
	registerFn("element-with-id", 1, 2, fnElementWithID)
	registerFn("collection", 0, 1, fnCollection)
	registerFn("uri-collection", 0, 1, fnURICollection)
}

func fnNodeName(ctx context.Context, args []Sequence) (Sequence, error) {
	n, err := nodeArgOrCtx(ctx, args)
	if err != nil {
		return nil, err
	}
	if n == nil {
		return validNilSequence, nil
	}
	switch n.Type() {
	case helium.ElementNode, helium.AttributeNode:
		local := ixpath.LocalNameOf(n)
		uri := ixpath.NodeNamespaceURI(n)
		prefix := ixpath.NodePrefix(n)
		return SingleAtomic(AtomicValue{
			TypeName: TypeQName,
			Value:    QNameValue{Prefix: prefix, Local: local, URI: uri},
		}), nil
	case helium.ProcessingInstructionNode:
		return SingleAtomic(AtomicValue{
			TypeName: TypeQName,
			Value:    QNameValue{Local: n.Name()},
		}), nil
	case helium.NamespaceNode:
		return SingleAtomic(AtomicValue{
			TypeName: TypeQName,
			Value:    QNameValue{Local: n.Name()},
		}), nil
	}
	return validNilSequence, nil
}

func fnNilled(ctx context.Context, args []Sequence) (Sequence, error) {
	n, err := nodeArgOrCtx(ctx, args)
	if err != nil {
		return nil, err
	}
	if n == nil {
		return validNilSequence, nil
	}
	// fn:nilled returns the nilled property of an element node, else () for a
	// non-element. The nilled property is the PSVI [nil] property: true only for
	// an element that the XSD validator confirmed carried a valid xsi:nil="true"
	// (recorded in the evaluator's NilledElements set). Without a schema-aware
	// nilled set every element is not-nilled → false.
	if n.Type() == helium.ElementNode {
		return SingleBoolean(nodeIsNilled(ctx, n)), nil
	}
	return validNilSequence, nil
}

// nodeIsNilled reports whether node n is in the active evaluation's
// schema-derived nilled-element set. It is safe when no set is configured
// (non-schema-aware evaluation): it returns false.
func nodeIsNilled(ctx context.Context, n helium.Node) bool {
	ec := getFnContext(ctx)
	if ec == nil || ec.nilledElements == nil {
		return false
	}
	_, ok := ec.nilledElements[n]
	return ok
}

func fnData(ctx context.Context, args []Sequence) (Sequence, error) {
	if len(args) == 0 {
		fc := getFnContext(ctx)
		if fc == nil {
			return nil, &XPathError{Code: errCodeXPDY0002, Message: "data() requires a context item"}
		}
		if fc.contextItem != nil {
			args = []Sequence{ItemSlice{fc.contextItem}}
		} else if fc.node != nil {
			args = []Sequence{ItemSlice{nodeItemFor(ctx, fc, fc.node)}}
		} else {
			return nil, &XPathError{Code: errCodeXPDY0002, Message: "data() requires a context item"}
		}
	}
	atoms, err := atomizeTypedValue(ctx, args[0])
	if err != nil {
		return nil, err
	}
	result := make(ItemSlice, len(atoms))
	for i, a := range atoms {
		result[i] = a
	}
	return result, nil
}

// atomizeTypedValue atomizes a sequence through the XDM 3.1 §5.15 typed-value
// rules, interleaving a typed-value check with atomization: a NILLED element has
// no typed value and is skipped (typed value ()); an element whose schema type
// annotation resolves to ELEMENT-ONLY complex content has no typed value and
// raises err:FOTY0012 (rather than fabricating an xs:untypedAtomic from its
// string value); an element with EMPTY complex content has typed value () and is
// skipped (contributes no atoms); mixed/simple content and everything else
// atomize normally. It backs fn:data AND the xs:string?-argument function
// conversion (coerceAtomizedString / seqToStringErr), since atomizing an
// element-only node has no typed value for ANY caller per XDM. The check is
// threaded through atomizeStreamCont, so it walks items in the SAME encounter
// order and with the SAME array recursion as atomization: the FIRST offending
// item wins (a map/function atomized earlier still raises FOTY0013 before a
// later element-only element), and nilled / element-only / empty nodes nested
// inside arrays are handled. The check fires only when a nilled-element set is
// configured or the active SchemaDeclarations implements the optional
// ContentTypeKindProvider; without either this is exactly AtomizeSequence, so
// other AtomizeItem callers (comparisons, casts) are unaffected.
func atomizeTypedValue(ctx context.Context, seq Sequence) ([]AtomicValue, error) {
	check := typedValueItemCheck(ctx)
	if check == nil {
		return AtomizeSequence(seq)
	}
	if seq == nil {
		return nil, nil
	}
	result := make([]AtomicValue, 0, seq.Len())
	_, err := atomizeStreamCont(seq, check, func(av AtomicValue) (bool, error) {
		result = append(result, av)
		return true, nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// atomizeItemCheck is a per-item pre-check consulted by atomizeStreamCont (the
// fn:data / typed-value path only) BEFORE an item is atomized, in the same
// encounter order and array recursion as atomization. It returns skip=true when
// the item has typed value () and contributes no atoms, or a non-nil error
// (e.g. FOTY0012) to reject the item.
type atomizeItemCheck func(item Item) (skip bool, err error)

// typedValueItemCheck builds the typed-value pre-check for the active
// evaluation, reading the evalContext stashed in ctx. It is the ctx-based entry
// point for callers past the function boundary (fn:data, coerceAtomizedString);
// the function-signature coercion, which runs BEFORE the fnContext is stashed in
// ctx, uses typedValueItemCheckFor with its explicit ec instead.
func typedValueItemCheck(ctx context.Context) atomizeItemCheck {
	return typedValueItemCheckFor(getFnContext(ctx))
}

// typedValueItemCheckFor builds the typed-value pre-check from an explicit
// evalContext, combining the two schema-aware sources: the nilled-element set (a
// nilled element has no typed value → skipped) and the optional
// ContentTypeKindProvider (element-only → FOTY0012, empty → skipped). It returns
// nil when neither is configured, so non-schema-aware atomization stays plain
// (AtomizeSequence). Nilled is checked first: a nilled element's content type is
// irrelevant, its typed value is always ().
func typedValueItemCheckFor(ec *evalContext) atomizeItemCheck {
	if ec == nil {
		return nil
	}
	provider := ec.contentKindProvider()
	nilled := ec.nilledElements
	if provider == nil && len(nilled) == 0 {
		return nil
	}
	return func(item Item) (bool, error) {
		ni, ok := item.(NodeItem)
		if !ok {
			return false, nil
		}
		if ni.Node != nil && ni.Node.Type() == helium.ElementNode {
			if _, isNilled := nilled[ni.Node]; isNilled {
				return true, nil
			}
		}
		if provider != nil {
			return checkContentKindItem(provider, item)
		}
		return false, nil
	}
}

// contentKindProvider returns the active ContentTypeKindProvider for this
// evalContext, or nil when no schema-aware provider is configured.
func (ec *evalContext) contentKindProvider() ContentTypeKindProvider {
	if ec == nil || ec.schemaDeclarations == nil {
		return nil
	}
	provider, _ := ec.schemaDeclarations.(ContentTypeKindProvider)
	return provider
}

// nodeStringValue returns the XPath dm:string-value of a node. In a schema-aware
// run it strips INSIGNIFICANT whitespace: a whitespace-only text-node child of an
// element whose type annotation resolves (via the ContentTypeKindProvider) to
// ELEMENT-ONLY complex content is not part of the string value (XDM 3.1 PSVI
// construction). Without a provider/annotations, or for any element that is not
// element-only, this is byte-identical to ixpath.StringValue.
func (ec *evalContext) nodeStringValue(n helium.Node) string {
	if n == nil {
		return ""
	}
	provider := ec.contentKindProvider()
	if provider == nil || ec.typeAnnotations == nil {
		return ixpath.StringValue(n)
	}
	switch n.Type() {
	case helium.DocumentNode, helium.ElementNode:
		// only element/document nodes can contain element-only descendants
	default:
		return ixpath.StringValue(n)
	}
	var b strings.Builder
	ec.appendSchemaStringValue(&b, n, provider)
	return b.String()
}

// appendSchemaStringValue walks descendants in document order (iteratively, so a
// deep tree does not truncate) concatenating text/CDATA content. A whitespace-only
// text-node child of an element-only element is skipped as insignificant.
func (ec *evalContext) appendSchemaStringValue(b *strings.Builder, root helium.Node, provider ContentTypeKindProvider) {
	stack := []helium.Node{root}
	for len(stack) > 0 {
		cur := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		switch cur.Type() {
		case helium.TextNode, helium.CDATASectionNode:
			b.Write(cur.Content())
		}

		elementOnly := cur.Type() == helium.ElementNode && ec.isElementOnlyContent(cur, provider)
		for child := cur.LastChild(); child != nil; child = child.PrevSibling() {
			if elementOnly && isInsignificantWhitespaceText(child) {
				continue
			}
			stack = append(stack, child)
		}
	}
}

// isElementOnlyContent reports whether n's resolved type annotation is an
// element-only complex content type.
func (ec *evalContext) isElementOnlyContent(n helium.Node, provider ContentTypeKindProvider) bool {
	ann := ec.typeAnnotations[n]
	if ann == "" {
		return false
	}
	kind, ok := provider.SchemaTypeContentKind(ann)
	return ok && kind == ContentTypeElementOnly
}

// isInsignificantWhitespaceText reports whether n is a text or CDATA-section node
// whose content is entirely XSD whitespace (space, tab, CR, LF). XSD treats
// whitespace-only text and CDATA identically for element-only content, and
// ixpath.StringValue includes CDATA as a text descendant, so both are skipped
// when a child of an element-only element (not part of its string value).
func isInsignificantWhitespaceText(n helium.Node) bool {
	switch n.Type() {
	case helium.TextNode, helium.CDATASectionNode:
		return xmlchar.IsAllSpace(n.Content())
	default:
		return false
	}
}

// checkContentKindItem resolves the typed-value ACTION for one item
// (XDM 3.1 §5.15). For an element node whose type annotation resolves (via the
// ContentTypeKindProvider) to a complex content kind it returns: element-only →
// err:FOTY0012 (no typed value); empty → skip=true (typed value (), no atoms).
// Mixed/simple content, an unresolved annotation, and every non-element /
// non-schema-aware item return skip=false, nil — left to normal atomization.
func checkContentKindItem(provider ContentTypeKindProvider, item Item) (skip bool, err error) {
	ni, ok := item.(NodeItem)
	if !ok {
		return false, nil
	}
	if ni.TypeAnnotation == "" || ni.Node == nil || ni.Node.Type() != helium.ElementNode {
		return false, nil
	}
	kind, found := provider.SchemaTypeContentKind(ni.TypeAnnotation)
	if !found {
		return false, nil
	}
	switch kind {
	case ContentTypeElementOnly:
		return false, &XPathError{
			Code:    errCodeFOTY0012,
			Message: "element with element-only complex content has no typed value",
		}
	case ContentTypeEmpty:
		return true, nil
	default:
		return false, nil
	}
}

func fnBaseURI(ctx context.Context, args []Sequence) (Sequence, error) {
	n, err := nodeArgOrCtx(ctx, args)
	if err != nil {
		return nil, err
	}
	if n == nil {
		return validNilSequence, nil
	}
	// Namespace nodes have no base URI per the XPath data model.
	if n.Type() == helium.NamespaceNode {
		return validNilSequence, nil
	}
	// Per XDM: attribute, text, comment, and PI nodes derive their base URI
	// from their parent element. If they have no parent, their base URI is
	// the empty sequence.
	nodeType := n.Type()
	if nodeType == helium.AttributeNode || nodeType == helium.TextNode ||
		nodeType == helium.CommentNode || nodeType == helium.ProcessingInstructionNode {
		if n.Parent() == nil {
			return validNilSequence, nil
		}
	}
	// Walk up the parent chain to find the actual document the node lives
	// in. OwnerDocument() may return a different document when nodes are
	// created in one document and then moved to another (e.g. in XSLT
	// xsl:result-document).
	var doc *helium.Document
	if d, ok := n.(*helium.Document); ok {
		doc = d
	} else {
		cur := n
		for cur.Parent() != nil {
			cur = cur.Parent()
		}
		if d, ok := cur.(*helium.Document); ok {
			doc = d
		} else {
			doc = n.OwnerDocument()
		}
	}
	base := helium.NodeGetBase(doc, n)
	if base == "" {
		return validNilSequence, nil
	}
	return SingleAtomic(AtomicValue{TypeName: TypeAnyURI, Value: base}), nil
}

func fnDocumentURI(ctx context.Context, args []Sequence) (Sequence, error) {
	n, err := nodeArgOrCtx(ctx, args)
	if err != nil {
		return nil, err
	}
	if n == nil {
		return validNilSequence, nil
	}
	if n.Type() != helium.DocumentNode {
		return validNilSequence, nil
	}
	if doc, ok := n.(*helium.Document); ok {
		if doc.HasProperty(helium.DocInternal) {
			return validNilSequence, nil
		}
		if uri := doc.URL(); uri != "" {
			return SingleAtomic(AtomicValue{TypeName: TypeAnyURI, Value: uri}), nil
		}
	}
	return validNilSequence, nil
}

func fnRoot(ctx context.Context, args []Sequence) (Sequence, error) {
	n, err := nodeArgOrCtx(ctx, args)
	if err != nil {
		return nil, err
	}
	if n == nil {
		return validNilSequence, nil
	}
	return SingleNode(ixpath.DocumentRoot(n)), nil
}

func fnPath(ctx context.Context, args []Sequence) (Sequence, error) {
	n, err := nodeArgOrCtx(ctx, args)
	if err != nil {
		return nil, err
	}
	if n == nil {
		return validNilSequence, nil
	}
	return SingleString(buildNodePath(n)), nil
}

func buildNodePath(n helium.Node) string {
	const fnRoot = "Q{http://www.w3.org/2005/xpath-functions}root()"
	if n.Type() == helium.DocumentNode {
		return "/"
	}
	var parts []string
	for cur := n; cur != nil && cur.Type() != helium.DocumentNode; cur = cur.Parent() {
		// Skip the root of a non-document tree — it is represented by Q{...}root()
		if cur.Parent() == nil {
			break
		}
		switch cur.Type() {
		case helium.ElementNode:
			local := ixpath.LocalNameOf(cur)
			uri := ixpath.NodeNamespaceURI(cur)
			pos := elementPosition(cur)
			parts = append(parts, fmt.Sprintf("Q{%s}%s[%d]", uri, local, pos))
		case helium.AttributeNode:
			local := ixpath.LocalNameOf(cur)
			uri := ixpath.NodeNamespaceURI(cur)
			if uri != "" {
				parts = append(parts, fmt.Sprintf("@Q{%s}%s", uri, local))
			} else {
				parts = append(parts, "@"+local)
			}
		case helium.TextNode, helium.CDATASectionNode:
			pos := textPosition(cur)
			parts = append(parts, fmt.Sprintf("text()[%d]", pos))
		case helium.CommentNode:
			pos := commentPosition(cur)
			parts = append(parts, fmt.Sprintf("comment()[%d]", pos))
		case helium.ProcessingInstructionNode:
			parts = append(parts, fmt.Sprintf("processing-instruction(%s)[1]", cur.Name()))
		case helium.NamespaceNode:
			if cur.Name() == "" {
				parts = append(parts, fmt.Sprintf(`namespace::*[Q{%s}local-name()=""]`, NSFn))
			} else {
				parts = append(parts, "namespace::"+cur.Name())
			}
		}
	}
	// Reverse
	for i, j := 0, len(parts)-1; i < j; i, j = i+1, j-1 {
		parts[i], parts[j] = parts[j], parts[i]
	}
	// Check if the node is rooted in a document node.
	// Per XPath 3.1: document root → path starts with "/",
	// non-document root (orphan tree) → path starts with "Q{...}root()".
	root := n
	for root.Parent() != nil {
		root = root.Parent()
	}
	if root.Type() == helium.DocumentNode {
		return "/" + strings.Join(parts, "/")
	}
	if len(parts) == 0 {
		return fnRoot
	}
	return fnRoot + "/" + strings.Join(parts, "/")
}

func elementPosition(n helium.Node) int {
	local := ixpath.LocalNameOf(n)
	uri := ixpath.NodeNamespaceURI(n)
	pos := 1
	for sib := n.PrevSibling(); sib != nil; sib = sib.PrevSibling() {
		if sib.Type() == helium.ElementNode &&
			ixpath.LocalNameOf(sib) == local &&
			ixpath.NodeNamespaceURI(sib) == uri {
			pos++
		}
	}
	return pos
}

func textPosition(n helium.Node) int {
	pos := 1
	for sib := n.PrevSibling(); sib != nil; sib = sib.PrevSibling() {
		if sib.Type() == helium.TextNode || sib.Type() == helium.CDATASectionNode {
			pos++
		}
	}
	return pos
}

func commentPosition(n helium.Node) int {
	pos := 1
	for sib := n.PrevSibling(); sib != nil; sib = sib.PrevSibling() {
		if sib.Type() == helium.CommentNode {
			pos++
		}
	}
	return pos
}

func fnHasChildren(ctx context.Context, args []Sequence) (Sequence, error) {
	n, err := nodeArgOrCtx(ctx, args)
	if err != nil {
		return nil, err
	}
	if n == nil {
		return SingleBoolean(false), nil
	}
	// Per XPath spec, only document and element nodes can have children
	switch n.(type) {
	case *helium.Document, *helium.Element:
		return SingleBoolean(n.FirstChild() != nil), nil
	default:
		return SingleBoolean(false), nil
	}
}

func fnInnermost(ctx context.Context, args []Sequence) (Sequence, error) {
	nodes, ok := NodesFrom(args[0])
	if !ok {
		return nil, &XPathError{Code: lexicon.ErrXPTY0004, Message: "innermost() requires node-set"}
	}
	nodeSet := make(map[helium.Node]bool, len(nodes))
	for _, n := range nodes {
		nodeSet[n] = true
	}
	// A node is innermost if it is not an ancestor of any other node in the set.
	// Collect all ancestors of nodes in the set, then exclude them.
	ancestors := make(map[helium.Node]bool)
	for _, n := range nodes {
		for p := n.Parent(); p != nil; p = p.Parent() {
			if ancestors[p] {
				break // already traced this path
			}
			ancestors[p] = true
		}
	}
	var result []helium.Node
	for _, n := range nodes {
		if !ancestors[n] {
			result = append(result, n)
		}
	}
	return sequenceFromDocOrderedNodes(ctx, result)
}

func fnOutermost(ctx context.Context, args []Sequence) (Sequence, error) {
	nodes, ok := NodesFrom(args[0])
	if !ok {
		return nil, &XPathError{Code: lexicon.ErrXPTY0004, Message: "outermost() requires node-set"}
	}
	nodeSet := make(map[helium.Node]bool, len(nodes))
	for _, n := range nodes {
		nodeSet[n] = true
	}
	var result []helium.Node
	for _, n := range nodes {
		isOuter := true
		for p := n.Parent(); p != nil; p = p.Parent() {
			if nodeSet[p] {
				isOuter = false
				break
			}
		}
		if isOuter {
			result = append(result, n)
		}
	}
	return sequenceFromDocOrderedNodes(ctx, result)
}

func fnLang(ctx context.Context, args []Sequence) (Sequence, error) {
	langArg, err := coerceArgToString(ctx, args[0])
	if err != nil {
		return nil, err
	}
	langArg = strings.ToLower(langArg)
	var n helium.Node
	if len(args) > 1 {
		nodes, ok := NodesFrom(args[1])
		if !ok || len(nodes) == 0 {
			return nil, &XPathError{Code: lexicon.ErrXPTY0004, Message: "fn:lang second argument must be a node"}
		}
		n = nodes[0]
	} else {
		fc := getFnContext(ctx)
		if fc == nil || (fc.contextItem == nil && fc.node == nil) {
			return nil, &XPathError{Code: errCodeXPDY0002, Message: errMsgContextItemAbsent}
		}
		if fc.node == nil {
			return nil, &XPathError{Code: lexicon.ErrXPTY0004, Message: "context item is not a node for fn:lang"}
		}
		n = fc.node
	}
	matched, _ := domutil.XMLLangMatches(n, langArg)
	return SingleBoolean(matched), nil
}

func fnLocalName(ctx context.Context, args []Sequence) (Sequence, error) {
	n, err := nodeArgOrCtx(ctx, args)
	if err != nil {
		return nil, err
	}
	if n == nil {
		return SingleString(""), nil
	}
	return SingleString(ixpath.LocalNameOf(n)), nil
}

func fnName(ctx context.Context, args []Sequence) (Sequence, error) {
	n, err := nodeArgOrCtx(ctx, args)
	if err != nil {
		return nil, err
	}
	if n == nil {
		return SingleString(""), nil
	}
	switch n.Type() {
	case helium.ElementNode, helium.AttributeNode,
		helium.ProcessingInstructionNode, helium.NamespaceNode:
		return SingleString(n.Name()), nil
	}
	return SingleString(""), nil
}

func fnNamespaceURI(ctx context.Context, args []Sequence) (Sequence, error) {
	n, err := nodeArgOrCtx(ctx, args)
	if err != nil {
		return nil, err
	}
	if n == nil {
		return SingleString(""), nil
	}
	return SingleString(ixpath.NodeNamespaceURI(n)), nil
}

func fnNumber(ctx context.Context, args []Sequence) (Sequence, error) {
	if len(args) == 0 {
		fc := getFnContext(ctx)
		if fc == nil || (fc.contextItem == nil && fc.node == nil) {
			return nil, &XPathError{Code: errCodeXPDY0002, Message: errMsgContextItemAbsent}
		}
		s, ok := fc.contextStringValue()
		if !ok {
			return SingleDouble(math.NaN()), nil
		}
		a, err := CastFromString(s, TypeDouble)
		if err != nil {
			return SingleDouble(math.NaN()), nil //nolint:nilerr // fn:number returns NaN on cast failure per spec
		}
		return SingleDouble(a.DoubleVal()), nil
	}
	if seqLen(args[0]) == 0 {
		return SingleDouble(math.NaN()), nil
	}
	a, err := AtomizeItem(args[0].Get(0))
	if err != nil {
		// FOTY0013 (atomizing function items) must propagate per XPath 3.1 §2.7.2
		var xpErr *XPathError
		if errors.As(err, &xpErr) && xpErr.Code == errCodeFOTY0013 {
			return nil, err
		}
		return SingleDouble(math.NaN()), nil
	}
	dbl, err := CastAtomic(a, TypeDouble)
	if err != nil {
		return SingleDouble(math.NaN()), nil //nolint:nilerr // fn:number returns NaN on cast failure per spec
	}
	return SingleDouble(dbl.DoubleVal()), nil
}

func fnGenerateID(ctx context.Context, args []Sequence) (Sequence, error) {
	n, err := nodeArgOrCtx(ctx, args)
	if err != nil {
		return nil, err
	}
	if n == nil {
		return SingleString(""), nil
	}
	return SingleString(StableNodeID(n)), nil
}

func fnParseXML(ctx context.Context, args []Sequence) (Sequence, error) {
	if seqLen(args[0]) == 0 {
		return validNilSequence, nil
	}
	s, err := coerceArgToString(ctx, args[0])
	if err != nil {
		return nil, err
	}
	ec := getFnContext(ctx)
	parser := ec.xmlParser()
	if ec != nil && ec.baseURI != "" {
		parser = parser.BaseURI(ec.baseURI)
	}
	doc, err := parser.Parse(ctx, []byte(s))
	if err != nil {
		return nil, &XPathError{Code: errCodeFODC0006, Message: fmt.Sprintf("parse-xml: %v", err)}
	}
	doc.SetProperties(doc.Properties() | helium.DocInternal)
	return ItemSlice{NodeItem{Node: doc}}, nil
}

func fnParseXMLFragment(ctx context.Context, args []Sequence) (Sequence, error) {
	if seqLen(args[0]) == 0 {
		return validNilSequence, nil
	}
	s, err := coerceArgToString(ctx, args[0])
	if err != nil {
		return nil, err
	}

	s, err = stripXMLTextDeclaration(s)
	if err != nil {
		return nil, err
	}

	doc := helium.NewDocument("1.0", "", helium.StandaloneImplicitNo)
	doc.SetProperties(doc.Properties() | helium.DocInternal)
	ec := getFnContext(ctx)
	if ec != nil && ec.baseURI != "" {
		doc.SetURL(ec.baseURI)
	}

	first, err := ec.xmlParser().ParseInNodeContext(ctx, doc, []byte(s))
	if err != nil {
		return nil, &XPathError{Code: errCodeFODC0006, Message: fmt.Sprintf("parse-xml-fragment: %v", err)}
	}

	for cur := first; cur != nil; {
		next := cur.NextSibling()
		mn := cur.(helium.MutableNode) //nolint:forcetypeassert
		mn.SetPrevSibling(nil)
		mn.SetNextSibling(nil)
		if err := doc.AddChild(cur); err != nil {
			return nil, &XPathError{Code: errCodeFODC0006, Message: fmt.Sprintf("parse-xml-fragment: %v", err)}
		}
		cur = next
	}

	return ItemSlice{NodeItem{Node: doc}}, nil
}

func stripXMLTextDeclaration(s string) (string, error) {
	if !strings.HasPrefix(s, "<?xml") {
		return s, nil
	}

	end := strings.Index(s, "?>")
	if end < 0 {
		return "", &XPathError{Code: errCodeFODC0006, Message: errMsgParseXMLFragmentMalformedTextDec}
	}

	decl := s[len("<?xml"):end]
	names, err := parseXMLPseudoAttributes(decl)
	if err != nil {
		return "", &XPathError{Code: errCodeFODC0006, Message: errMsgParseXMLFragmentMalformedTextDec}
	}

	if len(names) == 0 {
		return "", &XPathError{Code: errCodeFODC0006, Message: errMsgParseXMLFragmentMalformedTextDec}
	}

	switch len(names) {
	case 1:
		if names[0] != lexicon.DeclEncoding {
			return "", &XPathError{Code: errCodeFODC0006, Message: errMsgParseXMLFragmentMalformedTextDec}
		}
	case 2:
		if names[0] != lexicon.DeclVersion || names[1] != lexicon.DeclEncoding {
			return "", &XPathError{Code: errCodeFODC0006, Message: errMsgParseXMLFragmentMalformedTextDec}
		}
	default:
		return "", &XPathError{Code: errCodeFODC0006, Message: errMsgParseXMLFragmentMalformedTextDec}
	}

	return s[end+2:], nil
}

func parseXMLPseudoAttributes(s string) ([]string, error) {
	names := make([]string, 0, 2)
	i := 0
	for i < len(s) {
		for i < len(s) && isXMLSpace(s[i]) {
			i++
		}
		if i == len(s) {
			return names, nil
		}

		start := i
		for i < len(s) && isXMLNameChar(s[i]) {
			i++
		}
		if start == i {
			return nil, fmt.Errorf("expected name")
		}
		name := s[start:i]
		switch name {
		case lexicon.DeclVersion, "encoding":
		case "standalone":
			return nil, fmt.Errorf("standalone not allowed")
		default:
			return nil, fmt.Errorf("unknown pseudo-attribute")
		}
		names = append(names, name)

		for i < len(s) && isXMLSpace(s[i]) {
			i++
		}
		if i == len(s) || s[i] != '=' {
			return nil, fmt.Errorf("expected =")
		}
		i++
		for i < len(s) && isXMLSpace(s[i]) {
			i++
		}
		if i == len(s) || (s[i] != '\'' && s[i] != '"') {
			return nil, fmt.Errorf("expected quoted value")
		}
		quote := s[i]
		i++
		start = i
		for i < len(s) && s[i] != quote {
			i++
		}
		if i == len(s) {
			return nil, fmt.Errorf("unterminated value")
		}
		if start == i {
			return nil, fmt.Errorf("empty value")
		}
		i++
	}
	return names, nil
}

func isXMLSpace(b byte) bool {
	switch b {
	case ' ', '\t', '\r', '\n':
		return true
	default:
		return false
	}
}

func isXMLNameChar(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || b == '-' || b == '_'
}

func fnID(ctx context.Context, args []Sequence) (Sequence, error) {
	return idLookup(ctx, args, false)
}

// idLookup implements both fn:id and fn:element-with-id. The two functions
// agree whenever the is-id node is an attribute (the result is the element
// bearing the attribute) and differ when the is-id node is an element: fn:id
// returns that element itself, whereas fn:element-with-id returns its parent
// element. The elementWithID flag selects the latter behavior.
func idLookup(ctx context.Context, args []Sequence, elementWithID bool) (Sequence, error) {
	doc, err := resolveIDLookupDocument(ctx, args)
	if err != nil {
		return nil, err
	}

	tokens, err := idLookupTokens(args[0])
	if err != nil {
		return nil, err
	}
	if len(tokens) == 0 {
		return validNilSequence, nil
	}

	nodes := make([]helium.Node, 0, len(tokens))
	for _, token := range tokens {
		// GetElementByID resolves DTD-declared ID attributes; the returned
		// element already bears the ID attribute, so it is the correct
		// result for both fn:id and fn:element-with-id.
		if elem := doc.GetElementByID(token); elem != nil {
			nodes = append(nodes, elem)
		}
	}
	nodes = append(nodes, idElementsFromTypeAnnotations(doc, tokens, getFnContext(ctx), elementWithID)...)
	return sequenceFromDocOrderedNodes(ctx, nodes)
}

func idElementsFromTypeAnnotations(doc *helium.Document, tokens []string, ec *evalContext, elementWithID bool) []helium.Node {
	if ec == nil || len(tokens) == 0 {
		return nil
	}
	if ec.typeAnnotations == nil && ec.preservedIDAnnotations == nil && ec.idNodes == nil {
		return nil
	}

	wanted := make(map[string]struct{}, len(tokens))
	for _, token := range tokens {
		wanted[token] = struct{}{}
	}

	// Gather every candidate is-id node exactly once. A node is an is-id node
	// when its type annotation is xs:ID (or derived from it) OR the xsd validator
	// flagged it in the PSVI is-id set — the latter covers a singleton list of
	// xs:ID and a union that selects an xs:ID-derived member, neither of which is
	// a name-level subtype of xs:ID. Both regular and preserved ID annotations are
	// consulted (the latter are kept when input-type-annotations="strip" removes
	// regular annotations but retains the is-id/is-idref properties).
	candidates := make(map[helium.Node]struct{})
	for _, annMap := range []map[helium.Node]string{ec.typeAnnotations, ec.preservedIDAnnotations} {
		for node, typeName := range annMap {
			if annotationMatchesIDType(typeName, ec) {
				candidates[node] = struct{}{}
			}
		}
	}
	for node := range ec.idNodes {
		candidates[node] = struct{}{}
	}

	seen := make(map[helium.Node]struct{})
	var nodes []helium.Node
	for node := range candidates {
		if ixpath.DocumentRoot(node) != doc {
			continue
		}
		result, ok := idNodeResult(node, wanted, elementWithID)
		if !ok {
			continue
		}
		if _, dup := seen[result]; dup {
			continue
		}
		seen[result] = struct{}{}
		nodes = append(nodes, result)
	}
	return nodes
}

// idNodeResult maps an is-id node to the node fn:id / fn:element-with-id should
// return for it, or (nil, false) when the node's value is not among wanted (or
// it has no element parent where one is required). For an ID attribute both
// functions return the bearing element. For an ID-typed element, fn:id returns
// that element itself and fn:element-with-id (elementWithID) returns its parent.
func idNodeResult(node helium.Node, wanted map[string]struct{}, elementWithID bool) (helium.Node, bool) {
	switch typed := node.(type) {
	case *helium.Attribute:
		if _, ok := wanted[strings.TrimSpace(ixpath.StringValue(typed))]; !ok {
			return nil, false
		}
		parent, ok := typed.Parent().(*helium.Element)
		if !ok {
			return nil, false
		}
		return parent, true
	case *helium.Element:
		if _, ok := wanted[strings.TrimSpace(ixpath.StringValue(typed))]; !ok {
			return nil, false
		}
		if elementWithID {
			parent, ok := typed.Parent().(*helium.Element)
			if !ok {
				return nil, false
			}
			return parent, true
		}
		return typed, true
	}
	return nil, false
}

func annotationMatchesIDType(typeName string, ec *evalContext) bool {
	if typeName == "" {
		return false
	}
	if isSubtypeOf(typeName, TypeID) {
		return true
	}
	if ec == nil || ec.schemaDeclarations == nil {
		return false
	}
	return ec.schemaDeclarations.IsSubtypeOf(typeName, TypeID)
}

// annotationMatchesIDRefType checks if an attribute node has an IDREF/IDREFS
// type annotation in either typeAnnotations or preservedIDAnnotations.
func annotationMatchesIDRefType(ec *evalContext, attr *helium.Attribute) bool {
	for _, annMap := range []map[helium.Node]string{ec.typeAnnotations, ec.preservedIDAnnotations} {
		if ann, ok := annMap[attr]; ok && isIDRefAnnotation(ann, ec) {
			return true
		}
	}
	return false
}

// isIDRefAnnotation returns true if the type name is xs:IDREF, xs:IDREFS, or
// a subtype thereof.
func isIDRefAnnotation(typeName string, ec *evalContext) bool {
	if typeName == "" {
		return false
	}
	if isSubtypeOf(typeName, TypeIDREF) || isSubtypeOf(typeName, TypeIDREFS) {
		return true
	}
	if ec != nil && ec.schemaDeclarations != nil {
		return ec.schemaDeclarations.IsSubtypeOf(typeName, TypeIDREF) || ec.schemaDeclarations.IsSubtypeOf(typeName, TypeIDREFS)
	}
	return false
}

func fnIDRef(ctx context.Context, args []Sequence) (Sequence, error) {
	doc, err := resolveIDLookupDocument(ctx, args)
	if err != nil {
		return nil, err
	}

	// Unlike fn:id, fn:idref does NOT tokenize its argument strings.
	// Each string in the input sequence is matched as-is against individual
	// IDREF/IDREFS tokens in the document.
	atoms, err := AtomizeSequence(args[0])
	if err != nil {
		return nil, err
	}
	if len(atoms) == 0 {
		return validNilSequence, nil
	}

	wanted := make(map[string]struct{}, len(atoms))
	for _, atom := range atoms {
		s, sErr := atomicToString(atom)
		if sErr != nil {
			return nil, sErr
		}
		if s != "" {
			wanted[s] = struct{}{}
		}
	}

	ec := getFnContext(ctx)
	var nodes []helium.Node
	walkErr := helium.Walk(doc, helium.NodeWalkerFunc(func(n helium.Node) error {
		// Honor context cancellation and the established op-limit on every
		// visited node so a large document cannot run unbounded.
		if ec != nil {
			if err := ec.countOps(ctx, 1); err != nil {
				return err
			}
		} else if err := ctx.Err(); err != nil {
			return err
		}
		elem, ok := n.(*helium.Element)
		if !ok {
			return nil
		}
		for _, attr := range elem.Attributes() {
			isIDRef := attr.AType() == enum.AttrIDRef || attr.AType() == enum.AttrIDRefs
			if !isIDRef && ec != nil {
				isIDRef = annotationMatchesIDRefType(ec, attr)
			}
			if !isIDRef {
				continue
			}
			for token := range strings.FieldsSeq(attr.Value()) {
				if _, ok := wanted[token]; ok {
					nodes = append(nodes, attr)
					break
				}
			}
		}
		// Also check element content for IDREF type annotations.
		if ec != nil {
			isIDRef := false
			for _, annMap := range []map[helium.Node]string{ec.typeAnnotations, ec.preservedIDAnnotations} {
				if ann, ok := annMap[elem]; ok && isIDRefAnnotation(ann, ec) {
					isIDRef = true
					break
				}
			}
			if isIDRef {
				for token := range strings.FieldsSeq(strings.TrimSpace(ixpath.StringValue(elem))) {
					if _, ok := wanted[token]; ok {
						nodes = append(nodes, elem)
						break
					}
				}
			}
		}
		return nil
	}))
	if walkErr != nil {
		return nil, walkErr
	}
	return sequenceFromDocOrderedNodes(ctx, nodes)
}

func fnElementWithID(ctx context.Context, args []Sequence) (Sequence, error) {
	return idLookup(ctx, args, true)
}

func fnCollection(ctx context.Context, args []Sequence) (Sequence, error) {
	uri, hasURI, err := collectionURIArg(ctx, args, "fn:collection")
	if err != nil {
		return nil, err
	}
	if hasURI {
		uri, err = resolveCollectionURI(ctx, uri, "fn:collection")
		if err != nil {
			return nil, err
		}
	} else {
		uri = ""
	}

	ec := getFnContext(ctx)
	if ec == nil || ec.collectionResolver == nil {
		return nil, &XPathError{Code: errCodeFODC0002, Message: "fn:collection: no collection resolver configured"}
	}
	seq, err := ec.collectionResolver.ResolveCollection(uri)
	if err != nil {
		return nil, &XPathError{Code: errCodeFODC0002, Message: fmt.Sprintf("fn:collection: cannot resolve collection: %v", err)}
	}
	return seq, nil
}

func fnURICollection(ctx context.Context, args []Sequence) (Sequence, error) {
	uri, hasURI, err := collectionURIArg(ctx, args, "fn:uri-collection")
	if err != nil {
		return nil, err
	}
	if hasURI {
		uri, err = resolveCollectionURI(ctx, uri, "fn:uri-collection")
		if err != nil {
			return nil, err
		}
	} else {
		uri = ""
	}

	ec := getFnContext(ctx)
	if ec == nil || ec.collectionResolver == nil {
		return nil, &XPathError{Code: errCodeFODC0002, Message: "fn:uri-collection: no collection resolver configured"}
	}
	uris, err := ec.collectionResolver.ResolveURICollection(uri)
	if err != nil {
		return nil, &XPathError{Code: errCodeFODC0002, Message: fmt.Sprintf("fn:uri-collection: cannot resolve collection: %v", err)}
	}

	result := make(ItemSlice, 0, len(uris))
	for _, uri := range uris {
		result = append(result, AtomicValue{TypeName: TypeAnyURI, Value: uri})
	}
	return result, nil
}

func fnDoc(ctx context.Context, args []Sequence) (Sequence, error) {
	if seqLen(args[0]) == 0 {
		return validNilSequence, nil
	}
	uri, err := docURIArg(ctx, args[0], "fn:doc")
	if err != nil {
		return nil, err
	}
	doc, err := loadDoc(ctx, uri)
	if err != nil {
		return nil, err
	}
	return SingleNode(doc), nil
}

func fnDocAvailable(ctx context.Context, args []Sequence) (Sequence, error) {
	if seqLen(args[0]) == 0 {
		return SingleBoolean(false), nil
	}
	uri, err := docURIArg(ctx, args[0], "fn:doc-available")
	if err != nil {
		return nil, err
	}
	_, err = loadDoc(ctx, uri)
	return SingleBoolean(err == nil), nil
}

func loadDoc(ctx context.Context, uri string) (helium.Node, error) {
	if strings.Contains(uri, "#") {
		return nil, &XPathError{Code: errCodeFODC0005, Message: "fn:doc: URI must not contain a fragment identifier"}
	}

	resolved, err := resolveDocURI(ctx, uri)
	if err != nil {
		return nil, err
	}
	// An empty argument resolves to the base URI verbatim, which may itself
	// carry a fragment identifier. Re-check the resolved URI so doc("") with a
	// fragmented base URI is rejected the same as a fragmented argument.
	if strings.Contains(resolved, "#") {
		return nil, &XPathError{Code: errCodeFODC0005, Message: "fn:doc: URI must not contain a fragment identifier"}
	}
	ec := getFnContext(ctx)
	if ec != nil {
		if doc, ok := ec.docCache[resolved]; ok {
			return doc, nil
		}
	}

	data, err := unparsedtext.ReadURI(ctx, unparsedTextConfig(ctx), resolved)
	if err != nil {
		return nil, &XPathError{Code: errCodeFODC0002, Message: fmt.Sprintf("fn:doc: cannot retrieve resource: %v", err)}
	}

	// The parser governs external entity expansion and network access for the
	// retrieved document. The default helium.NewParser() is safe-by-default
	// (XXE blocked, network disabled); an injected parser's policy wins.
	//
	// The retrieved resource is already bounded by the resource read cap
	// (ec.maxResourceBytes), so the parser's separate per-node content cap is
	// redundant on this path. Align it with the resource cap so a raised
	// MaxResourceBytes actually accepts large single-node content instead of
	// being silently overridden by the parser's default node-content cap. The
	// 0/negative/positive convention (default / unbounded / explicit) matches
	// between MaxResourceBytes and MaxNodeContentSize.
	p := ec.xmlParser().MaxNodeContentSize(clampInt64ToInt(ec.maxResourceBytes))
	doc, err := p.Parse(ctx, data)
	if err != nil {
		return nil, &XPathError{Code: errCodeFODC0002, Message: fmt.Sprintf("fn:doc: cannot parse document: %v", err)}
	}
	doc.SetURL(resolved)
	if ec != nil {
		ec.docCache[resolved] = doc
	}
	return doc, nil
}

// clampInt64ToInt narrows an int64 limit to int without wrapping on 32-bit
// platforms. The sign is preserved so the 0/negative/positive limit convention
// (default / unbounded / explicit) survives the conversion.
func clampInt64ToInt(n int64) int {
	if n > int64(math.MaxInt) {
		return math.MaxInt
	}
	if n < int64(math.MinInt) {
		return math.MinInt
	}
	return int(n)
}

func docURIArg(ctx context.Context, seq Sequence, fnName string) (string, error) {
	if seqLen(seq) > 1 {
		return "", &XPathError{Code: lexicon.ErrXPTY0004, Message: fnName + ": expected xs:string?, got sequence of length > 1"}
	}
	return coerceArgToString(ctx, seq)
}

func collectionURIArg(ctx context.Context, args []Sequence, fnName string) (string, bool, error) {
	if len(args) == 0 || seqLen(args[0]) == 0 {
		return "", false, nil
	}
	uri, err := docURIArg(ctx, args[0], fnName)
	if err != nil {
		return "", false, err
	}
	return uri, true, nil
}

func resolveCollectionURI(ctx context.Context, uri, fnName string) (string, error) {
	if uri == "" {
		return "", nil
	}

	ref, err := parseURIReference(uri)
	if err != nil {
		return "", &XPathError{Code: errCodeFODC0004, Message: fmt.Sprintf("%s: invalid URI: %v", fnName, err)}
	}
	if ref.IsAbs() {
		return uri, nil
	}

	if baseURI := baseURIFromContext(ctx); baseURI != "" {
		if _, err := parseURIReference(baseURI); err != nil {
			return "", &XPathError{Code: errCodeFODC0004, Message: fmt.Sprintf("%s: invalid base URI: %v", fnName, err)}
		}
		resolved, err := resolveURIReference(baseURI, uri)
		if err != nil {
			return "", &XPathError{Code: errCodeFODC0004, Message: fmt.Sprintf("%s: invalid base URI: %v", fnName, err)}
		}
		return resolved, nil
	}

	return uri, nil
}

func resolveDocURI(ctx context.Context, uri string) (string, error) {
	if uri == "" {
		// doc("") returns the document at the base URI (XSLT §14.1, XPath §14.1.1)
		if ec := getFnContext(ctx); ec != nil && ec.baseURI != "" {
			return ec.baseURI, nil
		}
		return "", &XPathError{Code: errCodeFODC0002, Message: "fn:doc: empty URI"}
	}

	// Absolute file path — use directly
	if strings.HasPrefix(uri, "/") {
		return uri, nil
	}

	parsed, err := parseURIReference(uri)
	if err != nil {
		return "", &XPathError{Code: errCodeFODC0002, Message: fmt.Sprintf("fn:doc: invalid URI: %s", uri)}
	}

	if parsed.Scheme != "" {
		if !isSupportedResourceScheme(parsed.Scheme) {
			return "", &XPathError{Code: errCodeFODC0002, Message: fmt.Sprintf("fn:doc: unsupported URI scheme: %s", parsed.Scheme)}
		}
		return uri, nil
	}

	// Relative URI — resolve against base URI
	if baseURI := baseURIFromContext(ctx); baseURI != "" {
		resolved, err := resolveURIReference(baseURI, uri)
		if err == nil {
			return resolved, nil
		}
	}
	return uri, nil
}

func resolveIDLookupDocument(ctx context.Context, args []Sequence) (*helium.Document, error) {
	var node helium.Node
	if len(args) > 1 {
		if seqLen(args[1]) != 1 {
			return nil, &XPathError{Code: lexicon.ErrXPTY0004, Message: "fn:id second argument must be a single node"}
		}
		ni, ok := args[1].Get(0).(NodeItem)
		if !ok {
			return nil, &XPathError{Code: lexicon.ErrXPTY0004, Message: "fn:id second argument must be a node"}
		}
		node = ni.Node
	} else {
		fc := getFnContext(ctx)
		switch {
		case fc == nil || (fc.node == nil && fc.contextItem == nil):
			return nil, &XPathError{Code: errCodeXPDY0002, Message: errMsgContextItemAbsent}
		case fc.node != nil:
			node = fc.node
		default:
			return nil, &XPathError{Code: lexicon.ErrXPTY0004, Message: "context item is not a node"}
		}
	}

	root := ixpath.DocumentRoot(node)
	doc, ok := root.(*helium.Document)
	if !ok {
		return nil, &XPathError{Code: errCodeFODC0001, Message: "fn:id requires a node whose root is a document node"}
	}
	return doc, nil
}

func idLookupTokens(seq Sequence) ([]string, error) {
	atoms, err := AtomizeSequence(seq)
	if err != nil {
		return nil, err
	}

	var tokens []string
	for _, atom := range atoms {
		s, err := atomicToString(atom)
		if err != nil {
			return nil, err
		}
		for token := range strings.FieldsSeq(s) {
			if xmlchar.IsValidNCName(token) {
				tokens = append(tokens, token)
			}
		}
	}
	return tokens, nil
}

func sequenceFromDocOrderedNodes(ctx context.Context, nodes []helium.Node) (Sequence, error) {
	if len(nodes) == 0 {
		return validNilSequence, nil
	}

	cache := &ixpath.DocOrderCache{}
	maxNodes := maxNodeSetLength
	fc := getFnContext(ctx)
	if fc != nil {
		cache = fc.docOrder
		maxNodes = fc.maxNodes
	}

	deduped, err := ixpath.DeduplicateNodes(nodes, cache, maxNodes)
	if err != nil {
		return nil, err
	}

	result := make(ItemSlice, 0, len(deduped))
	for _, node := range deduped {
		result = append(result, nodeItemFor(ctx, fc, node))
	}
	return result, nil
}

// nodeArgOrCtx extracts a node from the first argument or falls back to the context node.
func nodeArgOrCtx(ctx context.Context, args []Sequence) (helium.Node, error) {
	if len(args) == 0 {
		fc := getFnContext(ctx)
		if fc == nil {
			return nil, &XPathError{Code: errCodeXPDY0002, Message: errMsgContextItemAbsent}
		}
		if fc.node == nil {
			return nil, &XPathError{Code: lexicon.ErrXPTY0004, Message: "context item is not a node"}
		}
		return fc.node, nil
	}
	if seqLen(args[0]) == 0 {
		return nil, nil //nolint:nilnil
	}
	if seqLen(args[0]) > 1 {
		// XPath 1.0 compatibility mode uses the first node; otherwise a >1 node
		// argument is a type error.
		if !getFnContext(ctx).xpath10CompatMode() {
			return nil, &XPathError{Code: lexicon.ErrXPTY0004, Message: "expected single node, got sequence of length > 1"}
		}
	}
	ni, ok := args[0].Get(0).(NodeItem)
	if !ok {
		return nil, &XPathError{Code: lexicon.ErrXPTY0004, Message: "expected node"}
	}
	return ni.Node, nil
}
