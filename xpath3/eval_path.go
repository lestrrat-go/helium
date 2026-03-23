package xpath3

import (
	"fmt"
	"math"
	"math/big"
	"strings"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/lexicon"
	ixpath "github.com/lestrrat-go/helium/internal/xpath"
)

func evalLiteral(e LiteralExpr) (Sequence, error) {
	switch v := e.Value.(type) {
	case string:
		return SingleString(v), nil
	case *big.Int:
		return SingleIntegerBig(v), nil
	case *big.Rat:
		return SingleDecimal(v), nil
	case float64:
		return SingleDouble(v), nil
	}
	return nil, fmt.Errorf("%w: literal %T", ErrUnsupportedExpr, e.Value)
}

func evalVariable(ec *evalContext, e VariableExpr) (Sequence, error) {
	if ec.vars != nil {
		// Try exact name first
		if v, ok := ec.vars.Lookup(e.Name); ok {
			return enrichNodeItems(ec, v), nil
		}
		// If EQName (Q{uri}local), normalize to {uri}local and retry
		if strings.HasPrefix(e.Name, "Q{") {
			resolved := e.Name[1:] // strip leading "Q"
			if v, ok := ec.vars.Lookup(resolved); ok {
				return enrichNodeItems(ec, v), nil
			}
		}
		// If prefixed, resolve to {uri}local and retry
		if e.Prefix != "" {
			if uri, ok := ec.namespaces[e.Prefix]; ok {
				local := e.Name[len(e.Prefix)+1:] // strip "prefix:"
				resolved := "{" + uri + "}" + local
				if v, ok := ec.vars.Lookup(resolved); ok {
					return enrichNodeItems(ec, v), nil
				}
			}
		}
	}
	// Fallback to lazy variable resolver (e.g. for XSLT global variables)
	if ec.variableResolver != nil {
		if v, ok, err := ec.variableResolver.ResolveVariable(ec.goCtx, e.Name); err != nil {
			return nil, err
		} else if ok {
			return enrichNodeItems(ec, v), nil
		}
	}
	return nil, fmt.Errorf("%w: $%s", ErrUndefinedVariable, e.Name)
}

// enrichNodeItems ensures NodeItems in a sequence have their TypeAnnotation
// and related fields set from the evalContext's type annotations map. This is
// needed because variables may store NodeItems created before type annotations
// were available (e.g., validated after construction).
func enrichNodeItems(ec *evalContext, seq Sequence) Sequence {
	if ec == nil || ec.typeAnnotations == nil {
		return seq
	}
	needsEnrich := false
	for item := range seqItems(seq) {
		if ni, ok := item.(NodeItem); ok && ni.TypeAnnotation == "" {
			if _, hasAnn := ec.typeAnnotations[ni.Node]; hasAnn {
				needsEnrich = true
				break
			}
		}
	}
	if !needsEnrich {
		return seq
	}
	result := make(ItemSlice, seqLen(seq))
	i := 0
	for item := range seqItems(seq) {
		if ni, ok := item.(NodeItem); ok && ni.TypeAnnotation == "" {
			enriched := nodeItemFor(ec, ni.Node)
			result[i] = enriched
		} else {
			result[i] = item
		}
		i++
	}
	return result
}

func evalSequenceExpr(evalFn exprEvaluator, ec *evalContext, e SequenceExpr) (Sequence, error) {
	if len(e.Items) == 0 {
		return nil, nil
	}
	var result ItemSlice
	for _, item := range e.Items {
		seq, err := evalFn(ec, item)
		if err != nil {
			return nil, err
		}
		result = append(result, seqMaterialize(seq)...)
	}
	return result, nil
}

