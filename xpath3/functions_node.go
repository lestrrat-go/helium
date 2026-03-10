package xpath3

import (
	"context"
	"fmt"
	"math"
	"strings"

	"github.com/lestrrat-go/helium"
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
		return nil, nil
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
	return nil, nil
}

func fnNilled(ctx context.Context, args []Sequence) (Sequence, error) {
	n, err := nodeArgOrCtx(ctx, args)
	if err != nil {
		return nil, err
	}
	if n == nil {
		return nil, nil
	}
	// Helium doesn't support schema-validated nilled; always false for elements.
	if n.Type() == helium.ElementNode {
		return SingleBoolean(false), nil
	}
	return nil, nil
}

func fnData(_ context.Context, args []Sequence) (Sequence, error) {
	if len(args) == 0 {
		return nil, &XPathError{Code: "XPTY0004", Message: "data() requires an argument when context item is absent"}
	}
	atoms, err := AtomizeSequence(args[0])
	if err != nil {
		return nil, err
	}
	result := make(Sequence, len(atoms))
	for i, a := range atoms {
		result[i] = a
	}
	return result, nil
}

func fnBaseURI(ctx context.Context, args []Sequence) (Sequence, error) {
	n, err := nodeArgOrCtx(ctx, args)
	if err != nil {
		return nil, err
	}
	if n == nil {
		return nil, nil
	}
	// Walk up looking for xml:base
	for cur := n; cur != nil; cur = cur.Parent() {
		if elem, ok := cur.(*helium.Element); ok {
			for _, attr := range elem.Attributes() {
				if attr.LocalName() == "base" && attr.URI() == helium.XMLNamespace {
					return SingleString(attr.Value()), nil
				}
			}
		}
	}
	return SingleString(""), nil
}

func fnDocumentURI(ctx context.Context, args []Sequence) (Sequence, error) {
	n, err := nodeArgOrCtx(ctx, args)
	if err != nil {
		return nil, err
	}
	if n == nil {
		return nil, nil
	}
	if n.Type() != helium.DocumentNode {
		return nil, nil
	}
	if doc, ok := n.(*helium.Document); ok {
		if uri := doc.URL(); uri != "" {
			return SingleAtomic(AtomicValue{TypeName: TypeAnyURI, Value: uri}), nil
		}
	}
	return nil, nil
}

func fnRoot(ctx context.Context, args []Sequence) (Sequence, error) {
	n, err := nodeArgOrCtx(ctx, args)
	if err != nil {
		return nil, err
	}
	if n == nil {
		return nil, nil
	}
	return SingleNode(ixpath.DocumentRoot(n)), nil
}

func fnPath(ctx context.Context, args []Sequence) (Sequence, error) {
	n, err := nodeArgOrCtx(ctx, args)
	if err != nil {
		return nil, err
	}
	if n == nil {
		return nil, nil
	}
	return SingleString(buildNodePath(n)), nil
}

func buildNodePath(n helium.Node) string {
	if n.Type() == helium.DocumentNode {
		return "/"
	}
	var parts []string
	for cur := n; cur != nil && cur.Type() != helium.DocumentNode; cur = cur.Parent() {
		switch cur.Type() {
		case helium.ElementNode:
			local := ixpath.LocalNameOf(cur)
			uri := ixpath.NodeNamespaceURI(cur)
			pos := elementPosition(cur)
			if uri != "" {
				parts = append(parts, fmt.Sprintf("Q{%s}%s[%d]", uri, local, pos))
			} else {
				parts = append(parts, fmt.Sprintf("%s[%d]", local, pos))
			}
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
		}
	}
	// Reverse
	for i, j := 0, len(parts)-1; i < j; i, j = i+1, j-1 {
		parts[i], parts[j] = parts[j], parts[i]
	}
	return "/" + strings.Join(parts, "/")
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
	return SingleBoolean(n.FirstChild() != nil), nil
}

func fnInnermost(_ context.Context, args []Sequence) (Sequence, error) {
	nodes, ok := NodesFrom(args[0])
	if !ok {
		return nil, &XPathError{Code: "XPTY0004", Message: "innermost() requires node-set"}
	}
	nodeSet := make(map[helium.Node]bool, len(nodes))
	for _, n := range nodes {
		nodeSet[n] = true
	}
	var result Sequence
	for _, n := range nodes {
		isInner := true
		for p := n.Parent(); p != nil; p = p.Parent() {
			if nodeSet[p] {
				isInner = false
				break
			}
		}
		if isInner {
			result = append(result, NodeItem{Node: n})
		}
	}
	return result, nil
}

func fnOutermost(_ context.Context, args []Sequence) (Sequence, error) {
	nodes, ok := NodesFrom(args[0])
	if !ok {
		return nil, &XPathError{Code: "XPTY0004", Message: "outermost() requires node-set"}
	}
	nodeSet := make(map[helium.Node]bool, len(nodes))
	for _, n := range nodes {
		nodeSet[n] = true
	}
	var result Sequence
	for _, n := range nodes {
		isOuter := true
		for p := n.Parent(); p != nil; p = p.Parent() {
			if nodeSet[p] {
				isOuter = false
				break
			}
		}
		if isOuter {
			result = append(result, NodeItem{Node: n})
		}
	}
	return result, nil
}

