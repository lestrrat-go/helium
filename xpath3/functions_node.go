package xpath3

import (
	"context"
	"fmt"
	"math"
	"net/url"
	"strings"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/enum"
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

func fnData(ctx context.Context, args []Sequence) (Sequence, error) {
	if len(args) == 0 {
		fc := getFnContext(ctx)
		if fc == nil {
			return nil, &XPathError{Code: errCodeXPDY0002, Message: "data() requires a context item"}
		}
		if fc.contextItem != nil {
			args = []Sequence{{fc.contextItem}}
		} else if fc.node != nil {
			args = []Sequence{{NodeItem{Node: fc.node}}}
		} else {
			return nil, &XPathError{Code: errCodeXPDY0002, Message: "data() requires a context item"}
		}
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
	if doc, ok := n.(*helium.Document); ok {
		if uri := doc.URL(); uri != "" {
			return SingleAtomic(AtomicValue{TypeName: TypeAnyURI, Value: uri}), nil
		}
	}
	// Walk up looking for xml:base
	for cur := n; cur != nil; cur = cur.Parent() {
		if elem, ok := cur.(*helium.Element); ok {
			for _, attr := range elem.Attributes() {
				if attr.LocalName() == "base" && attr.URI() == helium.XMLNamespace {
					return SingleAtomic(AtomicValue{TypeName: TypeAnyURI, Value: attr.Value()}), nil
				}
			}
		}
	}
	if doc := n.OwnerDocument(); doc != nil {
		if uri := doc.URL(); uri != "" {
			return SingleAtomic(AtomicValue{TypeName: TypeAnyURI, Value: uri}), nil
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
		if doc.HasProperty(helium.DocInternal) {
			return nil, nil
		}
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
	// Per XPath spec, only document and element nodes can have children
	switch n.(type) {
	case *helium.Document, *helium.Element:
		return SingleBoolean(n.FirstChild() != nil), nil
	default:
		return SingleBoolean(false), nil
	}
}

func fnInnermost(_ context.Context, args []Sequence) (Sequence, error) {
	nodes, ok := NodesFrom(args[0])
	if !ok {
		return nil, &XPathError{Code: errCodeXPTY0004, Message: "innermost() requires node-set"}
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
	var result Sequence
	for _, n := range nodes {
		if !ancestors[n] {
			result = append(result, NodeItem{Node: n})
		}
	}
	return result, nil
}

func fnOutermost(_ context.Context, args []Sequence) (Sequence, error) {
	nodes, ok := NodesFrom(args[0])
	if !ok {
		return nil, &XPathError{Code: errCodeXPTY0004, Message: "outermost() requires node-set"}
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
	langArg, err := coerceArgToString(args[0])
	if err != nil {
		return nil, err
	}
	langArg = strings.ToLower(langArg)
	var n helium.Node
	if len(args) > 1 {
		nodes, ok := NodesFrom(args[1])
		if !ok || len(nodes) == 0 {
			return nil, &XPathError{Code: errCodeXPTY0004, Message: "fn:lang second argument must be a node"}
		}
		n = nodes[0]
	} else {
		fc := getFnContext(ctx)
		if fc == nil || (fc.contextItem == nil && fc.node == nil) {
			return nil, &XPathError{Code: errCodeXPDY0002, Message: "context item is absent"}
		}
		if fc.node == nil {
			return nil, &XPathError{Code: errCodeXPTY0004, Message: "context item is not a node for fn:lang"}
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
		if fc == nil || (fc.contextItem == nil && fc.node == nil) {
			return nil, &XPathError{Code: errCodeXPDY0002, Message: "context item is absent"}
		}
		s, ok := fc.contextStringValue()
		if !ok {
			return SingleDouble(math.NaN()), nil
		}
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
	dbl, err := CastAtomic(a, TypeDouble)
	if err != nil {
		return SingleDouble(math.NaN()), nil
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
	return SingleString(stableNodeID(n)), nil
}

func fnParseXML(ctx context.Context, args []Sequence) (Sequence, error) {
	if len(args[0]) == 0 {
		return nil, nil
	}
	s, err := coerceArgToString(args[0])
	if err != nil {
		return nil, err
	}
	parser := helium.NewParser()
	if ec := getFnContext(ctx); ec != nil && ec.baseURI != "" {
		parser.SetBaseURI(ec.baseURI)
	}
	doc, err := parser.Parse(ctx, []byte(s))
	if err != nil {
		return nil, &XPathError{Code: errCodeFODC0006, Message: fmt.Sprintf("parse-xml: %v", err)}
	}
	doc.SetProperties(doc.Properties() | helium.DocInternal)
	return Sequence{NodeItem{Node: doc}}, nil
}

func fnParseXMLFragment(ctx context.Context, args []Sequence) (Sequence, error) {
	if len(args[0]) == 0 {
		return nil, nil
	}
	s, err := coerceArgToString(args[0])
	if err != nil {
		return nil, err
	}

	s, err = stripXMLTextDeclaration(s)
	if err != nil {
		return nil, err
	}

	doc := helium.NewDocument("1.0", "", helium.StandaloneImplicitNo)
	doc.SetProperties(doc.Properties() | helium.DocInternal)
	if ec := getFnContext(ctx); ec != nil && ec.baseURI != "" {
		doc.SetURL(ec.baseURI)
	}

	first, err := helium.ParseInNodeContext(ctx, doc, []byte(s))
	if err != nil {
		return nil, &XPathError{Code: errCodeFODC0006, Message: fmt.Sprintf("parse-xml-fragment: %v", err)}
	}

	for cur := first; cur != nil; {
		next := cur.NextSibling()
		cur.SetPrevSibling(nil)
		cur.SetNextSibling(nil)
		if err := doc.AddChild(cur); err != nil {
			return nil, &XPathError{Code: errCodeFODC0006, Message: fmt.Sprintf("parse-xml-fragment: %v", err)}
		}
		cur = next
	}

	return Sequence{NodeItem{Node: doc}}, nil
}

func stripXMLTextDeclaration(s string) (string, error) {
	if !strings.HasPrefix(s, "<?xml") {
		return s, nil
	}

	end := strings.Index(s, "?>")
	if end < 0 {
		return "", &XPathError{Code: errCodeFODC0006, Message: "parse-xml-fragment: malformed text declaration"}
	}

	decl := s[len("<?xml"):end]
	names, err := parseXMLPseudoAttributes(decl)
	if err != nil {
		return "", &XPathError{Code: errCodeFODC0006, Message: "parse-xml-fragment: malformed text declaration"}
	}

	if len(names) == 0 {
		return "", &XPathError{Code: errCodeFODC0006, Message: "parse-xml-fragment: malformed text declaration"}
	}

	switch len(names) {
	case 1:
		if names[0] != "encoding" {
			return "", &XPathError{Code: errCodeFODC0006, Message: "parse-xml-fragment: malformed text declaration"}
		}
	case 2:
		if names[0] != "version" || names[1] != "encoding" {
			return "", &XPathError{Code: errCodeFODC0006, Message: "parse-xml-fragment: malformed text declaration"}
		}
	default:
		return "", &XPathError{Code: errCodeFODC0006, Message: "parse-xml-fragment: malformed text declaration"}
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
		case "version", "encoding":
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
	doc, err := resolveIDLookupDocument(ctx, args)
	if err != nil {
		return nil, err
	}

	tokens, err := idLookupTokens(args[0])
	if err != nil {
		return nil, err
	}
	if len(tokens) == 0 {
		return nil, nil
	}

	nodes := make([]helium.Node, 0, len(tokens))
	for _, token := range tokens {
		if elem := doc.GetElementByID(token); elem != nil {
			nodes = append(nodes, elem)
		}
	}
	return sequenceFromDocOrderedNodes(ctx, nodes)
}

func fnIDRef(ctx context.Context, args []Sequence) (Sequence, error) {
	doc, err := resolveIDLookupDocument(ctx, args)
	if err != nil {
		return nil, err
	}

	tokens, err := idLookupTokens(args[0])
	if err != nil {
		return nil, err
	}
	if len(tokens) == 0 {
		return nil, nil
	}

	wanted := make(map[string]struct{}, len(tokens))
	for _, token := range tokens {
		wanted[token] = struct{}{}
	}

	var nodes []helium.Node
	_ = helium.Walk(doc, func(n helium.Node) error {
		elem, ok := n.(*helium.Element)
		if !ok {
			return nil
		}
		for _, attr := range elem.Attributes() {
			if attr.AType() != enum.AttrIDRef && attr.AType() != enum.AttrIDRefs {
				continue
			}
			for _, token := range strings.Fields(attr.Value()) {
				if _, ok := wanted[token]; ok {
					nodes = append(nodes, attr)
					break
				}
			}
		}
		return nil
	})
	return sequenceFromDocOrderedNodes(ctx, nodes)
}

func fnElementWithID(ctx context.Context, args []Sequence) (Sequence, error) {
	return fnID(ctx, args)
}

func fnCollection(ctx context.Context, args []Sequence) (Sequence, error) {
	uri, hasURI, err := collectionURIArg(args, "fn:collection")
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
	uri, hasURI, err := collectionURIArg(args, "fn:uri-collection")
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

	result := make(Sequence, 0, len(uris))
	for _, uri := range uris {
		result = append(result, AtomicValue{TypeName: TypeAnyURI, Value: uri})
	}
	return result, nil
}

func fnDoc(ctx context.Context, args []Sequence) (Sequence, error) {
	if len(args[0]) == 0 {
		return nil, nil
	}
	uri, err := docURIArg(args[0], "fn:doc")
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
	if len(args[0]) == 0 {
		return SingleBoolean(false), nil
	}
	uri, err := docURIArg(args[0], "fn:doc-available")
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
	if ec := getFnContext(ctx); ec != nil {
		if doc, ok := ec.docCache[resolved]; ok {
			return doc, nil
		}
	}

	data, err := readUnparsedTextURI(ctx, resolved)
	if err != nil {
		return nil, &XPathError{Code: errCodeFODC0002, Message: fmt.Sprintf("fn:doc: cannot retrieve resource: %v", err)}
	}

	doc, err := helium.Parse(ctx, data)
	if err != nil {
		return nil, &XPathError{Code: errCodeFODC0002, Message: fmt.Sprintf("fn:doc: cannot parse document: %v", err)}
	}
	doc.SetURL(resolved)
	if ec := getFnContext(ctx); ec != nil {
		ec.docCache[resolved] = doc
	}
	return doc, nil
}

func docURIArg(seq Sequence, fnName string) (string, error) {
	if len(seq) > 1 {
		return "", &XPathError{Code: errCodeXPTY0004, Message: fnName + ": expected xs:string?, got sequence of length > 1"}
	}
	return coerceArgToString(seq)
}

func collectionURIArg(args []Sequence, fnName string) (string, bool, error) {
	if len(args) == 0 || len(args[0]) == 0 {
		return "", false, nil
	}
	uri, err := docURIArg(args[0], fnName)
	if err != nil {
		return "", false, err
	}
	return uri, true, nil
}

func resolveCollectionURI(ctx context.Context, uri, fnName string) (string, error) {
	if uri == "" {
		return "", nil
	}

	ref, err := url.Parse(uri)
	if err != nil {
		return "", &XPathError{Code: errCodeFODC0004, Message: fmt.Sprintf("%s: invalid URI: %v", fnName, err)}
	}
	if ref.IsAbs() {
		return ref.String(), nil
	}

	if ec := getFnContext(ctx); ec != nil && ec.baseURI != "" {
		base, berr := url.Parse(ec.baseURI)
		if berr != nil {
			return "", &XPathError{Code: errCodeFODC0004, Message: fmt.Sprintf("%s: invalid base URI: %v", fnName, berr)}
		}
		return base.ResolveReference(ref).String(), nil
	}

	return uri, nil
}

func resolveDocURI(ctx context.Context, uri string) (string, error) {
	if uri == "" {
		return "", &XPathError{Code: errCodeFODC0002, Message: "fn:doc: empty URI"}
	}

	// Absolute file path — use directly
	if strings.HasPrefix(uri, "/") {
		return uri, nil
	}

	parsed, err := url.Parse(uri)
	if err != nil {
		return "", &XPathError{Code: errCodeFODC0002, Message: fmt.Sprintf("fn:doc: invalid URI: %s", uri)}
	}

	if parsed.Scheme != "" {
		switch parsed.Scheme {
		case "file", "http", "https":
			return uri, nil
		default:
			return "", &XPathError{Code: errCodeFODC0002, Message: fmt.Sprintf("fn:doc: unsupported URI scheme: %s", parsed.Scheme)}
		}
	}

	// Relative URI — resolve against base URI
	ec := getFnContext(ctx)
	if ec != nil && ec.baseURI != "" {
		base, berr := url.Parse(ec.baseURI)
		if berr == nil {
			return base.ResolveReference(parsed).String(), nil
		}
	}
	return uri, nil
}

func resolveIDLookupDocument(ctx context.Context, args []Sequence) (*helium.Document, error) {
	var node helium.Node
	if len(args) > 1 {
		if len(args[1]) != 1 {
			return nil, &XPathError{Code: errCodeXPTY0004, Message: "fn:id second argument must be a single node"}
		}
		ni, ok := args[1][0].(NodeItem)
		if !ok {
			return nil, &XPathError{Code: errCodeXPTY0004, Message: "fn:id second argument must be a node"}
		}
		node = ni.Node
	} else {
		fc := getFnContext(ctx)
		switch {
		case fc == nil || (fc.node == nil && fc.contextItem == nil):
			return nil, &XPathError{Code: errCodeXPDY0002, Message: "context item is absent"}
		case fc.node != nil:
			node = fc.node
		default:
			return nil, &XPathError{Code: errCodeXPTY0004, Message: "context item is not a node"}
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
		for _, token := range strings.Fields(s) {
			if isValidNCName(token) {
				tokens = append(tokens, token)
			}
		}
	}
	return tokens, nil
}

func sequenceFromDocOrderedNodes(ctx context.Context, nodes []helium.Node) (Sequence, error) {
	if len(nodes) == 0 {
		return nil, nil
	}

	cache := &ixpath.DocOrderCache{}
	maxNodes := maxNodeSetLength
	if fc := getFnContext(ctx); fc != nil {
		cache = fc.docOrder
		maxNodes = fc.maxNodes
	}

	deduped, err := ixpath.DeduplicateNodes(nodes, cache, maxNodes)
	if err != nil {
		return nil, err
	}

	result := make(Sequence, 0, len(deduped))
	for _, node := range deduped {
		result = append(result, NodeItem{Node: node})
	}
	return result, nil
}

// nodeArgOrCtx extracts a node from the first argument or falls back to the context node.
func nodeArgOrCtx(ctx context.Context, args []Sequence) (helium.Node, error) {
	if len(args) == 0 {
		fc := getFnContext(ctx)
		if fc == nil {
			return nil, &XPathError{Code: errCodeXPDY0002, Message: "context item is absent"}
		}
		if fc.node == nil {
			return nil, &XPathError{Code: errCodeXPTY0004, Message: "context item is not a node"}
		}
		return fc.node, nil
	}
	if len(args[0]) == 0 {
		return nil, nil
	}
	if len(args[0]) > 1 {
		return nil, &XPathError{Code: errCodeXPTY0004, Message: "expected single node, got sequence of length > 1"}
	}
	ni, ok := args[0][0].(NodeItem)
	if !ok {
		return nil, &XPathError{Code: errCodeXPTY0004, Message: "expected node"}
	}
	return ni.Node, nil
}