func evalLocationPath(evalFn exprEvaluator, ec *evalContext, lp *LocationPath) (Sequence, error) {
	var nodes []helium.Node

	if lp.Absolute {
		if ixpath.IsNilNode(ec.node) {
			return nil, &XPathError{Code: errCodeXPDY0002, Message: "context item is absent"}
		}
		root := ixpath.DocumentRoot(ec.node)
		// XPDY0050: the root of the context node's tree must be a document node.
		if root.Type() != helium.DocumentNode && root.Type() != helium.HTMLDocumentNode {
			return nil, &XPathError{Code: errCodeXPDY0050, Message: "root of the tree containing the context node is not a document node"}
		}
		nodes = []helium.Node{root}
	} else {
		if ixpath.IsNilNode(ec.node) {
			return nil, &XPathError{Code: errCodeXPDY0002, Message: "context item is absent"}
		}
		nodes = []helium.Node{ec.node}
	}

	var err error
	for _, step := range lp.Steps {
		if len(step.Predicates) > 0 {
			nodes, err = evalStepWithPredicates(evalFn, ec, nodes, step)
		} else {
			nodes, err = evalStepNoPredicates(ec, nodes, step)
		}
		if err != nil {
			return nil, err
		}
	}

	result := make(ItemSlice, len(nodes))
	for i, n := range nodes {
		result[i] = nodeItemFor(ec, n)
	}
	return result, nil
}

func evalVMLocationPath(evalFn exprEvaluator, ec *evalContext, lp vmLocationPathExpr) (Sequence, error) {
	var nodes []helium.Node

	if lp.Absolute {
		if ixpath.IsNilNode(ec.node) {
			return nil, &XPathError{Code: errCodeXPDY0002, Message: "context item is absent"}
		}
		root := ixpath.DocumentRoot(ec.node)
		// XPDY0050: the root of the context node's tree must be a document node.
		if root.Type() != helium.DocumentNode && root.Type() != helium.HTMLDocumentNode {
			return nil, &XPathError{Code: errCodeXPDY0050, Message: "root of the tree containing the context node is not a document node"}
		}
		nodes = []helium.Node{root}
	} else {
		if ixpath.IsNilNode(ec.node) {
			return nil, &XPathError{Code: errCodeXPDY0002, Message: "context item is absent"}
		}
		nodes = []helium.Node{ec.node}
	}

	var err error
	for _, step := range lp.Steps {
		if len(step.Predicates) > 0 {
			nodes, err = evalVMStepWithPredicates(evalFn, ec, nodes, step)
		} else {
			nodes, err = evalVMStepNoPredicates(ec, nodes, step)
		}
		if err != nil {
			return nil, err
		}
	}

	result := make(ItemSlice, len(nodes))
	for i, n := range nodes {
		result[i] = nodeItemFor(ec, n)
	}
	return result, nil
}

func nodeItemFor(ec *evalContext, n helium.Node) NodeItem {
	ni := NodeItem{Node: n}
	if ec == nil || ec.typeAnnotations == nil {
		return ni
	}
	ni.TypeAnnotation = ec.typeAnnotations[n]
	ni.AtomizedType = atomizedTypeForAnnotation(ni.TypeAnnotation, ec.schemaDeclarations)
	if ec.schemaDeclarations != nil && ni.TypeAnnotation != "" {
		if itemType, ok := ec.schemaDeclarations.ListItemType(ni.TypeAnnotation); ok {
			ni.ListItemType = itemType
		}
	}
	return ni
}

func evalStepWithPredicates(evalFn exprEvaluator, ec *evalContext, nodes []helium.Node, step Step) ([]helium.Node, error) {
	var allFiltered []helium.Node
	for _, n := range nodes {
		matched, traversed, err := appendAxisNodeMatches(nil, ec, n, step.Axis, step.NodeTest)
		if err != nil {
			return nil, err
		}
		if err := ec.countOps(traversed); err != nil {
			return nil, err
		}
		for _, pred := range step.Predicates {
			matched, err = applyPredicate(evalFn, ec, matched, pred)
			if err != nil {
				return nil, err
			}
		}
		allFiltered = append(allFiltered, matched...)
	}
	return ixpath.DeduplicateNodes(allFiltered, ec.docOrder, ec.maxNodes)
}