func fnLang(ctx context.Context, args []Sequence) (Sequence, error) {
	langArg := strings.ToLower(seqToString(args[0]))
	var n helium.Node
	if len(args) > 1 {
		nodes, ok := NodesFrom(args[1])
		if !ok || len(nodes) == 0 {
			return SingleBoolean(false), nil
		}
		n = nodes[0]
	} else {
		fc := getFnContext(ctx)
		if fc == nil || fc.node == nil {
			return SingleBoolean(false), nil
		}
		n = fc.node
	}
	for cur := n; cur != nil; cur = cur.Parent() {
		elem, ok := cur.(*helium.Element)
		if !ok {
			continue
		}
		for _, attr := range elem.Attributes() {
			if attr.LocalName() == "lang" && attr.URI() == helium.XMLNamespace {
				val := strings.ToLower(attr.Value())
				if val == langArg || strings.HasPrefix(val, langArg+"-") {
					return SingleBoolean(true), nil
				}
				return SingleBoolean(false), nil
			}
		}
	}
	return SingleBoolean(false), nil
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
		if fc == nil || fc.node == nil {
			return SingleDouble(math.NaN()), nil
		}
		s := ixpath.StringValue(fc.node)
		a, err := CastFromString(s, TypeDouble)
		if err != nil {
			return SingleDouble(math.NaN()), nil
		}
		return SingleDouble(a.DoubleVal()), nil
	}
	if len(args[0]) == 0 {
		return SingleDouble(math.NaN()), nil
	}
	a, err := AtomizeItem(args[0][0])
	if err != nil {
		return SingleDouble(math.NaN()), nil
	}
	return SingleDouble(a.ToFloat64()), nil
}

func fnGenerateID(ctx context.Context, args []Sequence) (Sequence, error) {
	n, err := nodeArgOrCtx(ctx, args)
	if err != nil {
		return nil, err
	}
	if n == nil {
		return SingleString(""), nil
	}
	return SingleString(fmt.Sprintf("id%p", n)), nil
}

func fnParseXML(_ context.Context, args []Sequence) (Sequence, error) {
	if len(args[0]) == 0 {
		return nil, nil
	}
	s := seqToString(args[0])
	doc, err := helium.Parse(context.Background(), []byte(s))
	if err != nil {
		return nil, &XPathError{Code: "FODC0006", Message: fmt.Sprintf("parse-xml: %v", err)}
	}
	return Sequence{NodeItem{Node: doc}}, nil
}

func fnParseXMLFragment(_ context.Context, args []Sequence) (Sequence, error) {
	if len(args[0]) == 0 {
		return nil, nil
	}
	s := seqToString(args[0])
	// Wrap in a root element to parse as fragment
	wrapped := "<_fragment_>" + s + "</_fragment_>"
	doc, err := helium.Parse(context.Background(), []byte(wrapped))
	if err != nil {
		return nil, &XPathError{Code: "FODC0006", Message: fmt.Sprintf("parse-xml-fragment: %v", err)}
	}
	return Sequence{NodeItem{Node: doc}}, nil
}

func fnID(_ context.Context, _ []Sequence) (Sequence, error) {
	// Requires DTD/schema info for ID attribute detection
	return nil, nil
}

func fnIDRef(_ context.Context, _ []Sequence) (Sequence, error) {
	return nil, nil
}

func fnElementWithID(_ context.Context, _ []Sequence) (Sequence, error) {
	return nil, nil
}

func fnCollection(_ context.Context, _ []Sequence) (Sequence, error) {
	return nil, &XPathError{Code: "FODC0002", Message: "fn:collection: not supported"}
}

func fnURICollection(_ context.Context, _ []Sequence) (Sequence, error) {
	return nil, &XPathError{Code: "FODC0002", Message: "fn:uri-collection: not supported"}
}

func fnDoc(_ context.Context, _ []Sequence) (Sequence, error) {
	return nil, &XPathError{Code: "FODC0002", Message: "fn:doc: URI resolution not supported"}
}

func fnDocAvailable(_ context.Context, _ []Sequence) (Sequence, error) {
	return SingleBoolean(false), nil
}

// nodeArgOrCtx extracts a node from the first argument or falls back to the context node.
func nodeArgOrCtx(ctx context.Context, args []Sequence) (helium.Node, error) {
	if len(args) == 0 {
		fc := getFnContext(ctx)
		if fc == nil || fc.node == nil {
			return nil, nil
		}
		return fc.node, nil
	}
	if len(args[0]) == 0 {
		return nil, nil
	}
	ni, ok := args[0][0].(NodeItem)
	if !ok {
		return nil, &XPathError{Code: "XPTY0004", Message: "expected node"}
	}
	return ni.Node, nil
}