func evalStepNoPredicates(ec *evalContext, nodes []helium.Node, step Step) ([]helium.Node, error) {
	var next []helium.Node
	for _, n := range nodes {
		var traversed int
		var err error
		next, traversed, err = appendAxisNodeMatches(next, ec, n, step.Axis, step.NodeTest)
		if err != nil {
			return nil, err
		}
		if err := ec.countOps(traversed); err != nil {
			return nil, err
		}
	}
	return ixpath.DeduplicateNodes(next, ec.docOrder, ec.maxNodes)
}

func evalVMStepWithPredicates(evalFn exprEvaluator, ec *evalContext, nodes []helium.Node, step vmLocationStep) ([]helium.Node, error) {
	var allFiltered []helium.Node
	for _, n := range nodes {
		matched, traversed, err := appendAxisNodeMatches(nil, ec, n, step.Axis, step.NodeTest)
		if err != nil {
			return nil, err
		}
		if err := ec.countOps(traversed); err != nil {
			return nil, err
		}
		for _, pred := range step.Predicates {
			matched, err = applyVMPredicate(evalFn, ec, matched, pred)
			if err != nil {
				return nil, err
			}
		}
		allFiltered = append(allFiltered, matched...)
	}
	return ixpath.DeduplicateNodes(allFiltered, ec.docOrder, ec.maxNodes)
}

func applyVMPredicate(evalFn exprEvaluator, ec *evalContext, nodes []helium.Node, pred Expr) ([]helium.Node, error) {
	switch p := pred.(type) {
	case vmPositionPredicateExpr:
		return applyVMPositionPredicate(nodes, p), nil
	case vmAttributeExistsPredicateExpr:
		return applyVMAttributeExistsPredicate(ec, nodes, p), nil
	case vmAttributeEqualsStringPredicateExpr:
		return applyVMAttributeEqualsStringPredicate(evalFn, ec, nodes, p)
	default:
		return applyPredicate(evalFn, ec, nodes, pred)
	}
}

func applyVMPositionPredicate(nodes []helium.Node, pred vmPositionPredicateExpr) []helium.Node {
	if pred.Position <= 0 || pred.Position > len(nodes) {
		return nil
	}
	return nodes[pred.Position-1 : pred.Position]
}

func applyVMAttributeExistsPredicate(ec *evalContext, nodes []helium.Node, pred vmAttributeExistsPredicateExpr) []helium.Node {
	var result []helium.Node
	for _, n := range nodes {
		if nodeHasMatchingAttribute(ec, n, pred.NodeTest) {
			result = append(result, n)
		}
	}
	return result
}

func applyVMAttributeEqualsStringPredicate(evalFn exprEvaluator, ec *evalContext, nodes []helium.Node, pred vmAttributeEqualsStringPredicateExpr) ([]helium.Node, error) {
	size := len(nodes)
	var result []helium.Node
	for i, n := range nodes {
		match, ok := vmAttributeEqualsStringPredicateMatches(ec, n, pred)
		if !ok {
			frame := ec.pushNodeContext(n, i+1, size)
			r, err := evalFn(ec, pred.Fallback)
			ec.restoreContext(frame)
			if err != nil {
				return nil, err
			}
			predMatch, err := predicateTrue(r, i+1)
			if err != nil {
				return nil, err
			}
			match = predMatch
		}
		if match {
			result = append(result, n)
		}
	}
	return result, nil
}

func vmAttributeEqualsStringPredicateMatches(ec *evalContext, node helium.Node, pred vmAttributeEqualsStringPredicateExpr) (bool, bool) {
	elem, ok := node.(*helium.Element)
	if !ok {
		return false, true
	}

	mustFallback := false
	matched := false
	elem.ForEachAttribute(func(attr *helium.Attribute) bool {
		if !matchNodeTest(pred.NodeTest, attr, AxisAttribute, ec) {
			return true
		}
		if nodeTypeAnnotation(attr, ec) != "" {
			mustFallback = true
			return false
		}
		if attr.Value() == pred.Value {
			matched = true
			return false
		}
		return true
	})
	if mustFallback {
		return false, false
	}
	return matched, true
}

func nodeHasMatchingAttribute(ec *evalContext, node helium.Node, test NodeTest) bool {
	elem, ok := node.(*helium.Element)
	if !ok {
		return false
	}
	found := false
	elem.ForEachAttribute(func(attr *helium.Attribute) bool {
		if !matchNodeTest(test, attr, AxisAttribute, ec) {
			return true
		}
		found = true
		return false
	})
	return found
}

func evalVMStepNoPredicates(ec *evalContext, nodes []helium.Node, step vmLocationStep) ([]helium.Node, error) {
	var next []helium.Node
	for _, n := range nodes {
		var traversed int
		var err error
		next, traversed, err = appendAxisNodeMatches(next, ec, n, step.Axis, step.NodeTest)
		if err != nil {
			return nil, err
		}
		if err := ec.countOps(traversed); err != nil {
			return nil, err
		}
	}
	return ixpath.DeduplicateNodes(next, ec.docOrder, ec.maxNodes)
}

func appendAxisNodeMatches(dst []helium.Node, ec *evalContext, node helium.Node, axis AxisType, nodeTest NodeTest) ([]helium.Node, int, error) {
	switch axis {
	case AxisChild:
		if _, ok := node.(*helium.Attribute); ok {
			return dst, 0, nil
		}
		traversed := 0
		for child := range helium.Children(node) {
			if !ixpath.IsXDMChild(child) {
				continue
			}
			traversed++
			if matchNodeTest(nodeTest, child, axis, ec) {
				dst = append(dst, child)
			}
		}
		return dst, traversed, nil
	case AxisAttribute:
		elem, ok := node.(*helium.Element)
		if !ok {
			return dst, 0, nil
		}
		traversed := 0
		elem.ForEachAttribute(func(attr *helium.Attribute) bool {
			traversed++
			if matchNodeTest(nodeTest, attr, axis, ec) {
				dst = append(dst, attr)
			}
			return true
		})
		return dst, traversed, nil
	case AxisSelf:
		if matchNodeTest(nodeTest, node, axis, ec) {
			dst = append(dst, node)
		}
		return dst, 1, nil
	case AxisParent:
		parent := node.Parent()
		if parent == nil {
			return dst, 0, nil
		}
		if matchNodeTest(nodeTest, parent, axis, ec) {
			dst = append(dst, parent)
		}
		return dst, 1, nil
	default:
		candidates, err := ixpath.TraverseAxis(axis, node, ec.maxNodes)
		if err != nil {
			return nil, 0, err
		}
		for _, candidate := range candidates {
			if matchNodeTest(nodeTest, candidate, axis, ec) {
				dst = append(dst, candidate)
			}
		}
		return dst, len(candidates), nil
	}
}

func matchNodeTest(nt NodeTest, n helium.Node, axis AxisType, ec *evalContext) bool {
	switch test := nt.(type) {
	case NameTest:
		return matchNameTest(test, n, axis, ec)
	case TypeTest:
		return matchTypeTest(test, n)
	case PITest:
		if n.Type() != helium.ProcessingInstructionNode {
			return false
		}
		if test.Target == "" {
			return true
		}
		return n.Name() == test.Target
	case ElementTest:
		if n.Type() != helium.ElementNode {
			return false
		}
		if test.Name != "" && test.Name != "*" {
			if !matchElementOrAttributeName(test.Name, n, ec) {
				return false
			}
		}
		if test.TypeName != "" {
			ann := nodeTypeAnnotation(n, ec)
			if ann == "" {
				ann = TypeUntyped // elements default to xs:untyped
			}
			target := resolveTestTypeName(test.TypeName, ec)
			if !isSubtypeOf(ann, target) {
				if ec == nil || ec.schemaDeclarations == nil || !ec.schemaDeclarations.IsSubtypeOf(ann, target) {
					return false
				}
			}
		}
		return true
	case AttributeTest:
		if _, ok := n.(*helium.Attribute); !ok {
			return false
		}
		if test.Name != "" && test.Name != "*" {
			if !matchElementOrAttributeName(test.Name, n, ec) {
				return false
			}
		}
		if test.TypeName != "" {
			ann := nodeTypeAnnotation(n, ec)
			if ann == "" {
				ann = TypeUntypedAtomic
			}
			target := resolveTestTypeName(test.TypeName, ec)
			if !isSubtypeOf(ann, target) {
				if ec == nil || ec.schemaDeclarations == nil || !ec.schemaDeclarations.IsSubtypeOf(ann, target) {
					return false
				}
			}
		}
		return true
	case DocumentTest:
		if n.Type() != helium.DocumentNode {
			return false
		}
		if test.Inner != nil {
			// document-node(E) matches when the document has exactly one element
			// child, no text node children, and that element matches E.
			var elemCount int
			var matchedInner bool
			for c := range helium.Children(n) {
				switch c.Type() {
				case helium.ElementNode:
					elemCount++
					if elemCount > 1 {
						return false
					}
					if matchNodeTest(test.Inner, c, AxisChild, ec) {
						matchedInner = true
					}
				case helium.TextNode:
					return false
				}
			}
			return elemCount == 1 && matchedInner
		}
		return true
	case SchemaElementTest:
		if n.Type() != helium.ElementNode {
			return false
		}
		if ec == nil || ec.schemaDeclarations == nil {
			// Without schema, match by name only.
			if test.Name == "" || test.Name == "*" {
				return true
			}
			_, local := splitQName(test.Name)
			return ixpath.LocalNameOf(n) == local
		}
		local, ns := resolveSchemaTestName(test.Name, ec)
		if ixpath.LocalNameOf(n) != local || ixpath.NodeNamespaceURI(n) != ns {
			return false
		}
		typeName, found := ec.schemaDeclarations.LookupSchemaElement(local, ns)
		if !found {
			return false
		}
		ann := nodeTypeAnnotation(n, ec)
		if ann == "" {
			ann = TypeUntyped
		}
		if isSubtypeOf(ann, typeName) {
			return true
		}
		return ec.schemaDeclarations.IsSubtypeOf(ann, typeName)
	case NamespaceNodeTest:
		return n.Type() == helium.NamespaceNode
	case AnyItemTest:
		return true
	}
	return false
}

// matchElementOrAttributeName matches an element/attribute name from an
// ElementTest or AttributeTest against a node. Handles URIQualifiedNames
// (Q{uri}local), prefixed names (prefix:local), and plain local names.
func matchElementOrAttributeName(name string, n helium.Node, ec *evalContext) bool {
	if strings.HasPrefix(name, "Q{") {
		if idx := strings.Index(name, "}"); idx >= 0 {
			uri := name[2:idx]
			local := name[idx+1:]
			return ixpath.LocalNameOf(n) == local && ixpath.NodeNamespaceURI(n) == uri
		}
	}
	prefix, local := splitQName(name)
	if ixpath.LocalNameOf(n) != local {
		return false
	}
	if prefix != "" {
		return matchPrefix(prefix, n, ec)
	}
	return true
}

func matchNameTest(test NameTest, n helium.Node, axis AxisType, ec *evalContext) bool {
	switch axis {
	case AxisAttribute:
		if _, ok := n.(*helium.Attribute); !ok {
			return false
		}
	case AxisNamespace:
		if n.Type() != helium.NamespaceNode {
			return false
		}
		if test.Local == "*" {
			return true
		}
		// Namespace nodes have simple NCName prefixes, not QNames.
		// A name test with a prefix (e.g. namespace::b:a) cannot
		// match any namespace node.
		if test.Prefix != "" {
			return false
		}
		return n.Name() == test.Local
	default:
		if n.Type() != helium.ElementNode {
			return false
		}
	}

	if test.Local == "*" {
		if test.URI != "" {
			return ixpath.NodeNamespaceURI(n) == test.URI
		}
		if test.Prefix == "" {
			return true
		}
		return matchPrefix(test.Prefix, n, ec)
	}

	if ixpath.LocalNameOf(n) != test.Local {
		return false
	}

	if test.URI != "" {
		return ixpath.NodeNamespaceURI(n) == test.URI
	}
	if test.Prefix == "*" {
		// *:local matches any namespace
		return true
	}
	if test.Prefix != "" {
		return matchPrefix(test.Prefix, n, ec)
	}
	// Check for default element namespace (xpath-default-namespace).
	// Only applies to element axis, not attributes.
	// Per XPath 3.1 §3.3.2.1: when default element namespace is absent,
	// unprefixed names match only no-namespace elements.
	if axis == AxisAttribute {
		// Per XPath 3.1 §3.3.2.1: an unprefixed attribute name test
		// matches only attributes with no namespace URI.
		return ixpath.NodeNamespaceURI(n) == ""
	}
	if ec.namespaces != nil {
		return ixpath.NodeNamespaceURI(n) == ec.namespaces[""]
	}
	// No namespace context at all: permissive match (any namespace).
	return true
}

func matchPrefix(prefix string, n helium.Node, ec *evalContext) bool {
	if ec.namespaces != nil {
		if uri, ok := ec.namespaces[prefix]; ok {
			return ixpath.NodeNamespaceURI(n) == uri
		}
	}
	// The xml prefix is always bound per the XML Namespaces spec.
	if prefix == "xml" {
		return ixpath.NodeNamespaceURI(n) == helium.XMLNamespace
	}
	// Check built-in XPath prefixes (fn, xs, math, map, array, err).
	if uri, ok := defaultPrefixNS[prefix]; ok {
		return ixpath.NodeNamespaceURI(n) == uri
	}
	// Per XPath spec, prefix resolution must come from the static/evaluation
	// namespace bindings, not the document's lexical prefixes. If the prefix
	// is not declared in the namespace context, it cannot match.
	return false
}

func matchTypeTest(test TypeTest, n helium.Node) bool {
	switch test.Kind {
	case NodeKindNode:
		return true
	case NodeKindText:
		return n.Type() == helium.TextNode || n.Type() == helium.CDATASectionNode
	case NodeKindComment:
		return n.Type() == helium.CommentNode
	case NodeKindProcessingInstruction:
		return n.Type() == helium.ProcessingInstructionNode
	}
	return false
}

func applyPredicate(evalFn exprEvaluator, ec *evalContext, nodes []helium.Node, pred Expr) ([]helium.Node, error) {
	if err := ec.countOps(len(nodes)); err != nil {
		return nil, err
	}
	size := len(nodes)
	var result []helium.Node
	for i, n := range nodes {
		frame := ec.pushNodeContext(n, i+1, size)
		r, err := evalFn(ec, pred)
		ec.restoreContext(frame)
		if err != nil {
			return nil, err
		}
		match, err := predicateTrue(r, i+1)
		if err != nil {
			return nil, err
		}
		if match {
			result = append(result, n)
		}
	}
	return result, nil
}

// nodeTypeAnnotation returns the type annotation for a node from the
// evalContext's type annotation map.
func nodeTypeAnnotation(n helium.Node, ec *evalContext) string {
	if ec == nil || ec.typeAnnotations == nil {
		return ""
	}
	return ec.typeAnnotations[n]
}

// resolveSchemaTestName resolves a name from a SchemaElementTest or
// SchemaAttributeTest. Handles Q{uri}local (EQName) and prefix:local forms.
func resolveSchemaTestName(name string, ec *evalContext) (local, ns string) {
	if strings.HasPrefix(name, "Q{") {
		if idx := strings.Index(name, "}"); idx >= 0 {
			return name[idx+1:], name[2:idx]
		}
	}
	prefix, loc := splitQName(name)
	if prefix != "" && ec != nil && ec.namespaces != nil {
		return loc, ec.namespaces[prefix]
	}
	if prefix == "" && ec != nil && ec.namespaces != nil {
		return loc, ec.namespaces[""]
	}
	return loc, ""
}

// resolveTestTypeName normalizes a type name from an ElementTest/AttributeTest
// to the canonical annotation format:
//   - "xs:localName" for types in the XSD namespace
//   - "Q{ns}localName" for types in any other namespace
//   - the raw name for names without a prefix (treated as no-namespace)
func resolveTestTypeName(raw string, ec *evalContext) string {
	if strings.HasPrefix(raw, "xs:") {
		return raw
	}
	if idx := strings.IndexByte(raw, ':'); idx >= 0 {
		prefix := raw[:idx]
		local := raw[idx+1:]
		if prefix == "xsd" {
			return "xs:" + local
		}
		if ec != nil && ec.namespaces != nil {
			if uri, ok := ec.namespaces[prefix]; ok {
				if uri == lexicon.NamespaceXSD {
					return "xs:" + local
				}
				return QAnnotation(uri, local)
			}
		}
	}
	// Unprefixed type names: resolve using the default element namespace
	// (same as unprefixed element names in XPath). This handles types like
	// "addressType" in element(address, addressType) when xpath-default-namespace
	// provides the namespace.
	if ec != nil && ec.namespaces != nil {
		if defNS, ok := ec.namespaces[""]; ok && defNS != "" {
			return QAnnotation(defNS, raw)
		}
	}
	// No namespace: use Q{} annotation form for consistency with
	// xsdTypeNameFromDef which produces Q{}local for no-namespace types.
	return "Q{}" + raw
}

// predicateTrue evaluates a predicate result per XPath spec:
// numeric → compare to position, otherwise → EBV.
func predicateTrue(r Sequence, position int) (bool, error) {
	if seqLen(r) == 1 {
		if av, ok := r.Get(0).(AtomicValue); ok && av.IsNumeric() {
			return av.ToFloat64() == float64(position), nil
		}
	}
	return EBV(r)
}

func vmPredicatePosition(expr Expr) (int, bool) {
	switch e := expr.(type) {
	case LiteralExpr:
		return vmPositionFromLiteral(e)
	case *LiteralExpr:
		if e == nil {
			return 0, false
		}
		return vmPositionFromLiteral(*e)
	case BinaryExpr:
		return vmPredicatePositionFromBinary(e)
	case *BinaryExpr:
		if e == nil {
			return 0, false
		}
		return vmPredicatePositionFromBinary(*e)
	default:
		return 0, false
	}
}

func vmPredicatePositionFromBinary(expr BinaryExpr) (int, bool) {
	if expr.Op != TokenEquals && expr.Op != TokenEq {
		return 0, false
	}
	if positionCall(expr.Left) {
		return vmPositionFromExpr(expr.Right)
	}
	if positionCall(expr.Right) {
		return vmPositionFromExpr(expr.Left)
	}
	return 0, false
}

func vmPositionFromExpr(expr Expr) (int, bool) {
	switch e := expr.(type) {
	case LiteralExpr:
		return vmPositionFromLiteral(e)
	case *LiteralExpr:
		if e == nil {
			return 0, false
		}
		return vmPositionFromLiteral(*e)
	default:
		return 0, false
	}
}

func vmPositionFromLiteral(expr LiteralExpr) (int, bool) {
	switch v := expr.Value.(type) {
	case *big.Int:
		if !v.IsInt64() {
			return 0, false
		}
		n := v.Int64()
		if n <= 0 || n > math.MaxInt {
			return 0, false
		}
		return int(n), true
	case float64:
		if v <= 0 || v != math.Trunc(v) || v > math.MaxInt {
			return 0, false
		}
		return int(v), true
	default:
		return 0, false
	}
}

func positionCall(expr Expr) bool {
	switch e := expr.(type) {
	case FunctionCall:
		return len(e.Args) == 0 && e.Name == "position" && (e.Prefix == "" || e.Prefix == "fn")
	case *FunctionCall:
		if e == nil {
			return false
		}
		return len(e.Args) == 0 && e.Name == "position" && (e.Prefix == "" || e.Prefix == "fn")
	default:
		return false
	}
}

func vmPredicateAttributeExists(expr Expr) (NodeTest, bool) {
	return vmSingleRelativeAttributePath(expr)
}

func vmPredicateAttributeEqualsString(expr Expr) (NodeTest, string, bool) {
	switch e := expr.(type) {
	case BinaryExpr:
		return vmPredicateAttributeEqualsStringBinary(e)
	case *BinaryExpr:
		if e == nil {
			return nil, "", false
		}
		return vmPredicateAttributeEqualsStringBinary(*e)
	default:
		return nil, "", false
	}
}

func vmPredicateAttributeEqualsStringBinary(expr BinaryExpr) (NodeTest, string, bool) {
	if expr.Op != TokenEquals && expr.Op != TokenEq {
		return nil, "", false
	}
	if test, ok := vmSingleRelativeAttributePath(expr.Left); ok {
		if value, ok := vmStringLiteralValue(expr.Right); ok {
			return test, value, true
		}
	}
	if test, ok := vmSingleRelativeAttributePath(expr.Right); ok {
		if value, ok := vmStringLiteralValue(expr.Left); ok {
			return test, value, true
		}
	}
	return nil, "", false
}

func vmSingleRelativeAttributePath(expr Expr) (NodeTest, bool) {
	switch e := expr.(type) {
	case LocationPath:
		return vmAttributeNodeTestFromLocationPath(e.Absolute, e.Steps)
	case *LocationPath:
		if e == nil {
			return nil, false
		}
		return vmAttributeNodeTestFromLocationPath(e.Absolute, e.Steps)
	case vmLocationPathExpr:
		return vmAttributeNodeTestFromVMLocationPath(e.Absolute, e.Steps)
	case *vmLocationPathExpr:
		if e == nil {
			return nil, false
		}
		return vmAttributeNodeTestFromVMLocationPath(e.Absolute, e.Steps)
	default:
		return nil, false
	}
}

func vmAttributeNodeTestFromLocationPath(absolute bool, steps []Step) (NodeTest, bool) {
	if absolute || len(steps) != 1 {
		return nil, false
	}
	step := steps[0]
	if step.Axis != AxisAttribute || len(step.Predicates) != 0 {
		return nil, false
	}
	return step.NodeTest, true
}

func vmAttributeNodeTestFromVMLocationPath(absolute bool, steps []vmLocationStep) (NodeTest, bool) {
	if absolute || len(steps) != 1 {
		return nil, false
	}
	step := steps[0]
	if step.Axis != AxisAttribute || len(step.Predicates) != 0 {
		return nil, false
	}
	return step.NodeTest, true
}

func vmStringLiteralValue(expr Expr) (string, bool) {
	switch e := expr.(type) {
	case LiteralExpr:
		value, ok := e.Value.(string)
		return value, ok
	case *LiteralExpr:
		if e == nil {
			return "", false
		}
		value, ok := e.Value.(string)
		return value, ok
	default:
		return "", false
	}
}
